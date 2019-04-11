// Modified for MassNet
// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package chainindexer

import (
	"container/heap"
	"math"
	"runtime"
	"sync"
	"sync/atomic"

	"massnet.org/mass-wallet/database"
	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/txscript"
	"massnet.org/mass-wallet/wire"

	"github.com/btcsuite/golangcrypto/ripemd160"
	"massnet.org/mass-wallet/logging"
)

type indexState int

const (
	indexCatchUp indexState = iota
	indexMaintain
)

var (
	numCatchUpWorkers = runtime.NumCPU() * 3
	zeroHash          = &wire.Hash{}
)

// indexBlockMsg packages a request to have the addresses of a block indexed.
type indexBlockMsg struct {
	blk  *massutil.Block
	done chan struct{}
}

// writeIndexReq represents a request to have a completed address index
// committed to the database.
type writeIndexReq struct {
	blk       *massutil.Block
	addrIndex database.BlockAddrIndex
}

type Server interface {
	Stop() error
}

type BlockLogger interface {
	LogBlockHeight(blk *massutil.Block)
}

type Chain interface {
	GetBlockByHeight(height uint64) (*massutil.Block, error)
	NewestSha() (sha *wire.Hash, height uint64, err error)
}

// AddrIndexer provides a concurrent service for indexing the transactions of
// target blocks based on the addresses involved in the transaction.
type AddrIndexer struct {
	started         int32
	shutdown        int32
	state           indexState
	server          Server
	BlockLogger     BlockLogger
	Chain           Chain
	quit            chan struct{}
	wg              sync.WaitGroup
	addrIndexJobs   chan *indexBlockMsg
	writeRequests   chan *writeIndexReq
	db              database.Db
	currentIndexTip int32
	chainTip        int32
	sync.Mutex
}

// newAddrIndexer creates a new block address indexer.
// Use Start to begin processing incoming index jobs.
func NewAddrIndexer(db database.Db, server Server) (*AddrIndexer, error) {
	_, chainHeight, err := db.NewestSha()
	if err != nil {
		return nil, err
	}

	_, lastIndexedHeight, err := db.FetchAddrIndexTip()
	if err != nil && err != database.ErrAddrIndexDoesNotExist {
		return nil, err
	}

	var state indexState
	if chainHeight == lastIndexedHeight {
		state = indexMaintain
	} else {
		state = indexCatchUp
	}

	ai := &AddrIndexer{
		db:              db,
		quit:            make(chan struct{}),
		state:           state,
		server:          server,
		BlockLogger:     NewBlockProgressLogger("process"),
		addrIndexJobs:   make(chan *indexBlockMsg),
		writeRequests:   make(chan *writeIndexReq, numCatchUpWorkers),
		currentIndexTip: lastIndexedHeight,
		chainTip:        chainHeight,
	}
	return ai, nil
}

// Start begins processing of incoming indexing jobs.
func (a *AddrIndexer) Start() {
	if atomic.AddInt32(&a.started, 1) != 1 {
		return
	}
	logging.CPrint(logging.TRACE, "Starting address indexer", logging.LogFormat{})
	a.wg.Add(2)
	go a.indexManager()
	go a.indexWriter()
}

// Stop gracefully shuts down the address indexer by stopping all ongoing
// worker goroutines, waiting for them to finish their current task.
func (a *AddrIndexer) Stop() error {
	if atomic.AddInt32(&a.shutdown, 1) != 1 {
		logging.CPrint(logging.WARN, "Address indexer is already in the process of shutting down", logging.LogFormat{})
		return nil
	}
	logging.CPrint(logging.INFO, "Address indexer shutting down", logging.LogFormat{})
	close(a.quit)
	a.wg.Wait()
	return nil
}

// IsCaughtUp returns a bool representing if the address indexer has
// caught up with the best height on the main chain.
func (a *AddrIndexer) IsCaughtUp() bool {
	a.Lock()
	defer a.Unlock()
	return a.state == indexMaintain
}

// indexManager creates, and oversees worker index goroutines.
// indexManager is the main goroutine for the addresses indexer.
// It creates, and oversees worker goroutines to index incoming blocks, with
// the exact behavior depending on the current index state
// (catch up, vs maintain). Completion of catch-up mode is always proceeded by
// a gracefull transition into "maintain" mode.
// NOTE: Must be run as a goroutine.
func (a *AddrIndexer) indexManager() {
	if a.state == indexCatchUp {
		logging.CPrint(logging.INFO, "Building up address index", logging.LogFormat{"start height": a.currentIndexTip + 1, "target height": a.chainTip})
		runningWorkers := make([]chan struct{}, 0, numCatchUpWorkers)
		shutdownWorkers := func() {
			for _, quit := range runningWorkers {
				close(quit)
			}
		}
		criticalShutdown := func() {
			shutdownWorkers()
		}

		var workerWg sync.WaitGroup
		catchUpChan := make(chan *indexBlockMsg)
		for i := 0; i < numCatchUpWorkers; i++ {
			quit := make(chan struct{})
			runningWorkers = append(runningWorkers, quit)
			workerWg.Add(1)
			go a.indexCatchUpWorker(catchUpChan, &workerWg, quit)
		}

		lastBlockIdxHeight := a.currentIndexTip + 1
		for lastBlockIdxHeight <= a.chainTip {
			targetBlock, err := a.Chain.GetBlockByHeight(uint64(lastBlockIdxHeight))
			if err != nil {
				logging.CPrint(logging.ERROR, "Unable to look up the next target block", logging.LogFormat{"height": lastBlockIdxHeight, "error": err})
				criticalShutdown()
				goto fin
			}

			indexJob := &indexBlockMsg{blk: targetBlock}
			select {
			case catchUpChan <- indexJob:
				lastBlockIdxHeight++
			case <-a.quit:
				shutdownWorkers()
				goto fin
			}
			_, chainTip, err := a.Chain.NewestSha()
			if err != nil {
				logging.CPrint(logging.ERROR, "Unable to get latest block height", logging.LogFormat{"error": err})
				criticalShutdown()
				goto fin
			}
			a.chainTip = int32(chainTip)
		}

		a.Lock()
		a.state = indexMaintain
		a.Unlock()

		shutdownWorkers()
		workerWg.Wait()
	}

	logging.CPrint(logging.INFO, "Address indexer has caught up to best height, entering maintenance mode", logging.LogFormat{})

	for {
		select {
		case indexJob := <-a.addrIndexJobs:
			addrIndex, err := a.indexBlockAddrs(indexJob.blk)
			if err != nil {
				logging.CPrint(logging.ERROR,
					"Unable to index transactions of block",
					logging.LogFormat{
						"block hash": indexJob.blk.Hash().String(),
						"height":     indexJob.blk.MsgBlock().Header.Height,
						"error":      err})
				a.stopServer()
				goto fin
			}
			a.writeRequests <- &writeIndexReq{blk: indexJob.blk,
				addrIndex: addrIndex}
		case <-a.quit:
			goto fin
		}
	}
fin:
	a.wg.Done()
}

// UpdateAddressIndex asynchronously queues a newly solved block to have its
// transactions indexed by address.
func (a *AddrIndexer) UpdateAddressIndex(block *massutil.Block) {
	go func() {
		job := &indexBlockMsg{blk: block}
		a.addrIndexJobs <- job
	}()
}

// pendingIndexWrites writes is a priority queue which is used to ensure the
// wallet index of the block height N+1 is written when our wallet tip is at
// height N. This ordering is necessary to maintain index consistency in face
// of our concurrent workers, which may not necessarily finish in the order the
// jobs are handed out.
type pendingWriteQueue []*writeIndexReq

// Len returns the number of items in the priority queue. It is part of the
// heap.Interface implementation.
func (pq pendingWriteQueue) Len() int { return len(pq) }

// Less returns whether the item in the priority queue with index i should sort
// before the item with index j. It is part of the heap.Interface implementation.
func (pq pendingWriteQueue) Less(i, j int) bool {
	return pq[i].blk.Height() < pq[j].blk.Height()
}

// Swap swaps the items at the passed indices in the priority queue. It is
// part of the heap.Interface implementation.
func (pq pendingWriteQueue) Swap(i, j int) { pq[i], pq[j] = pq[j], pq[i] }

// Push pushes the passed item onto the priority queue. It is part of the
// heap.Interface implementation.
func (pq *pendingWriteQueue) Push(x interface{}) {
	*pq = append(*pq, x.(*writeIndexReq))
}

// Pop removes the highest priority item (according to Less) from the priority
// queue and returns it.  It is part of the heap.Interface implementation.
func (pq *pendingWriteQueue) Pop() interface{} {
	n := len(*pq)
	item := (*pq)[n-1]
	(*pq)[n-1] = nil
	*pq = (*pq)[0 : n-1]
	return item
}

// indexWriter commits the populated address indexes created by the
// catch up workers to the database. Since we have concurrent workers, the writer
// ensures indexes are written in ascending order to avoid a possible gap in the
// address index triggered by an unexpected shutdown.
// NOTE: Must be run as a goroutine
func (a *AddrIndexer) indexWriter() {
	var pendingWrites pendingWriteQueue
	minHeightWrite := make(chan *writeIndexReq)
	workerQuit := make(chan struct{})
	writeFinished := make(chan struct{}, 1)

	go func() {
		for {
		top:
			select {
			case incomingWrite := <-a.writeRequests:
				heap.Push(&pendingWrites, incomingWrite)

				writeReq := heap.Pop(&pendingWrites).(*writeIndexReq)
				_, addrTip, _ := a.db.FetchAddrIndexTip()
				for writeReq.blk.Height() == (addrTip+1) ||
					writeReq.blk.Height() <= addrTip {
					minHeightWrite <- writeReq

					<-writeFinished

					if pendingWrites.Len() == 0 {
						break top
					}

					writeReq = heap.Pop(&pendingWrites).(*writeIndexReq)
					_, addrTip, _ = a.db.FetchAddrIndexTip()
				}

				heap.Push(&pendingWrites, writeReq)
			case <-workerQuit:
				return
			}
		}
	}()

out:
	for {
		select {
		case nextWrite := <-minHeightWrite:
			sha := nextWrite.blk.Hash()
			height := nextWrite.blk.Height()
			err := a.db.UpdateAddrIndexForBlock(sha, height,
				nextWrite.addrIndex)
			if err != nil {
				logging.CPrint(logging.ERROR, "Unable to write index for block", logging.LogFormat{"block hash": sha.String(), "height": height})
				a.stopServer()
				break out
			}
			writeFinished <- struct{}{}
			a.logBlockHeight(nextWrite.blk)
		case <-a.quit:
			break out
		}

	}
	close(workerQuit)
	a.wg.Done()
}

// indexCatchUpWorker indexes the transactions of previously validated and
// stored blocks.
// NOTE: Must be run as a goroutine
func (a *AddrIndexer) indexCatchUpWorker(workChan chan *indexBlockMsg,
	wg *sync.WaitGroup, quit chan struct{}) {
out:
	for {
		select {
		case indexJob := <-workChan:
			addrIndex, err := a.indexBlockAddrs(indexJob.blk)
			if err != nil {
				logging.CPrint(logging.ERROR, "Unable to index transactions of block",
					logging.LogFormat{"block hash": indexJob.blk.Hash().String(), "height": indexJob.blk.MsgBlock().Header.Height, "error": err})
				a.stopServer()
				break out
			}
			a.writeRequests <- &writeIndexReq{blk: indexJob.blk,
				addrIndex: addrIndex}
		case <-quit:
			break out
		}
	}
	wg.Done()
}

// indexScriptPubKey indexes all data pushes greater than 8 bytes within the
// passed SPK. Our "address" index is actually a hash160 index, where in the
// ideal case the data push is the hash160 of a witness script (P2WSH).
func indexScriptPubKey(addrIndex database.BlockAddrIndex, scriptPubKey []byte,
	locInBlock *wire.TxLoc, index uint32) error {
	dataPushes, err := txscript.PushedData(scriptPubKey)
	if err != nil {
		logging.CPrint(logging.TRACE, "Couldn't get pushes", logging.LogFormat{"error": err})
		return err
	}

	for _, data := range dataPushes {
		if len(data) < 8 {
			continue
		}

		var indexKey [ripemd160.Size]byte
		if len(data) <= 20 {
			copy(indexKey[:], data)
		} else {
			copy(indexKey[:], massutil.Hash160(data))
		}

		addrIndex[indexKey] = append(addrIndex[indexKey], &database.AddrIndexOutPoint{TxLoc: locInBlock, Index: index})
	}
	return nil
}

// indexBlockAddrs returns a populated index of the all the transactions in the
// passed block based on the addresses involved in each transaction.
func (a *AddrIndexer) indexBlockAddrs(blk *massutil.Block) (database.BlockAddrIndex, error) {
	addrIndex := make(database.BlockAddrIndex)
	txLocs, err := blk.TxLoc()
	if err != nil {
		return nil, err
	}

	for txIdx, tx := range blk.Transactions() {
		locInBlock := &txLocs[txIdx]

		for index, txOut := range tx.MsgTx().TxOut {
			indexScriptPubKey(addrIndex, txOut.PkScript, locInBlock, uint32(index))
		}
	}
	return addrIndex, nil
}

func (a *AddrIndexer) stopServer() {
	a.server.Stop()
}

func (a *AddrIndexer) logBlockHeight(blk *massutil.Block) {
	a.BlockLogger.LogBlockHeight(blk)
}

func IsCoinBase(tx *massutil.Tx) bool {
	prevOut := &tx.MsgTx().TxIn[0].PreviousOutPoint
	if prevOut.Index != math.MaxUint32 || !prevOut.Hash.IsEqual(zeroHash) {
		return false
	}

	return true
}
