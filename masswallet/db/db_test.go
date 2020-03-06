package db_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	walletdb "massnet.org/mass-wallet/masswallet/db"

	"github.com/stretchr/testify/assert"
	_ "massnet.org/mass-wallet/masswallet/db/ldb"
	_ "massnet.org/mass-wallet/masswallet/db/rdb"
)

const testDbRoot = "testDbs"

var (
	dbtype          = ""
	triggerRollback = errors.New("trigger rollback")
)

// filesExists returns whether or not the named file or directory exists.
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func GetDb(dbName string) (walletdb.DB, func(), error) {
	// Create the root directory for test databases.
	if !fileExists(testDbRoot) {
		if err := os.MkdirAll(testDbRoot, 0700); err != nil {
			err := fmt.Errorf("unable to create test db "+
				"root: %v", err)
			return nil, nil, err
		}
	}
	dbPath := filepath.Join(testDbRoot, dbName)
	if err := os.RemoveAll(dbPath); err != nil {
		err := fmt.Errorf("cannot remove old db: %v", err)
		return nil, nil, err
	}
	db, err := walletdb.CreateDB(dbtype, dbPath)
	if err != nil {
		fmt.Println("create db error: ", err)
		return nil, nil, err
	}
	tearDown := func() {
		dbVersionPath := filepath.Join(testDbRoot, dbName+".ver")
		db.Close()
		os.RemoveAll(dbPath)
		os.Remove(dbVersionPath)
		os.RemoveAll(testDbRoot)
	}
	return db, tearDown, nil
}

// NOTE: use flag '-tags rocksdb' to enable running tests with rocksdb
func TestAll(t *testing.T) {
	for _, tp := range walletdb.RegisteredDbTypes() {
		dbtype = tp
		t.Logf("run tests with %s...", dbtype)
		testDB_NewRootBucket(t)
		testDB_RootBucket(t)
		testDB_RootBucketNames(t)
		testDB_DeleteRootBucket(t)
		testBucket_NewSubBucket(t)
		testBucket_SubBucket(t)
		testBucket_SubBucketNames(t)
		testBucket_DeleteSubBucket(t)
		testBucket_Put(t)
		testBucket_Get(t)
		testBucket_Delete(t)
		testBucket_Clear(t)
		testCreateOrOpenDB(t)
		testGetByPrefix(t)
		testIterator(t)
		testSeek(t)
	}
}

//test create a new bucket
func testDB_NewRootBucket(t *testing.T) {
	tests := []struct {
		name               string
		dbName             string
		newRootBucketTimes int
		err                string
	}{
		{
			"valid",
			"123",
			1,
			"",
		}, {
			"duplicate bucket",
			"123",
			2,
			"bucket already exist",
		},
	}

	db, tearDown, err := GetDb("Tst_NewRootBucket")
	if err != nil {
		t.Errorf("init db error:%v", err)
	}
	defer tearDown()

	err = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
		_, err := tx.CreateTopLevelBucket("123")
		if err != nil {
			t.Errorf("New RootBucket error:%v", err)
		}
		return err
	})
	assert.Nil(t, err)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.newRootBucketTimes != 1 {

				_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
					_, err := tx.CreateTopLevelBucket("123")
					if err != nil {
						assert.Equal(t, test.err, err.Error())
					}
					return err
				})
			}
		})
	}
}

func testDB_RootBucket(t *testing.T) {
	tests := []struct {
		name       string
		bucketName string
		err        string
	}{
		{
			"valid",
			"123",
			"",
		}, {
			"invalid bucket name",
			"",
			"invalid bucket name",
		},
	}
	db, tearDown, err := GetDb("Tst_RootBucket")
	if err != nil {
		t.Errorf("init db error:%v", err)
	}
	defer tearDown()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
				bucket := tx.TopLevelBucket("123")
				assert.Nil(t, bucket)
				return nil
			})
			// _, err = walletdb.Bucket(test.bucketName)
			// if err != nil {
			// 	assert.Equal(t, test.err, err.Error())
			// }
		})
	}
}

func testDB_RootBucketNames(t *testing.T) {
	tests := []struct {
		name          string
		bucketName    []string
		GetBucketName []string
	}{
		{
			"valid",
			[]string{"123", "456"},
			[]string{"123", "456"},
		},
	}
	db, tearDown, err := GetDb("Tst_RootBucketNames")
	if err != nil {
		t.Errorf("init db error:%v", err)
	}
	defer tearDown()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, num := range test.bucketName {
				err = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
					_, err := tx.CreateTopLevelBucket(num)
					return err
				})
				assert.Nil(t, err)
			}
			_ = walletdb.View(db, func(tx walletdb.ReadTransaction) error {
				s, err := tx.BucketNames()
				assert.Nil(t, err)
				assert.Equal(t, test.GetBucketName, s)
				return nil
			})
		})
	}
}

func testDB_DeleteRootBucket(t *testing.T) {
	tests := []struct {
		name       string
		rootBucket string
		error      string
	}{
		{
			"valid",
			"123",
			"not supported",
		},
	}
	db, tearDown, err := GetDb("Tst_DeleteRootBucket")
	if err != nil {
		t.Errorf("init db error:%v", err)
	}
	defer tearDown()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
				_, _ = tx.CreateTopLevelBucket(test.rootBucket)
				err := tx.DeleteTopLevelBucket(test.rootBucket)
				assert.Equal(t, test.error, err.Error())
				return nil
			})
		})
	}
}

func testBucket_NewSubBucket(t *testing.T) {
	tests := []struct {
		name              string
		dbName            string
		newSubBucketTimes int
		err               string
	}{
		{
			"valid",
			"123",
			1,
			"",
		}, {
			"duplicate bucket",
			"123",
			2,
			"bucket already exist",
		},
	}

	db, tearDown, err := GetDb("Tst_NewSubBucket")
	if err != nil {
		t.Errorf("init db error:%v", err)
	}
	defer tearDown()

	_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
		bucket, err := tx.CreateTopLevelBucket("root")
		assert.Nil(t, err, fmt.Sprintf("New RootBucket error:%v", err))
		if err != nil {
			return err
		}

		bucket, err = bucket.NewBucket("123")
		assert.Nil(t, err)
		t.Logf("create sub bucket %v", bucket.GetBucketMeta().Paths())
		return err
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.newSubBucketTimes != 1 {
				_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
					bucket := tx.TopLevelBucket("root")
					assert.NotNil(t, bucket)
					_, err = bucket.NewBucket("123")
					assert.Equal(t, test.err, err.Error())
					return err
				})
			}
		})
	}
}

func testBucket_SubBucket(t *testing.T) {
	tests := []struct {
		name       string
		bucketName string
		err        string
	}{
		{
			"valid",
			"123",
			"",
		}, {
			"invalid bucket name",
			"",
			"invalid bucket name",
		},
	}

	db, tearDown, err := GetDb("Tst_SubBucket")
	if err != nil {
		t.Errorf("init db error:%v", err)
	}
	defer tearDown()

	_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
		_, err := tx.CreateTopLevelBucket("abc")
		if err != nil {
			t.Errorf("New RootBucket error:%v", err)
			return err
		}
		return nil
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
				bucket := tx.TopLevelBucket("abc")
				assert.NotNil(t, bucket)
				_, err = bucket.NewBucket(test.bucketName)
				if err != nil {
					assert.Equal(t, test.err, err.Error())
				}
				return nil
			})
		})
	}
}

func testBucket_SubBucketNames(t *testing.T) {
	tests := []struct {
		name          string
		bucketName    []string
		GetBucketName []string
	}{
		{
			"valid",
			[]string{"123", "456"},
			[]string{"123", "456"},
		},
	}
	db, tearDown, err := GetDb("Tst_SubBucketNames")
	if err != nil {
		t.Errorf("init db error:%v", err)
	}
	defer tearDown()

	_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
		_, err := tx.CreateTopLevelBucket("abc")
		if err != nil {
			t.Errorf("New RootBucket error:%v", err)
			return err
		}
		return nil
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
				bucket := tx.TopLevelBucket("abc")
				assert.NotNil(t, bucket)
				for _, num := range test.bucketName {
					_, err := bucket.NewBucket(num)
					assert.Nil(t, err)
				}
				return nil
			})

			_ = walletdb.View(db, func(tx walletdb.ReadTransaction) error {
				bucket := tx.TopLevelBucket("abc")
				assert.NotNil(t, bucket)
				s, _ := bucket.BucketNames()
				assert.Equal(t, test.GetBucketName, s)
				return nil
			})
		})
	}
}

func testBucket_DeleteSubBucket(t *testing.T) {
	tests := []struct {
		name              string
		bucketName        string
		bucketToDelete    string
		bucketAfterDelete []string
		err               string
	}{
		{
			"valid",
			"123",
			"123",
			[]string{},
			"",
		}, {
			"invalid bucket name",
			"123",
			"",
			[]string{"123"},
			"invalid bucket name",
		}, {
			"invalid bucket name",
			"123",
			"456",
			[]string{"123"},
			"",
		},
	}

	db, tearDown, err := GetDb("Tst_DeleteSubBucket")
	if err != nil {
		t.Errorf("init db error:%v", err)
	}
	defer tearDown()

	_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
		_, err := tx.CreateTopLevelBucket("abc")
		if err != nil {
			t.Errorf("New RootBucket error:%v", err)
			return err
		}
		return nil
	})

	walletdb.View(db, func(tx walletdb.ReadTransaction) error {
		names, err := tx.BucketNames()
		assert.Nil(t, err)
		assert.Equal(t, []string{"abc"}, names)

		bucket := tx.TopLevelBucket("abc")
		names, err = bucket.BucketNames()
		assert.Nil(t, err)
		assert.Equal(t, []string{}, names)
		return nil
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
				bucket := tx.TopLevelBucket("abc")
				assert.NotNil(t, bucket)

				_, err := bucket.NewBucket(test.bucketName)
				assert.Nil(t, err)
				s1, _ := bucket.BucketNames()
				assert.NotNil(t, s1)
				err = bucket.DeleteBucket(test.bucketToDelete)
				if err != nil {
					assert.Equal(t, test.err, err.Error())
				}
				bucket = tx.TopLevelBucket("abc")
				assert.NotNil(t, bucket)
				s, _ := bucket.BucketNames()
				assert.Equal(t, test.bucketAfterDelete, s)

				return triggerRollback
			})
		})
	}
}

func testBucket_Put(t *testing.T) {
	tests := []struct {
		name  string
		key   []byte
		value []byte
		err   string
	}{
		{
			"valid",
			[]byte{123},
			[]byte{123},
			"",
		}, {
			"illegal key",
			[]byte{},
			[]byte{123},
			"illegal key",
		}, {
			"illegal value",
			[]byte{123},
			[]byte{},
			"illegal value",
		},
	}
	db, tearDown, err := GetDb("Tst_Put")
	if err != nil {
		t.Errorf("init db error:%v", err)
	}
	defer tearDown()

	_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
		_, err := tx.CreateTopLevelBucket("abc")
		if err != nil {
			t.Errorf("New RootBucket error:%v", err)
			return err
		}
		return nil
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
				bucket := tx.TopLevelBucket("abc")
				assert.NotNil(t, bucket)
				err := bucket.Put(test.key, test.value)
				if err != nil {
					assert.Equal(t, test.err, err.Error())
				} else {
					v, err := bucket.Get(test.key)
					assert.Nil(t, err)
					assert.Equal(t, test.value, v)
				}
				return triggerRollback
			})
		})
	}
}

func testBucket_Get(t *testing.T) {
	tests := []struct {
		name        string
		key         []byte
		value       []byte
		keyToFind   []byte
		valueToFind []byte
		err         string
	}{
		{
			"valid",
			[]byte{123},
			[]byte{123},
			[]byte{123},
			[]byte{123},
			"",
		}, {
			"invalid key",
			[]byte{123},
			[]byte{123},
			[]byte{},
			nil,
			"illegal key",
		}, {
			"key not found",
			[]byte{123},
			[]byte{123},
			[]byte{45},
			nil,
			"",
		},
	}
	db, tearDown, err := GetDb("Tst_Get")
	if err != nil {
		t.Errorf("init db error:%v", err)
	}
	defer tearDown()

	_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
		_, err := tx.CreateTopLevelBucket("abc")
		if err != nil {
			t.Errorf("New RootBucket error:%v", err)
			return err
		}
		return nil
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
				bucket := tx.TopLevelBucket("abc")
				assert.NotNil(t, bucket)

				bucket.Put(test.key, test.value)
				value, err := bucket.Get(test.keyToFind)
				if err != nil {
					assert.Equal(t, test.err, err.Error())
				} else {
					assert.Equal(t, test.valueToFind, value)
				}
				return triggerRollback
			})
		})
	}
}

func testBucket_Delete(t *testing.T) {
	tests := []struct {
		name        string
		key         []byte
		value       []byte
		keyToDelete []byte
		valueToFind []byte
		err         string
	}{
		{
			"valid",
			[]byte{123},
			[]byte{123},
			[]byte{123},
			nil,
			"",
		}, {
			"invalid key",
			[]byte{123},
			[]byte{123},
			[]byte{},
			[]byte{123},
			"illegal key",
		}, {
			"key not found",
			[]byte{123},
			[]byte{123},
			[]byte{45},
			[]byte{123},
			"",
		},
	}
	db, tearDown, err := GetDb("Tst_Delete")
	if err != nil {
		t.Errorf("init db error:%v", err)
	}
	defer tearDown()

	_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
		_, err := tx.CreateTopLevelBucket("abc")
		if err != nil {
			t.Errorf("New RootBucket error:%v", err)
			return err
		}
		return nil
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
				bucket := tx.TopLevelBucket("abc")
				assert.NotNil(t, bucket)
				bucket.Put(test.key, test.value)
				err := bucket.Delete(test.keyToDelete)
				if err != nil {
					assert.Equal(t, test.err, err.Error())
				}
				value, err := bucket.Get(test.key)
				assert.Nil(t, err)
				assert.Equal(t, test.valueToFind, value)
				return err
			})
		})
	}
}

func testBucket_Clear(t *testing.T) {
	tests := []struct {
		name        string
		key         []byte
		value       []byte
		keyToFind   []byte
		valueToFind []byte
		err         string
	}{
		{
			"valid",
			[]byte{123},
			[]byte{123},
			[]byte{123},
			nil,
			"",
		}, {
			"invalid key",
			[]byte{123},
			[]byte{123},
			[]byte{},
			nil,
			"illegal key",
		}, {
			"key not found",
			[]byte{123},
			[]byte{123},
			[]byte{45},
			nil,
			"",
		},
	}
	db, tearDown, err := GetDb("Tst_Clear")
	if err != nil {
		t.Errorf("init db error:%v", err)
	}
	defer tearDown()

	_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
		_, err := tx.CreateTopLevelBucket("abc")
		if err != nil {
			t.Errorf("New RootBucket error:%v", err)
			return err
		}
		return nil
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
				bucket := tx.TopLevelBucket("abc")
				assert.NotNil(t, bucket)
				bucket.Put(test.key, test.value)
				err := bucket.Clear()
				if err != nil {
					assert.Equal(t, test.err, err.Error())
				}
				value, _ := bucket.Get(test.keyToFind)
				assert.Equal(t, test.valueToFind, value)
				return err
			})
		})
	}
}

func testCreateOrOpenDB(t *testing.T) {
	tests := []struct {
		name   string
		dbPath string
		create bool
		err    error
	}{
		{
			"create new db",
			testDbRoot + "/Tst_NewDB",
			true,
			nil,
		}, {
			"create existed db",
			testDbRoot + "/Tst_NewDB",
			true,
			walletdb.ErrCreateDBFailed,
		}, {
			"create invalid dbpath",
			"",
			true,
			walletdb.ErrCreateDBFailed,
		}, {
			"open existing db",
			testDbRoot + "/Tst_NewDB",
			false,
			nil,
		}, {
			"open non-existent db",
			testDbRoot + "/Tst_NonExistent",
			false,
			walletdb.ErrOpenDBFailed,
		}, {
			"open invalid dbpath",
			"",
			false,
			walletdb.ErrOpenDBFailed,
		},
	}

	if !fileExists(testDbRoot) {
		if err := os.MkdirAll(testDbRoot, 0700); err != nil {
			t.Errorf("unable to create test db root: %v", err)
			return
		}
	}
	defer func() {
		os.RemoveAll(testDbRoot)
	}()

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				db  walletdb.DB
				err error
			)
			if test.create {
				db, err = walletdb.CreateDB(dbtype, test.dbPath)
			} else {
				db, err = walletdb.OpenDB(dbtype, test.dbPath)
			}

			assert.Equal(t, test.err, err)
			defer func() {
				if db != nil {
					db.Close()
				}
			}()
		})
	}
}

func testGetByPrefix(t *testing.T) {
	tests := []struct {
		name   string
		bucket string
		puts   map[string]string
		prefix string
		expect map[string]string
		err    error
	}{
		{
			"case 1",
			"case1",
			map[string]string{"a123": "123", "b456": "456", "c789": "789"},
			"d",
			map[string]string{},
			nil,
		},
		{
			"case 2",
			"case2",
			map[string]string{"a123": "123", "a223": "223", "b456": "456", "c789": "789"},
			"a",
			map[string]string{"a123": "123", "a223": "223"},
			nil,
		},
		{
			"case 3",
			"case3",
			map[string]string{"a123": "123", "a223": "223", "b456": "456", "c789": "789"},
			"a2",
			map[string]string{"a223": "223"},
			nil,
		},
		{
			"case 4",
			"case4",
			map[string]string{"a123": "123", "a223": "223", "b456": "456", "c789": "789"},
			"",
			map[string]string{"a123": "123", "a223": "223", "b456": "456", "c789": "789"},
			nil,
		},
		{
			"case 5",
			"case5",
			map[string]string{"a123": "123", "a1231": "1231", "b456": "456", "c789": "789"},
			"a123",
			map[string]string{"a123": "123", "a1231": "1231"},
			nil,
		},
	}

	db, tearDown, err := GetDb("Tst_GetByPrefix")
	if err != nil {
		t.Errorf("init db error:%v", err)
	}
	defer tearDown()

	_ = walletdb.Update(db, func(tx walletdb.DBTransaction) error {
		_, err := tx.CreateTopLevelBucket("root")
		if err != nil {
			t.Errorf("New RootBucket error:%v", err)
			return err
		}
		return nil
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			walletdb.Update(db, func(tx walletdb.DBTransaction) error {
				root := tx.TopLevelBucket("root")
				assert.NotNil(t, root)
				bucket, err := root.NewBucket(test.bucket)
				if err != nil {
					t.FailNow()
				}
				for key, value := range test.puts {
					err := bucket.Put([]byte(key), []byte(value))
					assert.Nil(t, err)
				}
				return nil
			})

			walletdb.View(db, func(tx walletdb.ReadTransaction) error {
				root := tx.TopLevelBucket("root")
				assert.NotNil(t, root)
				bucket := root.Bucket(test.bucket)
				assert.NotNil(t, bucket)

				iter := bucket.NewIterator(walletdb.BytesPrefix([]byte(test.prefix)))
				defer iter.Release()

				actual := make(map[string]string)
				for iter.Next() {
					actual[string(iter.Key())] = string(iter.Value())
				}

				assert.Equal(t, len(test.expect), len(actual))
				for key, value := range actual {
					assert.Equal(t, test.puts[key], value)
					fmt.Println("equal", key, value)
				}
				return nil
			})
		})
	}
}

func testIterator(t *testing.T) {

	puts := map[string]string{
		"a":      "a",
		"a1":     "a1",
		"a123":   "a123",
		"b123":   "b123",
		"b2":     "b2",
		"b4":     "b4",
		"b45":    "b45",
		"b45123": "b45123",
		"b456":   "b456",
		"b6":     "b6",
		"b756":   "b756",
		"c789":   "c789",
	}

	tests := []struct {
		name   string
		slice  *walletdb.Range
		expect map[string]string
		err    error
	}{
		{
			"nil range",
			nil,
			puts,
			nil,
		},
		{
			"by prefix - empty",
			walletdb.BytesPrefix([]byte("")),
			puts,
			nil,
		},
		{
			"by prefix - no match",
			walletdb.BytesPrefix([]byte("ddd")),
			nil,
			nil,
		},
		{
			"invalid range",
			&walletdb.Range{Start: []byte("b451233"), Limit: []byte("b451233")},
			nil,
			nil,
		},
		{
			"by prefix - normal",
			walletdb.BytesPrefix([]byte("b4")),
			map[string]string{
				"b45":    "b45",
				"b45123": "b45123",
				"b456":   "b456",
				"b4":     "b4",
			},
			nil,
		},
		{
			"by range - no item",
			&walletdb.Range{Start: []byte("b3"), Limit: []byte("b4")},
			nil,
			nil,
		},
		{
			"by range - 1",
			&walletdb.Range{Start: []byte("b3"), Limit: []byte("b40")},
			map[string]string{
				"b4": "b4",
			},
			nil,
		},
		{
			"by range - 2",
			&walletdb.Range{Start: []byte("b4"), Limit: []byte("b40")},
			map[string]string{
				"b4": "b4",
			},
			nil,
		},
		{
			"by range - 3",
			&walletdb.Range{Start: []byte("b2"), Limit: []byte("b4")},
			map[string]string{
				"b2": "b2",
			},
			nil,
		},
		{
			"by range - 4",
			&walletdb.Range{Start: []byte("b"), Limit: []byte("b5")},
			map[string]string{
				"b123":   "b123",
				"b2":     "b2",
				"b4":     "b4",
				"b45":    "b45",
				"b45123": "b45123",
				"b456":   "b456",
			},
			nil,
		},
		{
			"by range - 5",
			&walletdb.Range{Start: []byte("b451233"), Limit: []byte("b5")},
			map[string]string{
				"b456": "b456",
			},
			nil,
		},
	}

	stor, tearDown, err := GetDb("Tst_Iterator")
	if err != nil {
		t.Errorf("init db error:%v", err)
	}
	defer tearDown()

	_ = walletdb.Update(stor, func(tx walletdb.DBTransaction) error {
		root, err := tx.CreateTopLevelBucket("root")
		if err != nil {
			t.Errorf("New RootBucket error:%v", err)
			return err
		}
		sub, err := root.NewBucket("sub")
		if err != nil {
			t.Errorf("new sub bucket error: %v", err)
			return err
		}

		for k, v := range puts {
			err := sub.Put([]byte(k), []byte(v))
			assert.Nil(t, err)
		}
		t.Log("init bucket&data success")
		return nil
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			walletdb.View(stor, func(tx walletdb.ReadTransaction) error {
				root := tx.TopLevelBucket("root")
				assert.NotNil(t, root)
				bucket := root.Bucket("sub")
				assert.NotNil(t, bucket)

				iter := bucket.NewIterator(test.slice)
				defer iter.Release()

				actual := make(map[string]string)
				for iter.Next() {
					actual[string(iter.Key())] = string(iter.Value())
				}

				assert.Equal(t, len(test.expect), len(actual))
				for key, value := range actual {
					assert.Equal(t, test.expect[key], value)
					fmt.Println("equal", key, value)
				}
				return nil
			})
		})
	}
}

func testSeek(t *testing.T) {
	puts := map[string]string{
		"a":      "a",
		"a1":     "a1",
		"a123":   "a123",
		"b123":   "b123",
		"b2":     "b2",
		"b4":     "b4",
		"b45":    "b45",
		"b45123": "b45123",
		"b456":   "b456",
		"b6":     "b6",
		"b756":   "b756",
		"c789":   "c789",
	}

	type seekData struct {
		key    string
		expect bool
	}

	tests := []struct {
		name   string
		rg     *walletdb.Range
		seek   *seekData
		expect map[string]string
		err    error
	}{
		{
			name:   "case 1",
			rg:     nil,
			seek:   nil, // no seek
			expect: puts,
			err:    nil,
		},
		{
			name:   "case 2",
			rg:     &walletdb.Range{Start: []byte(""), Limit: []byte("")},
			seek:   nil, // no seek
			expect: puts,
			err:    nil,
		},
		{
			name: "case 3",
			rg:   &walletdb.Range{Start: []byte("b"), Limit: []byte("c")},
			seek: nil,
			expect: map[string]string{
				"b123":   "b123",
				"b2":     "b2",
				"b4":     "b4",
				"b45":    "b45",
				"b45123": "b45123",
				"b456":   "b456",
				"b6":     "b6",
				"b756":   "b756",
			},
			err: nil,
		},
		{
			name: "case 4",
			rg:   &walletdb.Range{Start: []byte("b"), Limit: []byte("c")},
			seek: &seekData{
				key:    "b2",
				expect: true,
			},
			expect: map[string]string{
				"b2":     "b2",
				"b4":     "b4",
				"b45":    "b45",
				"b45123": "b45123",
				"b456":   "b456",
				"b6":     "b6",
				"b756":   "b756",
			},
			err: nil,
		},
		{
			name: "case 5",
			rg:   &walletdb.Range{Start: []byte("b"), Limit: nil},
			seek: &seekData{
				key:    "b45",
				expect: true,
			},
			expect: map[string]string{
				"b45":    "b45",
				"b45123": "b45123",
				"b456":   "b456",
				"b6":     "b6",
				"b756":   "b756",
				"c789":   "c789",
			},
			err: nil,
		},
		{
			name: "case 6",
			rg:   &walletdb.Range{Start: []byte("b"), Limit: []byte("c")},
			seek: &seekData{
				key:    "c",
				expect: false,
			},
			expect: nil,
			err:    nil,
		},
	}

	stor, tearDown, err := GetDb("Tst_Seek")
	if err != nil {
		t.Errorf("init db error:%v", err)
	}
	defer tearDown()

	_ = walletdb.Update(stor, func(tx walletdb.DBTransaction) error {
		root, err := tx.CreateTopLevelBucket("root")
		if err != nil {
			t.Errorf("New RootBucket error:%v", err)
			return err
		}
		sub, err := root.NewBucket("sub")
		if err != nil {
			t.Errorf("new sub bucket error: %v", err)
			return err
		}

		for k, v := range puts {
			err := sub.Put([]byte(k), []byte(v))
			assert.Nil(t, err)
		}
		t.Log("init bucket&data success")
		return nil
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			walletdb.View(stor, func(tx walletdb.ReadTransaction) error {
				root := tx.TopLevelBucket("root")
				assert.NotNil(t, root)
				bucket := root.Bucket("sub")
				assert.NotNil(t, bucket)

				it := bucket.NewIterator(test.rg)
				defer it.Release()

				sought := false
				if test.seek != nil {
					sought = it.Seek([]byte(test.seek.key))
					assert.Equal(t, test.seek.expect, sought)
				}

				actual := make(map[string]string)
				if sought {
					actual[string(it.Key())] = string(it.Value())
				}
				for it.Next() {
					actual[string(it.Key())] = string(it.Value())
				}

				assert.Equal(t, len(test.expect), len(actual))
				for ek, ev := range test.expect {
					av, ok := actual[ek]
					assert.True(t, ok && ev == av, fmt.Sprintf("expect %s:%s, actual %s", ek, ev, av))
				}
				return nil
			})
		})
	}
}
