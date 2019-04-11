// Modified for MassNet
// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package ldb

import (
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/massnetorg/MassNet-wallet/logging"

	"math"

	"github.com/massnetorg/MassNet-wallet/database"
	"github.com/massnetorg/MassNet-wallet/massutil"
	"github.com/massnetorg/MassNet-wallet/wire"

	"github.com/btcsuite/goleveldb/leveldb"
	"github.com/btcsuite/goleveldb/leveldb/opt"
)

// LevelDb holds internal state for databse.
type LevelDb struct {
	// lock preventing multiple entry
	dbLock sync.Mutex

	// leveldb pieces
	lDb *leveldb.DB
	ro  *opt.ReadOptions
	wo  *opt.WriteOptions

	lbatch *leveldb.Batch

	nextBlock int32

	lastBlkShaCached bool
	lastBlkSha       wire.Hash
	lastBlkIdx       int32

	lastAddrIndexBlkSha wire.Hash
	lastAddrIndexBlkIdx int32

	txUpdateMap      map[wire.Hash]*txUpdateObj
	txSpentUpdateMap map[wire.Hash]*spentTxUpdate
}

var self = database.DriverDB{DbType: "leveldb", CreateDB: CreateDB, OpenDB: OpenDB}

func init() {
	database.AddDBDriver(self)
}

// parseArgs parses the arguments from the database package Open/Create methods.
func parseArgs(funcName string, args ...interface{}) (string, error) {
	if len(args) != 1 {
		return "", fmt.Errorf("Invalid arguments to ldb.%s -- "+
			"expected database path string", funcName)
	}
	dbPath, ok := args[0].(string)
	if !ok {
		return "", fmt.Errorf("First argument to ldb.%s is invalid -- "+
			"expected database path string", funcName)
	}
	return dbPath, nil
}

// CurrentDBVersion is the database version.
var CurrentDBVersion int32 = 1

// OpenDB opens an existing database for use.
func OpenDB(args ...interface{}) (database.Db, error) {
	dbpath, err := parseArgs("OpenDB", args...)
	if err != nil {
		return nil, err
	}

	db, err := openDB(dbpath, false)
	if err != nil {
		return nil, err
	}

	// Need to find last block and tx
	var lastknownblock, nextunknownblock, testblock int32

	increment := int32(100000)
	ldb := db.(*LevelDb)

	var lastSha *wire.Hash
	// forward scan
blockforward:
	for {

		sha, err := ldb.fetchBlockShaByHeight(testblock)
		if err == nil {
			// block is found
			lastSha = sha
			lastknownblock = testblock
			testblock += increment
		} else {
			if testblock == 0 {
				//no blocks in db, odd but ok.
				lastknownblock = -1
				nextunknownblock = 0
				var emptysha wire.Hash
				lastSha = &emptysha
			} else {
				nextunknownblock = testblock
			}
			break blockforward
		}
	}

	// narrow search
blocknarrow:
	for lastknownblock != -1 {
		testblock = (lastknownblock + nextunknownblock) / 2
		sha, err := ldb.fetchBlockShaByHeight(testblock)
		if err == nil {
			lastknownblock = testblock
			lastSha = sha
		} else {
			nextunknownblock = testblock
		}
		if lastknownblock+1 == nextunknownblock {
			break blocknarrow
		}
	}

	logging.CPrint(logging.INFO, "checking address index", logging.LogFormat{})

	// Load the last block whose transactions have been indexed by address.
	if sha, idx, err := ldb.fetchAddrIndexTip(); err == nil {
		if err = ldb.checkAddrIndexVersion(); err == nil {
			ldb.lastAddrIndexBlkSha = *sha
			ldb.lastAddrIndexBlkIdx = idx
			logging.CPrint(logging.INFO, "address index good, continuing", logging.LogFormat{})
		} else {
			logging.CPrint(logging.INFO, "address index in old, incompatible format, dropping...", logging.LogFormat{})
			ldb.deleteOldAddrIndex()
			ldb.DeleteAddrIndex()
			logging.CPrint(logging.INFO, "old, incompatible address index dropped and can now be rebuilt", logging.LogFormat{})
		}
	} else {
		ldb.lastAddrIndexBlkIdx = -1
	}

	ldb.lastBlkSha = *lastSha
	ldb.lastBlkIdx = lastknownblock
	ldb.nextBlock = lastknownblock + 1

	return db, nil
}

func openDB(dbpath string, create bool) (pbdb database.Db, err error) {
	var db LevelDb
	var tlDb *leveldb.DB
	var dbversion int32

	defer func() {
		if err == nil {
			db.lDb = tlDb

			db.txUpdateMap = map[wire.Hash]*txUpdateObj{}
			db.txSpentUpdateMap = make(map[wire.Hash]*spentTxUpdate)

			pbdb = &db
		}
	}()

	if create == true {
		err = os.Mkdir(dbpath, 0750)
		if err != nil {
			logging.CPrint(logging.ERROR, "mkdir failed", logging.LogFormat{"dbPath": dbpath, "err": err})
			return
		}
	} else {
		_, err = os.Stat(dbpath)
		if err != nil {
			err = database.ErrDbDoesNotExist
			return
		}
	}

	needVersionFile := false
	verfile := dbpath + ".ver"
	fi, ferr := os.Open(verfile)
	if ferr == nil {
		defer fi.Close()

		ferr = binary.Read(fi, binary.LittleEndian, &dbversion)
		if ferr != nil {
			dbversion = ^0
		}
	} else {
		if create == true {
			needVersionFile = true
			dbversion = CurrentDBVersion
		}
	}

	opts := &opt.Options{
		BlockCacher:     opt.DefaultBlockCacher,
		Compression:     opt.NoCompression,
		OpenFilesCacher: opt.DefaultOpenFilesCacher,
	}

	switch dbversion {
	case 0:
		opts = &opt.Options{}
	case 1:
		// uses defaults from above
	default:
		err = fmt.Errorf("unsupported db version %v", dbversion)
		return
	}

	tlDb, err = leveldb.OpenFile(dbpath, opts)
	if err != nil {
		return
	}

	// If we opened the database successfully on 'create'
	// update the
	if needVersionFile {
		fo, ferr := os.Create(verfile)
		if ferr != nil {
			err = ferr
			return
		}
		defer fo.Close()
		err = binary.Write(fo, binary.LittleEndian, dbversion)
		if err != nil {
			return
		}
	}

	return
}

// CreateDB creates, initializes and opens a database for use.
func CreateDB(args ...interface{}) (database.Db, error) {
	dbpath, err := parseArgs("Create", args...)
	if err != nil {
		return nil, err
	}

	// No special setup needed, just OpenBB
	db, err := openDB(dbpath, true)
	if err == nil {
		ldb := db.(*LevelDb)
		ldb.lastBlkIdx = -1
		ldb.lastAddrIndexBlkIdx = -1
		ldb.nextBlock = 0
	}
	return db, err
}

func (db *LevelDb) close() error {
	return db.lDb.Close()
}

// Sync verifies that the database is coherent on disk,
// and no outstanding transactions are in flight.
func (db *LevelDb) Sync() error {

	// while specified by the API, does nothing
	// however does grab lock to verify it does not return until other operations are complete.
	return nil
}

// Close cleanly shuts down database, syncing all data.
func (db *LevelDb) Close() error {

	return db.close()
}

// DropAfterBlockBySha will remove any blocks from the database after
// the given block.
func (db *LevelDb) DropAfterBlockBySha(sha *wire.Hash) (rerr error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	defer func() {
		if rerr == nil {
			rerr = db.processBatches()
		} else {
			db.lBatch().Reset()
		}
	}()

	startheight := db.nextBlock - 1

	keepidx, err := db.getBlkLoc(sha)
	if err != nil {
		logging.CPrint(logging.TRACE, "block loc failed", logging.LogFormat{"sha": sha})
		return err
	}

	for height := startheight; height > keepidx; height = height - 1 {
		var blk *massutil.Block
		blksha, buf, err := db.getBlkByHeight(height)
		if err != nil {
			return err
		}
		blk, err = massutil.NewBlockFromBytes(buf, wire.DB)
		if err != nil {
			return err
		}

		for _, tx := range blk.MsgBlock().Transactions {
			err = db.unSpend(tx)
			if err != nil {
				return err
			}
		}

		if err != nil {
			return err
		}

		// rather than iterate the list of tx backward, do it twice.
		for _, tx := range blk.Transactions() {
			var txUo txUpdateObj
			txUo.delete = true
			db.txUpdateMap[*tx.Hash()] = &txUo
		}

		db.lBatch().Delete(shaBlkToKey(blksha))
		db.lBatch().Delete(int64ToKey(int64(height)))
	}

	// update the last block cache
	db.lastBlkShaCached = true
	db.lastBlkSha = *sha
	db.lastBlkIdx = keepidx
	db.nextBlock = keepidx + 1

	return nil
}

// InsertBlock inserts raw block and transaction data from a block into the
// database.  The first block inserted into the database will be treated as the
// genesis block.  Every subsequent block insert requires the referenced parent
// block to already exist.
func (db *LevelDb) InsertBlock(block *massutil.Block) (height int32, rerr error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	defer func() {
		if rerr == nil {
			rerr = db.processBatches()
		} else {
			db.lBatch().Reset()
		}
	}()

	blocksha := block.Hash()
	mblock := block.MsgBlock()
	rawMsg, err := block.Bytes(wire.DB)
	if err != nil {
		logging.CPrint(logging.WARN, "failed to obtain raw block hash", logging.LogFormat{"hash": blocksha})
		return 0, err
	}
	txloc, err := block.TxLoc()
	if err != nil {
		logging.CPrint(logging.WARN, "failed to obtain raw block hash", logging.LogFormat{"hash": blocksha})
		return 0, err
	}

	// Insert block into database
	newheight, err := db.insertBlockData(blocksha, &mblock.Header.Previous,
		rawMsg)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to insert block",
			logging.LogFormat{"blocksha": blocksha, "previous": &mblock.Header.Previous, "err": err})
		return 0, err
	}

	// At least two blocks in the long past were generated by faulty
	// miners, the sha of the transaction exists in a previous block,
	// detect this condition and 'accept' the block.
	for txidx, tx := range mblock.Transactions {
		txsha, err := block.TxHash(txidx)
		if err != nil {
			logging.CPrint(logging.WARN, "failed to compute tx name", logging.LogFormat{"block": blocksha, "idx": txidx, "err": err})
			return 0, err
		}
		spentbuflen := (len(tx.TxOut) + 7) / 8
		spentbuf := make([]byte, spentbuflen, spentbuflen)
		if len(tx.TxOut)%8 != 0 {
			for i := uint(len(tx.TxOut) % 8); i < 8; i++ {
				spentbuf[spentbuflen-1] |= (byte(1) << i)
			}
		}

		err = db.insertTx(txsha, newheight, txloc[txidx].TxStart, txloc[txidx].TxLen, spentbuf)
		if err != nil {
			logging.CPrint(logging.WARN, "failed to insert tx",
				logging.LogFormat{"block": blocksha, "newHeight": newheight, "txHash": &txsha, "txidx": txidx, "err": err})
			return 0, err
		}

		err = db.doSpend(tx)
		if err != nil {
			logging.CPrint(logging.WARN, "failed to spend tx",
				logging.LogFormat{"block": blocksha, "newHeight": newheight, "txHash": &txsha, "txidx": txidx, "err": err})
			return 0, err
		}
	}
	return newheight, nil
}

func (db *LevelDb) doSpend(tx *wire.MsgTx) error {

	for txinidx := range tx.TxIn {
		if isCoinBaseTx(tx) {
			continue
		}
		txin := tx.TxIn[txinidx]

		inTxSha := txin.PreviousOutPoint.Hash
		inTxidx := txin.PreviousOutPoint.Index

		if inTxidx == ^uint32(0) {
			continue
		}

		err := db.setSpentData(&inTxSha, inTxidx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *LevelDb) unSpend(tx *wire.MsgTx) error {
	for txinidx := range tx.TxIn {
		txin := tx.TxIn[txinidx]

		inTxSha := txin.PreviousOutPoint.Hash
		inTxidx := txin.PreviousOutPoint.Index

		if inTxidx == ^uint32(0) {
			continue
		}

		err := db.clearSpentData(&inTxSha, inTxidx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (db *LevelDb) setSpentData(sha *wire.Hash, idx uint32) error {
	return db.setclearSpentData(sha, idx, true)
}

func (db *LevelDb) clearSpentData(sha *wire.Hash, idx uint32) error {
	return db.setclearSpentData(sha, idx, false)
}

func (db *LevelDb) setclearSpentData(txsha *wire.Hash, idx uint32, set bool) error {
	var txUo *txUpdateObj
	var ok bool

	if txUo, ok = db.txUpdateMap[*txsha]; !ok {
		// not cached, load from db
		var txU txUpdateObj
		blkHeight, txOff, txLen, spentData, err := db.getTxData(txsha)
		if err != nil {
			// setting a fully spent tx is an error.
			if set == true {
				return err
			}
			// if we are clearing a tx and it wasn't found
			// in the tx table, it could be in the fully spent
			// (duplicates) table.
			spentTxList, err := db.getTxFullySpent(txsha)
			if err != nil {
				return err
			}

			// need to reslice the list to exclude the most recent.
			sTx := spentTxList[len(spentTxList)-1]
			spentTxList[len(spentTxList)-1] = nil
			if len(spentTxList) == 1 {
				db.txSpentUpdateMap[*txsha] = &spentTxUpdate{delete: true}
			} else {
				return fmt.Errorf("fully-spent tx %v does not have 1 record: "+
					"%v", txsha, len(spentTxList))
			}

			// Create 'new' Tx update data.
			blkHeight = sTx.blkHeight
			txOff = sTx.txoff
			txLen = sTx.txlen
			spentbuflen := (sTx.numTxO + 7) / 8
			spentData = make([]byte, spentbuflen, spentbuflen)
			for i := range spentData {
				spentData[i] = ^byte(0)
			}
		}

		txU.txSha = txsha
		txU.blkHeight = blkHeight
		txU.txoff = txOff
		txU.txlen = txLen
		txU.spentData = spentData

		txUo = &txU
	}

	byteidx := idx / 8
	byteoff := idx % 8

	if set {
		txUo.spentData[byteidx] |= (byte(1) << byteoff)
	} else {
		txUo.spentData[byteidx] &= ^(byte(1) << byteoff)
	}

	// check for fully spent Tx
	fullySpent := true
	for _, val := range txUo.spentData {
		if val != ^byte(0) {
			fullySpent = false
			break
		}
	}
	if fullySpent {
		var txSu *spentTxUpdate
		// Look up Tx in fully spent table
		if txSuOld, ok := db.txSpentUpdateMap[*txsha]; ok {
			txSu = txSuOld
		} else {
			var txSuStore spentTxUpdate
			txSu = &txSuStore

			txSuOld, err := db.getTxFullySpent(txsha)
			if err == nil {
				txSu.txl = txSuOld
			}
		}

		// Fill in spentTx
		var sTx spentTx
		sTx.blkHeight = txUo.blkHeight
		sTx.txoff = txUo.txoff
		sTx.txlen = txUo.txlen
		// XXX -- there is no way to comput the real TxOut
		// from the spent array.
		sTx.numTxO = 8 * len(txUo.spentData)

		// append this txdata to fully spent txlist
		txSu.txl = append(txSu.txl, &sTx)

		// mark txsha as deleted in the txUpdateMap
		logging.CPrint(logging.TRACE, "tx %v is fully spent", logging.LogFormat{"txHash": txsha})

		db.txSpentUpdateMap[*txsha] = txSu

		txUo.delete = true
		db.txUpdateMap[*txsha] = txUo
	} else {
		db.txUpdateMap[*txsha] = txUo
	}

	return nil
}

func int64ToKey(keyint int64) []byte {
	key := strconv.FormatInt(keyint, 10)
	return []byte(key)
}

func shaBlkToKey(sha *wire.Hash) []byte {
	return sha[:]
}

// These are used here and in tx.go's deleteOldAddrIndex() to prevent deletion
// of indexes other than the addrindex now.
var recordSuffixTx = []byte{'t', 'x'}
var recordSuffixSpentTx = []byte{'s', 'x'}

func shaTxToKey(sha *wire.Hash) []byte {
	key := make([]byte, len(sha)+len(recordSuffixTx))
	copy(key, sha[:])
	copy(key[len(sha):], recordSuffixTx)
	return key
}

func shaSpentTxToKey(sha *wire.Hash) []byte {
	key := make([]byte, len(sha)+len(recordSuffixSpentTx))
	copy(key, sha[:])
	copy(key[len(sha):], recordSuffixSpentTx)
	return key
}

func isCoinBaseTx(msgTx *wire.MsgTx) bool {
	prevOut := &msgTx.TxIn[0].PreviousOutPoint
	if prevOut.Index != math.MaxUint32 || !prevOut.Hash.IsEqual(&wire.Hash{}) {
		return false
	}

	return true
}

func (db *LevelDb) lBatch() *leveldb.Batch {
	if db.lbatch == nil {
		db.lbatch = new(leveldb.Batch)
	}
	return db.lbatch
}

func (db *LevelDb) processBatches() error {
	var err error

	if len(db.txUpdateMap) != 0 || len(db.txSpentUpdateMap) != 0 || db.lbatch != nil {
		if db.lbatch == nil {
			db.lbatch = new(leveldb.Batch)
		}

		defer db.lbatch.Reset()

		for txSha, txU := range db.txUpdateMap {
			key := shaTxToKey(&txSha)
			if txU.delete {
				db.lbatch.Delete(key)
			} else {
				txdat := db.formatTx(txU)
				db.lbatch.Put(key, txdat)
			}
		}

		for txSha, txSu := range db.txSpentUpdateMap {
			key := shaSpentTxToKey(&txSha)
			if txSu.delete {
				db.lbatch.Delete(key)
			} else {
				txdat := db.formatTxFullySpent(txSu.txl)
				db.lbatch.Put(key, txdat)
			}
		}

		err = db.lDb.Write(db.lbatch, db.wo)
		if err != nil {
			logging.CPrint(logging.TRACE, "batch failed", logging.LogFormat{"err": err})
			return err
		}
		db.txUpdateMap = map[wire.Hash]*txUpdateObj{}
		db.txSpentUpdateMap = make(map[wire.Hash]*spentTxUpdate)

	}

	return nil
}

// RollbackClose this is part of the database.Db interface and should discard
// recent changes to the db and the close the db.  This currently just does
// a clean shutdown.
func (db *LevelDb) RollbackClose() error {

	return db.close()
}
