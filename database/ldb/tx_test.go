package ldb

import (
	"testing"

	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/database"
	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/wire"
)

func fetchBlockTx(db database.Db, height int32) (*wire.Hash, error) {
	sha, err := db.FetchBlockShaByHeight(height)
	if err != nil {
		return nil, err
	}
	blk, err := db.FetchBlockBySha(sha)
	if err != nil {
		return nil, err
	}
	hash, err := blk.TxHash(0)
	if err != nil {
		return nil, err
	}
	return hash, nil
}

func TestLevelDb_FetchTxByShaList(t *testing.T) {
	db, tearDown, err := GetDb("DbTest")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	hash1, err := fetchBlockTx(db, 1)
	if err != nil {
		t.Errorf("fetchBlockByHeight 1 error : %v", err)
	}

	hash2, err := fetchBlockTx(db, 2)
	if err != nil {
		t.Errorf("fetchBlockByHeight 2 error : %v", err)
	}
	hash3, err := fetchBlockTx(db, 3)
	if err != nil {
		t.Errorf("fetchBlockByHeight 1 error : %v", err)
	}
	var txList []*wire.Hash
	txList = append(txList, hash1)
	txList = append(txList, hash2)
	txList = append(txList, hash3)
	txReply := db.FetchTxByShaList(txList)
	if len(txReply) != 3 {
		t.Errorf("FetchTxByShaList error : %v", err)
	}

}

func TestLevelDb_FetchUnSpentTxByShaList(t *testing.T) {
	db, tearDown, err := GetDb("DbTest")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	hash1, err := fetchBlockTx(db, 1)
	if err != nil {
		t.Errorf("fetchBlockByHeight 1 error : %v", err)
	}

	hash2, err := fetchBlockTx(db, 2)
	if err != nil {
		t.Errorf("fetchBlockByHeight 2 error : %v", err)
	}
	hash3, err := fetchBlockTx(db, 3)
	if err != nil {
		t.Errorf("fetchBlockByHeight 1 error : %v", err)
	}
	var txList []*wire.Hash
	txList = append(txList, hash1)
	txList = append(txList, hash2)
	txList = append(txList, hash3)
	txReply := db.FetchUnSpentTxByShaList(txList)
	if len(txReply) != 3 {
		t.Errorf("FetchTxByShaList error : %v", err)
	}

}

func TestLevelDb_FetchTxBySha(t *testing.T) {
	db, tearDown, err := GetDb("DbTest")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	hash1, err := fetchBlockTx(db, 1)
	if err != nil {
		t.Errorf("fetchBlockByHeight 1 error : %v", err)
	}
	txReply, err := db.FetchTxBySha(hash1)
	if err != nil {
		t.Errorf("FetchTxBySha error : %v", err)
	}
	tx := txReply[len(txReply)-1]
	if !tx.Sha.IsEqual(hash1) {
		t.Errorf("FetchTxBySha failed")
	}
}

func TestLevelDb_ExistsTxSha(t *testing.T) {
	db, tearDown, err := GetDb("DbTest")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	blockHash, _, err := db.NewestSha()
	if err != nil {
		t.Errorf("fetchBlockByHeight 1 error : %v", err)
	}
	exist, err := db.ExistsSha(blockHash)
	if !exist || err != nil {
		t.Errorf("ExistsSha error : %v", err)
	}
}

func TestLevelDb_FetchUtxosForAddrs(t *testing.T) {
	var address []massutil.Address
	db, tearDown, err := GetDb("DbTestFetchUtxosForAddrs")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	for _, addr := range addressList {
		add, err := massutil.DecodeAddress(addr, &config.ChainParams)
		if err != nil {
			t.Errorf("DecodeAddress error : %v", err)
		}
		address = append(address, add)
	}
	_, err = db.FetchUtxosForAddrs(address, &config.ChainParams)
	if err != nil {
		t.Errorf("FetchUtxosForAddrs error : %v", err)
	}
}

func TestLevelDb_InsertWitnessAddr(t *testing.T) {
	db, tearDown, err := GetDb("DbTestInsertWitnessAddr")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	err = db.InsertWitnessAddr(addressList[0], []byte{1})
	if err != nil {
		t.Errorf("InsertWitnessAddr error : %v", err)
	}
}

func TestLevelDb_FetchWitnessAddrToRedeem(t *testing.T) {
	db, tearDown, err := GetDb("DbTestInsertWitnessAddr")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	_, err = db.FetchWitnessAddrToRedeem()
	if err != nil {
		t.Errorf("FetchWitnessAddrToRedeem error:%v", err)
	}
}

func TestLevelDb_ImportChildKeyNum(t *testing.T) {
	db, tearDown, err := GetDb("DbTestImportChildKeyNum")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	err = db.ImportChildKeyNum("", "")
	if err != nil {
		t.Errorf("ImportChildKeyNum error:%v", err)
	}
}

func TestLevelDb_FetchAllRootPkStr(t *testing.T) {
	db, tearDown, err := GetDb("DbTestImportChildKeyNum")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	_, err = db.FetchAllRootPkStr()
	if err != nil {
		t.Errorf("FetchAllRootPkStr error:%v", err)
	}
}

func TestLevelDb_DeleteAddrIndex(t *testing.T) {
	db, tearDown, err := GetDb("DeleteAddrIndex")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	err = db.DeleteAddrIndex()
	if err != nil {
		t.Errorf("DeleteAddrIndex error:%v", err)
	}
}

func TestLevelDb_FetchTxsForAddr(t *testing.T) {
	var address []massutil.Address
	db, tearDown, err := GetDb("FetchTxsForAddr")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	for _, addr := range addressList {
		add, err := massutil.DecodeAddress(addr, &config.ChainParams)
		if err != nil {
			t.Errorf("DecodeAddress error : %v", err)
		}
		address = append(address, add)
	}
	_, _, err = db.FetchTxsForAddr(address[0], 1, 2, false)
	if err != nil {
		t.Errorf("FetchTxsForAddr error:%v", err)
	}
}
