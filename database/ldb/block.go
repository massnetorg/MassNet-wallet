package ldb

import (
	"bytes"
	"encoding/binary"
	"math"

	"massnet.org/mass-wallet/database/storage"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/txscript"
	wirepb "massnet.org/mass-wallet/wire/pb"

	"github.com/golang/protobuf/proto"

	"massnet.org/mass-wallet/database"
	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/wire"
)

var (
	blockHeightKeyPrefix = []byte("BLKHGT")
	blockShaKeyPrefix    = []byte("BLKSHA")
)

const (
	blockHeightKeyPrefixLength = 6
	blockHeightKeyLength       = blockHeightKeyPrefixLength + 8
	blockShaKeyPrefixLength    = 6
	blockShaKeyLength          = blockShaKeyPrefixLength + 32
	UnknownHeight              = math.MaxUint64
)

func (db *ChainDb) SubmitBlock(block *massutil.Block) error {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	batch := db.Batch(blockBatch)
	batch.Set(*block.Hash())

	if err := db.preSubmit(blockBatch); err != nil {
		return err
	}

	if err := db.submitBlock(block); err != nil {
		batch.Reset()
		return err
	}

	batch.Done()
	return nil
}

func (db *ChainDb) submitBlock(block *massutil.Block) (err error) {

	defer func() {
		if err == nil {
			err = db.processBlockBatch()
		}
	}()

	batch := db.Batch(blockBatch).Batch()
	blockHash := block.Hash()
	mBlock := block.MsgBlock()

	rawMsg, err := mBlock.Bytes(wire.DB)
	if err != nil {
		logging.CPrint(logging.WARN, "failed to obtain raw block bytes", logging.LogFormat{"block": blockHash})
		return err
	}
	txLoc, err := block.TxLoc()
	if err != nil {
		logging.CPrint(logging.WARN, "failed to obtain block txLoc", logging.LogFormat{"block": blockHash})
		return err
	}

	// put block into blockBatch
	newHeight, err := db.insertBlockData(batch, blockHash, &mBlock.Header.Previous, rawMsg)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to insert block",
			logging.LogFormat{"block": blockHash, "previous": &mBlock.Header.Previous, "err": err})
		return err
	}

	// put Fault PubKey into blockBatch, and delete from punishments
	faultPks := block.MsgBlock().Proposals.PunishmentArea
	if err = insertFaultPks(batch, newHeight, block.MsgBlock().Proposals.PunishmentArea); err != nil {
		return err
	}
	if err = dropPunishments(batch, faultPks); err != nil {
		return err
	}

	// put pk, bitLength and height into batch
	if err = db.insertPubkblToBatch(batch, block.MsgBlock().Header.PubKey, block.MsgBlock().Header.Proof.BitLength, newHeight); err != nil {
		return err
	}

	// put index for pubKey->Block
	if err = updateMinedBlockIndex(batch, true, block.MsgBlock().Header.PubKey, newHeight); err != nil {
		return err
	}

	// At least two blocks in the long past were generated by faulty
	// miners, the sha of the transaction exists in a previous block,
	// detect this condition and 'accept' the block.
	for txidx, tx := range mBlock.Transactions {
		txsha, err := block.TxHash(txidx)
		if err != nil {
			logging.CPrint(logging.WARN, "failed to compute tx hash", logging.LogFormat{"block": blockHash, "idx": txidx, "err": err})
			return err
		}
		spentbuflen := (len(tx.TxOut) + 7) / 8
		spentbuf := make([]byte, spentbuflen)
		if block.Height() == 0 {
			for _, b := range spentbuf {
				for i := uint(0); i < 8; i++ {
					b |= byte(1) << i
				}
			}
		} else {
			if len(tx.TxOut)%8 != 0 {
				for i := uint(len(tx.TxOut) % 8); i < 8; i++ {
					spentbuf[spentbuflen-1] |= (byte(1) << i)
				}
			}
		}

		// find and insert staking tx
		for i, txOut := range tx.TxOut {
			class, pops := txscript.GetScriptInfo(txOut.PkScript)
			if class == txscript.StakingScriptHashTy {
				frozenPeriod, rsh, err := txscript.GetParsedOpcode(pops, class)
				if err != nil {
					return err
				}
				logging.CPrint(logging.DEBUG, "Insert StakingTx", logging.LogFormat{
					"txid":          txsha,
					"index":         i,
					"block height":  block.Height(),
					"frozen period": frozenPeriod,
					"scriptHash":    rsh,
				})
				err = db.insertStakingTx(txsha, uint32(i), frozenPeriod, newHeight, rsh, txOut.Value)
				if err != nil {
					return err
				}
			}
		}

		err = db.insertTx(txsha, newHeight, txLoc[txidx].TxStart, txLoc[txidx].TxLen, spentbuf)
		if err != nil {
			logging.CPrint(logging.WARN, "failed to insert tx",
				logging.LogFormat{"block": blockHash, "newHeight": newHeight, "txHash": &txsha, "txidx": txidx, "err": err})
			return err
		}

		err = db.doSpend(tx)
		if err != nil {
			logging.CPrint(logging.WARN, "failed to spend tx",
				logging.LogFormat{"block": blockHash, "newHeight": newHeight, "txHash": &txsha, "txidx": txidx, "err": err})
			return err
		}

		err = db.expire(newHeight)
		if err != nil {
			logging.CPrint(logging.ERROR, "block failed to expire tx", logging.LogFormat{"block": blockHash, "blockHeigh": newHeight, "txid": txsha, "txidx": txidx})
			return err
		}
	}
	return nil
}

func (db *ChainDb) DeleteBlock(hash *wire.Hash) error {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	batch := db.Batch(blockBatch)
	batch.Set(*hash)

	if err := db.preSubmit(blockBatch); err != nil {
		return err
	}

	if err := db.deleteBlock(hash); err != nil {
		batch.Reset()
		return err
	}

	batch.Done()
	return nil
}

func (db *ChainDb) deleteBlock(hash *wire.Hash) (err error) {
	defer func() {
		if err == nil {
			err = db.processBlockBatch()
		}
	}()

	if !hash.IsEqual(&db.dbStorageMeta.currentHash) {
		logging.CPrint(logging.ERROR, "fail on delete block",
			logging.LogFormat{
				"err":         err,
				"currentHash": db.dbStorageMeta.currentHash,
				"deleteHash":  hash,
			})
		return database.ErrDeleteNonNewestBlock
	}

	var batch = db.Batch(blockBatch).Batch()
	var blk *massutil.Block
	height, buf, err := db.getBlk(hash)
	if err != nil {
		return err
	}
	blk, err = massutil.NewBlockFromBytes(buf, wire.DB)
	if err != nil {
		return err
	}

	if err = insertPunishments(batch, blk.MsgBlock().Proposals.PunishmentArea); err != nil {
		return err
	}

	if err = db.dropFaultPksByHeight(batch, height); err != nil {
		return err
	}

	for _, tx := range blk.MsgBlock().Transactions {
		if err = db.unSpend(tx); err != nil {
			return err
		}
	}

	if err = db.freeze(height); err != nil {
		return err
	}

	// remove pk, bitLength and height
	if err = db.removePubkblWithCheck(batch, blk.MsgBlock().Header.PubKey, blk.MsgBlock().Header.Proof.BitLength, height); err != nil {
		return err
	}

	if err = updateMinedBlockIndex(batch, false, blk.MsgBlock().Header.PubKey, height); err != nil {
		return err
	}

	// rather than iterate the list of tx backward, do it twice.
	for _, tx := range blk.Transactions() {
		var txUo txUpdateObj
		var txStk stakingTx
		txUo.delete = true
		db.txUpdateMap[*tx.Hash()] = &txUo

		// delete insert stakingTx in the block
		for i, txOut := range tx.MsgTx().TxOut {
			class, pushData := txscript.GetScriptInfo(txOut.PkScript)
			if class == txscript.StakingScriptHashTy {
				frozenPeriod, _, err := txscript.GetParsedOpcode(pushData, class)
				if err != nil {
					return err
				}
				txStk.expiredHeight = height + frozenPeriod
				txStk.delete = true
				var key = stakingTxMapKey{
					blockHeight: height,
					txID:        *tx.Hash(),
					index:       uint32(i),
				}
				db.stakingTxMap[key] = &txStk
			}
		}
	}

	batch.Delete(makeBlockShaKey(hash))
	batch.Delete(makeBlockHeightKey(height))
	// If height is 0, reset dbStorageMetaDataKey to initial value.
	// See NewChainDb(...)
	newStorageMeta := dbStorageMeta{
		currentHeight: UnknownHeight,
	}
	if height != 0 {
		lastHash, _, err := db.getBlkByHeight(height - 1)
		if err != nil {
			return err
		}
		newStorageMeta = dbStorageMeta{
			currentHeight: height - 1,
			currentHash:   *lastHash,
		}
	}
	batch.Put(dbStorageMetaDataKey, encodeDBStorageMetaData(newStorageMeta))

	return nil
}

func (db *ChainDb) processBlockBatch() error {
	var batch = db.Batch(blockBatch).Batch()

	if len(db.txUpdateMap) != 0 || len(db.txSpentUpdateMap) != 0 || len(db.stakingTxMap) != 0 || len(db.expiredStakingTxMap) != 0 {
		for txSha, txU := range db.txUpdateMap {
			key := shaTxToKey(&txSha)
			if txU.delete {
				//log.Tracef("deleting tx %v", txSha)
				batch.Delete(key)
			} else {
				//log.Tracef("inserting tx %v", txSha)
				txdat := db.formatTx(txU)
				batch.Put(key, txdat)
			}
		}

		for txSha, txSu := range db.txSpentUpdateMap {
			key := shaSpentTxToKey(&txSha)
			if txSu.delete {
				//log.Tracef("deleting tx %v", txSha)
				batch.Delete(key)
			} else {
				//log.Tracef("inserting tx %v", txSha)
				txdat := db.formatTxFullySpent(txSu.txl)
				batch.Put(key, txdat)
			}
		}

		for mapKey, txL := range db.stakingTxMap {
			key := heightStakingTxToKey(txL.expiredHeight, mapKey)
			if txL.delete {
				batch.Delete(key)
			} else {
				txdat := db.formatSTx(txL)
				batch.Put(key, txdat)
			}
		}

		for mapKey, txU := range db.expiredStakingTxMap {
			key := heightExpiredStakingTxToKey(txU.expiredHeight, mapKey)
			if txU.delete {
				batch.Delete(key)
			} else {
				txdat := db.formatSTx(txU)
				batch.Put(key, txdat)
			}
		}

		db.txUpdateMap = map[wire.Hash]*txUpdateObj{}
		db.txSpentUpdateMap = make(map[wire.Hash]*spentTxUpdate)

		db.stakingTxMap = map[stakingTxMapKey]*stakingTx{}
		db.expiredStakingTxMap = map[stakingTxMapKey]*stakingTx{}
	}

	return nil
}

func (db *ChainDb) InitByGenesisBlock(block *massutil.Block) error {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	if err := db.submitBlock(block); err != nil {
		return err
	}

	db.dbStorageMeta.currentHash = *block.Hash()
	db.dbStorageMeta.currentHeight = 0

	return db.stor.Write(db.Batch(blockBatch).Batch())
}

// FetchBlockBySha - return a massutil Block
func (db *ChainDb) FetchBlockBySha(sha *wire.Hash) (blk *massutil.Block, err error) {
	return db.fetchBlockBySha(sha)
}

// fetchBlockBySha - return a massutil Block
// Must be called with db lock held.
func (db *ChainDb) fetchBlockBySha(sha *wire.Hash) (blk *massutil.Block, err error) {
	buf, height, err := db.fetchSha(sha)
	if err != nil {
		return
	}

	blk, err = massutil.NewBlockFromBytes(buf, wire.DB)
	if err != nil {
		return
	}
	blk.SetHeight(height)

	return
}

// FetchBlockHeightBySha returns the block height for the given hash.  This is
// part of the database.Db interface implementation.
func (db *ChainDb) FetchBlockHeightBySha(sha *wire.Hash) (uint64, error) {
	return db.getBlkLoc(sha)
}

// FetchBlockHeaderBySha - return a Hash
func (db *ChainDb) FetchBlockHeaderBySha(sha *wire.Hash) (bh *wire.BlockHeader, err error) {

	// Read the raw block from the database.
	buf, _, err := db.fetchSha(sha)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(buf)

	// Only deserialize the header portion and ensure the transaction count
	// is zero since this is a standalone header.
	// Read BLockBase length
	blockBaseLength, _, err := wire.ReadUint64(r, 0)
	if err != nil {
		return nil, err
	}

	// Read BlockBase
	baseData := make([]byte, blockBaseLength)
	_, err = r.Read(baseData)
	if err != nil {
		return nil, err
	}
	basePb := new(wirepb.BlockBase)
	err = proto.Unmarshal(baseData, basePb)
	if err != nil {
		return nil, err
	}
	base, err := wire.NewBlockBaseFromProto(basePb)
	if err != nil {
		return nil, err
	}

	bh = &base.Header

	return bh, nil
}

func (db *ChainDb) getBlkLoc(sha *wire.Hash) (uint64, error) {
	key := makeBlockShaKey(sha)

	data, err := db.stor.Get(key)
	if err != nil {
		if err == storage.ErrNotFound {
			err = database.ErrBlockShaMissing
		}
		return 0, err
	}

	// deserialize
	blkHeight := binary.LittleEndian.Uint64(data)

	return blkHeight, nil
}

func (db *ChainDb) getBlkByHeight(blkHeight uint64) (rsha *wire.Hash, rbuf []byte, err error) {
	var blkVal []byte

	key := makeBlockHeightKey(blkHeight)

	blkVal, err = db.stor.Get(key)
	if err != nil {
		logging.CPrint(logging.TRACE, "failed to find block on height", logging.LogFormat{"height": blkHeight})
		return
	}

	var sha wire.Hash

	sha.SetBytes(blkVal[0:32])

	blockData := make([]byte, len(blkVal[32:]))
	copy(blockData[:], blkVal[32:])

	return &sha, blockData, nil
}

func (db *ChainDb) getBlk(sha *wire.Hash) (rBlkHeight uint64, rBuf []byte, err error) {
	var blkHeight uint64

	blkHeight, err = db.getBlkLoc(sha)
	if err != nil {
		return
	}

	var buf []byte

	_, buf, err = db.getBlkByHeight(blkHeight)
	if err != nil {
		return
	}
	return blkHeight, buf, nil
}

func setBlk(batch storage.Batch, sha *wire.Hash, blkHeight uint64, buf []byte) {
	// serialize
	var lw [8]byte
	binary.LittleEndian.PutUint64(lw[0:8], uint64(blkHeight))

	shaKey := makeBlockShaKey(sha)
	blkKey := makeBlockHeightKey(blkHeight)

	blkVal := make([]byte, len(sha)+len(buf))
	copy(blkVal[0:], sha[:])
	copy(blkVal[len(sha):], buf)

	batch.Put(shaKey, lw[:])
	batch.Put(blkKey, blkVal)
}

// insertSha stores a block hash and its associated data block with a
// previous sha of `prevSha'.
// insertSha shall be called with db lock held
func (db *ChainDb) insertBlockData(batch storage.Batch, sha *wire.Hash, prevSha *wire.Hash, buf []byte) (uint64, error) {
	oBlkHeight, err := db.getBlkLoc(prevSha)
	if err != nil {
		var zeroHash = wire.Hash{}
		if *prevSha != zeroHash {
			return 0, err
		}
		oBlkHeight = UnknownHeight
	}

	blkHeight := oBlkHeight + 1
	setBlk(batch, sha, blkHeight, buf)
	newStorageMeta := dbStorageMeta{
		currentHeight: blkHeight,
		currentHash:   *sha,
	}
	batch.Put(dbStorageMetaDataKey, encodeDBStorageMetaData(newStorageMeta))

	return blkHeight, nil
}

// fetchSha returns the datablock for the given Hash.
func (db *ChainDb) fetchSha(sha *wire.Hash) (rBuf []byte,
	rBlkHeight uint64, err error) {
	var blkHeight uint64
	var buf []byte

	blkHeight, buf, err = db.getBlk(sha)
	if err != nil {
		return
	}

	return buf, blkHeight, nil
}

// ExistsSha looks up the given block hash
// returns true if it is present in the database.
func (db *ChainDb) ExistsSha(sha *wire.Hash) (bool, error) {

	// not in cache, try database
	return db.blkExistsSha(sha)
}

// blkExistsSha looks up the given block hash
// returns true if it is present in the database.
// CALLED WITH LOCK HELD
func (db *ChainDb) blkExistsSha(sha *wire.Hash) (bool, error) {
	key := makeBlockShaKey(sha)

	return db.stor.Has(key)
}

// FetchBlockShaByHeight returns a block hash based on its height in the
// block chain.
func (db *ChainDb) FetchBlockShaByHeight(height uint64) (sha *wire.Hash, err error) {

	return db.fetchBlockShaByHeight(height)
}

// fetchBlockShaByHeight returns a block hash based on its height in the
// block chain.
func (db *ChainDb) fetchBlockShaByHeight(height uint64) (rsha *wire.Hash, err error) {
	key := makeBlockHeightKey(height)

	blkVal, err := db.stor.Get(key)
	if err != nil {
		logging.CPrint(logging.TRACE, "failed to find block on height", logging.LogFormat{"height": height})
		return // exists ???
	}

	var sha wire.Hash
	sha.SetBytes(blkVal[0:32])

	return &sha, nil
}

// FetchHeightRange looks up a range of blocks by the start and ending
// heights.  Fetch is inclusive of the start height and exclusive of the
// ending height.
func (db *ChainDb) FetchHeightRange(startHeight, endHeight uint64) ([]wire.Hash, error) {
	shalist := make([]wire.Hash, 0, endHeight-startHeight)
	for height := startHeight; height < endHeight; height++ {
		key := makeBlockHeightKey(height)

		blkVal, err := db.stor.Get(key)
		if err != nil {
			return nil, err
		}

		var sha wire.Hash
		sha.SetBytes(blkVal[0:32])
		shalist = append(shalist, sha)
	}

	return shalist, nil
}

// NewestSha returns the hash and block height of the most recent (end) block of
// the block chain.  It will return the zero hash, UnknownHeight for the block height, and
// no error (nil) if there are not any blocks in the database yet.
func (db *ChainDb) NewestSha() (rSha *wire.Hash, rBlkHeight uint64, err error) {

	if db.dbStorageMeta.currentHeight == UnknownHeight {
		return &wire.Hash{}, UnknownHeight, nil
	}
	sha := db.dbStorageMeta.currentHash

	return &sha, db.dbStorageMeta.currentHeight, nil
}

// transition code that will be removed soon
func (db *ChainDb) IndexPubkbl(rebuild bool) error {
	if db.dbStorageMeta.currentHeight == UnknownHeight {
		return nil
	}
	height, err := db.fetchPubkblIndexProgress()
	if err != nil {
		return err
	}
	logging.CPrint(logging.INFO, "build block-bl index start", logging.LogFormat{
		"current": height,
		"rebuild": rebuild,
	})

	if rebuild {
		err = db.deletePubkblIndexProgress()
		if err != nil {
			logging.CPrint(logging.ERROR, "deletePubkblIndexProgress error", logging.LogFormat{
				"err": err,
			})
			return err
		}
	}

	// If height is 0, make sure to clear residual pubkbl within last rebuild call.
	if rebuild || height == 0 {
		err = db.clearPubkbl()
		if err != nil {
			logging.CPrint(logging.ERROR, "clearPubkbl error", logging.LogFormat{
				"err": err,
			})
			return err
		}
	}

	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	i := uint64(0)
	if height != 0 {
		i = height + 1
	}
	for ; i <= db.dbStorageMeta.currentHeight; i++ {
		if i%1000 == 0 {
			logging.CPrint(logging.DEBUG, "updating pk & bl index", logging.LogFormat{
				"height": i,
			})
		}
		_, buf, err := db.getBlkByHeight(i)
		if err != nil {
			return err
		}
		block, err := massutil.NewBlockFromBytes(buf, wire.DB)
		if err != nil {
			return err
		}

		err = db.insertPubkbl(block.MsgBlock().Header.PubKey, block.MsgBlock().Header.Proof.BitLength, block.Height())
		if err != nil {
			return err
		}
		err = db.updatePubkblIndexProgress(block.Height())
		if err != nil {
			return err
		}
	}
	logging.CPrint(logging.INFO, "build block-bl index done", logging.LogFormat{
		"current": i - 1,
		"rebuild": rebuild,
	})
	return db.deletePubkblIndexProgress()
}
