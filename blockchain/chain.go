// Modified for MassNet
// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"container/list"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"time"

	"github.com/massnetorg/MassNet-wallet/btcec"
	"github.com/massnetorg/MassNet-wallet/config"
	"github.com/massnetorg/MassNet-wallet/logging"

	"github.com/massnetorg/MassNet-wallet/chainindexer"
	"github.com/massnetorg/MassNet-wallet/database"
	"github.com/massnetorg/MassNet-wallet/massutil"
	"github.com/massnetorg/MassNet-wallet/txscript"
	"github.com/massnetorg/MassNet-wallet/wire"
)

const (
	maxOrphanBlocks = 100
	minMemoryNodes  = 2000
)

// ErrIndexAlreadyInitialized describes an error that indicates the block index
// is already initialized.
var ErrIndexAlreadyInitialized = errors.New("the block index can only be " +
	"initialized before it has been modified")

type blockNode struct {
	inMainChain bool
	parent      *blockNode
	children    []*blockNode

	version    uint64
	hash       *wire.Hash
	parentHash *wire.Hash
	height     int32
	timestamp  time.Time
	target     *big.Int
	capSum     *big.Int
}

// Ancestor returns the ancestor block node at the provided height by following
// the chain backwards from this node.  The returned block will be nil when a
// height is requested that is after the height of the passed node or is less
// than zero.
//
// This function is safe for concurrent access.
func (node *blockNode) Ancestor(height int32) *blockNode {
	if height < 0 || height > node.height {
		return nil
	}

	n := node
	for ; n != nil && n.height != height; n = n.parent {
		// Intentionally left blank
	}

	return n
}

// newBlockNode returns a new block node for the given block header.  It is
// completely disconnected from the chain and the capSum value is just the work
// for the passed block.  The work sum is updated accordingly when the node is
// inserted into a chain.
func newBlockNode(blockHeader *wire.BlockHeader, blockSha *wire.Hash, height int32) *blockNode {
	// Make a copy of the hash so the node doesn't keep a reference to part
	// of the full block/block header preventing it from being garbage
	// collected.
	prevHash := blockHeader.Previous
	banList := make([]*btcec.PublicKey, 0, len(blockHeader.BanList))
	copy(banList, blockHeader.BanList)
	node := blockNode{
		hash:       blockSha,
		parentHash: &prevHash,
		capSum:     new(big.Int).Set(blockHeader.Target),
		height:     height,
		version:    blockHeader.Version,
		target:     blockHeader.Target,
		timestamp:  blockHeader.Timestamp,
	}
	return &node
}

// addChildrenWork adds the passed work amount to all children all the way
// down the chain.  It is used primarily to allow a new node to be dynamically
// inserted from the database into the memory chain prior to nodes we already
// have and update their work values accordingly.
func addChildrenWork(node *blockNode, work *big.Int) {
	for _, childNode := range node.children {
		childNode.capSum.Add(childNode.capSum, work)
		addChildrenWork(childNode, work)
	}
}

// removeChildNode deletes node from the provided slice of child block
// nodes.  It ensures the final pointer reference is set to nil to prevent
// potential memory leaks.  The original slice is returned unmodified if node
// is invalid or not in the slice.
func removeChildNode(children []*blockNode, node *blockNode) []*blockNode {
	if node == nil {
		return children
	}

	for i := 0; i < len(children); i++ {
		if children[i].hash.IsEqual(node.hash) {
			copy(children[i:], children[i+1:])
			children[len(children)-1] = nil
			return children[:len(children)-1]
		}
	}
	return children
}

type chainInfo struct {
	genesisBlock *massutil.Block
	genesisHash  wire.Hash
	genesisTime  time.Time
	chainID      wire.Hash
}

type BlockChain struct {
	db                  database.Db
	checkpointsByHeight map[int32]*config.Checkpoint
	notifications       NotificationCallback
	info                *chainInfo
	root                *blockNode
	bestChain           *blockNode
	index               map[wire.Hash]*blockNode
	depNodes            map[wire.Hash][]*blockNode
	blockCache          map[wire.Hash]*massutil.Block
	noVerify            bool
	noCheckpoints       bool
	nextCheckpoint      *config.Checkpoint
	checkpointBlock     *massutil.Block
	sigCache            *txscript.SigCache
	hashCache           *txscript.HashCache

	orphanBlockPool *OrphanBlockPool
	TxPool          *TxPool
	addrIndexer     *chainindexer.AddrIndexer

	nodeLock   sync.RWMutex
	accessLock sync.Mutex
}

// DisableVerify provides a mechanism to disable transaction script validation
// which you DO NOT want to do in production as it could allow double spends
// and othe undesirable things.  It is provided only for debug purposes since
// script validation is extremely intensive and when debugging it is sometimes
// nice to quickly get the chain.
func (b *BlockChain) DisableVerify(disable bool) {
	b.noVerify = disable
}

// HaveBlock returns whether or not the chain instance has the block represented
// by the passed hash.  This includes checking the various places a block can
// be like part of the main chain, on a side chain, or in the orphan pool.
//
// This function is NOT safe for concurrent access.
func (b *BlockChain) HaveBlock(hash *wire.Hash) (bool, error) {
	exists, err := b.blockExists(hash)
	if err != nil {
		return false, err
	}
	return b.orphanBlockPool.IsKnownOrphan(hash) || exists, nil
}

// SequenceLock represents the converted relative lock-time in seconds, and
// absolute block-height for a transaction input's relative lock-times.
// According to SequenceLock, after the referenced input has been confirmed
// within a block, a transaction spending that input can be included into a
// block either after 'seconds' (according to past median time), or once the
// 'BlockHeight' has been reached.
type SequenceLock struct {
	Seconds     int64
	BlockHeight int32
}

// CalcSequenceLock computes a relative lock-time SequenceLock for the passed
// transaction using the passed UtxoViewpoint to obtain the past median time
// for blocks in which the referenced inputs of the transactions were included
// within. The generated SequenceLock lock can be used in conjunction with a
// block height, and adjusted median block time to determine if all the inputs
// referenced within a transaction have reached sufficient maturity allowing
// the candidate transaction to be included in a block.
//
// This function is safe for concurrent access.
func (b *BlockChain) CalcSequenceLock(tx *massutil.Tx, txStore TxStore) (*SequenceLock, error) {
	return b.calcSequenceLock(b.bestChain, tx, txStore)
}

// calcSequenceLock computes the relative lock-times for the passed
// transaction. See the exported version, CalcSequenceLock for further details.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockChain) calcSequenceLock(node *blockNode, tx *massutil.Tx, txStore TxStore) (*SequenceLock, error) {
	sequenceLock := &SequenceLock{Seconds: -1, BlockHeight: -1}
	mTx := tx.MsgTx()
	nextHeight := node.height + 1

	if IsCoinBase(tx) {
		return sequenceLock, nil
	}

	for txInIndex, txIn := range mTx.TxIn {
		utxo := txStore[txIn.PreviousOutPoint.Hash]
		if utxo == nil {
			str := fmt.Sprintf("output %v referenced from "+
				"transaction %s:%d either does not exist or "+
				"has already been spent", txIn.PreviousOutPoint,
				tx.Hash(), txInIndex)
			return sequenceLock, ruleError(ErrMissingTxOut, str)
		}

		inputHeight := utxo.BlockHeight
		if inputHeight == 0x7fffffff {
			inputHeight = nextHeight
		}

		sequenceNum := txIn.Sequence
		relativeLock := int64(sequenceNum & wire.SequenceLockTimeMask)

		switch {

		case sequenceNum&wire.SequenceLockTimeDisabled == wire.SequenceLockTimeDisabled:
			continue

		case sequenceNum&wire.SequenceLockTimeIsSeconds == wire.SequenceLockTimeIsSeconds:
			prevInputHeight := inputHeight - 1
			if prevInputHeight < 0 {
				prevInputHeight = 0
			}
			blockNode := node.Ancestor(prevInputHeight)
			medianTime, _ := b.calcPastMedianTime(blockNode)
			timeLockSeconds := (relativeLock << wire.SequenceLockTimeGranularity) - 1
			timeLock := medianTime.Unix() + timeLockSeconds
			if timeLock > sequenceLock.Seconds {
				sequenceLock.Seconds = timeLock
			}

		default:
			blockHeight := inputHeight + int32(relativeLock-1)
			if blockHeight > sequenceLock.BlockHeight {
				sequenceLock.BlockHeight = blockHeight
			}
		}
	}

	return sequenceLock, nil
}

// GenerateInitialIndex is an optional function which generates the required
// number of initial block nodes in an optimized fashion.  This is optional
// because the memory block index is sparse and previous nodes are dynamically
// loaded as needed.  However, during initial startup (when there are no nodes
// in memory yet), dynamically loading all of the required nodes on the fly in
// the usual way is much slower than preloading them.
//
// This function can only be called once and it must be called before any nodes
// are added to the block index.  ErrIndexAlreadyInitialized is returned if
// the former is not the case.  In practice, this means the function should be
// called directly after New.
func (b *BlockChain) GenerateInitialIndex() error {
	if b.root != nil {
		return ErrIndexAlreadyInitialized
	}

	_, endHeight, err := b.db.NewestSha()
	if err != nil {
		return err
	}

	startHeight := endHeight - minMemoryNodes
	if startHeight < 0 {
		startHeight = 0
	}

	for start := startHeight; start <= endHeight; {
		hashList, err := b.db.FetchHeightRange(start, endHeight+1)
		if err != nil {
			return err
		}

		if len(hashList) == 0 {
			break
		}

		for _, hash := range hashList {
			hashCopy := hash
			node, err := b.loadBlockNode(&hashCopy)
			if err != nil {
				return err
			}

			b.bestChain = node
		}

		start += int32(len(hashList))
	}

	return nil
}

// loadBlockNode loads the block identified by hash from the block database,
// creates a block node from it, and updates the memory block chain accordingly.
// It is used mainly to dynamically load previous blocks from database as they
// are needed to avoid needing to put the entire block chain in memory.
func (b *BlockChain) loadBlockNode(hash *wire.Hash) (*blockNode, error) {
	blockHeader, err := b.db.FetchBlockHeaderBySha(hash)
	if err != nil {
		return nil, err
	}
	blockHeight, err := b.db.FetchBlockHeightBySha(hash)
	if err != nil {
		return nil, err
	}

	node := newBlockNode(blockHeader, hash, blockHeight)
	node.inMainChain = true

	// Add the node to the chain.
	// There are several possibilities here:
	//  1) This node is a child of an existing block node
	//  2) This node is the parent of one or more nodes
	//  3) Neither 1 or 2 is true, and this is not the first node being
	//     added to the tree which implies it's an orphan block and
	//     therefore is an error to insert into the chain
	//  4) Neither 1 or 2 is true, but this is the first node being added
	//     to the tree, so it's the root.
	prevHash := &blockHeader.Previous
	if parentNode, ok := b.index[*prevHash]; ok {
		// Case 1 -- This node is a child of an existing block node.
		// Update the node's work sum with the sum of the parent node's
		// work sum and this node's work, append the node as a child of
		// the parent node and set this node's parent to the parent
		// node.
		node.capSum = node.capSum.Add(parentNode.capSum, node.capSum)
		parentNode.children = append(parentNode.children, node)
		node.parent = parentNode

	} else if childNodes, ok := b.depNodes[*hash]; ok {
		// Case 2 -- This node is the parent of one or more nodes.
		// Connect this block node to all of its children and update
		// all of the children (and their children) with the new work
		// sums.
		for _, childNode := range childNodes {
			childNode.parent = node
			node.children = append(node.children, childNode)
			addChildrenWork(childNode, node.capSum)
			b.root = node
		}

	} else {
		// Case 3 -- The node does't have a parent and is not the parent
		// of another node.  This is only acceptable for the first node
		// inserted into the chain.  Otherwise it means an arbitrary
		// orphan block is trying to be loaded which is not allowed.
		if b.root != nil {
			str := "loadBlockNode: attempt to insert orphan block %v"
			return nil, fmt.Errorf(str, hash)
		}

		// Case 4 -- This is the root since it's the first and only node.
		b.root = node
	}

	// Add the new node to the indices for faster lookups.
	b.index[*hash] = node
	b.depNodes[*prevHash] = append(b.depNodes[*prevHash], node)

	return node, nil
}

// getPrevNodeFromBlock returns a block node for the block previous to the
// passed block (the passed block's parent).  When it is already in the memory
// block chain, it simply returns it.  Otherwise, it loads the previous block
// from the block database, creates a new block node from it, and returns it.
// The returned node will be nil if the genesis block is passed.
func (b *BlockChain) getPrevNodeFromBlock(block *massutil.Block) (*blockNode, error) {
	prevHash := &block.MsgBlock().Header.Previous
	if prevHash.IsEqual(zeroHash) {
		return nil, nil
	}

	if bn, ok := b.index[*prevHash]; ok {
		return bn, nil
	}

	prevBlockNode, err := b.loadBlockNode(prevHash)
	if err != nil {
		return nil, err
	}
	return prevBlockNode, nil
}

// getPrevNodeFromNode returns a block node for the block previous to the
// passed block node (the passed block node's parent).  When the node is already
// connected to a parent, it simply returns it.  Otherwise, it loads the
// associated block from the database to obtain the previous hash and uses that
// to dynamically create a new block node and return it.  The memory block
// chain is updated accordingly.  The returned node will be nil if the genesis
// block is passed.
func (b *BlockChain) getPrevNodeFromNode(node *blockNode) (*blockNode, error) {
	if node.parent != nil {
		return node.parent, nil
	}

	if node.hash.IsEqual(config.ChainParams.GenesisHash) {
		return nil, nil
	}

	prevBlockNode, err := b.loadBlockNode(node.parentHash)
	if err != nil {
		return nil, err
	}

	return prevBlockNode, nil
}

// removeBlockNode removes the passed block node from the memory chain by
// unlinking all of its children and removing it from the the node and
// dependency indices.
func (b *BlockChain) removeBlockNode(node *blockNode) error {
	if node.parent != nil {
		return fmt.Errorf("removeBlockNode must be called with a "+
			" node at the front of the chain - node %v", node.hash)
	}

	delete(b.index, *node.hash)

	for _, child := range node.children {
		child.parent = nil
	}
	node.children = nil

	prevHash := node.parentHash
	if children, ok := b.depNodes[*prevHash]; ok {
		b.depNodes[*prevHash] = removeChildNode(children, node)
		if len(b.depNodes[*prevHash]) == 0 {
			delete(b.depNodes, *prevHash)
		}
	}

	return nil
}

// pruneBlockNodes removes references to old block nodes which are no longer
// needed so they may be garbage collected.  In order to validate block rules
// and choose the best chain, only a portion of the nodes which form the block
// chain are needed in memory.  This function walks the chain backwards from the
// current best chain to find any nodes before the first needed block node.
func (b *BlockChain) pruneBlockNodes() error {
	if b.bestChain == nil {
		return nil
	}

	newRootNode := b.bestChain
	for i := int32(0); i < minMemoryNodes-1 && newRootNode != nil; i++ {
		newRootNode = newRootNode.parent
	}

	if newRootNode == nil || newRootNode.parent == nil {
		return nil
	}

	deleteNodes := list.New()
	for node := newRootNode.parent; node != nil; node = node.parent {
		deleteNodes.PushFront(node)
	}

	for e := deleteNodes.Front(); e != nil; e = e.Next() {
		node := e.Value.(*blockNode)
		err := b.removeBlockNode(node)
		if err != nil {
			return err
		}
	}

	b.root = newRootNode

	return nil
}

// isMajorityVersion determines if a previous number of blocks in the chain
// starting with startNode are at least the minimum passed version.
func (b *BlockChain) isMajorityVersion(minVer uint64, startNode *blockNode,
	numRequired uint64) bool {

	numFound := uint64(0)
	iterNode := startNode
	for i := uint64(0); i < config.ChainParams.BlockUpgradeNumToCheck &&
		numFound < numRequired && iterNode != nil; i++ {
		if iterNode.version >= minVer {
			numFound++
		}

		var err error
		iterNode, err = b.getPrevNodeFromNode(iterNode)
		if err != nil {
			break
		}
	}

	return numFound >= numRequired
}

// calcPastMedianTime calculates the median time of the previous few blocks
// prior to, and including, the passed block node.  It is primarily used to
// validate new blocks have sane timestamps.
func (b *BlockChain) calcPastMedianTime(startNode *blockNode) (time.Time, error) {
	if startNode == nil {
		return b.info.genesisTime, nil
	}

	timestamps := make([]time.Time, medianTimeBlocks)
	numNodes := 0
	iterNode := startNode
	for i := 0; i < medianTimeBlocks && iterNode != nil; i++ {
		timestamps[i] = iterNode.timestamp
		numNodes++

		var err error
		iterNode, err = b.getPrevNodeFromNode(iterNode)
		if err != nil {
			logging.CPrint(logging.ERROR, "getPrevNodeFromNode", logging.LogFormat{"err": err, "iterNode": iterNode})
			return time.Time{}, err
		}
	}

	timestamps = timestamps[:numNodes]
	sort.Sort(timeSorter(timestamps))

	medianTimestamp := timestamps[numNodes/2]
	return medianTimestamp, nil
}

// CalcPastMedianTime calculates the median time of the previous few blocks
// prior to, and including, the end of the current best chain.  It is primarily
// used to ensure new blocks have sane timestamps.
//
// This function is NOT safe for concurrent access.
func (b *BlockChain) CalcPastMedianTime() (time.Time, error) {
	return b.calcPastMedianTime(b.bestChain)
}

// getReorganizeNodes finds the fork point between the main chain and the passed
// node and returns a list of block nodes that would need to be detached from
// the main chain and a list of block nodes that would need to be attached to
// the fork point (which will be the end of the main chain after detaching the
// returned list of block nodes) in order to reorganize the chain such that the
// passed node is the new end of the main chain.  The lists will be empty if the
// passed node is not on a side chain.
func (b *BlockChain) getReorganizeNodes(node *blockNode) (*list.List, *list.List) {
	attachNodes := list.New()
	detachNodes := list.New()
	if node == nil {
		return detachNodes, attachNodes
	}

	ancestor := node
	for ; ancestor.parent != nil; ancestor = ancestor.parent {
		if ancestor.inMainChain {
			break
		}
		attachNodes.PushFront(ancestor)
	}

	for n := b.bestChain; n != nil && n.parent != nil; n = n.parent {
		if n.hash.IsEqual(ancestor.hash) {
			break
		}
		detachNodes.PushBack(n)
	}

	return detachNodes, attachNodes
}

// connectBlock handles connecting the passed node/block to the end of the main
// (best) chain.
func (b *BlockChain) connectBlock(node *blockNode, block *massutil.Block) error {
	prevHash := &block.MsgBlock().Header.Previous
	if b.bestChain != nil && !prevHash.IsEqual(b.bestChain.hash) {
		return fmt.Errorf("connectBlock must be called with a block " +
			"that extends the main chain")
	}

	_, err := b.db.InsertBlock(block)
	if err != nil {
		return err
	}

	node.inMainChain = true
	b.index[*node.hash] = node
	b.depNodes[*prevHash] = append(b.depNodes[*prevHash], node)
	b.bestChain = node

	b.connectNotification(block)

	return nil
}

func (b *BlockChain) connectNotification(block *massutil.Block) {
	for _, tx := range block.Transactions()[1:] {
		b.TxPool.RemoveTransaction(tx, false)
		b.TxPool.RemoveDoubleSpends(tx)
		b.TxPool.RemoveOrphan(tx.Hash())
		b.TxPool.ProcessOrphans(tx.Hash())
	}
	if config.AddrIndex && b.addrIndexer.IsCaughtUp() {
		b.addrIndexer.UpdateAddressIndex(block)
	}
}

// disconnectBlock handles disconnecting the passed node/block from the end of
// the main (best) chain.
func (b *BlockChain) disconnectBlock(node *blockNode, block *massutil.Block) error {
	if b.bestChain == nil || !node.hash.IsEqual(b.bestChain.hash) {
		return fmt.Errorf("disconnectBlock must be called with the " +
			"block at the end of the main chain")
	}

	prevNode, err := b.getPrevNodeFromNode(node)
	if err != nil {
		return err
	}

	err = b.db.DropAfterBlockBySha(prevNode.hash)
	if err != nil {
		return err
	}

	node.inMainChain = false
	b.blockCache[*node.hash] = block
	b.bestChain = node.parent

	b.disconnectNotification(block)

	return nil
}

func (b *BlockChain) disconnectNotification(block *massutil.Block) {
	for _, tx := range block.Transactions()[1:] {
		_, err := b.TxPool.MaybeAcceptTransaction(tx,
			false, false)
		if err != nil {
			b.TxPool.RemoveTransaction(tx, true)
		}
	}
}

// reorganizeChain reorganizes the block chain by disconnecting the nodes in the
// detachNodes list and connecting the nodes in the attach list.  It expects
// that the lists are already in the correct order and are in sync with the
// end of the current best chain.  Specifically, nodes that are being
// disconnected must be in reverse order (think of popping them off
// the end of the chain) and nodes the are being attached must be in forwards
// order (think pushing them onto the end of the chain).
func (b *BlockChain) reorganizeChain(detachNodes, attachNodes *list.List, flags BehaviorFlags) error {
	for e := attachNodes.Front(); e != nil; e = e.Next() {
		n := e.Value.(*blockNode)
		if _, exists := b.blockCache[*n.hash]; !exists {
			return fmt.Errorf("block %v is missing from the side "+
				"chain block cache", n.hash)
		}
	}

	for e := attachNodes.Front(); e != nil; e = e.Next() {
		n := e.Value.(*blockNode)
		block := b.blockCache[*n.hash]
		err := b.checkConnectBlock(n, block)
		if err != nil {
			return err
		}
	}

	if flags&BFDryRun == BFDryRun {
		return nil
	}

	b.nodeLock.Lock()
	defer b.nodeLock.Unlock()
	for e := detachNodes.Front(); e != nil; e = e.Next() {
		n := e.Value.(*blockNode)
		block, err := b.db.FetchBlockBySha(n.hash)
		if err != nil {
			return err
		}
		err = b.disconnectBlock(n, block)
		if err != nil {
			return err
		}
	}

	for e := attachNodes.Front(); e != nil; e = e.Next() {
		n := e.Value.(*blockNode)
		block := b.blockCache[*n.hash]
		err := b.connectBlock(n, block)
		if err != nil {
			return err
		}
		delete(b.blockCache, *n.hash)
	}

	firstAttachNode := attachNodes.Front().Value.(*blockNode)
	forkNode, err := b.getPrevNodeFromNode(firstAttachNode)
	if err == nil {
		logging.CPrint(logging.INFO, "REORGANIZE: Chain forks", logging.LogFormat{"forkNode": forkNode.hash})
	}

	firstDetachNode := detachNodes.Front().Value.(*blockNode)
	lastAttachNode := attachNodes.Back().Value.(*blockNode)
	logging.CPrint(logging.INFO, "REORGANIZE", logging.LogFormat{
		"old_best_chain_head": firstDetachNode.hash,
	})
	logging.CPrint(logging.INFO, "REORGANIZE", logging.LogFormat{
		"new_best_chain_head": lastAttachNode.hash,
	})

	return nil
}

// connectBestChain handles connecting the passed block to the chain while
// respecting proper chain selection according to the chain with the most
// proof of work.  In the typical case, the new block simply extends the main
// chain.  However, it may also be extending (or creating) a side chain (fork)
// which may or may not end up becoming the main chain depending on which fork
// cumulatively has the most proof of work.
func (b *BlockChain) connectBestChain(node *blockNode, block *massutil.Block, flags BehaviorFlags) error {
	fastAdd := flags&BFFastAdd == BFFastAdd
	dryRun := flags&BFDryRun == BFDryRun

	if b.bestChain == nil || node.parent.hash.IsEqual(b.bestChain.hash) {
		if !fastAdd {
			err := b.checkConnectBlock(node, block)
			if err != nil {
				return err
			}
		}

		if dryRun {
			return nil
		}

		b.nodeLock.Lock()
		defer b.nodeLock.Unlock()
		err := b.connectBlock(node, block)
		if err != nil {
			return err
		}

		if node.parent != nil {
			node.parent.children = append(node.parent.children, node)
		}

		return nil
	}
	if fastAdd {
		logging.CPrint(logging.WARN, "fastAdd set in the side chain case?", logging.LogFormat{
			"block": block.Hash(),
		})
	}

	if !dryRun {
		logging.CPrint(logging.DEBUG, "Adding block to side chain cache", logging.LogFormat{
			"block": node.hash,
		})
	}
	b.blockCache[*node.hash] = block
	b.index[*node.hash] = node

	node.inMainChain = false
	node.parent.children = append(node.parent.children, node)

	if dryRun {
		defer func() {
			children := node.parent.children
			children = removeChildNode(children, node)
			node.parent.children = children

			delete(b.index, *node.hash)
			delete(b.blockCache, *node.hash)
		}()
	}

	if ok, _ := b.IsPotentialNewBestChain(node); !ok {
		if dryRun {
			return nil
		}

		fork := node
		for ; fork.parent != nil; fork = fork.parent {
			if fork.inMainChain {
				break
			}
		}

		if fork.hash.IsEqual(node.parent.hash) {
			logging.CPrint(logging.INFO, "FORK: Block forks the chain at height, but does not cause a reorganize",
				logging.LogFormat{
					"block":  node.hash,
					"height": fork.height,
					"fork":   fork.hash,
				})
		} else {
			logging.CPrint(logging.INFO, "EXTEND FORK: Block extends a side chain which forks the chain",
				logging.LogFormat{
					"block":  node.hash,
					"height": fork.height,
					"fork":   fork.hash,
				})
		}

		return nil
	}

	detachNodes, attachNodes := b.getReorganizeNodes(node)

	if !dryRun {
		logging.CPrint(logging.INFO, "REORGANIZE: Block is causing a reorganize.", logging.LogFormat{
			"Block": node.hash,
		})
	}

	err := b.reorganizeChain(detachNodes, attachNodes, flags)
	if err != nil {
		return err
	}

	return nil
}

// IsPotentialNewBestChain returns whether or not the side chain can be a new
// best chain. The Boolean is valid only when error is nil.
func (b *BlockChain) IsPotentialNewBestChain(sideChain *blockNode) (bool, error) {
	bestChain := b.bestChain
	if sideChain == nil || bestChain == nil {
		return false, fmt.Errorf("figure potential best chain: Cannot decide invalid chain")
	}

	if sideChain.capSum.Cmp(bestChain.capSum) != 0 {
		if sideChain.capSum.Cmp(bestChain.capSum) < 0 {
			return false, nil
		}
		return true, nil
	}

	if sideChain.timestamp.Unix() != bestChain.timestamp.Unix() {
		if sideChain.timestamp.Unix() > bestChain.timestamp.Unix() {
			return false, nil
		}
		return true, nil
	}

	if new(big.Int).SetBytes(sideChain.hash.Bytes()).Cmp(new(big.Int).SetBytes(bestChain.hash.Bytes())) < 0 {
		return true, nil
	}

	return false, nil
}

// GetBlockTimestamp
func (b *BlockChain) GetBlockTimestamp(hash wire.Hash) time.Time {
	b.accessLock.Lock()
	defer b.accessLock.Unlock()

	return b.index[hash].timestamp
}

// GetBestChainHash
func (b *BlockChain) GetBestChainHash() *wire.Hash {
	b.accessLock.Lock()
	defer b.accessLock.Unlock()

	return b.bestChain.hash
}

func (b *BlockChain) GetBestChainHeader() *wire.BlockHeader {
	b.nodeLock.RLock()
	defer b.nodeLock.RUnlock()

	sha := b.bestChain.hash
	bh, err := b.db.FetchBlockHeaderBySha(sha)
	if err != nil {
		logging.CPrint(logging.ERROR, "b.BlockChain.GetBestChainHeader() cannot fetch bestBlockHeader",
			logging.LogFormat{"blockHash": sha.String()})
		return wire.NewEmptyBlockHeader()
	}
	return bh
}

func (b *BlockChain) GetBestChainHeight() uint64 {
	return uint64(b.bestChain.height)
}

func (b *BlockChain) InMainChain(hash wire.Hash) bool {
	if bn, exists := b.index[hash]; exists {
		return bn.inMainChain
	}
	height, err := b.db.FetchBlockHeightBySha(&hash)
	if err != nil {
		return false
	}
	dbHash, err := b.db.FetchBlockShaByHeight(height)
	if err != nil {
		return false
	}
	return *dbHash == hash
}

// GetBlockByHash return a block by given hash
func (b *BlockChain) GetBlockByHash(hash *wire.Hash) (*massutil.Block, error) {
	return b.db.FetchBlockBySha(hash)
}

// GetBlockByHeight return a block header by given height
func (b *BlockChain) GetBlockByHeight(height uint64) (*massutil.Block, error) {
	b.nodeLock.RLock()
	defer b.nodeLock.RUnlock()
	hash, err := b.db.FetchBlockShaByHeight(int32(height))
	if err != nil {
		return nil, err
	}
	return b.GetBlockByHash(hash)
}

func (b *BlockChain) NewestSha() (sha *wire.Hash, height uint64, err error) {
	b.nodeLock.RLock()
	defer b.nodeLock.RUnlock()
	sha, blkHeight, err := b.db.NewestSha()
	return sha, uint64(blkHeight), err
}

func (b *BlockChain) ChainID() *wire.Hash {
	return wire.NewHashFromHash(b.info.chainID)
}

func New(db database.Db, sigCache *txscript.SigCache, hashCache *txscript.HashCache, txpool *TxPool, addrIndexer *chainindexer.AddrIndexer) (*BlockChain, error) {
	var checkpointsByHeight map[int32]*config.Checkpoint
	if len(config.ChainParams.Checkpoints) > 0 {
		checkpointsByHeight = make(map[int32]*config.Checkpoint)
		for i := range config.ChainParams.Checkpoints {
			checkpoint := &config.ChainParams.Checkpoints[i]
			checkpointsByHeight[int32(checkpoint.Height)] = checkpoint
		}
	}

	b := &BlockChain{
		db:                  db,
		sigCache:            sigCache,
		hashCache:           hashCache,
		checkpointsByHeight: checkpointsByHeight,
		root:                nil,
		bestChain:           nil,
		index:               make(map[wire.Hash]*blockNode),
		depNodes:            make(map[wire.Hash][]*blockNode),
		blockCache:          make(map[wire.Hash]*massutil.Block),
		orphanBlockPool:     newOrphanBlockPool(),
		TxPool:              txpool,
		addrIndexer:         addrIndexer,
	}

	b.TxPool.BlockChain = b
	b.addrIndexer.Chain = b

	err := b.GenerateInitialIndex()
	if err != nil {
		return nil, err
	}

	genesisHash, err := b.db.FetchBlockShaByHeight(0)
	if err != nil {
		return nil, err
	}
	genesisBlock, err := b.db.FetchBlockBySha(genesisHash)
	if err != nil {
		return nil, err
	}
	b.info = &chainInfo{
		genesisBlock: genesisBlock,
		genesisHash:  *genesisBlock.Hash(),
		genesisTime:  genesisBlock.MsgBlock().Header.Timestamp,
		chainID:      genesisBlock.MsgBlock().Header.ChainID,
	}

	return b, nil
}

func (b *BlockChain) Start() {
	if config.AddrIndex {
		b.addrIndexer.Start()
	}
}

func (b *BlockChain) Stop() {
	if config.AddrIndex {
		b.addrIndexer.Stop()
	}
}
