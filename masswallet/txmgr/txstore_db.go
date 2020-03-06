package txmgr

import (
	"encoding/binary"
	"fmt"
	"time"

	"massnet.org/mass-wallet/logging"
	mwdb "massnet.org/mass-wallet/masswallet/db"
	"massnet.org/mass-wallet/wire"
)

func existsRawUnmined(ns mwdb.Bucket, k []byte) ([]byte, error) {
	v, err := ns.Get(k)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func putRawUnmined(ns mwdb.Bucket, k, v []byte) error {
	return ns.Put(k, v)
}

func deleteRawUnmined(ns mwdb.Bucket, k []byte) error {
	return ns.Delete(k)
}

func valueTxRecord(rec *TxRecord) ([]byte, error) {
	bs, err := rec.MsgTx.Bytes(wire.DB)
	if err != nil {
		return nil, err
	}
	v := make([]byte, 8+len(bs))
	copy(v[8:], bs)
	binary.BigEndian.PutUint64(v, uint64(rec.Received.Unix()))
	return v, nil
}

func keyTxRecord(txHash *wire.Hash, block *BlockMeta) []byte {
	k := make([]byte, 72)
	copy(k, txHash[:])
	binary.BigEndian.PutUint64(k[32:40], block.Height)
	copy(k[40:72], block.Hash[:])
	return k
}

func putTxRecord(ns mwdb.Bucket, rec *TxRecord, block *BlockMeta) error {
	k := keyTxRecord(&rec.Hash, block)
	v, err := valueTxRecord(rec)
	if err != nil {
		return err
	}
	err = ns.Put(k, v)
	if err != nil {
		return fmt.Errorf("failed to put tx record: %v, err: %v", rec.Hash, err)
	}
	return nil
}

func existsTxRecord(ns mwdb.Bucket, txHash *wire.Hash, block *BlockMeta) (k, v []byte) {
	k = keyTxRecord(txHash, block)
	v, _ = ns.Get(k)
	return
}

func existsRawTxRecord(ns mwdb.Bucket, k []byte) ([]byte, error) {
	return ns.Get(k)
}

func fetchRawTxRecordByTxHashHeight(ns mwdb.Bucket, txHash *wire.Hash, height uint64) ([]byte, error) {
	if len(txHash) != 32 {
		return nil, fmt.Errorf("short hash value (expected 32 bytes, read %d)", len(txHash))
	}
	prefix := make([]byte, 40)
	copy(prefix[0:32], txHash[:])
	binary.BigEndian.PutUint64(prefix[32:40], height)
	entries, err := ns.GetByPrefix(prefix)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		return entry.Value, nil
	}
	return nil, nil
}

func fetchRawTxRecordByHashHeight(ns mwdb.Bucket, txHash *wire.Hash, height uint64) (*mwdb.Entry, error) {
	if len(txHash) != 32 || height == 0 {
		return nil, fmt.Errorf("short hash value (expected 32 bytes, read %d) "+
			"or invalid height %d", len(txHash), height)
	}
	prefix := make([]byte, 40)
	copy(prefix[0:32], txHash[:])
	binary.BigEndian.PutUint64(prefix[32:], height)
	entries, err := ns.GetByPrefix(prefix)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}
	if len(entries) > 1 {
		logging.CPrint(logging.ERROR, "duplicate tx record",
			logging.LogFormat{
				"tx":     txHash.String(),
				"height": height,
			})
		return nil, fmt.Errorf("duplicate tx record (tx %s, height %d)",
			txHash.String(), height)
	}
	return entries[0], nil
}

func fetchLatestRawTxRecordOfHash(ns mwdb.Bucket, txHash *wire.Hash) (*mwdb.Entry, error) {
	if len(txHash) != 32 {
		return nil, fmt.Errorf("short hash value (expected 32 bytes, read %d)", len(txHash))
	}
	entries, err := ns.GetByPrefix(txHash[:])
	if err != nil {
		return nil, err
	}

	var found *mwdb.Entry
	foundHeight := uint64(0)
	for _, entry := range entries {
		h := binary.BigEndian.Uint64(entry.Key[32:40])
		if h > foundHeight {
			foundHeight = h
			found = entry
		}
	}
	return found, nil
}

// func existsRawTxRecord(ns mwdb.Bucket, k []byte) (v []byte) {
// 	v, _ = ns.Get(k)
// 	return
// }

func deleteTxRecord(ns mwdb.Bucket, txHash *wire.Hash, block *BlockMeta) error {
	k := keyTxRecord(txHash, block)
	return ns.Delete(k)
}

func deleteRawTxRecord(ns mwdb.Bucket, k []byte) error {
	if len(k) != 72 {
		return fmt.Errorf("short k (expected %d bytes, read %d)", 72, len(k))
	}
	return ns.Delete(k)
}

func readTxRecordKey(k []byte) (height uint64, blkHash []byte, err error) {
	if len(k) < 72 {
		return 0, nil, fmt.Errorf("short k value (expected %d bytes, read %d)", 72, len(k))
	}
	return binary.BigEndian.Uint64(k[32:40]), k[40:72], nil
}

func readRawTxRecordValue(v []byte, rec *TxRecord) error {
	if len(v) < 8 {
		return fmt.Errorf("%s: short read (expected %d bytes, read %d)",
			bucketTxRecords, 8, len(v))
	}
	rec.Received = time.Unix(int64(binary.BigEndian.Uint64(v)), 0)
	err := rec.MsgTx.SetBytes(v[8:], wire.DB)
	if err != nil {
		return fmt.Errorf("failed to deserialize transaction: %v", err)
	}
	return nil
}

func keyBlockRecord(height uint64) []byte {
	k := make([]byte, 8)
	binary.BigEndian.PutUint64(k, height)
	return k
}

func valueBlockRecord(block *BlockMeta, txHash *wire.Hash) []byte {
	v := make([]byte, 76)
	copy(v, block.Hash[:])
	binary.BigEndian.PutUint64(v[32:40], uint64(block.Timestamp.Unix()))
	binary.BigEndian.PutUint32(v[40:44], 1)
	copy(v[44:76], txHash[:])
	return v
}

// appendRawBlockRecord returns a new block record value with a transaction
// hash appended to the end and an incremented number of transactions.
func appendRawBlockRecord(v []byte, txHash *wire.Hash) ([]byte, error) {
	if len(v) < 44 {
		return nil, fmt.Errorf("%s: short read (expected %d bytes, read %d)",
			bucketBlocks, 44, len(v))
	}
	newv := append(v[:len(v):len(v)], txHash[:]...)
	n := binary.BigEndian.Uint32(newv[40:44])
	binary.BigEndian.PutUint32(newv[40:44], n+1)
	return newv, nil
}

func putRawBlockRecord(ns mwdb.Bucket, k, v []byte) error {
	return ns.Put(k, v)
}

func putBlockRecord(ns mwdb.Bucket, block *BlockMeta, txHash *wire.Hash) error {
	k := keyBlockRecord(block.Height)
	v := valueBlockRecord(block, txHash)
	return putRawBlockRecord(ns, k, v)
}

func updateBlockRecord(ns mwdb.Bucket, block *BlockMeta, txHashes []wire.Hash) (err error) {
	k := keyBlockRecord(block.Height)
	v := valueBlockRecord(block, &txHashes[0])
	for i := 1; i < len(txHashes); i++ {
		v, err = appendRawBlockRecord(v, &txHashes[i])
		if err != nil {
			return
		}
	}
	return putRawBlockRecord(ns, k, v)
}

func existsBlockRecord(ns mwdb.Bucket, height uint64) (k, v []byte, err error) {
	k = keyBlockRecord(height)
	v, err = ns.Get(k)
	return
}

func deleteBlockRecord(ns mwdb.Bucket, height uint64) error {
	k := keyBlockRecord(height)
	return ns.Delete(k)
}

func readBlockHashFromValue(v []byte) (blkHash wire.Hash, err error) {
	if len(v) < 44 {
		return wire.Hash{}, fmt.Errorf("nsBlocks: invalid value length %d", len(v))
	}
	copy(blkHash[:], v[0:32])
	return
}

func readRawBlockRecord(k, v []byte, block *blockRecord) error {
	if len(k) < 8 {
		return fmt.Errorf("%s: short key (expected %d bytes, read %d)",
			bucketBlocks, 8, len(k))
	}
	if len(v) < 44 {
		return fmt.Errorf("%s: short value (expected %d bytes, read %d)",
			bucketBlocks, 44, len(v))
	}
	numTransactions := int(binary.BigEndian.Uint32(v[40:44]))
	expectedLen := 44 + wire.HashSize*numTransactions
	if len(v) < expectedLen {
		return fmt.Errorf("%s: short read (expected %d bytes, read %d)",
			bucketBlocks, expectedLen, len(v))
	}

	block.Height = binary.BigEndian.Uint64(k)
	copy(block.Hash[:], v)
	block.Timestamp = time.Unix(int64(binary.BigEndian.Uint64(v[32:40])), 0)
	block.transactions = make([]wire.Hash, numTransactions)
	off := 44
	for i := range block.transactions {
		copy(block.transactions[i][:], v[off:])
		off += wire.HashSize
	}

	return nil
}

func fetchBlockRecord(ns mwdb.Bucket, height uint64) (block *blockRecord, err error) {
	k, v, err := existsBlockRecord(ns, height)
	if err != nil {
		return
	}
	if v == nil {
		return
	}
	block = &blockRecord{}
	err = readRawBlockRecord(k, v, block)
	if err != nil {
		block = nil
	}
	return
}
