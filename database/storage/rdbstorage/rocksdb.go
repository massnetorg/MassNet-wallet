// +build darwin linux
// +build rocksdb

package rdbstorage

import (
	"runtime"

	"massnet.org/mass-wallet/database/storage"
	"massnet.org/mass-wallet/logging"

	"github.com/tecbot/gorocksdb"
)

type rocksDB struct {
	db *gorocksdb.DB

	ro *gorocksdb.ReadOptions
	wo *gorocksdb.WriteOptions
}

type rocksBatch struct {
	b *gorocksdb.WriteBatch
}

type rocksIterator struct {
	started bool
	slice   *storage.Range
	iter    *gorocksdb.Iterator
}

func init() {
	storage.RegisterDriver(storage.StorageDriver{
		DbType:        "rocksdb",
		OpenStorage:   OpenDB,
		CreateStorage: CreateDB,
	})
}

func CreateDB(path string, args ...interface{}) (storage.Storage, error) {
	return newRocksDB(path, true)
}

func OpenDB(path string, args ...interface{}) (storage.Storage, error) {
	return newRocksDB(path, false)
}

func newRocksDB(path string, create bool) (storage.Storage, error) {

	filter := gorocksdb.NewBloomFilter(10)
	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetFilterPolicy(filter)
	cache := gorocksdb.NewLRUCache(512 << 20)
	bbto.SetBlockCache(cache)
	bbto.SetBlockSize(16 * storage.KiB)

	opts := gorocksdb.NewDefaultOptions()
	opts.SetBlockBasedTableFactory(bbto)
	if create {
		opts.SetCreateIfMissing(true)
		opts.SetErrorIfExists(true)
	}
	opts.SetMaxOpenFiles(500)
	opts.SetWriteBufferSize(64 * storage.MiB)
	opts.IncreaseParallelism(runtime.NumCPU())

	db, err := gorocksdb.OpenDb(opts, path)
	if err != nil {
		logging.CPrint(logging.ERROR, "init rocksdb error", logging.LogFormat{
			"path":   path,
			"create": create,
			"err":    err,
		})
		return nil, err
	}

	rdb := &rocksDB{
		db: db,
		ro: gorocksdb.NewDefaultReadOptions(),
		wo: gorocksdb.NewDefaultWriteOptions(),
	}
	logging.CPrint(logging.INFO, "init rocksdb", logging.LogFormat{
		"path":   path,
		"create": create,
	})
	return rdb, nil
}

func (r *rocksDB) Close() error {
	r.db.Close()
	return nil
}

func (r *rocksDB) Get(key []byte) ([]byte, error) {
	value, err := r.db.Get(r.ro, key)
	if err != nil {
		return nil, err
	}
	defer value.Free()
	if !value.Exists() {
		return nil, storage.ErrNotFound
	}
	ret := make([]byte, value.Size(), value.Size())
	copy(ret, value.Data())
	return ret, nil
}

func (r *rocksDB) Put(key, value []byte) error {
	if len(key) == 0 {
		return storage.ErrInvalidKey
	}
	return r.db.Put(r.wo, key, value)
}

func (r *rocksDB) Has(key []byte) (bool, error) {
	_, err := r.Get(key)
	if err != nil {
		if err == storage.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *rocksDB) Delete(key []byte) error {
	return r.db.Delete(r.wo, key)
}

func (r *rocksDB) NewBatch() storage.Batch {
	return &rocksBatch{
		b: gorocksdb.NewWriteBatch(),
	}
}

func (r *rocksDB) Write(batch storage.Batch) error {
	rb, ok := batch.(*rocksBatch)
	if !ok {
		return storage.ErrInvalidBatch
	}
	return r.db.Write(r.wo, rb.b)
}

func (r *rocksDB) NewIterator(slice *storage.Range) storage.Iterator {
	if slice == nil {
		slice = &storage.Range{}
	} else {
		if len(slice.Start) == 0 {
			slice.Start = nil
		}
		if len(slice.Limit) == 0 {
			slice.Limit = nil
		}
	}
	ro := gorocksdb.NewDefaultReadOptions()
	ro.SetIterateUpperBound(slice.Limit)
	return &rocksIterator{
		started: false,
		slice:   slice,
		iter:    r.db.NewIterator(ro),
	}
}

// -------------rocksBatch-------------

func (b *rocksBatch) Put(key, value []byte) error {
	if len(key) == 0 {
		return storage.ErrInvalidKey
	}
	b.b.Put(key, value)
	return nil
}

func (b *rocksBatch) Delete(key []byte) error {
	if len(key) == 0 {
		return storage.ErrInvalidKey
	}
	b.b.Delete(key)
	return nil
}

func (b *rocksBatch) Reset() {
	b.b.Clear()
}

func (b *rocksBatch) Release() {
	b.b.Destroy()
	b.b = nil
}

// -----------------rocksIterator-----------------

func (it *rocksIterator) Seek(key []byte) bool {
	it.iter.Seek(key)
	it.started = true
	return it.iter.Valid()
}

func (it *rocksIterator) Next() bool {
	if !it.started {
		it.iter.Seek(it.slice.Start)
		it.started = true
	} else {
		it.iter.Next()
	}
	return it.iter.Valid()
}

func (it *rocksIterator) Key() []byte {
	k := it.iter.Key()
	defer k.Free()
	if k.Size() == 0 {
		return nil
	}
	ret := make([]byte, k.Size(), k.Size())
	copy(ret, k.Data())
	return ret
}

func (it *rocksIterator) Value() []byte {
	v := it.iter.Value()
	defer v.Free()
	if v.Size() == 0 {
		return nil
	}
	ret := make([]byte, v.Size(), v.Size())
	copy(ret, v.Data())
	return ret
}

func (it *rocksIterator) Release() {
	it.iter.Close()
}

func (it *rocksIterator) Error() error {
	return it.iter.Err()
}
