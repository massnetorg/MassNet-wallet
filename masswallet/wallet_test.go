package masswallet

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"encoding/hex"

	"github.com/massnetorg/mass-core/blockchain"
	"github.com/massnetorg/mass-core/database"
	"github.com/massnetorg/mass-core/database/ldb"
	"github.com/massnetorg/mass-core/database/memdb"
	"github.com/massnetorg/mass-core/massutil"
	"github.com/massnetorg/mass-core/netsync"
	"github.com/massnetorg/mass-core/txscript"
	"github.com/massnetorg/mass-core/wire"
	"github.com/massnetorg/mass-core/wire/mock"
	"massnet.org/mass-wallet/config"
	mwdb "massnet.org/mass-wallet/masswallet/db"
	_ "massnet.org/mass-wallet/masswallet/db/ldb"
	"massnet.org/mass-wallet/masswallet/keystore"
	"massnet.org/mass-wallet/masswallet/txmgr"
	"massnet.org/mass-wallet/masswallet/utils"
)

const (
	testDbRoot     = "testDbs"
	defaultBitSize = 128
)

var (
	DbType = "leveldb"

	pubPassphrase   = "DJr6BomK"
	privPassphrase  = "81lUHXXd7O9xylj"
	pubPassphrase2  = []byte("fuGpbNZI")
	privPassphrase2 = "lwxo0Psl"

	block15         *wire.MsgBlock
	block15T2       *wire.MsgTx
	block15T3       *wire.MsgTx
	block15Meta     *txmgr.BlockMeta
	b15T2O0PkScript []byte
	b15T3O0PkScript []byte
	b15T2Hash       wire.Hash
	b15T3Hash       wire.Hash

	allTxs map[wire.Hash]*wire.MsgTx

	blks200 []*massutil.Block

	cfg *config.Config
)

func init() {
	f, err := os.Open("./data/mockBlks.dat")
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
		blks200 = append(blks200, blk)
	}
	block15 = blks200[25].MsgBlock()
	block15T2 = block15.Transactions[2]
	block15T3 = block15.Transactions[3]
	block15Meta = &txmgr.BlockMeta{
		Height:    block15.Header.Height,
		Hash:      block15.BlockHash(),
		Timestamp: block15.Header.Timestamp,
	}
	b15T2O0PkScript = block15T2.TxOut[0].PkScript
	b15T3O0PkScript = block15T3.TxOut[0].PkScript
	b15T2Hash = block15T2.TxHash()
	b15T3Hash = block15T3.TxHash()
	allTxs = make(map[wire.Hash]*wire.MsgTx)
	for i := 1; i < 25; i++ {
		blk := blks200[i]
		for _, tx := range blk.Transactions() {
			allTxs[*tx.Hash()] = tx.MsgTx()
		}
	}

	cfg = &config.Config{
		Core:        config.NewDefCoreConfig(),
		Wallet:      config.NewDefWalletConfig(),
		ShowVersion: false,
		Create:      false,
	}

}

type mockServer struct {
	db database.Db
}

func (s *mockServer) Blockchain() *blockchain.Blockchain {
	return nil
}
func (s *mockServer) ChainDB() database.Db {
	return s.db
}
func (s *mockServer) TxMemPool() *blockchain.TxPool {
	return blockchain.NewTxPool(nil, nil, nil)
}

func (s *mockServer) SyncManager() *netsync.SyncManager {
	return nil
}

// filesExists returns whether or not the named file or directory exists.
func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func testDB(dbName string) (mwdb.DB, func(), error) {
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
	db, err := mwdb.CreateDB(DbType, dbPath)
	return db, func() { os.RemoveAll(dbPath) }, err
}

func testBlocks(maturity uint64, height int64, tx int) (*mock.Chain, error) {
	opt := &mock.Option{
		Mode:        mock.Auto,
		TotalHeight: height,
		TxPerBlock:  tx,
	}

	chain, err := mock.NewMockedChain(opt)
	if err != nil {
		return nil, err
	}

	return chain, nil
}

// initializes chain db with first(excluding genesis) n blocks
func newTestChainDB(n int) (database.Db, func(), error) {
	db, err := memdb.NewMemDb()
	if err != nil {
		return nil, nil, err
	}
	err = db.InitByGenesisBlock(massutil.NewBlock(config.ChainParams.GenesisBlock))
	if err != nil {
		return nil, nil, err
	}
	for i := 1; i < n; i++ {
		err = db.SubmitBlock(blks200[i])
		if err != nil {
			return nil, nil, err
		}
		db.(*ldb.ChainDb).Batch(1).Set(blks200[i].MsgBlock().BlockHash())
		db.(*ldb.ChainDb).Batch(1).Done()
		err = db.Commit(blks200[i].MsgBlock().BlockHash())
		if err != nil {
			return nil, nil, err
		}
	}
	return db, func() {
		db.Close()
		os.RemoveAll("./blocks")
	}, err
}

func TestNewWallet(t *testing.T) {
	databaseDb, close, err := newTestChainDB(0)
	if err != nil {
		t.Fatal("new databaseDb error")
	}
	defer close()
	walletDb, teardown, err := testDB("testNewWallet")
	if err != nil {
		t.Fatal("new walletDb error")
	}
	defer teardown()
	_, err = NewWalletManager(&mockServer{databaseDb}, walletDb, cfg, config.ChainParams, pubPassphrase)
	if err != nil {
		t.Fatal("new wallet error", err.Error())
	}
}

//TODO: removeWallet
func TestWalletManager_CreateWallet_UseWallet_Wallets_WalletBalance(t *testing.T) {
	databaseDb, close, err := newTestChainDB(0)
	if err != nil {
		t.Fatal("new databaseDb error")
	}
	defer close()
	walletDb, teardown, err := testDB("testNewWallet")
	if err != nil {
		t.Fatal("new walletDb error")
	}
	defer teardown()
	w, err := NewWalletManager(&mockServer{databaseDb}, walletDb, cfg, config.ChainParams, pubPassphrase)
	if err != nil {
		t.Fatal("new wallet error", err.Error())
	}

	walletId1, _, version, err := w.CreateWallet(privPassphrase, "", defaultBitSize)
	if err != nil {
		t.Fatal("create wallet error", err.Error())
	}
	assert.True(t, version == keystore.KeystoreVersionLatest.Value())
	t.Log("wallet_1_Id: ", walletId1)
	walletId2, _, version, err := w.CreateWallet(privPassphrase2, "", defaultBitSize)
	if err != nil {
		t.Fatal("create wallet error", err.Error())
	}
	assert.True(t, version == keystore.KeystoreVersionLatest.Value())
	t.Log("wallet_2_Id: ", walletId2)

	wallets, err := w.Wallets()
	if err != nil {
		t.Fatal("get wallets error", err.Error())
	}
	for index, wallet := range wallets {
		t.Log()
		t.Logf("wallet%v_id: %v", index, wallet.WalletID)
		t.Logf("wallet%v_type: %v", index, wallet.Type)
		t.Logf("wallet%v_remark: %v", index, wallet.Remarks)
	}
	t.Log("beforeUseWallet_currentKeystore:", w.ksmgr.CurrentKeystore())
	wInfo, err := w.UseWallet(walletId2)
	if err != nil {
		t.Fatal("use wallet error", err.Error())
	}
	t.Log("afterUseWallet_currentKeystore:", w.ksmgr.CurrentKeystore().Name())
	t.Log("wInfo_id: ", wInfo.WalletID)
	t.Log("wInfo_remarks: ", wInfo.Remarks)
	t.Log("wInfo_type: ", wInfo.Type)
	t.Log("wInfo_chainId: ", wInfo.ChainID)
	t.Log("wInfo_externalKeyCount: ", wInfo.ExternalKeyCount)
	t.Log("wInfo_internalKeyCount: ", wInfo.InternalKeyCount)
	t.Log("wInfo_balance: ", wInfo.TotalBalance)
	wBal, err := w.WalletBalance(0, true)
	if err != nil {
		t.Fatal("get wallet balance error", err.Error())
	}
	t.Log("walletBalance_id: ", wBal.WalletID)
	t.Log("walletBalance_total: ", wBal.Total)
	t.Log("walletBalance_spendable: ", wBal.Spendable)
	//err = w.RemoveWallet(walletId1, privPassphrase)
	//if err != nil {
	//	t.Fatal("remove wallet error", err.Error())
	//}
	//t.Log("remove wallet")
	//wallets0 := w.Wallets()
	//for index, wallet := range wallets0 {
	//	t.Log()
	//	t.Logf("wallet%v_id: %v", index, wallet.WalletID)
	//	t.Logf("wallet%v_type: %v", index, wallet.Type)
	//	t.Logf("wallet%v_remark: %v", index, wallet.Remarks)
	//}

}

func TestWalletManager_NewAddress_GetAddresses(t *testing.T) {
	databaseDb, close, err := newTestChainDB(0)
	if err != nil {
		t.Fatal("new databaseDb error")
	}
	defer close()
	walletDb, teardown, err := testDB("testNewWallet")
	if err != nil {
		t.Fatal("new walletDb error")
	}
	defer teardown()
	w, err := NewWalletManager(&mockServer{databaseDb}, walletDb, cfg, config.ChainParams, pubPassphrase)
	if err != nil {
		t.Fatal("new wallet error", err.Error())
	}

	walletId1, _, version, err := w.CreateWallet(privPassphrase, "", defaultBitSize)
	if err != nil {
		t.Fatal("create wallet error", err.Error())
	}
	assert.True(t, version == keystore.KeystoreVersionLatest.Value())
	t.Log("wallet_1_Id: ", walletId1)
	walletId2, _, version, err := w.CreateWallet(privPassphrase2, "", defaultBitSize)
	if err != nil {
		t.Fatal("create wallet error", err.Error())
	}
	assert.True(t, version == keystore.KeystoreVersionLatest.Value())
	t.Log("wallet_2_Id: ", walletId2)

	// error_test_1 ErrCurrentKeystoreNotFound
	_, err = w.NewAddress(0)
	if err != ErrNoWalletInUse {
		t.Fatalf("NewAddress: mismatched error -- got: %v, want: %v", err, ErrNoWalletInUse)
	}

	//wallet2
	_, err = w.UseWallet(walletId2)
	if err != nil {
		t.Fatal("use wallet error", err.Error())
	}
	addr2, err := w.NewAddress(0)
	if err != nil {
		t.Fatal("new addr error", err.Error())
	}
	lockAddr2, err := w.NewAddress(1)
	if err != nil {
		t.Fatal("new addr error", err.Error())
	}
	t.Log()
	t.Log("walletId:", walletId2)
	t.Log("Test_NewAddress_GetAddress")
	t.Logf("generate addr:%v,lockAddr:%v", addr2, lockAddr2)
	getAddrs2, err := w.GetAddresses(math.MaxUint16)
	if err != nil {
		t.Fatal("get addresses error", err.Error())
	}
	t.Log("get from walletdb")
	for index, getAddr := range getAddrs2 {
		t.Logf("%v_addr: %v", index, getAddr.Address)
		t.Logf("%v_version: %v", index, getAddr.AddressClass)
		t.Logf("%v_used: %v", index, getAddr.Used)
	}

	//wallet1
	_, err = w.UseWallet(walletId1)
	if err != nil {
		t.Fatal("use wallet error", err.Error())
	}

	addr1, err := w.NewAddress(0)
	if err != nil {
		t.Fatal("new addr error", err.Error())
	}
	stakingAddr, err := w.NewAddress(1)
	if err != nil {
		t.Fatal("new addr error", err.Error())
	}
	t.Log()
	t.Log("walletId:", walletId1)
	t.Log("Test_NewAddress_GetAddress")
	t.Logf("generate addr:%v,lockAddr:%v", addr1, stakingAddr)
	getAddrs1, err := w.GetAddresses(math.MaxUint16)
	if err != nil {
		t.Fatal("get addresses error", err.Error())
	}
	t.Log("get from walletdb")
	for index, getAddr := range getAddrs1 {
		t.Logf("%v_addr: %v", index, getAddr.Address)
		t.Logf("%v_version: %v", index, getAddr.AddressClass)
		t.Logf("%v_used: %v", index, getAddr.Used)
	}

}

//TODO: importWallet
func TestWalletManager_ExportWallet_ImportWallet(t *testing.T) {
	databaseDb, close, err := newTestChainDB(0)
	if err != nil {
		t.Fatal("new databaseDb error")
	}
	defer close()
	walletDb1, teardown, err := testDB("testNewWallet1")
	if err != nil {
		t.Fatal("new walletDb error")
	}
	defer teardown()
	w, err := NewWalletManager(&mockServer{databaseDb}, walletDb1, cfg, config.ChainParams, pubPassphrase)
	if err != nil {
		t.Fatal("new wallet error", err.Error())
	}
	walletId1, _, version, err := w.CreateWallet(privPassphrase, "", defaultBitSize)
	if err != nil {
		t.Fatal("create wallet error", err.Error())
	}
	assert.True(t, version == keystore.KeystoreVersionLatest.Value())
	t.Log("wallet_1_Id: ", walletId1)
	_, err = w.UseWallet(walletId1)
	if err != nil {
		t.Fatal("use wallet error", err.Error())
	}
	addr1, err := w.NewAddress(0)
	if err != nil {
		t.Fatal("new addr error", err.Error())
	}
	stakingAddr, err := w.NewAddress(1)
	if err != nil {
		t.Fatal("new addr error", err.Error())
	}
	t.Log("wallet1_addr1:", addr1)
	t.Log("wallet1_lockAddr1:", stakingAddr)
	//export
	wJson, err := w.ExportWallet(walletId1, privPassphrase)
	if err != nil {
		t.Fatal("export wallet error", err.Error())
	}
	t.Log("export_walletJson: ", wJson)

	//import
	walletDb2, teardown2, err := testDB("testNewWallet2")
	if err != nil {
		t.Fatal("new walletDb error")
	}
	defer teardown2()
	w, err = NewWalletManager(&mockServer{databaseDb}, walletDb2, cfg, config.ChainParams, pubPassphrase)
	if err != nil {
		t.Fatal("new wallet error", err.Error())
	}
	walletId2, _, version, err := w.CreateWallet(privPassphrase2, "", defaultBitSize)
	if err != nil {
		t.Fatal("create wallet error", err.Error())
	}
	assert.True(t, version == keystore.KeystoreVersionLatest.Value())
	t.Log("wallet_2_Id: ", walletId2)
	_, err = w.UseWallet(walletId2)
	if err != nil {
		t.Fatal("use wallet error", err.Error())
	}

	_, err = w.NewAddress(0)
	if err != nil {
		t.Fatal("new addr error", err.Error())
	}
}

func decodeHexStr(hexStr string) ([]byte, error) {
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}

func simpleFilterTx(rec *txmgr.TxRecord, tx *wire.MsgTx, walletId string) (*txmgr.TxRecord, error) {
	// // check TxIn
	// if !blockchain.IsCoinBaseTx(tx) {
	// 	for i, txIn := range tx.TxIn {
	// 		prevTx := allTxs[txIn.PreviousOutPoint.Hash]

	// 		pkScript := prevTx.TxOut[txIn.PreviousOutPoint.Index].PkScript
	// 		ps, err := utils.ParsePkScript(pkScript, &config.ChainParams)
	// 		if err != nil {
	// 			logging.CPrint(logging.ERROR, "failed to parse txin pkscript",
	// 				logging.LogFormat{
	// 					"tx":        tx.TxHash().String(),
	// 					"txInIndex": i,
	// 					"err":       err,
	// 				})
	// 			return nil, err
	// 		}

	// 		rec.HasBindingIn = ps.IsBinding()
	// 		rec.RelevantTxIn = append(rec.RelevantTxIn,
	// 			&txmgr.RelevantMeta{
	// 				Index:    i,
	// 				PkScript: ps,
	// 				WalletId: walletId,
	// 			})
	// 	}
	// }

	// check TxOut
	for i, txOut := range tx.TxOut {
		ps, err := utils.ParsePkScript(txOut.PkScript, config.ChainParams)
		if err != nil {

			return nil, err
		}
		rec.HasBindingOut = ps.IsBinding()
		rec.RelevantTxOut = append(rec.RelevantTxOut,
			&txmgr.RelevantMeta{
				Index:    i,
				PkScript: ps,
				WalletId: walletId,
			})
	}
	return rec, nil
}

func TestWalletManager_EstimateTxFee(t *testing.T) {
	chainDb, close, err := newTestChainDB(0)
	if err != nil {
		t.Fatal("newTestChainDB error:", err)
	}
	defer close()
	walletDb1, teardown, err := testDB("testNewWallet1")
	if err != nil {
		t.Fatal("new walletDb error")
	}
	defer teardown()
	w, err := NewWalletManager(&mockServer{chainDb}, walletDb1, cfg, config.ChainParams, pubPassphrase)
	if err != nil {
		t.Fatal("new wallet error", err.Error())
	}
	walletId1, _, version, err := w.CreateWallet(privPassphrase, "", defaultBitSize)
	if err != nil {
		t.Fatal("create wallet error", err.Error())
	}
	assert.True(t, version == keystore.KeystoreVersionLatest.Value())
	t.Log("wallet_1_Id: ", walletId1)
	_, err = w.UseWallet(walletId1)
	if err != nil {
		t.Fatal("use wallet error", err.Error())
	}

	addr1, err := w.NewAddress(0)
	if err != nil {
		t.Fatal("new addr error", err.Error())
	}
	addr2, err := w.NewAddress(0)
	if err != nil {
		t.Fatal("new addr error", err.Error())
	}
	addrInterface, err := massutil.DecodeAddress(addr1, w.chainParams)
	if err != nil {
		t.Fatal("decode addr error", err.Error())
	}

	scriptHash := addrInterface.ScriptAddress()
	pkScript, err := txscript.PayToWitnessScriptHashScript(scriptHash)
	if err != nil {
		t.Fatal("get pkscript error", err.Error())
	}

	// replace some transactions
	block15T2.TxOut[0].PkScript = pkScript
	block15T3.TxOut[0].PkScript = pkScript
	newBlock15T2Hash := block15T2.TxHash()
	newBlock15T3Hash := block15T3.TxHash()
	for i := 1; i <= int(block15Meta.Height); i++ {
		blk := blks200[i]
		blk.ResetGenerated()
		for _, tx := range blk.MsgBlock().Transactions {
			for _, txin := range tx.TxIn {
				if bytes.Equal(txin.PreviousOutPoint.Hash[:], b15T2Hash[:]) {
					txin.PreviousOutPoint.Hash = newBlock15T2Hash
					continue
				}
				if bytes.Equal(txin.PreviousOutPoint.Hash[:], b15T3Hash[:]) {
					txin.PreviousOutPoint.Hash = newBlock15T3Hash
				}
			}
		}
		err = chainDb.SubmitBlock(blk)
		if err != nil {
			t.Fatal("init db error:", err)
		}
		chainDb.(*ldb.ChainDb).Batch(1).Set(blk.MsgBlock().BlockHash())
		chainDb.(*ldb.ChainDb).Batch(1).Done()
		err = chainDb.Commit(blk.MsgBlock().BlockHash())
		if err != nil {
			t.Fatal("init db error:", err)
		}
	}
	defer func() {
		block15T2.TxOut[0].PkScript = b15T2O0PkScript
		block15T3.TxOut[0].PkScript = b15T3O0PkScript
		for i := 1; i <= int(block15Meta.Height); i++ {
			blk := blks200[i]
			for _, tx := range blk.MsgBlock().Transactions {
				for _, txin := range tx.TxIn {
					if bytes.Equal(txin.PreviousOutPoint.Hash[:], newBlock15T2Hash[:]) {
						txin.PreviousOutPoint.Hash = b15T2Hash
						continue
					}
					if bytes.Equal(txin.PreviousOutPoint.Hash[:], newBlock15T3Hash[:]) {
						txin.PreviousOutPoint.Hash = b15T3Hash
					}
				}
			}
		}
	}()

	rec, err := txmgr.NewTxRecordFromMsgTx(block15T2, block15.Header.Timestamp)
	if err != nil {
		t.Fatal("get txRecord error", err.Error())
	}
	rec, err = simpleFilterTx(rec, block15T2, walletId1)
	rec0, err := txmgr.NewTxRecordFromMsgTx(block15T3, block15.Header.Timestamp)
	if err != nil {
		t.Fatal("get txRecord error", err.Error())
	}
	rec0, err = simpleFilterTx(rec0, block15T3, walletId1)
	allBalances := map[string]massutil.Amount{
		walletId1: massutil.ZeroAmount(),
	}
	blkLoc, err := w.chainFetcher.FetchBlockLocByHeight(block15Meta.Height)
	if err != nil {
		t.Fatal("FetchBlockLocByHeight error:", err)
	} else {
		t.Log("block15 location:", blkLoc)
	}
	block15Meta.Loc = blkLoc
	txLocs, err := blks200[block15Meta.Height].TxLoc()
	if err != nil {
		t.Fatal("TxLoc error", err)
	}
	rec.TxLoc = &txLocs[2]
	rec0.TxLoc = &txLocs[3]
	err = mwdb.Update(walletDb1, func(tx mwdb.DBTransaction) error {
		err = w.txStore.AddRelevantTx(tx, allBalances, rec, block15Meta)
		if err != nil {
			t.Fatal("add relevantTx error", err.Error())
		}
		err = w.txStore.AddRelevantTx(tx, allBalances, rec0, block15Meta)
		if err != nil {
			t.Fatal("add relevantTx error", err.Error())
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	addrs := make([]string, 0)
	addrs = append(addrs, addr1)
	addrBals, err := w.AddressBalance(0, addrs)
	if err != nil {
		t.Fatal("get addresses balance error", err.Error())
	}
	t.Log()
	t.Log("Test_AddressBalance")
	for index, addrBal := range addrBals {
		t.Logf("%v_addr: %v", index, addrBal.Address)
		t.Logf("%v_total: %v", index, addrBal.Total)
		t.Logf("%v_spendable: %v", index, addrBal.Spendable)
	}

	amt2, err := massutil.NewAmountFromUint(20e8)
	assert.Nil(t, err)
	amt1, err := massutil.NewAmountFromUint(4e8)
	assert.Nil(t, err)

	txOuts := map[string]massutil.Amount{
		addr2: amt2,
		addr1: amt1,
	}
	//minTxFee
	incompleteTx, txFee, err := w.EstimateTxFee(txOuts, 0, massutil.ZeroAmount(), "", "", nil)
	if err != nil {
		t.Fatal("estimate txFee error", err.Error())
	}
	t.Log("/*minTxFee*/")
	t.Log("incompleteTx: ", incompleteTx)
	t.Log("txFee: ", txFee)
	for index, inctx := range incompleteTx.TxOut {
		t.Logf("txOut_%v_value:%v", index, inctx.Value)
	}
	//SetTxFee
	amt, err := massutil.NewAmountFromUint(10e8)
	assert.Nil(t, err)
	incompleteTx0, txFee0, err := w.EstimateTxFee(txOuts, 0, amt, "", "", nil)
	if err != nil {
		t.Fatal("estimate txFee error", err.Error())
	}
	t.Log()
	t.Log("/*user set txfee*/")
	t.Log("incompleteTx: ", incompleteTx0)
	t.Log("txFee0: ", txFee0)
	for index0, inctx0 := range incompleteTx0.TxOut {
		t.Logf("txOut_%v_value:%v", index0, inctx0.Value)
	}

}

func TestWalletManager_AutoConstructTx(t *testing.T) {
	chainDb, close, err := newTestChainDB(0)
	if err != nil {
		t.Fatal("new chainDb error")
	}
	defer close()
	walletDb1, teardown, err := testDB("testNewWallet1")
	if err != nil {
		t.Fatal("new walletDb error")
	}
	defer teardown()
	w, err := NewWalletManager(&mockServer{chainDb}, walletDb1, cfg, config.ChainParams, pubPassphrase)
	if err != nil {
		t.Fatal("new wallet error", err.Error())
	}
	walletId1, _, version, err := w.CreateWallet(privPassphrase, "", defaultBitSize)
	if err != nil {
		t.Fatal("create wallet error", err.Error())
	}
	assert.True(t, version == keystore.KeystoreVersionLatest.Value())
	t.Log("wallet_1_Id: ", walletId1)
	_, err = w.UseWallet(walletId1)
	if err != nil {
		t.Fatal("use wallet error", err.Error())
	}

	addr1, err := w.NewAddress(0)
	if err != nil {
		t.Fatal("new addr error", err.Error())
	}
	addr2, err := w.NewAddress(0)
	if err != nil {
		t.Fatal("new addr error", err.Error())
	}
	addrInterface, err := massutil.DecodeAddress(addr1, w.chainParams)
	if err != nil {
		t.Fatal("decode addr error", err.Error())
	}

	scriptHash := addrInterface.ScriptAddress()
	pkScript, err := txscript.PayToWitnessScriptHashScript(scriptHash)
	if err != nil {
		t.Fatal("get pkscript error", err.Error())
	}

	// replace some transactions
	block15T2.TxOut[0].PkScript = pkScript
	block15T3.TxOut[0].PkScript = pkScript
	newBlock15T2Hash := block15T2.TxHash()
	newBlock15T3Hash := block15T3.TxHash()
	for i := 1; i <= int(block15Meta.Height); i++ {
		blk := blks200[i]
		blk.ResetGenerated()
		for _, tx := range blk.MsgBlock().Transactions {
			for _, txin := range tx.TxIn {
				if bytes.Equal(txin.PreviousOutPoint.Hash[:], b15T2Hash[:]) {
					txin.PreviousOutPoint.Hash = newBlock15T2Hash
					continue
				}
				if bytes.Equal(txin.PreviousOutPoint.Hash[:], b15T3Hash[:]) {
					txin.PreviousOutPoint.Hash = newBlock15T3Hash
				}
			}
		}
		err = chainDb.SubmitBlock(blk)
		if err != nil {
			t.Fatal("init db error:", err)
		}
		chainDb.(*ldb.ChainDb).Batch(1).Set(blk.MsgBlock().BlockHash())
		chainDb.(*ldb.ChainDb).Batch(1).Done()
		err = chainDb.Commit(blk.MsgBlock().BlockHash())
		if err != nil {
			t.Fatal("init db error:", err)
		}
	}
	defer func() {
		block15T2.TxOut[0].PkScript = b15T2O0PkScript
		block15T3.TxOut[0].PkScript = b15T3O0PkScript
		for i := 1; i <= int(block15Meta.Height); i++ {
			blk := blks200[i]
			for _, tx := range blk.MsgBlock().Transactions {
				for _, txin := range tx.TxIn {
					if bytes.Equal(txin.PreviousOutPoint.Hash[:], newBlock15T2Hash[:]) {
						txin.PreviousOutPoint.Hash = b15T2Hash
						continue
					}
					if bytes.Equal(txin.PreviousOutPoint.Hash[:], newBlock15T3Hash[:]) {
						txin.PreviousOutPoint.Hash = b15T3Hash
					}
				}
			}
		}
	}()

	rec, err := txmgr.NewTxRecordFromMsgTx(block15T2, block15.Header.Timestamp)
	if err != nil {
		t.Fatal("get txRecord error", err.Error())
	}
	rec, err = simpleFilterTx(rec, block15T2, walletId1)
	rec0, err := txmgr.NewTxRecordFromMsgTx(block15T3, block15.Header.Timestamp)
	if err != nil {
		t.Fatal("get txRecord error", err.Error())
	}
	rec0, err = simpleFilterTx(rec0, block15T3, walletId1)
	allBalances := map[string]massutil.Amount{
		walletId1: massutil.ZeroAmount(),
	}
	blkLoc, err := w.chainFetcher.FetchBlockLocByHeight(block15Meta.Height)
	if err != nil {
		t.Fatal("FetchBlockLocByHeight error:", err)
	} else {
		t.Log("block15 location:", blkLoc)
	}
	block15Meta.Loc = blkLoc
	txLocs, err := blks200[block15Meta.Height].TxLoc()
	if err != nil {
		t.Fatal("TxLoc error", err)
	}
	rec.TxLoc = &txLocs[2]
	rec0.TxLoc = &txLocs[3]
	err = mwdb.Update(walletDb1, func(tx mwdb.DBTransaction) error {
		err = w.txStore.AddRelevantTx(tx, allBalances, rec, block15Meta)
		if err != nil {
			t.Fatal("add relevantTx error", err.Error())
		}
		err = w.txStore.AddRelevantTx(tx, allBalances, rec0, block15Meta)
		if err != nil {
			t.Fatal("add relevantTx error", err.Error())
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	addrs := make([]string, 0)
	addrs = append(addrs, addr1)
	addrBals, err := w.AddressBalance(0, addrs)
	if err != nil {
		t.Fatal("get addresses balance error", err.Error())
	}
	t.Log()
	t.Log("Test_AddressBalance")
	for index, addrBal := range addrBals {
		t.Logf("%v_addr: %v", index, addrBal.Address)
		t.Logf("%v_total: %v", index, addrBal.Total)
		t.Logf("%v_spendable: %v", index, addrBal.Spendable)
	}

	amt2, err := massutil.NewAmountFromUint(20e8)
	assert.Nil(t, err)
	amt1, err := massutil.NewAmountFromUint(4e8)
	assert.Nil(t, err)

	txOuts := map[string]massutil.Amount{
		addr2: amt2,
		addr1: amt1,
	}
	amt, err := massutil.NewAmountFromUint(10e8)
	assert.Nil(t, err)
	txHex, txFee, err := w.AutoCreateRawTransaction(txOuts, 0, amt, "", "", nil)
	if err != nil {
		t.Fatal("AutoCreateRawTransaction error", err.Error())
	}
	t.Log()
	t.Log("txFee: ", txFee)
	t.Log("txHex:", txHex)

	serializedTx, err := decodeHexStr(txHex)
	if err != nil {
		t.Fatal("decode hexStr error", err.Error())
	}
	var mtx wire.MsgTx
	err = mtx.SetBytes(serializedTx, wire.Packet)
	if err != nil {
		t.Fatal("deserialize tx error", err.Error())
	}
	signedTx, err := w.SignRawTx([]byte(privPassphrase), "ALL", &mtx)
	if err != nil {
		t.Fatal("sign tx error", err.Error())
	}
	t.Log("signedTx:", signedTx)

}
