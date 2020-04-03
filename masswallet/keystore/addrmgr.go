package keystore

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/masswallet/db"
	"massnet.org/mass-wallet/masswallet/keystore/hdkeychain"
	"massnet.org/mass-wallet/masswallet/keystore/snacl"
	"massnet.org/mass-wallet/masswallet/keystore/zero"
)

type AddrManager struct {
	mu sync.Mutex

	keystoreName string
	remark       string

	// in number of second
	expires time.Duration
	index   map[uint32]string
	addrs   map[string]*ManagedAddress
	use     AddrUse

	acctInfo   *accountInfo
	branchInfo *branchInfo

	hdScope KeyScope

	storage db.BucketMeta

	unlocked bool

	// masterKeyPub is the secret key used to secure the cryptoKeyPub key
	// and masterKeyPriv is the secret key used to secure the cryptoKeyPriv
	// key.  This approach is used because it makes changing the passwords
	// much simpler as it then becomes just changing these keys.  It also
	// provides future flexibility.
	//
	// NOTE: This is not the same thing as BIP0032 master node extended
	// key.
	//
	// The underlying master private key will be zeroed when the address
	// manager is locked.
	masterKeyPub  *snacl.SecretKey
	masterKeyPriv *snacl.SecretKey

	// cryptoKeyPub is the key used to encrypt public extended keys and
	// addresses.
	cryptoKeyPub EncryptorDecryptor

	// cryptoKeyPriv is the key used to encrypt private data such as the
	// master hierarchical deterministic extended key.
	//
	// This key will be zeroed when the address manager is locked.
	cryptoKeyPrivEncrypted []byte
	cryptoKeyPriv          EncryptorDecryptor

	// cryptoKeyEntropyEncrypted is the encrypted entropy data.
	cryptoKeyEntropyEncrypted []byte

	// privPassphraseSalt and hashedPrivPassphrase allow for the secure
	// detection of a correct passphrase on manager unlock when the
	// manager is already unlocked.  The hash is zeroed each lock.
	privPassphraseSalt   [saltSize]byte
	hashedPrivPassphrase [sha512.Size]byte
}

type Keystore struct {
	Remarks string     `json:"remarks"`
	Crypto  cryptoJSON `json:"crypto"`
	HDpath  hdPath     `json:"hdPath"`
}

type hdPath struct {
	Purpose          uint32
	Coin             uint32 // 1-testnet,  297-mainnet
	Account          uint32
	ExternalChildNum uint32
	InternalChildNum uint32
}

type cryptoJSON struct {
	Cipher              string `json:"cipher,omitempty"`
	EntropyEnc          string `json:"entropyEnc"`
	KDF                 string `json:"kdf,omitempty"`
	PubParams           string `json:"pubParams,omitempty"`
	PrivParams          string `json:"privParams"`
	CryptoKeyPubEnc     string `json:"cryptoKeyPubEnc,omitempty"`
	CryptoKeyPrivEnc    string `json:"cryptoKeyPrivEnc,omitempty"`
	CryptoKeyEntropyEnc string `json:"cryptoKeyEntropyEnc"`
}

type accountInfo struct {
	acctType         uint32
	acctKeyEncrypted []byte
	acctKeyPriv      *hdkeychain.ExtendedKey
	acctKeyPub       *hdkeychain.ExtendedKey
}
type branchInfo struct {
	internalBranchPub  *hdkeychain.ExtendedKey
	internalBranchPriv *hdkeychain.ExtendedKey
	externalBranchPub  *hdkeychain.ExtendedKey
	externalBranchPriv *hdkeychain.ExtendedKey
	nextExternalIndex  uint32
	nextInternalIndex  uint32
}
type unlockDeriveInfo struct {
	managedAddr *ManagedAddress
	branch      uint32
	index       uint32
}

func (k *Keystore) Bytes() []byte {
	keysJson, err := json.Marshal(k)
	if err != nil {
		logging.CPrint(logging.ERROR, "json marshal failed",
			logging.LogFormat{
				"err": err,
			})
		return nil
	}

	return keysJson
}

func getKeystoreFromJson(keysJson []byte) (*Keystore, error) {
	keystore := &Keystore{}
	if err := json.Unmarshal(keysJson, keystore); err != nil {
		logging.CPrint(logging.ERROR, "json unmarshal failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, ErrInvalidKeystoreJson
	}

	return keystore, nil
}

// NOTE: this func will leave the masterKeyPriv derived
func (a *AddrManager) checkPassword(passphrase []byte) error {
	if a.unlocked {
		saltedPassphrase := append(a.privPassphraseSalt[:],
			passphrase...)
		hashedPassphrase := sha512.Sum512(saltedPassphrase)
		zero.Bytes(saltedPassphrase)
		if !bytes.Equal(hashedPassphrase[:], a.hashedPrivPassphrase[:]) {
			return ErrInvalidPassphrase
		}
		return nil
	} else {
		if err := a.masterKeyPriv.DeriveKey(&passphrase); err != nil {
			if err == snacl.ErrInvalidPassword {
				return ErrInvalidPassphrase
			}
			logging.CPrint(logging.ERROR, "DeriveKey failed",
				logging.LogFormat{
					"err": err,
				})
			return ErrDeriveMasterPrivKey
		}
		return nil
	}
}

func (a *AddrManager) safelyCheckPassword(privPass []byte) error {
	err := a.checkPassword(privPass)
	if err != nil {
		return err
	}
	a.masterKeyPriv.Zero()
	return nil
}

func unmarshalMasterPrivKey(masterPrivKey *snacl.SecretKey, privPass []byte, masterPrivParams []byte) error {
	err := masterPrivKey.Unmarshal(masterPrivParams)
	if err != nil {
		return err
	}
	err = masterPrivKey.DeriveKey(&privPass)
	if err != nil {
		if err == snacl.ErrInvalidPassword {
			return ErrInvalidPassphrase
		} else {
			return err
		}
	}
	return nil
}

func export(b db.Bucket, keyscope KeyScope) (*Keystore, error) {
	remarkBytes, err := fetchRemark(b)
	if err != nil {
		return nil, err
	}
	entropyEncBytes, err := fetchEntropy(b)
	if err != nil {
		return nil, err
	}
	usage, err := fetchAccountUsage(b)
	if err != nil {
		return nil, err
	}
	internalNum, externalNum, err := fetchChildNum(b)
	if err != nil {
		return nil, err
	}
	_, privParams, err := fetchMasterKeyParams(b)
	if err != nil {
		return nil, err
	}
	_, _, cEntropyEnc, err := fetchCryptoKeys(b)
	if err != nil {
		return nil, err
	}

	cryptoStruct := cryptoJSON{
		Cipher:              "Stream cipher",
		EntropyEnc:          hex.EncodeToString(entropyEncBytes),
		KDF:                 "scrypt",
		PrivParams:          hex.EncodeToString(privParams),
		CryptoKeyEntropyEnc: hex.EncodeToString(cEntropyEnc),
	}

	hd := hdPath{
		Purpose:          keyscope.Purpose,
		Coin:             keyscope.Coin,
		Account:          usage,
		ExternalChildNum: externalNum,
		InternalChildNum: internalNum,
	}
	exportedKeyStruct := &Keystore{
		Remarks: string(remarkBytes),
		Crypto:  cryptoStruct,
		HDpath:  hd,
	}
	return exportedKeyStruct, nil
}

func (a *AddrManager) exportKeystore(dbTransaction db.ReadTransaction, passphrase []byte) (*Keystore, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	err := a.safelyCheckPassword(passphrase)
	if err != nil {
		return nil, err
	}

	amBucket := dbTransaction.FetchBucket(a.storage)
	keystore, err := export(amBucket, a.hdScope)
	if err != nil {
		logging.CPrint(logging.ERROR, "dump rootKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	return keystore, nil
}

func (a *AddrManager) destroy(dbTransaction db.DBTransaction) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	amBucket := dbTransaction.FetchBucket(a.storage)
	return amBucket.Clear()
}

func (a *AddrManager) clearPrivKeys() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.unlocked = false

	for _, mAddr := range a.addrs {
		if mAddr.privKey != nil {
			zero.BigInt(mAddr.privKey.D)
			mAddr.privKey = nil
		}
	}
	if a.acctInfo.acctKeyPriv != nil {
		a.acctInfo.acctKeyPriv.Zero()
	}
	a.acctInfo.acctKeyPriv = nil

	if a.branchInfo.externalBranchPriv != nil {
		a.branchInfo.externalBranchPriv.Zero()
		a.branchInfo.externalBranchPriv = nil
	}

	if a.branchInfo.internalBranchPriv != nil {
		a.branchInfo.internalBranchPriv.Zero()
		a.branchInfo.internalBranchPriv = nil
	}

	if a.masterKeyPriv != nil {
		a.masterKeyPriv.Zero()
	}
	if a.cryptoKeyPriv != nil {
		a.cryptoKeyPriv.Zero()
	}
	zero.Bytea64(&a.hashedPrivPassphrase)
	return
}

// check the pass before. IMPORTANT!!!
func (a *AddrManager) updatePrivKeys(pass []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	cryptoKeyPrivDec, err := a.masterKeyPriv.Decrypt(a.cryptoKeyPrivEncrypted)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to decrypt",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	a.cryptoKeyPriv.CopyBytes(cryptoKeyPrivDec)
	zero.Bytes(cryptoKeyPrivDec)

	acctKeyBytes, err := a.cryptoKeyPriv.Decrypt(a.acctInfo.acctKeyEncrypted)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to decrypt",
			logging.LogFormat{
				"err": err,
			})
		return err
	}

	acctKeyExPriv, err := hdkeychain.NewKeyFromString(string(acctKeyBytes))
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to get private account key ", logging.LogFormat{
			"error": err,
		})
		return err
	}
	zero.Bytes(acctKeyBytes)

	inBranchKeyExPriv, err := acctKeyExPriv.Child(InternalBranch)
	if err != nil {
		logging.CPrint(logging.ERROR, "new internal childKey failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}
	defer inBranchKeyExPriv.Zero()
	exBranchKeyExPriv, err := acctKeyExPriv.Child(ExternalBranch)
	if err != nil {
		logging.CPrint(logging.ERROR, "new external childKey failed",
			logging.LogFormat{
				"err": err,
			})
		return err
	}
	defer exBranchKeyExPriv.Zero()

	a.acctInfo.acctKeyPriv = acctKeyExPriv

	for _, mAddr := range a.addrs {
		var exBKey *hdkeychain.ExtendedKey
		if mAddr.derivationPath.Branch == ExternalBranch {
			exBKey = exBranchKeyExPriv
		} else {
			exBKey = inBranchKeyExPriv
		}

		cKeyExPriv, err := exBKey.Child(mAddr.derivationPath.Index)
		if err != nil {
			logging.CPrint(logging.ERROR, "new childKey failed",
				logging.LogFormat{
					"err": err,
				})
			return err
		}

		cKeyBtPriv, err := cKeyExPriv.ECPrivKey()
		if err != nil {
			logging.CPrint(logging.ERROR, "exPrivKey->btcec.PrivKey failed",
				logging.LogFormat{
					"err": err,
				})
			return err
		}
		mAddr.privKey = cKeyBtPriv
		cKeyExPriv.Zero()
	}
	a.unlocked = true
	return nil
}

func (a *AddrManager) nextAddresses(dbTransaction db.DBTransaction, checkfunc func([]byte) (bool, error), internal bool, numAddresses, addressGapLimit uint32, net *config.Params, nRequired int, addressClass uint16) ([]*ManagedAddress, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	am := dbTransaction.FetchBucket(a.storage)
	var acctKey *hdkeychain.ExtendedKey
	acctKey = a.acctInfo.acctKeyPub
	// TODO: only 2^31 pk can be generate, using private pass to generate more?
	if a.acctInfo.acctKeyPriv != nil {
		acctKey = a.acctInfo.acctKeyPriv
	}

	var branch uint32
	if internal {
		branch = InternalBranch
	} else {
		branch = ExternalBranch
	}

	nextIndex, err := getChildNum(am, internal)
	if err != nil {
		logging.CPrint(logging.ERROR, "fetch db failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}
	if numAddresses > MaxAddressesPerAccount || numAddresses+nextIndex > MaxAddressesPerAccount {
		logging.CPrint(logging.ERROR, "required too many new addresses", logging.LogFormat{
			"error":     ErrExceedAllowedNumberPerAccount,
			"number":    numAddresses,
			"nextIndex": nextIndex,
			"maximum":   MaxAddressesPerAccount,
		})
		return nil, ErrExceedAllowedNumberPerAccount
	}

	branchKey, err := acctKey.Child(branch)
	if err != nil {
		logging.CPrint(logging.ERROR, "new childKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}
	defer branchKey.Zero()

	if numAddresses > addressGapLimit {
		return nil, ErrGapLimit
	}

	if nextIndex != 0 && nextIndex+numAddresses > addressGapLimit {
		startIndex := nextIndex + numAddresses - addressGapLimit - 1

		pass := false
		for i := startIndex; i < nextIndex; i++ {
			addr, ok := a.index[i]
			if !ok {
				continue
			}
			managedAddr, ok := a.addrs[addr]
			if !ok {
				return nil, errors.New(fmt.Sprintf("failed to get managedAddress, index: %v", i))
			}
			used, err := checkfunc(managedAddr.scriptHash)
			if err != nil {
				return nil, err
			}
			if used {
				pass = true
				break
			}
		}
		if !pass {
			return nil, ErrGapLimit
		}
	}

	addressInfo := make([]*unlockDeriveInfo, 0, numAddresses)
	for i := uint32(0); i < numAddresses; i++ {
		var nextKey *hdkeychain.ExtendedKey
		for {
			indexKey, err := branchKey.Child(nextIndex)
			if err != nil {
				if err == hdkeychain.ErrInvalidChild {
					nextIndex++
					continue
				}
				logging.CPrint(logging.ERROR, "new childKey failed",
					logging.LogFormat{
						"err": err,
					})
				return nil, err
			}
			indexKey.SetNet(net)
			nextIndex++
			nextKey = indexKey
			break
		}
		// Now that we know this key can be used, we'll create the
		// proper derivation path so this information can be available
		// to callers.
		derivationPath := DerivationPath{
			Account: a.acctInfo.acctType,
			Branch:  branch,
			Index:   nextIndex - 1,
		}

		// create a new managed address based on the private key.
		// Also, zero the next key after creating the managed address
		// from it.
		managedAddr, err := newManagedAddressFromExtKey(a.keystoreName, derivationPath, nextKey, nRequired, addressClass, net)
		if err != nil {
			logging.CPrint(logging.ERROR, "new managedAddress failed",
				logging.LogFormat{
					"err": err,
				})
			return nil, err
		}

		nextKey.Zero()

		info := unlockDeriveInfo{
			managedAddr: managedAddr,
			branch:      branch,
			index:       nextIndex - 1,
		}
		addressInfo = append(addressInfo, &info)
	}

	managedAddresses := make([]*ManagedAddress, 0, len(addressInfo))
	for _, info := range addressInfo {
		ma := info.managedAddr
		managedAddresses = append(managedAddresses, ma)
	}

	err = updateChildNum(am, internal, nextIndex)
	if err != nil {
		logging.CPrint(logging.ERROR, "put db failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	pkBucket, err := db.GetOrCreateBucket(am, pubKeyBucket)
	if err != nil {
		return nil, err
	}
	for _, info := range addressInfo {
		pubKeyBytes := info.managedAddr.pubKey.SerializeCompressed()
		pubKeyEnc, err := a.cryptoKeyPub.Encrypt(pubKeyBytes)
		if err != nil {
			return nil, err
		}
		err = putEncryptedPubKey(pkBucket, info.branch, info.index, pubKeyEnc)
		if err != nil {
			return nil, err
		}
	}

	return managedAddresses, nil
}

func (a *AddrManager) updateManagedAddress(dbTransaction db.DBTransaction, managedAddresses []*ManagedAddress) error {
	for _, managedAddress := range managedAddresses {
		a.addrs[managedAddress.address] = managedAddress
		a.index[managedAddress.derivationPath.Index] = managedAddress.address
	}

	am := dbTransaction.FetchBucket(a.storage)
	inChildNUm, exChildNum, err := fetchChildNum(am)
	if err != nil {
		return err
	}
	a.branchInfo.nextExternalIndex = exChildNum
	a.branchInfo.nextInternalIndex = inChildNUm
	return nil
}

func (a *AddrManager) getPrivKeyBtcec(addr string, password []byte) (*btcec.PrivateKey, error) {
	// get address index
	mAddr, ok := a.addrs[addr]
	if !ok {
		logging.CPrint(logging.ERROR, "address not found in address manager", logging.LogFormat{"address": addr})
		return nil, ErrAddressNotFound
	}
	internal := mAddr.derivationPath.Branch == InternalBranch

	if mAddr.privKey != nil {
		return mAddr.privKey, nil
	}
	if a.branchInfo.externalBranchPriv == nil || a.branchInfo.internalBranchPriv == nil {
		cryptoKeyPrivDec, err := a.masterKeyPriv.Decrypt(a.cryptoKeyPrivEncrypted)
		if err != nil {
			return nil, err
		}

		a.cryptoKeyPriv.CopyBytes(cryptoKeyPrivDec)
		zero.Bytes(cryptoKeyPrivDec)

		acctKeyBytes, err := a.cryptoKeyPriv.Decrypt(a.acctInfo.acctKeyEncrypted)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to decrypt",
				logging.LogFormat{
					"error": err,
				})
			return nil, err
		}
		a.cryptoKeyPriv.Zero()
		a.cryptoKeyPriv = &cryptoKey{}

		acctKeyExPriv, err := hdkeychain.NewKeyFromString(string(acctKeyBytes))
		if err != nil {
			return nil, err
		}
		zero.Bytes(acctKeyBytes)
		externalBranchHDPrivKey, err := acctKeyExPriv.Child(ExternalBranch)
		if err != nil {
			return nil, err
		}
		internalBranchHDPrivKey, err := acctKeyExPriv.Child(InternalBranch)
		if err != nil {
			return nil, err
		}
		acctKeyExPriv.Zero()
		acctKeyExPriv = nil

		a.branchInfo.externalBranchPriv = externalBranchHDPrivKey
		a.branchInfo.internalBranchPriv = internalBranchHDPrivKey
	}
	var branchHDPrivKey *hdkeychain.ExtendedKey
	if internal {
		branchHDPrivKey = a.branchInfo.internalBranchPriv
	} else {
		branchHDPrivKey = a.branchInfo.externalBranchPriv
	}

	childHDPrivKey, err := branchHDPrivKey.Child(mAddr.derivationPath.Index)
	if err != nil {
		return nil, err
	}
	childPrivKeyBtcec, err := childHDPrivKey.ECPrivKey()
	if err != nil {
		return nil, err
	}
	a.addrs[addr].privKey = childPrivKeyBtcec

	return childPrivKeyBtcec, nil
}

func (a *AddrManager) signBtcec(hash []byte, addr string, password []byte) (signed *btcec.Signature, err error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(hash) != 32 {
		logging.CPrint(logging.ERROR, "hash size is not 32",
			logging.LogFormat{
				"err": ErrInvalidDataHash,
			})
		return nil, ErrInvalidDataHash
	}

	// check private passphrase
	err = a.checkPassword(password)
	if err != nil {
		return nil, err
	}
	// update cache, mark unlocked
	saltPassphrase := append(a.privPassphraseSalt[:], password...)
	a.hashedPrivPassphrase = sha512.Sum512(saltPassphrase)
	zero.Bytes(saltPassphrase)
	a.unlocked = true

	var privKey *btcec.PrivateKey
	privKey, err = a.getPrivKeyBtcec(addr, password)
	if err != nil {
		logging.CPrint(logging.ERROR, "get privKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	signed, err = privKey.Sign(hash)
	if err != nil {
		logging.CPrint(logging.ERROR, "sign failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	return signed, nil
}

func (a *AddrManager) changeRemark(dbTransaction db.DBTransaction, newRemark string) error {
	amBucket := dbTransaction.FetchBucket(a.storage)
	if amBucket == nil {
		logging.CPrint(logging.ERROR, "failed to get related bucket", logging.LogFormat{"error": ErrUnexpecteDBError})
		return ErrUnexpecteDBError
	}
	if len(newRemark) == 0 {
		err := deleteRemark(amBucket)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to change remark in database", logging.LogFormat{"error": err})
			return err
		}
	} else {
		err := putRemark(amBucket, []byte(newRemark))
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to put remark into database", logging.LogFormat{"error": err})
			return err
		}
	}
	a.remark = newRemark
	return nil
}

func (a *AddrManager) getMnemonic(dbTransaction db.ReadTransaction, privpass []byte) (string, error) {
	amBucket := dbTransaction.FetchBucket(a.storage)
	entropyEnc, err := fetchEntropy(amBucket)
	if err != nil {
		return "", err
	}

	cryptoEntropyKeyBytes, err := a.masterKeyPriv.Decrypt(a.cryptoKeyEntropyEncrypted)
	if err != nil {
		return "", err
	}

	var cryptoEntropyKey cryptoKey
	cryptoEntropyKey.CopyBytes(cryptoEntropyKeyBytes)
	zero.Bytes(cryptoEntropyKeyBytes)
	defer cryptoEntropyKey.Zero()

	entropy, err := cryptoEntropyKey.Decrypt(entropyEnc)
	if err != nil {
		return "", err
	}

	mnemonic, err := NewMnemonic(entropy)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to new mnemonic", logging.LogFormat{"error": err})
		return "", err
	}
	zero.Bytes(entropy)

	return mnemonic, nil
}

func (a *AddrManager) Name() string {
	return a.keystoreName
}

func (a *AddrManager) Remarks() string {
	return a.remark
}

func (a *AddrManager) AddrUse() AddrUse {
	return a.use
}

func (a *AddrManager) KeyScope() KeyScope {
	return a.hdScope
}

func (a *AddrManager) CountAddresses() (external int, internal int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, ma := range a.addrs {
		if ma.IsChangeAddr() {
			internal++
			continue
		}
		external++
	}
	return
}

func (a *AddrManager) ListAddresses() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	addrs := make([]string, 0)
	for addr := range a.addrs {
		addrs = append(addrs, addr)
	}
	return addrs
}

func (a *AddrManager) ManagedAddresses() []*ManagedAddress {
	a.mu.Lock()
	defer a.mu.Unlock()
	mas := make([]*ManagedAddress, 0)
	for _, ma := range a.addrs {
		mas = append(mas, ma)
	}
	return mas
}

func (a *AddrManager) Address(addr string) (*ManagedAddress, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ma, ok := a.addrs[addr]
	if ok {
		return ma, nil
	}
	return nil, ErrAddressNotFound
}

// TODO: remove before release?
// for testing purpose
// addresses - <wit address, scripthash>
func NewMockAddrManager(name string, addresses map[string][]byte) *AddrManager {
	am := &AddrManager{
		keystoreName: name,
		addrs:        make(map[string]*ManagedAddress),
	}
	for ea, sh := range addresses {
		ma := &ManagedAddress{
			scriptHash: sh,
		}
		am.addrs[ea] = ma
	}
	return am
}
