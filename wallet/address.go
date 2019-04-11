package wallet

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"

	"github.com/massnetorg/MassNet-wallet/btcec"
	"github.com/massnetorg/MassNet-wallet/config"
	"github.com/massnetorg/MassNet-wallet/database"
	"github.com/massnetorg/MassNet-wallet/database/ldb"
	"github.com/massnetorg/MassNet-wallet/logging"
	"github.com/massnetorg/MassNet-wallet/wallet/hdkeychain"
	"github.com/massnetorg/MassNet-wallet/wallet/snacl"
)

const (
	walletDbName = "walletdb"
	dbtype       = "leveldb"
)

type ScryptOptions struct {
	N, R, P int
	Salt    []byte
}

var (
	self = database.DriverDB{DbType: "leveldb", CreateDB: ldb.CreateDB, OpenDB: ldb.OpenDB}

	fastScrypt = &ScryptOptions{
		N:    16,
		R:    8,
		P:    1,
		Salt: []byte("0"),
	}
)

func DefaultNewSecretKey(passphrase []byte, config *ScryptOptions) (*snacl.SecretKey, error) {
	//return snacl.NewSecretKey(&passphrase, config.N, config.R, config.P)
	return snacl.NewSecretKeyFromUnmarshal(passphrase, config.Salt, config.N, config.R, config.P)
}

func InitDb(dbDirPath string) error {
	dbPath := filepath.Join(dbDirPath, walletDbName)
	// Create the wallet database
	err := os.MkdirAll(dbDirPath, 0700)
	if err != nil {
		logging.CPrint(logging.ERROR, "create dir failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	database.AddDBDriver(self)
	db, err := database.CreateDB(dbtype, dbPath)
	if err != nil {
		logging.CPrint(logging.ERROR, "create db failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}
	defer db.Close()

	return nil
}

func OpenDB(dbDirPath string) (database.Db, error) {
	dbPath := filepath.Join(dbDirPath, walletDbName)
	db, err := database.OpenDB(dbtype, dbPath)
	if err != nil {
		logging.CPrint(logging.ERROR, "open db failed",
			logging.LogFormat{
				"err": err,
			})

		return nil, err
	}
	return db, nil
}

func GenerateRootKey(db database.Db, privPassphrase string, seed []byte) (string, error) {
	rootPrivkeyEx, err := hdkeychain.NewMaster(seed, &config.ChainParams)
	if err != nil {
		logging.CPrint(logging.ERROR, "create rootKey failed",
			logging.LogFormat{
				"err": err,
			})
		return "", err
	}

	masterPrivKey, err := DefaultNewSecretKey([]byte(privPassphrase), fastScrypt)
	if err != nil {
		logging.CPrint(logging.ERROR, "create a secretKey failed",
			logging.LogFormat{
				"err": err,
			})
		return "", err
	}
	rootPrivkeyExEnc, err := masterPrivKey.Encrypt([]byte(rootPrivkeyEx.String()))
	if err != nil {
		logging.CPrint(logging.ERROR, "encrypt failed",
			logging.LogFormat{
				"err": err,
			})
		return "", err
	}

	//rootPubkeyEx *ExtendedKey
	rootPubkeyEx, err := rootPrivkeyEx.Neuter()
	if err != nil {
		logging.CPrint(logging.ERROR, "rootPrivkeyExy->rootPubkeyEx failed",
			logging.LogFormat{
				"err": err,
			})
		return "", err
	}

	rootPkExStr := rootPubkeyEx.String()

	num := bytes.Compare([]byte(DefaultPassword), []byte(privPassphrase))
	if num == 0 {
		err = db.InsertRootKey(rootPrivkeyExEnc, rootPkExStr, true)
		if err != nil {

			logging.CPrint(logging.ERROR, "insert db failed",
				logging.LogFormat{
					"err": err,
				})
			return "", err
		}
	} else {
		err = db.InsertRootKey(rootPrivkeyExEnc, rootPkExStr, false)
		if err != nil {
			logging.CPrint(logging.ERROR, "insert db failed",
				logging.LogFormat{
					"err": err,
				})
			return "", err
		}
	}

	err = db.InitChildKeyNum(rootPkExStr)
	if err != nil {
		logging.CPrint(logging.ERROR, "insert db failed",
			logging.LogFormat{
				"err": err,
			})
		return "", err
	}

	return rootPkExStr, nil
}

func GetRootPrivKey(db database.Db, privPassphrase string, rootPkExStr string) (string, error) {
	rootKeyEnc, _, err := db.FetchRootKey(rootPkExStr)
	if err != nil {
		logging.CPrint(logging.ERROR, "fetch db failed",
			logging.LogFormat{
				"err": err,
			})
		return "", err
	}
	masterPrivKey, err := DefaultNewSecretKey([]byte(privPassphrase), fastScrypt)
	if err != nil {
		logging.CPrint(logging.ERROR, "create a secretKey failed",
			logging.LogFormat{
				"err": err,
			})
		return "", err
	}

	rootPrivkeyStrBytes, err := masterPrivKey.Decrypt(rootKeyEnc)
	if err != nil {
		logging.CPrint(logging.ERROR, "decrypt failed",
			logging.LogFormat{
				"err": err,
			})
		return "", err
	}

	return string(rootPrivkeyStrBytes), nil
}

func GetRootPubKeyExStrList(db database.Db) ([]string, int, error) {
	rootPkStrList, err := db.FetchAllRootPkStr()
	if err != nil {
		logging.CPrint(logging.ERROR, "fetch db failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, err
	}
	return rootPkStrList, len(rootPkStrList), nil
}

func GenerateChildKey(db database.Db, rootPkExStr string) (*btcec.PublicKey, int, error) {
	var (
		childPubkey *btcec.PublicKey
	)

	num, err := db.FetchChildKeyNum(rootPkExStr)
	if err != nil {
		logging.CPrint(logging.ERROR, "fetch db failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, err
	}

	rootPkEx, err := hdkeychain.NewKeyFromString(rootPkExStr)
	if err != nil {
		logging.CPrint(logging.ERROR, "rootPkExStr->rootPkEx failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, err
	}

	childPkEx, err := rootPkEx.Child(uint32(num))
	if err != nil {
		logging.CPrint(logging.ERROR, "create childKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, err
	}
	tmpChildPubkey, err := childPkEx.ECPubKey()
	if err != nil {
		logging.CPrint(logging.ERROR, "ExtendedPubKey->BtcecPubKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, err
	}
	childPubkey = tmpChildPubkey

	childKeyNum, err := db.UpdateChildKeyNum(rootPkExStr)
	if err != nil {
		logging.CPrint(logging.ERROR, "insert db failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, err
	}

	return childPubkey, childKeyNum, nil
}

func GenerateChildKeys(db database.Db, rootPkExStr string, childKeyNum int) ([]*btcec.PublicKey, int, error) {
	var (
		childPubKeyListForReturn = make([]*btcec.PublicKey, childKeyNum)
		childPubKeyNumReturn     int
	)

	for i := 0; i < childKeyNum; i++ {
		childPubKey, childPubKeyNum, err := GenerateChildKey(db, rootPkExStr)
		if err != nil {
			logging.CPrint(logging.ERROR, "create childKey failed",
				logging.LogFormat{
					"err": err,
				})
			return nil, 0, err
		}
		childPubKeyListForReturn[i] = childPubKey
		childPubKeyNumReturn = childPubKeyNum

	}

	return childPubKeyListForReturn, childPubKeyNumReturn, nil
}

func GetChildPubKey(db database.Db, rootPkExStr string, childindex int) (*btcec.PublicKey, int, error) {
	childKeyNum, err := db.FetchChildKeyNum(rootPkExStr)
	if err != nil {
		logging.CPrint(logging.ERROR, "fetch db failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, err
	}

	if childKeyNum == 0 {
		logging.VPrint(logging.INFO, "childKeyNum of rootKey is 0",
			logging.LogFormat{
				"childKeyNum": 0,
			})
		return nil, 0, nil
	}

	childPubkey, err := getChildPubKey(rootPkExStr, childindex, childKeyNum)
	if err != nil {
		logging.CPrint(logging.ERROR, "get childKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, err
	}

	return childPubkey, childKeyNum, nil
}

func getChildPubKey(rootPkExStr string, childindex int, childKeyNum int) (*btcec.PublicKey, error) {
	var (
		childPubkey *btcec.PublicKey
	)

	rootPkEx, err := hdkeychain.NewKeyFromString(rootPkExStr)
	if err != nil {
		logging.CPrint(logging.ERROR, "rootPkExStr->rootPkEx failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	if childindex < 0 || childindex > childKeyNum-1 {
		err := errors.New("out of index")
		logging.CPrint(logging.ERROR, "out of index",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	childPkEx, err := rootPkEx.Child(uint32(childindex))
	if err != nil {
		logging.CPrint(logging.ERROR, "create childKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}
	tmpChildPubkey, err := childPkEx.ECPubKey()
	if err != nil {
		logging.CPrint(logging.ERROR, "ExtendedPubKey->BtcecPubKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}
	childPubkey = tmpChildPubkey

	return childPubkey, nil
}

func GetChildPubKeyList(db database.Db, rootPkExStr string) ([]*btcec.PublicKey, int, error) {
	childKeyNum, err := db.FetchChildKeyNum(rootPkExStr)
	if err != nil {
		logging.CPrint(logging.ERROR, "fetch db failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, err
	}

	if childKeyNum == 0 {
		logging.VPrint(logging.INFO, "childKeyNum of rootKey is 0",
			logging.LogFormat{
				"childKeyNum": 0,
			})
		return nil, 0, nil
	}

	var childPubKeyList = make([]*btcec.PublicKey, childKeyNum)

	for i := 0; i < childKeyNum; i++ {
		childPubKey, err := getChildPubKey(rootPkExStr, i, childKeyNum)
		if err != nil {
			logging.CPrint(logging.ERROR, "get childKey failed",
				logging.LogFormat{
					"err": err,
				})
			return nil, 0, nil
		}
		childPubKeyList[i] = childPubKey
	}
	return childPubKeyList, childKeyNum, nil

}
