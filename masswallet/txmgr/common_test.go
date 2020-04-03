package txmgr

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"path/filepath"

	_ "massnet.org/mass-wallet/database/storage/ldbstorage"

	"massnet.org/mass-wallet/database"
	"massnet.org/mass-wallet/database/ldb"

	"massnet.org/mass-wallet/blockchain"
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/errors"
	mwdb "massnet.org/mass-wallet/masswallet/db"
	"massnet.org/mass-wallet/masswallet/ifc"
	"massnet.org/mass-wallet/masswallet/keystore"

	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/wire"
)

// testDbRoot is the root directory used to create all test databases.
const (
	testDbRoot      = "testDbs"
	dbtype          = "leveldb"
	addressGapLimit = uint32(20)
)

var (
	blks200 []*massutil.Block
)

var (
	DbType = "leveldb"

	pubPassphrase  = []byte("@DJr@fL4H0O#$%0^n@V1")
	privPassphrase = []byte("@#XXd7O9xyDIWIbXX$lj")
	// // fastScrypt are parameters used throughout the tests to speed up the
	// // scrypt operations.
	// fastScrypt = &keystore.ScryptOptions{
	// 	N: 16,
	// 	R: 8,
	// 	P: 1,
	// }
	ks_testnet = "7b2272656d61726b223a226669727374222c2263727970746f223a7b22636970686572223a2253747265616d20636970686572222c226d61737465724844" +
		"507269764b6579456e63223a22363336343439306139343330613665663635613937323863633663386238336232666337333234313366363366663464643935333734" +
		"62663862633861306438313262653162326261633364376636363065643531646232646330323634376233626366623838613839646137346234303237643461303431" +
		"653336646339643264366638363234363936313739303266353236396164666466373465336239373763653563366239656237396464316362656137653236303462396" +
		"135363863366164326161613430666236393236326464393939383739623235306136626134323939383230333664303836376437346337393561393233633335613763" +
		"61373032353830376261613931333530383436343863656138653237633964366366303065613035363238623331222c226b6466223a22736372797074222c227075625" +
		"06172616d73223a223734626234303531643339356361646562626135343364616464663835313463386262363563623139633061363336323665386635366232623136" +
		"386132656235303138663834643664613131613033663532333235316238363934343363616236373865373532613534653162613330353665363764663463373530376" +
		"139313030303030303030303030303030303038303030303030303030303030303030313030303030303030303030303030222c2270726976506172616d73223a223133" +
		"363663643662313437326266326565643937623164346138366637646236396261613136623763346435633333643963643837323930313134333466643966353163386" +
		"561646336386238613531653131613937653536373133663864636266373333326361376537353537323539633339363032316563306437663730313030303030303030" +
		"303030303030303038303030303030303030303030303030313030303030303030303030303030222c2263727970746f4b6579507562456e63223a22363862653430343" +
		"733353035346335663964323938356364623365356634303865646139633564666163356266653165343235396566373039326532313135646232636631663836613038" +
		"336163613264376438383737323761376334396537306666646466356234616466663361613935363635653961323139613964336430376365373836316532626462333" +
		"762222c2263727970746f4b657950726976456e63223a223532623436373961303435356632343862653434313364363861353033313733636530326364343437383035" +
		"613639613163306239323737616266383163396537306564303432303361386665616334663532316161633934613236646639653234616633313134646361366631633" +
		"33931313661636365333437626137313732373264366637643433343732373336222c2263727970746f4b6579536372697074456e63223a226534323663383664343735" +
		"353162623036353166363161626430336263613233646161356332313964623839653630336265636639663539373065376230386162643730356366343266306561303" +
		"23737323538333539393734346234633236366666633761646537623835306138393566323439663435633037393939323963663439626431386565326538316439227d" +
		"2c22686450617468223a7b22507572706f7365223a34342c22436f696e223a312c224163636f756e74223a312c2245787465726e616c4368696c644e756d223a33302c2" +
		"2496e7465726e616c4368696c644e756d223a307d7d"

	ks_mainnet             = "{\"remarks\":\"init run\",\"crypto\":{\"cipher\":\"Stream cipher\",\"entropyEnc\":\"408998f673619fafe25a820588f12c0b9fed25a0ec2fad33128abc62644cd9d80c5e9f2f1f23df1862058ff7622bb097185c45f6b59697ec\",\"kdf\":\"scrypt\",\"privParams\":\"9d5d2f6de075ed1f8c46d590a823c67bcbdb25159ba3caf50426c27b575821a95daa891a93be42c900f40c1c6f1ae72c19cf3ffbefe45bb3b67643988a517cb2000004000000000008000000000000000100000000000000\",\"cryptoKeyEntropyEnc\":\"8b5d8cf78697d88c7a9e3143862c8db45b7a9729e5976df99ef586c7ebfd3b35a3ab2d82b606eaa9ca1f7c7b0bf21a585e87aec423e48c1e4d0d45745b5a7d4ae5c1c688c2cd9ca1\"},\"hdPath\":{\"Purpose\":44,\"Coin\":297,\"Account\":1,\"ExternalChildNum\":5,\"InternalChildNum\":0}}"
	privPassphrase_mainnet = []byte("111111")
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
		blks200 = append(blks200, blk)
	}
}

// the 1st block is genesis
func loadNthBlk(nth int) (*massutil.Block, error) {
	if nth <= 0 || nth > len(blks200) {
		return nil, errors.New("invalid nth")
	}
	return blks200[nth-1], nil
}

// the 1st block is genesis
func loadTopNBlk(n int) ([]*massutil.Block, error) {
	if n <= 0 || n > len(blks200) {
		return nil, errors.New("invalid n")
	}

	return blks200[:n], nil
}

func insertBlock(db database.Db, blk *massutil.Block) error {
	err := db.SubmitBlock(blk)
	if err != nil {
		return err
	}
	cdb := db.(*ldb.ChainDb)
	cdb.Batch(1).Set(*blk.Hash())
	cdb.Batch(1).Done()
	return db.Commit(*blk.Hash())
}

func insertBlock2(bc *blockchain.Blockchain, num *wire.MsgBlock, db *database.Db) error {
	block := massutil.NewBlock(num)
	prevNode, err := bc.TstgetPrevNodeFromBlock(block)
	if err != nil {
		return err
	}

	blockHeader := num.Header
	hash := num.BlockHash()
	newNode := blockchain.NewBlockNode(&blockHeader, &hash, blockchain.BFNone)
	if prevNode != nil {
		newNode.Parent = prevNode
	}
	txInputStore, err := bc.TstFetchInputTransactions(newNode, block)
	if err != nil {
		return err
	}
	AddrIndexer, err := blockchain.NewAddrIndexer(*db, nil)
	if err != nil {
		return err
	}
	// Insert the block into the database which houses the main chain.
	if err = (*db).SubmitBlock(block); err != nil {
		return err
	}
	if err = AddrIndexer.SyncAttachBlock(block, txInputStore); err != nil {
		return err
	}
	if err = (*db).Commit(hash); err != nil {
		return err
	}
	return nil
}

func initBlocks(db database.Db, numBlks int) error {
	blks, err := loadTopNBlk(numBlks)
	if err != nil {
		return err
	}

	for i, blk := range blks {
		if i == 0 {
			continue
		}

		err = db.SubmitBlock(blk)
		if err != nil {
			return err
		}
		cdb := db.(*ldb.ChainDb)
		cdb.Batch(1).Set(*blk.Hash())
		cdb.Batch(1).Done()
		err = db.Commit(*blk.Hash())
		if err != nil {
			return err
		}
	}

	_, height, err := db.NewestSha()
	if err != nil {
		return err
	}
	if height != uint64(numBlks-1) {
		return errors.New("incorrect best height")
	}
	return nil
}

func GetDb(dbName string) (database.Db, func(), error) {
	// Create the root directory for test databases.
	if !fileExists(testDbRoot) {
		if err := os.MkdirAll(testDbRoot, 0700); err != nil {
			err := fmt.Errorf("unable to create test db "+"root: %v", err)
			return nil, nil, err
		}
	}
	dbPath := filepath.Join(testDbRoot, dbName)
	_ = os.RemoveAll(dbPath)
	db, err := database.CreateDB(DbType, dbPath)
	if err != nil {
		fmt.Println("create db error: ", err)
		return nil, nil, err
	}

	// Get the latest block height from the database.
	_, height, err := db.NewestSha()
	if err != nil {
		db.Close()
		return nil, nil, err
	}

	// Insert the appropriate genesis block for the Mass network being
	// connected to if needed.
	if height == math.MaxUint64 {
		blk, err := loadNthBlk(1)
		if err != nil {
			return nil, nil, err
		}
		err = db.InitByGenesisBlock(blk)
		if err != nil {
			db.Close()
			return nil, nil, err
		}
		height = blk.Height()
		copy(config.ChainParams.GenesisHash[:], blk.Hash()[:])
	}

	// Setup a tearDown function for cleaning up.  This function is
	// returned to the caller to be invoked when it is done testing.
	tearDown := func() {
		dbVersionPath := filepath.Join(testDbRoot, dbName+".ver")
		db.Sync()
		db.Close()
		os.RemoveAll(dbPath)
		os.Remove(dbVersionPath)
		os.RemoveAll(testDbRoot)
	}
	return db, tearDown, nil
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

func testTxStore(txStoreName string, databaseDb database.Db) (txstore *TxStore, walletDb mwdb.DB, tearDown func(), err error) {

	if !fileExists(testDbRoot) {
		if err := os.MkdirAll(testDbRoot, 0700); err != nil {
			err := fmt.Errorf("unable to create test db "+
				"root: %v", err)
			return nil, nil, nil, err
		}
	}
	dbPath := filepath.Join(testDbRoot, txStoreName)
	if err := os.RemoveAll(dbPath); err != nil {
		err := fmt.Errorf("cannot remove old db: %v", err)
		return nil, nil, nil, err
	}
	walletDb, err = mwdb.CreateDB(DbType, dbPath)
	if err != nil {
		return nil, nil, nil, err
	}
	chainFetcher := ifc.NewChainFetcher(databaseDb)
	var s *TxStore
	err = mwdb.Update(walletDb, func(tx mwdb.DBTransaction) error {
		// init KeystoreManager
		var err error
		bucket, err := mwdb.GetOrCreateTopLevelBucket(tx, keystoreBucket)
		if err != nil {
			return err
		}
		ksmgr, err := keystore.NewKeystoreManager(bucket, pubPassphrase, &config.ChainParams)
		if err != nil {
			return err
		}
		//init UtxoStore
		bucket, err = mwdb.GetOrCreateTopLevelBucket(tx, utxoBucket)
		if err != nil {
			return err
		}
		bm := &StoreBucketMeta{}
		utxoStore, err := NewUtxoStore(bucket, ksmgr, bm, &config.ChainParams)
		if err != nil {
			return err
		}

		// init SyncStore
		bucketS, err := mwdb.GetOrCreateTopLevelBucket(tx, syncBucket)
		if err != nil {
			return err
		}
		syncStore, err := NewSyncStore(bucket, bm, &config.ChainParams)
		if err != nil {
			return err
		}
		// init TxStore
		bucket, err = mwdb.GetOrCreateTopLevelBucket(tx, txBucket)
		if err != nil {
			return err
		}
		s, err = NewTxStore(chainFetcher, bucket,
			utxoStore, syncStore, ksmgr, bm, &config.ChainParams)
		if err != nil {
			return err
		}
		bucket0, err := mwdb.GetOrCreateBucket(bucket, bucketUnspent)
		if err != nil {
			return err
		}
		s.bucketMeta.nsUnspent = bucket0.GetBucketMeta()
		//bucketUnmined
		bucket0, err = mwdb.GetOrCreateBucket(bucket, bucketUnmined)
		if err != nil {
			return err
		}
		s.bucketMeta.nsUnmined = bucket0.GetBucketMeta()

		// bucketTxRecords
		bucket0, err = mwdb.GetOrCreateBucket(bucket, bucketTxRecords)
		if err != nil {
			return err
		}
		s.bucketMeta.nsTxRecords = bucket0.GetBucketMeta()
		bucket0, err = mwdb.GetOrCreateBucket(bucket, bucketUnminedInputs)
		if err != nil {
			return err
		}
		s.bucketMeta.nsUnminedInputs = bucket0.GetBucketMeta()
		// unmined credits
		bucket0, err = mwdb.GetOrCreateBucket(bucket, bucketUnminedCredits)
		if err != nil {
			return err
		}
		s.bucketMeta.nsUnminedCredits = bucket0.GetBucketMeta()

		// mined balance
		bucket0, err = mwdb.GetOrCreateBucket(bucket, bucketMinedBalance)
		if err != nil {
			return err
		}
		s.bucketMeta.nsMinedBalance = bucket0.GetBucketMeta()
		// credits
		bucket0, err = mwdb.GetOrCreateBucket(bucket, bucketCredits)
		if err != nil {
			return err
		}
		s.bucketMeta.nsCredits = bucket0.GetBucketMeta()
		// debits
		bucket0, err = mwdb.GetOrCreateBucket(bucket, bucketDebits)
		if err != nil {
			return err
		}
		s.bucketMeta.nsDebits = bucket0.GetBucketMeta()
		_, err = NewSyncStore(bucketS, s.bucketMeta, &config.ChainParams)
		if err != nil {
			return err
		}
		// res, err := hex.DecodeString(ks_testnet)
		// if err != nil {
		// 	return err
		// }
		_, err = s.ksmgr.ImportKeystore(tx, func(scriptHash []byte) (bool, error) { return false, nil },
			[]byte(ks_mainnet), privPassphrase_mainnet, addressGapLimit)
		if err != nil {
			return err
		}
		return err
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return s, walletDb, func() {
		walletDb.Close()
		os.Remove(dbPath)
	}, nil
}
