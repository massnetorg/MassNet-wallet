package txmgr

import (
	"bufio"
	"encoding/hex"
	"os"
	"testing"

	// "time"

	"github.com/stretchr/testify/assert"
	"massnet.org/mass-wallet/massutil"
	mwdb "massnet.org/mass-wallet/masswallet/db"
	"massnet.org/mass-wallet/wire"
)

const (
	TxDbTestDbRoot = "txStoreTestDb"
)

var (
	blks          []*massutil.Block
	allTxs        map[wire.Hash]*wire.MsgTx
	blk1          *massutil.Block
	blk2          *massutil.Block
	blk1TxRecords []*TxRecord
	tx1           *massutil.Tx
	tx2           *massutil.Tx
	walletID      string
)

func init() {
	f, err := os.Open("../data/mockBlks.dat")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		buf, err := hex.DecodeString(scanner.Text())
		if err != nil {
			panic(err)
		}
		blk, err := massutil.NewBlockFromBytes(buf, wire.Packet)
		if err != nil {
			panic(err)
		}
		blks = append(blks, blk)
	}
	allTxs = make(map[wire.Hash]*wire.MsgTx)
	for i := 1; i < 25; i++ {
		blk := blks[i]
		for _, tx := range blk.Transactions() {
			allTxs[*tx.Hash()] = tx.MsgTx()
		}
	}
	blk1 = blks[21]
	blk2 = blks[22]
	blk1Txs := blks[21].Transactions()
	for _, blk1Txs := range blk1Txs {
		rec, _ := NewTxRecordFromMsgTx(blk1Txs.MsgTx(), blk1.MsgBlock().Header.Timestamp)
		blk1TxRecords = append(blk1TxRecords, rec)
	}
	tx1 = blks[21].Transactions()[1]
	tx2 = blks[21].Transactions()[2]
	walletID = "ac10d05dhzvc7v3pynscd7mx0ynex6n842gsq74z8g"

}

// func TestSpend(t *testing.T){
// 	for _,blk := range blks{
// 		for _,tx := range blk.Transactions(){

// 		}
// 	}
// }
func TestGetTxByHashHeight(t *testing.T) {
	chainDb, chainDbTearDown, err := GetDb("ChainTestDb")
	if !assert.Nil(t, err) {
		t.Fatal(err)
	}
	defer chainDbTearDown()
	s, walletDb, teardown, err := testTxStore("TstUtxosDb", chainDb)
	if !assert.Nil(t, err) {
		t.Fatal(err)
	}
	defer teardown()
	_ = mwdb.Update(walletDb, func(tx mwdb.DBTransaction) error {
		bucket := tx.FetchBucket(s.bucketMeta.nsTxRecords)
		rec, err := NewTxRecordFromMsgTx(tx1.MsgTx(), blk1.MsgBlock().Header.Timestamp)
		if err != nil {
			t.Errorf("new txrecord error : %v", err)
		}
		err = putTxRecord(bucket, rec, &BlockMeta{Height: blk1.Height(), Hash: blk1.MsgBlock().BlockHash(), Timestamp: blk1.MsgBlock().Header.Timestamp})
		assert.Nil(t, err)
		value, err := fetchRawTxRecordByTxHashHeight(bucket, tx1.Hash(), blk1.MsgBlock().Header.Height)
		if err != nil {
			t.Errorf("first getch error : %v", err)
		}
		assert.NotNil(t, value)
		v, err := fetchLatestRawTxRecordOfHash(bucket, tx1.Hash())
		assert.Nil(t, err)
		t.Log(v)
		err = deleteTxRecord(bucket, tx1.Hash(), &BlockMeta{Height: blk1.Height(), Hash: blk1.MsgBlock().BlockHash(), Timestamp: blk1.MsgBlock().Header.Timestamp})
		assert.Nil(t, err)

		return nil
	})
}
func TestBlockRecord(t *testing.T) {
	chainDb, chainDbTearDown, err := GetDb("ChainTestDb")
	if !assert.Nil(t, err) {
		t.Fatal(err)
	}
	defer chainDbTearDown()
	s, walletDb, teardown, err := testTxStore("TstUtxosDb", chainDb)
	if !assert.Nil(t, err) {
		t.Fatal(err)
	}
	defer teardown()
	blockmeta := &BlockMeta{Height: blk1.Height(), Hash: blk1.MsgBlock().BlockHash(), Timestamp: blk1.MsgBlock().Header.Timestamp}
	brkey := keyBlockRecord(blk1.Height())
	brvalue := valueBlockRecord(blockmeta, &blk1TxRecords[1].Hash)

	apBlockRecord := make([]wire.Hash, 1)
	apBlockRecord = append(apBlockRecord, *tx2.Hash())
	_ = mwdb.Update(walletDb, func(tx mwdb.DBTransaction) error {
		blockH, err := readBlockHashFromValue(brvalue)
		assert.Nil(t, err)
		t.Log(blockH)
		bucket := tx.FetchBucket(s.bucketMeta.nsBlocks)
		err = putRawBlockRecord(bucket, brkey, brvalue)
		assert.Nil(t, err)
		_, v, err := existsBlockRecord(bucket, blk1.Height())
		assert.Nil(t, err)
		t.Log("first rec : ", v)
		updateBlockRecord(bucket, blockmeta, apBlockRecord)
		_, v, err = existsBlockRecord(bucket, blk1.Height())
		assert.Nil(t, err)
		t.Log("second rec : ", v)
		rRecord, err := fetchBlockRecord(bucket, blk1.Height())
		assert.Nil(t, err)
		t.Log(rRecord)
		return nil
	})
}
