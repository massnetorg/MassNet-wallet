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

func deleteKey(bucket mwdb.Bucket, k []byte) error {
	if len(k) == 0 {
		return errors.New("deleteKey: empty key not allowed")
	}
	return bucket.Delete(k)
}

func putKeyValue(bucket mwdb.Bucket, k, v []byte) error {
	if len(k) == 0 || len(v) == 0 {
		return errors.New("putKeyValue: empty key/value not allowed")
	}
	return bucket.Put(k, v)
}

func clearBucket(bucket mwdb.Bucket) error {
	return bucket.Clear()
}
