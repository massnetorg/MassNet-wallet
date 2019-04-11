package wallet

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"

	"github.com/massnetorg/MassNet-wallet/database"
	"github.com/massnetorg/MassNet-wallet/logging"
	"github.com/massnetorg/MassNet-wallet/wallet/hdkeychain"
)

// KeyImage is the struct for hold export key data
type DumpImage struct {
	Keys []exportedKeyJSON `json:"keys"`
}

type exportedKeyJSON struct {
	Crypto          cryptoJSON `json:"crypto"`
	ID              string     `json:"id"`
	RootPubKeyExStr string     `json:"rpub"`
	ChildKeyNum     string     `json:"childKeyNum"`
}

type cryptoJSON struct {
	Cipher     string `json:"cipher"`
	CipherText string `json:"ciphertext"`
	KDF        string `json:"kdf"`
	KDFParams  string `json:"kdfparams"`
}

func ImportWallet(db database.Db, dumpPath string, oldPassword string, newPassword string) error {
	//
	dumpFilePath := filepath.Join(dumpPath, "backups_keys")
	image, err := getDumpImageFromFile(dumpFilePath)
	if err != nil {
		logging.CPrint(logging.ERROR, "get jsonData failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	rootPkExStrList, _, err := GetRootPubKeyExStrList(db)
	if err != nil {
		logging.CPrint(logging.ERROR, "get rootPkH failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}
	for _, key := range image.Keys {
		importRootPkH := key.RootPubKeyExStr
		exist := false

		for _, rootPkH := range rootPkExStrList {
			resNum := bytes.Compare([]byte(importRootPkH), []byte(rootPkH))
			if resNum == 0 {
				exist = true
			}
		}

		if exist {
			logging.CPrint(logging.INFO, "rootKey exists",
				logging.LogFormat{})
			continue
		}

		importRootKey(db, &key, oldPassword, newPassword)
		if err != nil {
			logging.CPrint(logging.ERROR, "import rootKey failed",
				logging.LogFormat{
					"err": err,
				})
			return err
		}

	}
	return nil
}

func importRootKey(db database.Db, key *exportedKeyJSON, oldPassword string, newPassword string) error {
	masterPrivKey, err := DefaultNewSecretKey([]byte(oldPassword), fastScrypt)
	if err != nil {
		logging.CPrint(logging.ERROR, "create a secretKey failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	rootPrivkeyEncFromDump, err := hex.DecodeString(key.Crypto.CipherText)
	if err != nil {
		logging.CPrint(logging.ERROR, "fetch db failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	rootPrivkeyBytes, err := masterPrivKey.Decrypt(rootPrivkeyEncFromDump)
	if err != nil {
		logging.CPrint(logging.ERROR, "wrong password",
			logging.LogFormat{
				"err": err,
			})
		return err
	}
	rootPrivkeyEx, err := hdkeychain.NewKeyFromString(string(rootPrivkeyBytes))
	if err != nil {
		logging.CPrint(logging.ERROR, "rootPrivkeyBytes->rootPrivkeyEx failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}
	rootPkEx, err := rootPrivkeyEx.Neuter()
	if err != nil {
		logging.CPrint(logging.ERROR, "rootPrivkeyEx->rootPubKeyEx failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	masterPrivKey, err = DefaultNewSecretKey([]byte(newPassword), fastScrypt)
	if err != nil {
		logging.CPrint(logging.ERROR, "create masterKey failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}
	rootPrivkeyEnc, err := masterPrivKey.Encrypt([]byte(rootPrivkeyEx.String()))
	if err != nil {
		logging.CPrint(logging.ERROR, "encrypt failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	err = db.InsertRootKey(rootPrivkeyEnc, rootPkEx.String(), false)
	if err != nil {
		logging.CPrint(logging.ERROR, "insert db failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	err = db.ImportChildKeyNum(key.RootPubKeyExStr, key.ChildKeyNum)
	if err != nil {
		logging.CPrint(logging.ERROR, "insert db failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	return nil
}

func getDumpImageFromFile(filePath string) (*DumpImage, error) {

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		logging.CPrint(logging.ERROR, "read file failed",
			logging.LogFormat{
				"err": ErrReadFile,
			})
		return nil, ErrReadFile
	}

	walletImage := &DumpImage{}
	if err := json.Unmarshal(data, walletImage); err != nil {
		logging.CPrint(logging.ERROR, "json unmarshal failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	return walletImage, nil
}

func DumpWallet(dumpDirPath string, db database.Db) error {
	rootPkExStrList, _, err := GetRootPubKeyExStrList(db)
	if err != nil {
		logging.CPrint(logging.ERROR, "get rootPkH failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	keys := make([]exportedKeyJSON, 0)

	for k, rpk := range rootPkExStrList {
		childKeyNum, err := db.FetchChildKeyNum(rpk)
		if err != nil {
			logging.CPrint(logging.ERROR, "fetch db failed",
				logging.LogFormat{
					"err": err,
				})
			return err
		}
		rootKeyEnc, _, err := db.FetchRootKey(rpk)
		if err != nil {
			logging.CPrint(logging.ERROR, "fetch db failed",
				logging.LogFormat{
					"err": err,
				})
			return err
		}
		//var exKey *exportedKeyJSON
		exKey, err := dumpRootKey(k, rpk, rootKeyEnc, childKeyNum)
		if err != nil {
			logging.CPrint(logging.ERROR, "dump rootKey failed",
				logging.LogFormat{
					"err": err,
				})
			return err
		}

		keys = append(keys, *exKey)
	}

	dumpImageStruct := DumpImage{
		Keys: keys,
	}

	keysJson, err := json.Marshal(dumpImageStruct)
	if err != nil {
		logging.CPrint(logging.ERROR, "json marshal failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	fPath := filepath.Join(dumpDirPath, "backups_keys")
	err = writeKeyFile(fPath, keysJson)
	if err != nil {
		logging.CPrint(logging.ERROR, "write file failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}
	return nil

}

func dumpRootKey(id int, rootPkExStr string, rootKeyEnc []byte, cKeyNum int) (*exportedKeyJSON, error) {
	cryptoStruct := cryptoJSON{
		Cipher:     "Stream cipher",
		CipherText: hex.EncodeToString(rootKeyEnc),
		KDF:        "scrypt",
	}
	exportedKeyStruct := exportedKeyJSON{
		Crypto:          cryptoStruct,
		RootPubKeyExStr: rootPkExStr,
		ChildKeyNum:     strconv.Itoa(cKeyNum),
		ID:              strconv.Itoa(id),
	}
	return &exportedKeyStruct, nil

}

func writeKeyFile(file string, content []byte) error {
	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return err
	}
	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
	f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Close()
	return os.Rename(f.Name(), file)
}
