package txmgr

import (
	"errors"

	mwdb "massnet.org/mass-wallet/masswallet/db"
)

func fetchAllEntry(ns mwdb.Bucket) ([]*mwdb.Entry, error) {
	return ns.GetByPrefix(nil)
}

func existsValue(bucket mwdb.Bucket, k []byte) ([]byte, error) {
	if len(k) == 0 {
		return nil, errors.New("existsValue: empty key not allowed")
	}
	return bucket.Get(k)
}

func deleteByKey(bucket mwdb.Bucket, k []byte) error {
	if len(k) == 0 {
		return errors.New("deleteByKey: empty key not allowed")
	}
	return bucket.Delete(k)
}

func clearBucket(bucket mwdb.Bucket) error {
	return bucket.Clear()
}
