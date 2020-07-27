package ldb

import "massnet.org/mass-wallet/wire"

func (db *ChainDb) GetBlkLocByHeight(height uint64) (sha *wire.Hash, fileNo uint32, offset int64, size int64, err error) {
	return db.getBlkLocByHeight(height)
}
