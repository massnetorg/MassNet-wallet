package wallet

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"massnet.org/mass-wallet/btcec"
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/database"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/wallet/hdkeychain"
)

var (
	Params = &config.ChainParams

	DefaultPassword = "123456"

	ErrNilPointer = errors.New("the pointer is nil")
	ErrKeyNum     = errors.New("keyNum can not be <= 0")
	ErrKeyIndex   = errors.New("keyIndex can not be < 0")
	ErrReadFile   = errors.New("can not read file")
)

type Wallet struct {
	wMtx sync.Mutex
	Path string

	//
	RootPkExStrList              []string
	RootPkExStrToChildPkMap      map[string][]*btcec.PublicKey
	ChildPkHashToRootPkExStrMap  map[[32]byte]string
	ChildPkHashToChildPrivKeyMap map[[32]byte]*btcec.PrivateKey

	//p2wsh
	AddressList []string
	// witness
	WitnessMap map[string][]byte

	//walletdb
	WalletDb database.Db
}

func NewWallet(path string) (*Wallet, error) {

	ChildPkHashToRootPkExStrMap0 := make(map[[32]byte]string)
	RootPkExStrToChildPkMap0 := make(map[string][]*btcec.PublicKey)
	RootPkExStrList0 := make([]string, 0)
	AddressList0 := make([]string, 0)
	WitnessMap0 := make(map[string][]byte)
	ChildPkHashToChildPrivKeyMap0 := make(map[[32]byte]*btcec.PrivateKey)

	w := &Wallet{
		Path:                         path,
		RootPkExStrList:              RootPkExStrList0,
		RootPkExStrToChildPkMap:      RootPkExStrToChildPkMap0,
		ChildPkHashToRootPkExStrMap:  ChildPkHashToRootPkExStrMap0,
		ChildPkHashToChildPrivKeyMap: ChildPkHashToChildPrivKeyMap0,
		AddressList:                  AddressList0,
		WitnessMap:                   WitnessMap0,
	}
	err := initPath(w)
	if err != nil {
		logging.CPrint(logging.ERROR, "init path failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	walletDb, err := OpenWalletdb(w.Path)
	if err != nil {
		logging.CPrint(logging.ERROR, "open db failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	w.WalletDb = walletDb

	err = w.UpdateList()
	if err != nil {

		logging.CPrint(logging.ERROR, "update memory failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}
	return w, nil

}

func OpenWalletdb(path string) (database.Db, error) {

	wdb, err := OpenDB(path)
	if err != nil {
		logging.CPrint(logging.ERROR, "open db failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}
	return wdb, nil
}

func checkFileIsExist(dbDirPath string) bool {
	dbPath := filepath.Join(dbDirPath, walletDbName)
	var exist = true
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

func initPath(w *Wallet) error {
	exist := checkFileIsExist(w.Path)
	if !exist {

		err := InitDb(w.Path)
		if err != nil {
			logging.CPrint(logging.ERROR, "init db failed",
				logging.LogFormat{
					"err": err,
				})
			return err
		}

		walletDb, err := OpenWalletdb(w.Path)
		if err != nil {
			logging.CPrint(logging.ERROR, "open db failed",
				logging.LogFormat{
					"err": err,
				})
			return err
		}

		w.WalletDb = walletDb

		_, err = generateRootKeysForSpace(w, DefaultPassword, 1)
		if err != nil {
			logging.CPrint(logging.ERROR, "generate rootKey failed",
				logging.LogFormat{
					"err": err,
				})
			return err
		}

		_, err = generateNewAddress(w, 0)
		if err != nil {
			logging.CPrint(logging.ERROR, "generate address failed",
				logging.LogFormat{
					"err": err,
				})
			return err
		}
		w.Close()

	} else {
		logging.CPrint(logging.INFO, "You have an existing wallet data",
			logging.LogFormat{})
		return nil
	}
	return nil
}

func (w *Wallet) Close() {

	w.WalletDb.Close()
}

func (w *Wallet) GenerateRootKeysForSpace(password string, rootKeyNum int) ([]string, error) {
	w.wMtx.Lock()
	defer w.wMtx.Unlock()

	strList, err := generateRootKeysForSpace(w, password, rootKeyNum)
	if err != nil {
		logging.CPrint(logging.ERROR, "create seed failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	return strList, nil
}

func generateRootKeysForSpace(w *Wallet, password string, Num int) ([]string, error) {
	if Num <= 0 {
		logging.CPrint(logging.ERROR, "rootKeyNum can not be <= 0",
			logging.LogFormat{
				"err": ErrKeyNum,
			})
		return nil, ErrKeyNum
	}
	var (
		rootPkExStrListReturn = make([]string, Num)
	)

	for i := 0; i < Num; i++ {
		seed, err := hdkeychain.GenerateSeed(
			hdkeychain.RecommendedSeedLen)
		if err != nil {
			logging.CPrint(logging.ERROR, "create seed failed",
				logging.LogFormat{
					"err": err,
				})
			return nil, err
		}

		rootPkExStr, err := GenerateRootKey(w.WalletDb, password, seed)
		if err != nil {
			logging.CPrint(logging.ERROR, "create rootKey failed",
				logging.LogFormat{
					"err": err,
				})
			return nil, err
		}
		rootPkExStrListReturn[i] = rootPkExStr

		childkeyFromRootKey := make([]*btcec.PublicKey, 0)
		w.RootPkExStrToChildPkMap[rootPkExStr] = childkeyFromRootKey

		w.RootPkExStrList = append(w.RootPkExStrList, rootPkExStr)
	}

	return rootPkExStrListReturn, nil
}

func GenerateChildKeysForSpace(w *Wallet, rootPkExStr string, childKeyNum int) ([]*btcec.PublicKey, int, error) {

	if childKeyNum <= 0 {
		logging.CPrint(logging.ERROR, "childKeyNum can not be <=0",
			logging.LogFormat{
				"err": ErrKeyNum,
			})
		return nil, 0, ErrKeyNum
	}

	childKeysReturn, childPubKeyNum, err := GenerateChildKeys(w.WalletDb, rootPkExStr, childKeyNum)
	if err != nil {
		logging.CPrint(logging.ERROR, "create childKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, err
	}

	for i := 0; i < len(childKeysReturn); i++ {
		w.RootPkExStrToChildPkMap[rootPkExStr] = append(w.RootPkExStrToChildPkMap[rootPkExStr], childKeysReturn[i])

		w.ChildPkHashToRootPkExStrMap[sha256.Sum256(childKeysReturn[i].SerializeCompressed())] = rootPkExStr
	}

	return childKeysReturn, childPubKeyNum, nil

}

func (w *Wallet) GenerateChildKeysForSpace(rootPkExStr string, childKeyNum int) ([]*btcec.PublicKey, int, error) {
	w.wMtx.Lock()
	defer w.wMtx.Unlock()

	childKeysReturn, childPubKeyNum, err := GenerateChildKeysForSpace(w, rootPkExStr, childKeyNum)
	if err != nil {
		logging.CPrint(logging.ERROR, "create childKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, err
	}

	return childKeysReturn, childPubKeyNum, nil
}

func (w *Wallet) GetChildPubKeyForSpace(rootPkExStr string, childKeyIndex int) (*btcec.PublicKey, int, error) {
	w.wMtx.Lock()
	defer w.wMtx.Unlock()

	if childKeyIndex < 0 {
		logging.CPrint(logging.ERROR, "keyIndex can not be < 0",
			logging.LogFormat{
				"err": ErrKeyIndex,
			})
		return nil, 0, ErrKeyIndex
	}

	childPubKey, childNum, err := GetChildPubKey(w.WalletDb, rootPkExStr, childKeyIndex)
	if err != nil {
		logging.CPrint(logging.ERROR, "create childKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, err
	}
	return childPubKey, childNum, nil

}

func (w *Wallet) GetChildPubKeyListForSpace(rootPkExStr string) ([]*btcec.PublicKey, int, error) {
	w.wMtx.Lock()
	defer w.wMtx.Unlock()

	childPubKeyList, childNum, err := GetChildPubKeyList(w.WalletDb, rootPkExStr)
	if err != nil {
		logging.CPrint(logging.ERROR, "get childKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, err
	}

	return childPubKeyList, childNum, nil
}

func (w *Wallet) UpdateList() error {

	rootPkExStrList, _, err := GetRootPubKeyExStrList(w.WalletDb)
	if err != nil {
		logging.CPrint(logging.ERROR, "get rootKey failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}
	w.RootPkExStrList = rootPkExStrList

	for i := 0; i < len(rootPkExStrList); i++ {

		childPubKeylist, _, err := GetChildPubKeyList(w.WalletDb, rootPkExStrList[i])
		if err != nil {
			logging.CPrint(logging.ERROR, "get childKey failed",
				logging.LogFormat{
					"err": err,
				})
			return err
		}
		w.RootPkExStrToChildPkMap[rootPkExStrList[i]] = childPubKeylist

		for j := 0; j < len(childPubKeylist); j++ {
			w.ChildPkHashToRootPkExStrMap[sha256.Sum256(childPubKeylist[j].SerializeCompressed())] = rootPkExStrList[i]
		}

	}

	err = updateWitnessMap(w)
	if err != nil {
		logging.CPrint(logging.ERROR, "get witnessMap failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	return nil
}

func (w *Wallet) GetRootPrivKeyForSpace(rootPkExStr string, password string) (string, error) {
	w.wMtx.Lock()
	defer w.wMtx.Unlock()

	rootPrivKeyStr, err := GetRootPrivKey(w.WalletDb, password, rootPkExStr)
	if err != nil {
		logging.CPrint(logging.ERROR, "get rootPrivKey",
			logging.LogFormat{
				"err": err,
			})
		return "", err
	}

	return rootPrivKeyStr, nil
}

func (w *Wallet) GetRootPubKeyStringList() ([]string, int, error) {
	w.wMtx.Lock()
	defer w.wMtx.Unlock()

	rootPkExStrList, num, err := GetRootPubKeyExStrList(w.WalletDb)
	if err != nil {
		logging.CPrint(logging.ERROR, "get rootKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, err
	}

	return rootPkExStrList, num, nil

}

//Sign []byte
func (w *Wallet) SignBytes(data []byte, pubkey *btcec.PublicKey, password string) (*btcec.Signature, error) {
	w.wMtx.Lock()
	defer w.wMtx.Unlock()

	//verify nil pointer,avoid panic error
	if pubkey == nil {
		logging.CPrint(logging.ERROR, "input pointer is nil",
			logging.LogFormat{
				"err": ErrNilPointer,
			})
		return nil, ErrNilPointer
	}
	if data == nil {
		err := errors.New("input []byte is nil")
		logging.CPrint(logging.ERROR, "input []byte is nil",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	//get datahash 32bytes
	h1 := sha256.New()
	h1.Write([]byte(data))
	dataHash := h1.Sum(nil)

	return signHash(w, dataHash, pubkey, password)
}

//SignHash
func (w *Wallet) SignHash(dataHash []byte, pubkey *btcec.PublicKey, password string) (*btcec.Signature, error) {
	w.wMtx.Lock()
	defer w.wMtx.Unlock()

	//verify nil pointer,avoid panic error
	if pubkey == nil {
		logging.CPrint(logging.ERROR, "input pointer is nil",
			logging.LogFormat{
				"err": ErrNilPointer,
			})
		return nil, ErrNilPointer
	}
	if dataHash == nil {
		err := errors.New("input []byte is nil")
		logging.CPrint(logging.ERROR, "input []byte is nil",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	return signHash(w, dataHash, pubkey, password)

}
func signHash(w *Wallet, hash []byte, pubkey *btcec.PublicKey, password string) (*btcec.Signature, error) {

	if len(hash) != 32 {
		err := errors.New("invalid hash []byte, size is not 32")
		logging.CPrint(logging.ERROR, "hash size is not 32",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	childPrivkeyStruct, err := w.PrivKeyFromPubKey(pubkey, password)
	if err != nil {
		logging.CPrint(logging.ERROR, "pubKey->privKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	signatureStruct, err := childPrivkeyStruct.Sign(hash)
	if err != nil {
		logging.CPrint(logging.ERROR, "sign failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	return signatureStruct, nil

}

func (w *Wallet) PrivKeyFromPubKey(pubkey *btcec.PublicKey, password string) (*btcec.PrivateKey, error) {

	//verify nil pointer,avoid panic error
	if pubkey == nil {
		logging.CPrint(logging.ERROR, "input pointer is nil",
			logging.LogFormat{
				"err": ErrNilPointer,
			})
		return nil, ErrNilPointer
	}
	rootPkExStr, ok := w.ChildPkHashToRootPkExStrMap[sha256.Sum256(pubkey.SerializeCompressed())]
	if !ok {
		err := errors.New("pubkey is nonexistent")
		logging.CPrint(logging.ERROR, "pubkey is nonexistent",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	var childKeyIndex int
	for i := 0; i < len(w.RootPkExStrToChildPkMap[rootPkExStr]); i++ {
		buff1 := w.RootPkExStrToChildPkMap[rootPkExStr][i].SerializeCompressed()
		buff2 := pubkey.SerializeCompressed()
		res := bytes.Compare(buff1, buff2)
		if res == 0 {
			childKeyIndex = i
		}
	}

	rootPrivKeyExBytes, err := GetRootPrivKey(w.WalletDb, password, rootPkExStr)
	if err != nil {
		logging.CPrint(logging.ERROR, "get rootPrivKey",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	rootPrivkeyEx, err := hdkeychain.NewKeyFromString(string(rootPrivKeyExBytes))
	if err != nil {
		logging.CPrint(logging.ERROR, "rootPrivKeyBytes->rootPrivkey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}
	childPrivkey, err := rootPrivkeyEx.Child(uint32(childKeyIndex))
	if err != nil {
		logging.CPrint(logging.ERROR, "create childKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}
	tmpChildPrivkeyStruct, err := childPrivkey.ECPrivKey()
	if err != nil {
		logging.CPrint(logging.ERROR, "*ExtendedKey->*btcec.PrivateKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}
	childPrivkeyStruct := tmpChildPrivkeyStruct
	return childPrivkeyStruct, nil

}

//Verify Signature
func VerifyBytes(data []byte, sig *btcec.Signature, pubkey *btcec.PublicKey) (bool, error) {
	if data == nil {
		err := errors.New("input []byte is nil")
		logging.CPrint(logging.ERROR, "input []byte is nil",
			logging.LogFormat{
				"err": err,
			})
		return false, err
	}
	//verify nil pointer,avoid panic error
	if pubkey == nil || sig == nil {
		logging.CPrint(logging.ERROR, "input pointer is nil",
			logging.LogFormat{
				"err": ErrNilPointer,
			})
		return false, ErrNilPointer
	}

	//get datahash 32bytes
	h1 := sha256.New()
	h1.Write([]byte(data))
	dataHash := h1.Sum(nil)

	return verifyHash(sig, dataHash, pubkey)
}

func VerifyHash(dataHash []byte, sig *btcec.Signature, pubkey *btcec.PublicKey) (bool, error) {
	if dataHash == nil {
		err := errors.New("input []byte is nil")
		logging.CPrint(logging.ERROR, "input []byte is nil",
			logging.LogFormat{
				"err": err,
			})
		return false, err
	}
	//verify nil pointer,avoid panic error
	if pubkey == nil || sig == nil {
		logging.CPrint(logging.ERROR, "input pointer is nil",
			logging.LogFormat{
				"err": ErrNilPointer,
			})
		return false, ErrNilPointer
	}

	return verifyHash(sig, dataHash, pubkey)
}
func verifyHash(sig *btcec.Signature, hash []byte, pubkey *btcec.PublicKey) (bool, error) {
	if len(hash) != 32 {
		err := errors.New("invalid hash []byte, size is not 32")
		logging.CPrint(logging.ERROR, "hash size is not 32",
			logging.LogFormat{
				"err": err,
			})
		return false, err
	}

	boolReturn := sig.Verify(hash, pubkey)
	return boolReturn, nil
}

func generateAddressForEachPubKey(w *Wallet) error {

	for _, rPkStr := range w.RootPkExStrList {
		for j := 0; j < len(w.RootPkExStrToChildPkMap[rPkStr]); j++ {
			pubkeys := make([]*btcec.PublicKey, 0)
			pubkeys = append(pubkeys, w.RootPkExStrToChildPkMap[rPkStr][j])

			_, err := NewWitnessScriptAddress(w, pubkeys, 1, 0)
			if err != nil {
				logging.CPrint(logging.ERROR, "create witnessScriptAddress failed",
					logging.LogFormat{
						"err": err,
					})
				return err
			}

		}
	}
	return nil

}

func generateNewAddress(w *Wallet, version int) (massutil.Address, error) {
	if len(w.RootPkExStrList) == 0 {
		err := errors.New("there is no rootKey")
		logging.CPrint(logging.ERROR, "there is no rootKey",
			logging.LogFormat{
				"err": err,
			})
		return nil, err

	}

	pubKey, _, err := GenerateChildKeysForSpace(w, w.RootPkExStrList[0], 1)
	if err != nil {
		logging.CPrint(logging.ERROR, "generate childKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	addressStr, err := NewWitnessScriptAddress(w, pubKey, 1, version)
	if err != nil {
		logging.CPrint(logging.ERROR, "create witnessScriptAddress failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	address, err := massutil.DecodeAddress(addressStr, Params)
	if err != nil {
		logging.CPrint(logging.ERROR, "decode address failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	return address, nil
}

func (w *Wallet) GenerateNewAddress(version int) (massutil.Address, error) {
	w.wMtx.Lock()
	defer w.wMtx.Unlock()

	addr, err := generateNewAddress(w, version)
	if err != nil {
		logging.CPrint(logging.ERROR, "generate a new address failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	return addr, nil
}

func NewWitnessScriptAddress(w *Wallet, pubkeys []*btcec.PublicKey, nrequired int, version int) (string, error) {
	//verify nil pointer,avoid panic error
	if pubkeys == nil {
		logging.CPrint(logging.ERROR, "input pointer is nil",
			logging.LogFormat{
				"err": ErrNilPointer,
			})
		return "", ErrNilPointer
	}
	if nrequired <= 0 {
		err := errors.New("nrequired can not be <= 0")
		logging.CPrint(logging.ERROR, "nrequired can not be <= 0",
			logging.LogFormat{
				"err": err,
			})
		return "", err
	}
	redeemScript, witAddress, err := newWitnessScriptAddress(pubkeys, nrequired, version, Params)
	if err != nil {
		logging.CPrint(logging.ERROR, "create witnessScriptAddress failed",
			logging.LogFormat{
				"err": err,
			})
		return "", err
	}

	err = w.WalletDb.InsertWitnessAddr(witAddress, redeemScript)
	if err != nil {
		logging.CPrint(logging.ERROR, "insert db failed",
			logging.LogFormat{
				"err": err,
			})
		return "", err
	}

	w.WitnessMap[witAddress] = redeemScript
	for _, addrStr := range w.AddressList {
		num := bytes.Compare([]byte(addrStr), []byte(witAddress))
		if num == 0 {
			return witAddress, nil
		}
	}
	w.AddressList = append(w.AddressList, witAddress)

	return witAddress, nil

}

func (w *Wallet) NewWitnessScriptAddress(pubkeys []*btcec.PublicKey, nrequired, version int) (string, error) {
	w.wMtx.Lock()
	defer w.wMtx.Unlock()

	witAddr, err := NewWitnessScriptAddress(w, pubkeys, nrequired, version)
	if err != nil {
		logging.CPrint(logging.ERROR, "create witnessScriptAddress failed",
			logging.LogFormat{
				"err": err,
			})
		return "", err
	}

	return witAddr, nil
}

func updateWitnessMap(w *Wallet) error {
	AddressList0 := make([]string, 0)
	witnessMap, err := getWitnessMap(w.WalletDb)
	if err != nil {
		logging.CPrint(logging.ERROR, "get witnessMap failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}
	w.WitnessMap = witnessMap
	for k := range witnessMap {
		AddressList0 = append(AddressList0, k)
	}
	w.AddressList = AddressList0
	return nil
}

func (w *Wallet) ChangePassword(rootPkExStr string, oldPassword string, newPassword string) error {
	w.wMtx.Lock()
	defer w.wMtx.Unlock()
	masterPrivKey, err := DefaultNewSecretKey([]byte(oldPassword), fastScrypt)
	if err != nil {
		logging.CPrint(logging.ERROR, "create a secretKey failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	oldRootPrivKeyEnc, _, err := w.WalletDb.FetchRootKey(rootPkExStr)
	if err != nil {
		logging.CPrint(logging.ERROR, "fetch db failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	rootPrivkeyBytes, err := masterPrivKey.Decrypt(oldRootPrivKeyEnc)
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

	masterPrivKey, err = DefaultNewSecretKey([]byte(newPassword), fastScrypt)
	if err != nil {
		logging.CPrint(logging.ERROR, "create masterKey failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}
	newRootPrivKeyEnc, err := masterPrivKey.Encrypt([]byte(rootPrivkeyEx.String()))
	if err != nil {
		logging.CPrint(logging.ERROR, "encrypt failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	num := bytes.Compare([]byte(DefaultPassword), []byte(newPassword))
	if num == 0 {
		err = w.WalletDb.InsertRootKey(newRootPrivKeyEnc, rootPkExStr, true)
		if err != nil {

			logging.CPrint(logging.ERROR, "insert db failed",
				logging.LogFormat{
					"err": err,
				})
			return err
		}
	} else {
		err = w.WalletDb.InsertRootKey(newRootPrivKeyEnc, rootPkExStr, false)
		if err != nil {
			logging.CPrint(logging.ERROR, "insert db failed",
				logging.LogFormat{
					"err": err,
				})
			return err
		}
	}

	return nil
}

func (w *Wallet) DefaultPasswordUsed(rootPkExStr string) (bool, error) {
	w.wMtx.Lock()
	defer w.wMtx.Unlock()

	_, defaultPasswordUsed, err := w.WalletDb.FetchRootKey(rootPkExStr)
	if err != nil {
		logging.CPrint(logging.ERROR, "fetch db failed",
			logging.LogFormat{
				"err": err,
			})
		return false, err
	}
	return defaultPasswordUsed, nil
}

func (w *Wallet) DumpWallet(dumpDirPath string) error {
	w.wMtx.Lock()
	defer w.wMtx.Unlock()

	err := DumpWallet(dumpDirPath, w.WalletDb)
	if err != nil {
		logging.CPrint(logging.ERROR, "dump wallet failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	return nil
}

func (w *Wallet) ImportWallet(importDirPath string, oldPassword string, newPassword string) error {
	w.wMtx.Lock()
	defer w.wMtx.Unlock()
	err := ImportWallet(w.WalletDb, importDirPath, oldPassword, newPassword)
	if err != nil {
		logging.CPrint(logging.ERROR, "import wallet failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	err = w.UpdateList()
	if err != nil {
		logging.CPrint(logging.ERROR, "update memory failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	err = generateAddressForEachPubKey(w)
	if err != nil {
		logging.CPrint(logging.ERROR, "generate address for each pubKey failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	return nil
}

func (w *Wallet) Unlock(password string) error {
	w.wMtx.Lock()
	defer w.wMtx.Unlock()

	rootPkStrList, _, err := GetRootPubKeyExStrList(w.WalletDb)
	if err != nil {
		logging.CPrint(logging.ERROR, "get rootPubKeyStringList failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	for _, rPkStr := range rootPkStrList {
		err := unlock(rPkStr, password, w)
		if err != nil {
			logging.CPrint(logging.ERROR, "unlock failed",
				logging.LogFormat{
					"err": err,
				})
			return err
		}
	}

	return nil
}

func unlock(rootPkStr string, password string, w *Wallet) error {
	cPks, _, err := GetChildPubKeyList(w.WalletDb, rootPkStr)
	if err != nil {
		logging.CPrint(logging.ERROR, "get childPubKeyList failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	for _, cPk := range cPks {
		cPrivk, err := w.PrivKeyFromPubKey(cPk, password)
		if err != nil {
			logging.CPrint(logging.ERROR, "pubKey -> privKey failed",
				logging.LogFormat{
					"err": err,
				})
			return err
		}
		w.ChildPkHashToChildPrivKeyMap[sha256.Sum256(cPk.SerializeCompressed())] = cPrivk

	}

	return nil
}

func (w *Wallet) Lock() {
	w.wMtx.Lock()
	defer w.wMtx.Unlock()

	w.ChildPkHashToChildPrivKeyMap = map[[32]byte]*btcec.PrivateKey{}
}
