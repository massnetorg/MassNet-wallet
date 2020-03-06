package ldb

import (
	"bytes"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/syndtr/goleveldb/leveldb/iterator"

	"massnet.org/mass-wallet/masswallet/db"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"massnet.org/mass-wallet/logging"
)

const (

	// e.g.
	// 1. top level bucket (created in LevelDB) index entry is like (the 2nd number means bucket depth, just '1'):
	//          <b_1_top1, top1>
	//          <b_1_top2, top2>
	//
	//	  the k/v entry in top level bucket is like (the 1st number means bucket depth):
	//					<1_top1_key1, value1>
	//					<1_top1_key2, value2>
	//
	//
	// 2. sub bucket (created in Bucket) index entry is like (the 2nd number means bucket depth, from '2' on):
	//          <b_2_top1_sub1, sub1>
	//          <b_2_top1_sub2, sub2>
	//
	//	  the k/v entry in sub bucket is like:
	//					<2_top1_sub1_key1, value1>
	//					<2_top1_sub2_key1, value1>
	//
	bucketNameBucket    = "b"
	bucketPathSep       = "_"
	topLevelBucketDepth = "1"

	maxBucketNameLen = 256
)

var (
	innerBatch *leveldb.Batch
)

// LevelDB ...
type LevelDB struct {
	ldb  *leveldb.DB
	muTr sync.Mutex
}

type batchPutValue struct {
	data []byte
	seq  uint32
}

type batch struct {
	seqNo   uint32
	b       *leveldb.Batch
	puts    map[string]*batchPutValue
	deletes map[string]uint32
}

type transaction struct {
	readOnly bool
	b        *batch
	l        *LevelDB
}

func newBatch() *batch {
	if innerBatch == nil {
		innerBatch = new(leveldb.Batch)
		// increase memory in advance
		paddingKey := make([]byte, opt.KiB, opt.KiB)
		paddingValue := make([]byte, 64*opt.MiB, 64*opt.MiB)
		innerBatch.Put(paddingKey, paddingValue)
	}
	innerBatch.Reset()
	return &batch{
		b:       innerBatch,
		puts:    make(map[string]*batchPutValue),
		deletes: make(map[string]uint32),
	}
}

func (b *batch) Get(k []byte) (v []byte, deleted bool) {

	mapK := string(k)
	seqDel, okDel := b.deletes[mapK]
	putV, okPut := b.puts[mapK]

	if okDel {
		if !okPut || seqDel > putV.seq {
			return nil, true
		}
		return putV.data, false
	}
	if okPut {
		return putV.data, false
	}
	return nil, false
}

func (b *batch) GetNetPutsByPrefix(prefix []byte) map[string][]byte {
	result := make(map[string][]byte)
	for k, v := range b.puts {
		if len(prefix) != 0 && !bytes.HasPrefix([]byte(k), prefix) {
			continue
		}
		seqDel, ok := b.deletes[k]
		if !ok || v.seq > seqDel {
			result[k] = v.data
		}
	}
	return result
}

func (b *batch) Put(k, v []byte) {

	b.b.Put(k, v)
	b.seqNo++
	b.puts[string(k)] = &batchPutValue{
		data: v,
		seq:  b.seqNo,
	}
}

func (b *batch) Delete(k []byte) {
	b.b.Delete(k)
	b.seqNo++
	b.deletes[string(k)] = b.seqNo
}

func joinBucketPath(arr ...string) string {
	return strings.Join(arr, bucketPathSep)
}

// Close
// TODO: It is not safe to close a DB until all outstanding iterators are released.
func (l *LevelDB) Close() error {
	return l.ldb.Close()
}

// BeginTx ...
func (l *LevelDB) BeginTx() (db.DBTransaction, error) {
	l.muTr.Lock()

	return &transaction{
		readOnly: false,
		b:        newBatch(),
		l:        l,
	}, nil
}

// BeginReadTx ...
func (l *LevelDB) BeginReadTx() (db.ReadTransaction, error) {
	return &transaction{
		readOnly: true,
		l:        l,
	}, nil
}

// TopLevelBucket ...
func (tx *transaction) TopLevelBucket(name string) db.Bucket {
	bucketPath := joinBucketPath(topLevelBucketDepth, name)
	key := []byte(joinBucketPath(bucketNameBucket, bucketPath))

	_, err := tx.l.ldb.Get(key, nil)
	if !tx.readOnly && err == leveldb.ErrNotFound {
		if v, _ := tx.b.Get(key); v != nil {
			err = nil
		}
	}
	if err != nil {
		return nil
	}
	//TOCONFIRM: check value == name

	return &levelBucket{
		tx:      tx,
		name:    name,
		path:    bucketPath,
		pathLen: len(bucketPath),
		depth:   1,
	}
}

// BucketNames ...
func (tx *transaction) BucketNames() (names []string, err error) {

	prefix := []byte(joinBucketPath(bucketNameBucket, topLevelBucketDepth, ""))

	iter := tx.l.ldb.NewIterator(util.BytesPrefix(prefix), nil)
	defer iter.Release()

	names = make([]string, 0)
	set := make(map[string]struct{})
	for iter.Next() {
		key := string(iter.Key())
		value := string(iter.Value())

		if !tx.readOnly {
			v, deleted := tx.b.Get([]byte(key))
			if deleted {
				continue
			}
			if v != nil {
				value = string(v)
			}
		}

		ss := strings.Split(key, bucketPathSep)
		if len(ss) != 3 || ss[2] != value {
			return nil, db.ErrIllegalValue
		}
		names = append(names, value)
		set[value] = struct{}{}
	}
	if err := iter.Error(); err != nil {
		return nil, err
	}

	if !tx.readOnly {
		for k, v := range tx.b.GetNetPutsByPrefix(prefix) {
			key := k
			value := string(v)
			ss := strings.Split(key, bucketPathSep)
			if len(ss) != 3 || ss[2] != value {
				return nil, db.ErrIllegalValue
			}

			if _, ok := set[value]; ok {
				continue
			}
			names = append(names, value)
			set[value] = struct{}{}
		}
	}

	return names, nil
}

// FetchBucket ...
func (tx *transaction) FetchBucket(meta db.BucketMeta) db.Bucket {
	if meta == nil {
		return nil
	}
	path := joinBucketPath(meta.Paths()...)
	key := []byte(joinBucketPath(bucketNameBucket, path))

	_, err := tx.l.ldb.Get(key, nil)
	if !tx.readOnly && err == leveldb.ErrNotFound {
		if v, _ := tx.b.Get(key); v != nil {
			err = nil
		}
	}
	if err != nil {
		return nil
	}
	//TOCONFIRM: check value == name

	return &levelBucket{
		tx:      tx,
		name:    meta.Name(),
		path:    path,
		pathLen: len(path),
		depth:   meta.Depth(),
	}
}

// CreateTopLevelBucket ...
func (tx *transaction) CreateTopLevelBucket(name string) (db.Bucket, error) {
	if tx.readOnly {
		return nil, db.ErrWriteNotAllowed
	}

	if !isValidBucketName(name) {
		return nil, db.ErrInvalidBucketName
	}

	bucketPath := joinBucketPath(topLevelBucketDepth, name)
	key := []byte(joinBucketPath(bucketNameBucket, bucketPath))

	_, err := tx.l.ldb.Get(key, nil)
	if err == nil {
		_, deleted := tx.b.Get(key)
		if !deleted {
			return nil, db.ErrBucketExist
		} else {
			err = leveldb.ErrNotFound
		}
	}
	if err != leveldb.ErrNotFound {
		return nil, err
	}

	bucket := &levelBucket{
		tx:      tx,
		name:    name,
		path:    bucketPath,
		pathLen: len(bucketPath),
		depth:   1,
	}

	tx.b.Put(key, []byte(name))
	logging.VPrint(logging.DEBUG, "new top level bucket",
		logging.LogFormat{
			"bucket": string(key),
		})

	return bucket, nil
}

// DeleteTopLevelBucket ...
func (tx *transaction) DeleteTopLevelBucket(name string) error {
	return db.ErrNotSupported
}

// Rollback ...
func (tx *transaction) Rollback() error {
	if !tx.readOnly {
		tx.l.muTr.Unlock()
	}
	return nil
}

// Commit ...
func (tx *transaction) Commit() error {
	if tx.readOnly {
		return nil
	}
	err := tx.l.ldb.Write(tx.b.b, nil)
	tx.l.muTr.Unlock()
	return err
}

// levelBucket ...
type levelBucket struct {
	tx      *transaction
	name    string
	path    string
	pathLen int
	depth   int
}

// NewBucket create sub bucket
func (b *levelBucket) NewBucket(name string) (db.Bucket, error) {
	if b.tx.readOnly {
		return nil, db.ErrWriteNotAllowed
	}

	sub, err := b.subBucket(name)
	if err != nil {
		return nil, err
	}

	key := []byte(joinBucketPath(bucketNameBucket, sub.path))
	_, err = b.tx.l.ldb.Get(key, nil) // value == name
	if err == nil {
		_, deleted := b.tx.b.Get(key)
		if !deleted {
			return nil, db.ErrBucketExist
		} else {
			err = leveldb.ErrNotFound
		}
	}
	if err != leveldb.ErrNotFound {
		return nil, err
	}
	// if string(v) == name {
	// 	return nil, ErrBucketExist
	// }

	b.tx.b.Put(key, []byte(name))
	logging.VPrint(logging.DEBUG, "new sub bucket",
		logging.LogFormat{
			"bucket": string(key),
		})
	return sub, nil
}

// Bucket ...
func (b *levelBucket) Bucket(name string) db.Bucket {
	sub, err := b.subBucket(name)
	if err != nil {
		return nil
	}

	key := []byte(joinBucketPath(bucketNameBucket, sub.path))

	_, err = b.tx.l.ldb.Get(key, nil)
	if !b.tx.readOnly && err == leveldb.ErrNotFound {
		if v, _ := b.tx.b.Get(key); v != nil {
			err = nil
		}
	}
	if err != nil {
		return nil
	}
	// if err != nil || string(value) != name {
	// 	if !b.tx.readOnly {
	// 		value, _ = b.tx.b.Get(key)
	// 		if string(value) != name {
	// 			return nil
	// 		}
	// 	} else {
	// 		return nil
	// 	}
	// }
	return sub
}

func (b *levelBucket) subBucket(name string) (*levelBucket, error) {
	if !isValidBucketName(name) {
		return nil, db.ErrInvalidBucketName
	}
	sub := &levelBucket{
		tx:    b.tx,
		name:  name,
		depth: b.depth + 1,
	}
	ss := strings.Split(b.path, bucketPathSep)
	if len(ss) < 2 {
		return nil, db.ErrIllegalBucketPath
	}
	// e.g.  1_top  -->  2_top_child
	ss[0] = strconv.Itoa(sub.depth)
	ss = append(ss, sub.name)
	sub.path = joinBucketPath(ss...)
	sub.pathLen = len(sub.path)
	return sub, nil
}

// BucketNames ...
func (b *levelBucket) BucketNames() (names []string, err error) {
	ss := strings.Split(b.path, bucketPathSep)
	if len(ss) < 2 {
		return nil, db.ErrIllegalBucketPath
	}

	ss[0] = strconv.Itoa(b.depth + 1)
	ss = append(ss, "")
	prefix := []byte(joinBucketPath(bucketNameBucket, joinBucketPath(ss...)))

	iter := b.tx.l.ldb.NewIterator(util.BytesPrefix(prefix), nil)
	defer iter.Release()

	names = make([]string, 0)
	set := make(map[string]struct{})
	for iter.Next() {
		key := string(iter.Key())
		value := string(iter.Value())
		if !b.tx.readOnly {
			v, deleted := b.tx.b.Get([]byte(key))
			if deleted {
				continue
			}
			if v != nil {
				value = string(v)
			}
		}
		ss := strings.Split(key, bucketPathSep)
		if len(ss) != b.depth+3 || ss[b.depth+2] != value {
			return nil, db.ErrIllegalValue
		}
		names = append(names, value)
		set[value] = struct{}{}
	}

	if !b.tx.readOnly {
		for k, v := range b.tx.b.GetNetPutsByPrefix(prefix) {
			key := k
			value := string(v)
			ss := strings.Split(key, bucketPathSep)
			if len(ss) != b.depth+3 || ss[b.depth+2] != value {
				return nil, db.ErrIllegalValue
			}
			if _, ok := set[value]; ok {
				continue
			}
			names = append(names, value)
			set[value] = struct{}{}
		}
	}

	if err := iter.Error(); err != nil {
		return nil, err
	}
	return names, nil
}

// DeleteBucket ...
func (b *levelBucket) DeleteBucket(name string) error {
	if b.tx.readOnly {
		return db.ErrWriteNotAllowed
	}

	sub := b.Bucket(name)
	if sub == nil {
		return nil
	}
	return deleteBucket(sub.(*levelBucket))
}

func deleteBucket(b *levelBucket) error {
	if b.depth == 1 {
		return db.ErrNotSupported
	}

	// delete sub bucket
	subnames, err := b.BucketNames()
	if err != nil {
		return err
	}
	for _, subname := range subnames {
		sub := b.Bucket(subname)
		if sub == nil {
			continue
		}
		err = deleteBucket(sub.(*levelBucket))
		if err != nil {
			return err
		}
	}

	// delete k/v in bucket
	prefix := []byte(joinBucketPath(b.path, ""))
	iter := b.tx.l.ldb.NewIterator(util.BytesPrefix(prefix), nil)
	for iter.Next() {
		_, deleted := b.tx.b.Get(iter.Key())
		if deleted {
			continue
		}
		b.tx.b.Delete(iter.Key())
		logging.VPrint(logging.DEBUG, "deleting bucket item",
			logging.LogFormat{
				"key": string(iter.Key()),
			})
	}
	iter.Release()
	err = iter.Error()
	if err != nil {
		return err
	}
	for k, _ := range b.tx.b.GetNetPutsByPrefix(prefix) {
		b.tx.b.Delete([]byte(k))
	}

	// delete bucket name
	key := joinBucketPath(bucketNameBucket, b.path)
	logging.VPrint(logging.DEBUG, "deleting bucket",
		logging.LogFormat{
			"bucket": key,
		})
	b.tx.b.Delete([]byte(key))
	return nil
}

// Close ...
func (b *levelBucket) Close() error {
	// noting to do in sub bucket instance
	return nil
}

// Put ...
func (b *levelBucket) Put(key, value []byte) error {
	if b.tx.readOnly {
		return db.ErrWriteNotAllowed
	}
	if len(value) == 0 {
		return db.ErrIllegalValue
	}
	key, err := b.innerKey(key, false)
	if err != nil {
		return err
	}
	b.tx.b.Put(key, value)
	return nil
}

// Get returns nil if not found
func (b *levelBucket) Get(key []byte) ([]byte, error) {
	key, err := b.innerKey(key, false)
	if err != nil {
		return nil, nil
	}

	value, err := b.tx.l.ldb.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			if b.tx.readOnly {
				return nil, nil
			}
			value, _ = b.tx.b.Get(key)
		} else {
			return nil, err
		}
	} else {
		if !b.tx.readOnly {
			v, deleted := b.tx.b.Get(key)
			if deleted {
				return nil, nil
			}
			if v != nil {
				value = v
			}
		}
	}
	return value, nil
}

// Delete ...
func (b *levelBucket) Delete(key []byte) error {
	if b.tx.readOnly {
		return db.ErrWriteNotAllowed
	}
	key, err := b.innerKey(key, false)
	if err != nil {
		return nil
	}
	b.tx.b.Delete(key)
	return nil
}

// Clear ...
func (b *levelBucket) Clear() error {
	if b.tx.readOnly {
		return db.ErrWriteNotAllowed
	}
	prefix := []byte(joinBucketPath(b.path, ""))

	iter := b.tx.l.ldb.NewIterator(util.BytesPrefix(prefix), nil)
	defer iter.Release()

	for iter.Next() {
		_, deleted := b.tx.b.Get(iter.Key())
		if deleted {
			continue
		}
		b.tx.b.Delete(iter.Key())
	}

	for k, _ := range b.tx.b.GetNetPutsByPrefix(prefix) {
		b.tx.b.Delete([]byte(k))
	}

	logging.VPrint(logging.DEBUG, "clear bucket item",
		logging.LogFormat{
			"key":    string(iter.Key()),
			"bucket": b.path,
		})
	return iter.Error()
}

// GetByPrefix ...
func (b *levelBucket) GetByPrefix(prefix []byte) ([]*db.Entry, error) {
	innerPrefix, err := b.innerKey(prefix, true)
	if err != nil {
		return nil, nil
	}
	entries := make([]*db.Entry, 0)
	set := make(map[string]struct{})

	iter := b.tx.l.ldb.NewIterator(util.BytesPrefix(innerPrefix), nil)
	defer iter.Release()

	for iter.Next() {
		key := iter.Key()[b.pathLen+1:]
		value := iter.Value()[:]
		if !b.tx.readOnly {
			v, deleted := b.tx.b.Get(iter.Key())
			if deleted {
				continue
			}
			if v != nil {
				value = v
			}
		}
		entry := &db.Entry{
			Key:   make([]byte, len(key)),
			Value: make([]byte, len(value)),
		}

		copy(entry.Key, key)
		copy(entry.Value, value)
		entries = append(entries, entry)
		set[string(iter.Key())] = struct{}{}
	}

	if !b.tx.readOnly {
		for k, value := range b.tx.b.GetNetPutsByPrefix(innerPrefix) {
			if _, ok := set[k]; ok {
				continue
			}
			key := []byte(k)[b.pathLen+1:]
			entry := &db.Entry{
				Key:   make([]byte, len(key)),
				Value: make([]byte, len(value)),
			}

			copy(entry.Key, key)
			copy(entry.Value, value)
			entries = append(entries, entry)
			set[k] = struct{}{}
		}
	}

	err = iter.Error()
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// GetBucketMeta ...
func (b *levelBucket) GetBucketMeta() db.BucketMeta {
	return &levelBucketMeta{
		paths: strings.Split(b.path, bucketPathSep),
	}
}

type levelBucketMeta struct {
	paths []string
}

// Paths ...
func (m *levelBucketMeta) Paths() []string {
	return m.paths
}

// Name ...
func (m *levelBucketMeta) Name() string {
	return m.paths[m.Depth()]
}

// Depth ...
func (m *levelBucketMeta) Depth() int {
	depth, _ := strconv.Atoi(m.paths[0])
	return depth
}

func (b *levelBucket) innerKey(key []byte, asPrefix bool) ([]byte, error) {
	kl := len(key)
	if !asPrefix && kl == 0 {
		return nil, db.ErrIllegalKey
	}
	buf := make([]byte, b.pathLen+kl+1)
	copy(buf[:], []byte(b.path))
	copy(buf[b.pathLen:b.pathLen+1], []byte(bucketPathSep))
	if kl > 0 {
		copy(buf[b.pathLen+1:], key)
	}
	return buf, nil
}

// is valid bucket name
func isValidBucketName(name string) bool {
	return len(name) > 0 && len(name) <= maxBucketNameLen && strings.Index(name, bucketPathSep) < 0
}

func (b *levelBucket) innerKeyForIterator(key []byte) []byte {
	kl := len(key)
	buf := make([]byte, b.pathLen+kl+1)
	copy(buf[:], []byte(b.path))
	copy(buf[b.pathLen:b.pathLen+1], []byte(bucketPathSep))
	if kl > 0 {
		copy(buf[b.pathLen+1:], key)
	}
	return buf
}

// ------------------ batchIterator -------------------- //
type batchIterator struct {
	ptr   int
	keys  []string
	m     map[string][]byte
	start []byte
	limit []byte
}

func newBatchIterator(b *batch, start, limit []byte) *batchIterator {
	items := b.GetNetPutsByPrefix(nil)
	it := &batchIterator{
		ptr:   -1,
		keys:  make([]string, 0, len(items)),
		m:     make(map[string][]byte),
		start: start,
		limit: limit,
	}
	for k, v := range items {
		it.keys = append(it.keys, k)
		it.m[k] = v
	}
	sort.Slice(it.keys, func(i, j int) bool {
		return bytes.Compare([]byte(it.keys[i]), []byte(it.keys[j])) < 0
	})
	return it
}

func (bi *batchIterator) Seek(seekKey []byte) bool {
	bi.start = seekKey
	for i, key := range bi.keys {
		if bytes.Compare([]byte(key), bi.start) >= 0 && bytes.Compare([]byte(key), bi.limit) < 0 {
			bi.ptr = i
			return true
		}
	}
	bi.ptr = len(bi.keys)
	return false
}

func (bi *batchIterator) Next() bool {
	for i := bi.ptr + 1; i < len(bi.keys); i++ {
		key := bi.keys[i]
		if bytes.Compare([]byte(key), bi.start) >= 0 &&
			bytes.Compare([]byte(key), bi.limit) < 0 {
			bi.ptr = i
			return true
		}
	}
	bi.ptr = len(bi.keys)
	return false
}

func (bi *batchIterator) End() bool {
	return bi.ptr >= len(bi.keys)
}

func (bi *batchIterator) Key() []byte {
	if bi.ptr < 0 || bi.ptr >= len(bi.keys) {
		return nil
	}
	return []byte(bi.keys[bi.ptr])
}

func (bi *batchIterator) Value() []byte {
	if bi.ptr < 0 || bi.ptr >= len(bi.keys) {
		return []byte{}
	}
	return bi.m[bi.keys[bi.ptr]]
}

func (bi *batchIterator) Reset(start []byte) {
	bi.ptr = -1
	bi.start = start
}

// ------------------ levelIterator -------------------- //
type levelIterator struct {
	iter      iterator.Iterator
	iterEnd   bool
	batchIter *batchIterator
	slice     *db.Range
	b         *levelBucket
}

func (b *levelBucket) NewIterator(slice *db.Range) db.Iterator {
	if slice == nil {
		slice = &db.Range{}
	}
	slice.Start = b.innerKeyForIterator(slice.Start)
	if len(slice.Limit) == 0 {
		limit := b.innerKeyForIterator(slice.Limit)
		slice.Limit = db.BytesPrefix(limit).Limit
	} else {
		slice.Limit = b.innerKeyForIterator(slice.Limit)
	}

	it := &levelIterator{
		b:       b,
		slice:   slice,
		iterEnd: false,
		iter: b.tx.l.ldb.NewIterator(&util.Range{
			Start: slice.Start,
			Limit: slice.Limit,
		}, nil),
	}
	if !b.tx.readOnly {
		it.batchIter = newBatchIterator(b.tx.b, slice.Start, slice.Limit)
	}
	return it
}

func (it *levelIterator) Seek(key []byte) bool {
	ikey := it.b.innerKeyForIterator(key)
	sk := it.iter.Seek(ikey)
	if sk {
		it.iterEnd = false
		if !it.b.tx.readOnly {
			it.batchIter.Reset(ikey)
		}
	} else {
		it.iterEnd = true
		if !it.b.tx.readOnly {
			sk = it.batchIter.Seek(ikey)
		}
	}
	return sk
}

func (it *levelIterator) Next() bool {
	if it.iterEnd {
		if it.b.tx.readOnly || it.batchIter.End() {
			return false
		}
		return it.batchIter.Next()
	}
	hasNext := it.iter.Next()
	if !hasNext {
		it.iterEnd = true
		if it.b.tx.readOnly || it.batchIter.End() {
			return false
		}
		return it.batchIter.Next()
	}
	return hasNext
}

func (it *levelIterator) Key() []byte {
	var data []byte
	if !it.iterEnd {
		data = it.iter.Key()
	} else if !it.b.tx.readOnly && !it.batchIter.End() {
		data = it.batchIter.Key()
	}
	if len(data) > 0 {
		return data[it.b.pathLen+1:]
	}
	return nil
}

func (it *levelIterator) Value() []byte {
	var data []byte
	if !it.iterEnd {
		data = it.iter.Value()
	} else if !it.b.tx.readOnly && !it.batchIter.End() {
		data = it.batchIter.Value()
	}
	return data
}

func (it *levelIterator) Release() {
	it.iter.Release()
}

func (it *levelIterator) Error() error {
	return it.iter.Error()
}

// ------------------ Export -------------------- //

func init() {
	db.RegisterDriver(db.DBDriver{
		Type:     "leveldb",
		OpenDB:   OpenDB,
		CreateDB: CreateDB,
	})
}

func parseDbPath(args ...interface{}) (string, error) {
	if len(args) != 1 {
		return "", db.ErrInvalidArgument
	}
	path, ok := args[0].(string)
	if !ok {
		return "", db.ErrInvalidArgument
	}
	return path, nil
}

func CreateDB(args ...interface{}) (db.DB, error) {
	path, err := parseDbPath(args...)
	if err != nil {
		return nil, err
	}
	return newLevelDB(path, true)
}

func OpenDB(args ...interface{}) (db.DB, error) {
	path, err := parseDbPath(args...)
	if err != nil {
		return nil, err
	}
	return newLevelDB(path, false)
}

func newLevelDB(path string, create bool) (db.DB, error) {
	var ldb *leveldb.DB

	opts := &opt.Options{
		Filter:             filter.NewBloomFilter(10),
		WriteBuffer:        128 * opt.MiB,
		BlockSize:          32 * opt.KiB,
		BlockCacheCapacity: 32 * opt.MiB,
		BlockCacher:        opt.DefaultBlockCacher,
		OpenFilesCacher:    opt.DefaultOpenFilesCacher,
		ErrorIfMissing:     !create,
		ErrorIfExist:       create,
	}

	ldb, err := leveldb.OpenFile(path, opts)
	if err != nil {
		logging.CPrint(logging.ERROR, "newLevelDB failed",
			logging.LogFormat{
				"err":    err,
				"create": create,
				"path":   path,
			})
		if create {
			return nil, db.ErrCreateDBFailed
		}
		return nil, db.ErrOpenDBFailed
	}
	logging.CPrint(logging.INFO, "init leveldb", logging.LogFormat{
		"path":   path,
		"create": create,
	})
	return &LevelDB{ldb: ldb}, nil
}
