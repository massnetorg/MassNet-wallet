package txmgr

import (
	// "bufio"
	"encoding/binary"
	"encoding/hex"

	// "os"
	"testing"

	"github.com/stretchr/testify/assert"
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/massutil"
	mwdb "massnet.org/mass-wallet/masswallet/db"
	"massnet.org/mass-wallet/masswallet/utils"

	// "github.com/stretchr/testify/assert"

	"massnet.org/mass-wallet/wire"
)

func TestCanonicalKeyCreateAndRead(t *testing.T) {
	k := canonicalUnspentKey(walletID, tx1.Hash(), uint32(tx1.Index()))
	t.Log(len(k))
	var op wire.OutPoint
	readCanonicalUnspentKey(k, &op)
	t.Log("hash:", op.Hash, "index:", op.Index)
}
func TestDeleteUnspentKey(t *testing.T) {
	//getDb
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
	//getcanonicalUnspentKey
	key := canonicalUnspentKey(walletID, tx1.Hash(), uint32(tx1.Index()))
	value := make([]byte, 40)
	binary.BigEndian.PutUint64(value[0:8], blk1.Height())
	copy(value[8:40], blk1.Hash().Bytes())
	//put
	_ = mwdb.Update(walletDb, func(tx mwdb.DBTransaction) error {
		//fetch bucket
		bucket := tx.FetchBucket(s.bucketMeta.nsUnspent)
		credkey, err := existsRawUnspent(bucket, key)
		assert.Nil(t, err)
		t.Log("credkey:", credkey)
		//insert
		bucket.Put(key, value)
		credkey, err = existsRawUnspent(bucket, key)
		assert.Nil(t, err)
		credkeyHex := make([]byte, len(credkey)*2)
		hex.Encode(credkeyHex, credkey)
		t.Log("credkey:", string(credkeyHex))
		//delete
		err = deleteRawUnspent(bucket, key)
		assert.Nil(t, err)
		credkey, err = existsRawUnspent(bucket, key)
		assert.Nil(t, err)
		t.Log("credkey:", credkey)
		return nil
	})
}
func TestUnminedInput(t *testing.T) {

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
		bucket := tx.FetchBucket(s.bucketMeta.nsUnminedInputs)
		t.Log(tx1.Hash())
		//create key
		k := canonicalOutPoint(tx1.Hash(), uint32(tx1.Index()))
		//put
		putRawUnminedInput(bucket, k, tx2.Hash().Bytes())
		//get
		result := existsRawUnminedInput(bucket, k)
		t.Log("exists v:", result)
		deleteRawUnminedInput(bucket, k)
		result = existsRawUnminedInput(bucket, k)
		t.Log("delete v:", result)
		return nil
	})
}

func TestUnminedCredit(t *testing.T) {
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
	k := make([]byte, 36)
	copy(k[0:32], tx1.Hash().Bytes())
	binary.BigEndian.PutUint32(k[32:36], uint32(tx1.Index()))
	pkscript, err := utils.ParsePkScript(tx1.MsgTx().TxOut[0].PkScript, &config.ChainParams)
	assert.Nil(t, err)
	amount, err := massutil.NewAmountFromInt(tx1.MsgTx().TxOut[0].Value)
	assert.Nil(t, err)
	v, err := valueUnminedCredit(amount, !(pkscript.IsBinding() || pkscript.IsStaking()), uint32(pkscript.Maturity()), tx1.Hash().Bytes(), pkscript)
	assert.Nil(t, err)
	t.Log(v)
	var cred credit
	err = readUnminedCreditKey(k, &cred)
	assert.Nil(t, err)
	t.Log("hash : ", cred.outPoint.Hash, "index : ", cred.outPoint.Index)

	_ = mwdb.Update(walletDb, func(tx mwdb.DBTransaction) error {
		bucket := tx.FetchBucket(s.bucketMeta.nsUnminedCredits)
		err := putRawUnminedCredit(bucket, k, v)
		assert.Nil(t, err)
		res1V, err := existsRawUnminedCredit(bucket, k)
		t.Log("origin v :", v)
		t.Log("inserted result : ", res1V)
		err = deleteRawUnminedCredit(bucket, k)
		assert.Nil(t, err)
		res2V, err := existsRawUnminedCredit(bucket, k)
		t.Log("deleted result : ", res2V)
		return nil
	})
}
func TestCredit(t *testing.T) {
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
	var cred credit
	var sp indexedIncidence
	var blockmeta BlockMeta = BlockMeta{Height: blk1.Height(), Hash: blk1.MsgBlock().BlockHash(), Timestamp: blk1.MsgBlock().Header.Timestamp}
	unspentk := make([]byte, 76)
	copy(unspentk[0:32], tx1.Hash().Bytes())
	binary.BigEndian.PutUint32(unspentk[32:36], uint32(blk1.Height()))
	copy(unspentk[36:72], blk1.Hash().Bytes())
	binary.BigEndian.PutUint32(unspentk[72:76], uint32(tx1.Index()))
	credk := keyCredit(tx1.Hash(), uint32(tx1.Index()), &blockmeta)
	//unspent credit value
	cred.scriptHash = tx1.Hash().Bytes()
	cred.outPoint.Hash = tx1.MsgTx().TxIn[0].PreviousOutPoint.Hash
	cred.outPoint.Index = tx1.MsgTx().TxIn[0].PreviousOutPoint.Index
	cred.flags.Change = true
	cred.maturity = uint32(tx1.MsgTx().LockTime)
	amo, err := massutil.NewAmountFromInt(tx1.MsgTx().TxOut[0].Value)
	assert.Nil(t, err)
	cred.amount = amo
	credV, err := valueUnspentCredit(&cred)
	assert.Nil(t, err)
	sp.index = uint32(blk2.Transactions()[0].Index())
	sp.incidence.txHash.SetBytes(blk2.Transactions()[0].Hash().Bytes())
	sp.incidence.block = BlockMeta{Height: blk2.Height(), Hash: blk2.MsgBlock().BlockHash(), Timestamp: blk2.MsgBlock().Header.Timestamp}
	_ = mwdb.Update(walletDb, func(tx mwdb.DBTransaction) error {
		bucket := tx.FetchBucket(s.bucketMeta.nsCredits)
		debitBucket := tx.FetchBucket(s.bucketMeta.nsDebits)
		unspentBucket := tx.FetchBucket(s.bucketMeta.nsUnspent)
		err = putRawCredit(bucket, credk, credV)
		assert.Nil(t, err)
		value, err := existsRawCredit(bucket, credk)
		assert.Nil(t, err)
		t.Log("unSpent credit value : ", value)
		unspentValue1, err := fetchNsUnspentValueFromRawCredit(credk)
		assert.Nil(t, err)
		unspentValue2 := valueUnspent(&blockmeta)
		t.Log("unspent key1 : ", unspentValue1, "unspent key2 : ", unspentValue2)
		putRawUnspent(unspentBucket, unspentk, unspentValue1)
		var bm BlockMeta
		readBlockOfUnspent(unspentValue1, &bm)
		t.Log("block height : ", bm.Height, "block hash : ", bm.Hash)

		cCred, err := unspendRawCredit(bucket, credk)
		assert.Nil(t, err)
		t.Log("cCred maturity : ", cCred.maturity, "  cCred scripthash : ", cCred.scriptHash, "  cCred outpoint hash : ", cCred.outPoint.Hash, "  cCred outpoint index : ", cCred.outPoint.Index, "  cCred flag : ", cCred.flags.Change)
		spentAmount, err := spendCredit(bucket, credk, &sp)
		assert.Nil(t, err)
		t.Log("spentAmount : ", spentAmount)
		spentValue, err := existsRawCredit(bucket, credk)
		assert.Nil(t, err)
		t.Log("Spent credit value : ", spentValue)
		spentAmount, spent, err := fetchRawCreditAmountSpent(spentValue)
		assert.Nil(t, err)
		t.Log("read amount : ", spentAmount, "MASS  spentflag : ", spent)
		unminedV, err := valueUnminedCreditFromMined(spentValue)
		assert.Nil(t, err)
		t.Log("unmined value : ", unminedV)
		debitK1 := readCreditSpender(spentValue)
		t.Log("debitKey1 : ", debitK1)
		entrys, err := getCreditsByTxHashHeight(bucket, tx1.Hash(), blk1.Height())
		assert.Nil(t, err)
		for index, entry := range entrys {
			t.Log("credits search by hash height index : ", index, "  entry : ", entry)
		}
		entry, err := getLastCreditByTxHashIndexTillHeight(bucket, tx1.Hash(), uint32(tx1.Index()), blk1.Height())
		assert.Nil(t, err)
		t.Log("last credit entry by txhash,index : ", entry)
		txRecordK, err := fetchTxRecordKeyFromRawCreditKey(credk)
		assert.Nil(t, err)
		t.Log("txRecord key generated by creditK : ", txRecordK, "  creditK : ", credk)
		index, scripthash, err := fetchRawCreditMaturityScriptHash(spentValue)
		assert.Nil(t, err)
		t.Log("tx index by credit value : ", index, "  scripthash : ", scripthash)
		//test debit
		debitk2 := keyDebit(tx1.Hash(), uint32(tx1.Index()), &blockmeta)
		t.Log(debitk2)
		tx1Amount, err := massutil.NewAmountFromInt(tx1.MsgTx().TxOut[0].Value)
		err = putDebit(debitBucket, tx1.Hash(), uint32(tx1.Index()), tx1Amount, &blockmeta, credk)
		assert.Nil(t, err)
		debitK3, credK1, err := existsDebit(debitBucket, tx1.Hash(), uint32(tx1.Index()), &blockmeta)
		t.Log("debitK : ", debitK3, "  credk1", credK1)
		err = deleteRawDebit(debitBucket, debitK3)
		assert.Nil(t, err)
		debitK4, credK2, err := existsDebit(debitBucket, tx1.Hash(), uint32(tx1.Index()), &blockmeta)
		assert.Nil(t, err)
		t.Log("debitK : ", debitK4, "  credk1", credK2)
		return nil
	})
}

func TestLGHistory(t *testing.T) {
	tests := []struct {
		name string
		lgh  lgTxHistory
	}{
		{
			name: "false withdraw true binding",
			lgh: lgTxHistory{
				walletId:    walletID,
				txhash:      *tx1.Hash(),
				indexes:     []uint32{0},
				isWithdraw:  false,
				isBinding:   true,
				blockHeight: blk1.Height(),
			},
		},
		{
			name: "false withdraw false binding",
			lgh: lgTxHistory{
				walletId:    walletID,
				txhash:      *tx1.Hash(),
				indexes:     []uint32{0},
				isWithdraw:  false,
				isBinding:   false,
				blockHeight: blk1.Height(),
			},
		},
		{
			name: "true withdraw false binding",
			lgh: lgTxHistory{
				walletId:    walletID,
				txhash:      *tx1.Hash(),
				indexes:     []uint32{0},
				isWithdraw:  true,
				isBinding:   false,
				blockHeight: blk1.Height(),
			},
		},
		{
			name: "true withdraw true binding",
			lgh: lgTxHistory{
				walletId:    walletID,
				txhash:      *tx1.Hash(),
				indexes:     []uint32{0},
				isWithdraw:  true,
				isBinding:   true,
				blockHeight: blk1.Height(),
			},
		},
	}
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
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_ = mwdb.Update(walletDb, func(tx mwdb.DBTransaction) error {
				unminedLGbucket := tx.FetchBucket(s.bucketMeta.nsUnminedLGHistory)
				minedLGbucket := tx.FetchBucket(s.bucketMeta.nsLGHistory)
				minedLGKey := keyMinedLGHistory(&test.lgh)
				unminedLGKey := keyUnminedLGHistory(&test.lgh)
				LGvalue := valueLGHistory(&test.lgh)
				err := putUnminedLGHistory(unminedLGbucket, &test.lgh)
				assert.Nil(t, err)
				value, err := existsRawLGOutput(unminedLGbucket, unminedLGKey)
				t.Log("generate value :", LGvalue, "  saved value : ", value)
				err = deleteUnminedLGHistory(unminedLGbucket, &test.lgh)
				assert.Nil(t, err)
				err = putLGHistory(minedLGbucket, &test.lgh)
				assert.Nil(t, err)
				value, err = existsRawLGOutput(minedLGbucket, minedLGKey)
				t.Log("mined LG value : ", value)
				otType := func() outputType {
					if test.lgh.isBinding == true {
						return outputBinding
					}
					return outputStaking
				}()
				entries, err := fetchRawLGHistoryByWalletId(minedLGbucket, test.lgh.walletId, otType)
				for k, entry := range entries {
					t.Log("i : ", k, "  entry key : ", entry.Key, "  entry value : ", entry.Value)
				}
				var LGResult lgTxHistory
				readLGHistory(false, minedLGKey, LGvalue, &LGResult)
				t.Log("walletID : ", LGResult.walletId, "  txhash : ", LGResult.txhash, "")
				return nil
			})
		})
	}

}
