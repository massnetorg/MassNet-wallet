package ldb

import (
	"encoding/hex"
	"testing"

	"github.com/massnetorg/MassNet-wallet/wire"
)

func TestLevelDb_FetchBlockBySha(t *testing.T) {
	db, tearDown, err := GetDb("DbTestFetchBlockBySha")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	sha, _, err := db.NewestSha()
	if err != nil {
		t.Errorf("get newestSha error: %v", err)
	}
	blk, err := db.FetchBlockBySha(sha)
	if err != nil {
		t.Errorf("fetch block error: %v", err)
	}
	bys, err := blk.Bytes(wire.Packet)
	if err != nil {
		t.Errorf("serialize block error: %v", err)
	}
	if hex.EncodeToString(bys) != blkHex3 {
		t.Error("unmatch block hex")
	}
}

func TestLevelDb_FetchBlockShaByHeight(t *testing.T) {
	db, tearDown, err := GetDb("DbTestFetchBlockShaByHeight")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	shaNew, _, err := db.NewestSha()
	if err != nil {
		t.Errorf("get newestSha error: %v", err)
	}
	sha3, err := db.FetchBlockShaByHeight(3)
	if sha3.String() != shaNew.String() || err != nil {
		t.Errorf("get FetchBlockShaByHeight error: %v", err)
	}
}

func TestLevelDb_FetchBlockHeightBySha(t *testing.T) {
	db, tearDown, err := GetDb("DbTestFetchBlockHeightBySha")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	shaNew, _, err := db.NewestSha()
	if err != nil {
		t.Errorf("get newestSha error: %v", err)
	}
	height, err := db.FetchBlockHeightBySha(shaNew)
	if height != 3 || err != nil {
		t.Errorf("get FetchBlockHeightBySha error: %v", err)
	}
}

func TestLevelDb_FetchBlockHeaderBySha(t *testing.T) {
	db, tearDown, err := GetDb("DbTestFetchBlockHeaderBySha")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	shaNew, _, err := db.NewestSha()
	if err != nil {
		t.Errorf("get newestSha error: %v", err)
	}
	blk, err := db.FetchBlockBySha(shaNew)
	if err != nil {
		t.Errorf("fetch block error: %v", err)
	}
	blockHeader, err := db.FetchBlockHeaderBySha(shaNew)
	if blk.MsgBlock().Header.BlockHash() != blockHeader.BlockHash() || err != nil {
		t.Errorf("FetchBlockHeaderBySha error: %v", err)
	}
}

func TestLevelDb_ExistsSha(t *testing.T) {
	db, tearDown, err := GetDb("DbTestExistsSha")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	shaNew, _, err := db.NewestSha()
	if err != nil {
		t.Errorf("get newestSha error: %v", err)
	}
	exist, err := db.ExistsSha(shaNew)
	if !exist {
		t.Errorf("cannot find block sha error: %v", err)
	}
}

func TestLevelDb_FetchHeightRange(t *testing.T) {
	db, tearDown, err := GetDb("DbTestFetchHeightRange")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	blockList, err := db.FetchHeightRange(0, 2)
	if len(blockList) != 2 || err != nil {
		t.Errorf("FetchHeightRange error%v", err)
	}
}

func TestLevelDb_FetchAddrIndexTip(t *testing.T) {
	db, tearDown, err := GetDb("DbTestFetchAddrIndexTip")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	_, height, err := db.FetchAddrIndexTip()
	if height != -1 {
		t.Errorf("FetchAddrIndexTip error: %v", err)
	}
}
