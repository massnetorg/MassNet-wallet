package ldb

import (
	"bytes"
	"strconv"
)

// address_package related
func (db *LevelDb) InsertRootKey(rootKeyEnc []byte, rootKeyStr string, defaultPasswordUsed bool) error {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	key := rStrToKeyOfRootKey(rootKeyStr)

	value := valueOfRootKey(rootKeyEnc, defaultPasswordUsed)

	batch := db.lBatch()
	batch.Put(key, value)
	err := db.lDb.Write(batch, nil)
	if err != nil {
		return err
	}

	return nil
}

func valueOfRootKey(rootKeyEnc []byte, defaultPasswordUsed bool) []byte {

	var checkBytes []byte

	if defaultPasswordUsed {
		checkBytes = []byte("1")

	} else {
		checkBytes = []byte("0")

	}

	l1 := len(checkBytes)
	l2 := len(rootKeyEnc)
	l := l1 + l2
	value := make([]byte, l, l)

	copy(value[0:l1], checkBytes)
	copy(value[l1:], rootKeyEnc)

	return value
}

func rStrToKeyOfRootKey(rootKeyStr string) []byte {

	l1 := len(rootKeyPrefix)
	l2 := len([]byte(rootKeyStr))
	l := l1 + l2
	key := make([]byte, l, l)

	copy(key[0:l1], rootKeyPrefix[:])
	copy(key[l1:], []byte(rootKeyStr))

	return key
}

func rStrToKeyOfChildKeyNum(rootKeyStr string) []byte {
	l1 := len(childKeyNumPrefix)
	l2 := len([]byte(rootKeyStr))
	l := l1 + l2
	key := make([]byte, l, l)
	copy(key[:l1], childKeyNumPrefix)
	copy(key[l1:], []byte(rootKeyStr))

	return key
}

func witnessAddrToKey(witAddr string) []byte {
	l1 := len(witnessAddrPrefix)
	l2 := len([]byte(witAddr))
	key := make([]byte, l1+l2, l1+l2)
	copy(key[:l1], witnessAddrPrefix)
	copy(key[l1:], []byte(witAddr)[:])

	return key
}

func (db *LevelDb) FetchRootKey(rootKeyStr string) ([]byte, bool, error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	key := rStrToKeyOfRootKey(rootKeyStr)
	buf, err := db.lDb.Get(key, nil)
	if err != nil {
		return nil, false, err
	}

	checkBytes := make([]byte, 1, 1)
	copy(checkBytes, buf[:1])

	num := bytes.Compare(checkBytes, []byte("1"))

	if num == 0 {
		return buf[1:], true, nil
	}

	return buf[1:], false, nil
}

func (db *LevelDb) FetchAllRootPkStr() ([]string, error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	l1 := len(rootKeyPrefix)
	rootPkStrList := make([]string, 0)

	iter := db.lDb.NewIterator(bytesPrefix(rootKeyPrefix), nil)
	for iter.Next() {

		rootPkStr := string(iter.Key()[l1:])
		rootPkStrList = append(rootPkStrList, rootPkStr)
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return nil, err
	}

	return rootPkStrList, nil
}

func (db *LevelDb) InitChildKeyNum(rootKeyStr string) error {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	key := rStrToKeyOfChildKeyNum(rootKeyStr)

	batch := db.lBatch()
	batch.Put(key, []byte("0"))
	err := db.lDb.Write(batch, nil)
	if err != nil {
		return err
	}
	return nil
}

func (db *LevelDb) FetchChildKeyNum(rootKeyStr string) (int, error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	bufInt, err := fetchChildKeyNum(db, rootKeyStr)
	if err != nil {
		return 0, err
	}

	return bufInt, nil
}

func fetchChildKeyNum(db *LevelDb, rootKeyStr string) (int, error) {
	key := rStrToKeyOfChildKeyNum(rootKeyStr)

	buf, err := db.lDb.Get(key, nil)
	if err != nil {
		return 0, err
	}

	bufInt, err := strconv.Atoi(string(buf))
	if err != nil {
		return 0, err
	}

	return bufInt, nil
}

func (db *LevelDb) UpdateChildKeyNum(rootKeyStr string) (int, error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	bufInt, err := fetchChildKeyNum(db, rootKeyStr)
	if err != nil {
		return 0, err
	}
	bufInt += 1
	bufStr := strconv.Itoa(bufInt)

	key := rStrToKeyOfChildKeyNum(rootKeyStr)
	batch := db.lBatch()
	batch.Put(key, []byte(bufStr))
	err = db.lDb.Write(batch, nil)
	if err != nil {
		return 0, err
	}

	bufInt, err = fetchChildKeyNum(db, rootKeyStr)
	if err != nil {
		return 0, err
	}

	return bufInt, nil
}

func (db *LevelDb) InsertWitnessAddr(witAddr string, redeemScript []byte) error {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	key := witnessAddrToKey(witAddr)
	batch := db.lBatch()
	batch.Put(key, redeemScript)
	err := db.lDb.Write(batch, nil)
	if err != nil {
		return err
	}

	return nil
}

func (db *LevelDb) FetchWitnessAddrToRedeem() (map[string][]byte, error) {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	l1 := len(witnessAddrPrefix)
	witAddrToRedeem := make(map[string][]byte)

	iter := db.lDb.NewIterator(bytesPrefix(witnessAddrPrefix), nil)
	for iter.Next() {
		witAddrStr := string(iter.Key()[l1:])
		redeemStr := string(iter.Value())
		witAddrToRedeem[witAddrStr] = []byte(redeemStr)
	}
	iter.Release()
	if err := iter.Error(); err != nil {
		return nil, err
	}

	return witAddrToRedeem, nil
}

func (db *LevelDb) ImportChildKeyNum(rootKeyStr string, num string) error {
	db.dbLock.Lock()
	defer db.dbLock.Unlock()

	key := rStrToKeyOfChildKeyNum(rootKeyStr)

	batch := db.lBatch()
	batch.Put(key, []byte(num))
	err := db.lDb.Write(batch, nil)
	if err != nil {
		return err
	}
	return nil
}
