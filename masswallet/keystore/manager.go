package keystore

import (
	"sync"

	"crypto/rand"
	"fmt"

	"encoding/hex"

	"bytes"

	"math"

	"github.com/btcsuite/btcd/btcec"
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/errors"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/masswallet/db"
	"massnet.org/mass-wallet/masswallet/keystore/hdkeychain"
	"massnet.org/mass-wallet/masswallet/keystore/snacl"
	"massnet.org/mass-wallet/masswallet/keystore/zero"
)

const (
	PoCUsage    AddrUse = iota //0
	WalletUsage                //1

	ksMgrBucket     = "km"
	accountIDBucket = "aid"
	pubKeyBucket    = "pub"

	// MaxAccountNum is the maximum allowed account number.  This value was
	// chosen because accounts are hardened children and therefore must not
	// exceed the hardened child range of extended keys and it provides a
	// reserved account at the top of the range for supporting imported
	// addresses.
	MaxAccountNum = hdkeychain.HardenedKeyStart - 2 // 2^31 - 2

	// MaxAddressesPerAccount is the maximum allowed number of addresses
	// per account number.  This value is based on the limitation of the
	// underlying hierarchical deterministic key derivation.
	MaxAddressesPerAccount = hdkeychain.HardenedKeyStart - 1

	// ExternalBranch is the child number to use when performing BIP0044
	// style hierarchical deterministic key derivation for the external
	// branch.
	ExternalBranch uint32 = 0

	// InternalBranch is the child number to use when performing BIP0044
	// style hierarchical deterministic key derivation for the internal
	// branch.
	InternalBranch uint32 = 1

	// maxCoinType is the maximum allowed coin type used when structuring
	// the BIP0044 multi-account hierarchy.  This value is based on the
	// limitation of the underlying hierarchical deterministic key
	// derivation.
	maxCoinType = hdkeychain.HardenedKeyStart - 1

	// saltSize is the number of bytes of the salt used when hashing
	// private passphrases.
	saltSize = 32

	// nRequiredDefault is the default number of signatures for multisig
	nRequiredDefault = 1

	// defaultBitSize is the default length of entropy
	defaultBitSize = 128
)

type AddrUse uint8

type KeystoreManager struct {
	mu               sync.Mutex
	managedKeystores map[string]*AddrManager
	params           *config.Params
	ksMgrMeta        db.BucketMeta
	accountIDMeta    db.BucketMeta
	pubPassphrase    []byte
	currentKeystore  *currentKeystore
}

type currentKeystore struct {
	accountName string
}

// ScryptOptions is used to hold the scrypt parameters needed when deriving new
// passphrase keys.
type ScryptOptions struct {
	N, R, P int
}

// DefaultScryptOptions is the default options used with scrypt.
var DefaultScryptOptions = ScryptOptions{
	N: 262144, // 2^18
	R: 8,
	P: 1,
}

// defaultNewSecretKey returns a new secret key.  See newSecretKey.
func defaultNewSecretKey(passphrase *[]byte, config *ScryptOptions) (*snacl.SecretKey, error) {
	return snacl.NewSecretKey(passphrase, config.N, config.R, config.P)
}

var (
	// secretKeyGen is the inner method that is executed when calling
	// newSecretKey.
	secretKeyGen = defaultNewSecretKey
)

// EncryptorDecryptor provides an abstraction on top of snacl.CryptoKey so that
// our tests can use dependency injection to force the behaviour they need.
type EncryptorDecryptor interface {
	Encrypt(in []byte) ([]byte, error)
	Decrypt(in []byte) ([]byte, error)
	Bytes() []byte
	CopyBytes([]byte)
	Zero()
}

// cryptoKey extends snacl.CryptoKey to implement EncryptorDecryptor.
type cryptoKey struct {
	snacl.CryptoKey
}

// Bytes returns a copy of this crypto key's byte slice.
func (ck *cryptoKey) Bytes() []byte {
	return ck.CryptoKey[:]
}

// CopyBytes copies the bytes from the given slice into this CryptoKey.
func (ck *cryptoKey) CopyBytes(from []byte) {
	copy(ck.CryptoKey[:], from)
}

// defaultNewCryptoKey returns a new CryptoKey.  See newCryptoKey.
func defaultNewCryptoKey() (EncryptorDecryptor, error) {
	key, err := snacl.GenerateCryptoKey()
	if err != nil {
		return nil, err
	}
	return &cryptoKey{*key}, nil
}

// newCryptoKey is used as a way to replace the new crypto key generation
// function used so tests can provide a version that fails for testing error
// paths.
var newCryptoKey = defaultNewCryptoKey

// generateSeed generates seed following the suggestion in BIP-0039
func generateSeed(bitSize int, privpass []byte) ([]byte, string, []byte, error) {
	entropy, err := NewEntropy(bitSize)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to new entropy", logging.LogFormat{"error": err})
		return nil, "", nil, err
	}

	mnemonic, err := NewMnemonic(entropy)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to new mnemonic", logging.LogFormat{"error": err})
		return nil, "", nil, err
	}

	hdSeed := NewSeed(mnemonic, string(privpass))
	return entropy, mnemonic, hdSeed, nil
}

// createManagerKeyScope creates a new key scoped for a target manager's scope.
// This partitions key derivation for a particular purpose+coin tuple, allowing
// multiple address derivation schems to be maintained concurrently.
func createManagerKeyScope(km db.Bucket, root *hdkeychain.ExtendedKey,
	cryptoKeyPub, cryptoKeyPriv EncryptorDecryptor, hdpath *hdPath, checkfunc func([]byte) (bool, error),
	net *config.Params, addressGapLimit uint32) (db.BucketMeta, error) {

	scope := Net2KeyScope[net.HDCoinType]

	accountIDBucket, err := db.GetOrCreateBucket(km, accountIDBucket)
	if err != nil {
		return nil, err
	}

	// Derive the cointype key according to the passed scope.
	coinTypeKeyPriv, err := deriveCoinTypeKey(root, scope)
	if err != nil {
		return nil, err
	}
	defer coinTypeKeyPriv.Zero()

	// Derive the account key for the first account according our
	// BIP0044-like derivation.
	acctKeyPriv, err := deriveAccountKey(coinTypeKeyPriv, hdpath.Account)
	if err != nil {
		// The seed is unusable if the any of the children in the
		// required hierarchy can't be derived due to invalid child.
		if err == hdkeychain.ErrInvalidChild {
			str := "the provided seed is unusable"
			return nil, errors.New(str)
		}

		return nil, err
	}
	defer acctKeyPriv.Zero()

	// The address manager needs the first address for the account.
	acctKeyPub, err := acctKeyPriv.Neuter()
	if err != nil {
		return nil, fmt.Errorf("failed to convert private account key to public account key: %v", err)
	}

	// get the address of the pubKey
	acctEcPubKey, err := acctKeyPub.ECPubKey()
	if err != nil {
		return nil, err
	}

	accountID, err := pubKeyToAccountID(acctEcPubKey)
	if err != nil {
		return nil, err
	}

	// check for repeated seed
	value, _ := accountIDBucket.Get([]byte(accountID))
	if value != nil {
		return nil, ErrDuplicateSeed
	}

	// Ensure the branch keys can be derived for the provided seed according
	// to our BIP0044-like derivation.
	if err := checkBranchKeys(acctKeyPriv); err != nil {
		// The seed is unusable if the any of the children in the
		// required hierarchy can't be derived due to invalid child.
		if err == hdkeychain.ErrInvalidChild {
			str := "the provided seed is unusable"
			return nil, errors.New(str)
		}

		return nil, err
	}

	// put account PubKey into seed bucket
	err = putAccountID(accountIDBucket, []byte(accountID))
	if err != nil {
		return nil, err
	}
	// new account bucket
	accountBucket, err := km.NewBucket(accountID)
	if err != nil {
		return nil, err
	}

	// Encrypt the default account keys with the associated crypto keys.
	acctPubEnc, err := cryptoKeyPub.Encrypt([]byte(acctKeyPub.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt public account key: %v", err)
	}
	acctPrivEnc, err := cryptoKeyPriv.Encrypt([]byte(acctKeyPriv.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt private account key: %v", err)
	}

	err = putCoinType(accountBucket, scope.Coin)
	if err != nil {
		return nil, err
	}

	err = putAccountInfo(accountBucket, &scope, hdpath.Account, acctPubEnc, acctPrivEnc)
	if err != nil {
		return nil, err
	}

	// new external branch and save the pubkey
	internalBranchPrivKey, err := acctKeyPriv.Child(InternalBranch)
	if err != nil {
		logging.CPrint(logging.ERROR, "new childKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}
	defer internalBranchPrivKey.Zero()
	internalBranchPubKey, err := internalBranchPrivKey.Neuter()
	if err != nil {
		logging.CPrint(logging.ERROR, "exKey->exPubKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	externalBranchPrivKey, err := acctKeyPriv.Child(ExternalBranch)
	if err != nil {
		logging.CPrint(logging.ERROR, "new childKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}
	defer externalBranchPrivKey.Zero()
	externalBranchPubKey, err := externalBranchPrivKey.Neuter()
	if err != nil {
		logging.CPrint(logging.ERROR, "exKey->exPubKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	// Encrypt the default branch keys with the associated crypto keys.
	internalBranchPubEnc, err := cryptoKeyPub.Encrypt([]byte(internalBranchPubKey.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt internal public branch key: %v", err)
	}
	externalBranchPubEnc, err := cryptoKeyPub.Encrypt([]byte(externalBranchPubKey.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt external public branch key: %v", err)
	}

	err = putBranchPubKeys(accountBucket, internalBranchPubEnc, externalBranchPubEnc)
	if err != nil {
		logging.CPrint(logging.ERROR, "put db failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}
	err = initBranchChildNum(accountBucket)
	if err != nil {
		logging.CPrint(logging.ERROR, "put db failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	var safeUint32Add = func(a, b uint32) uint32 {
		if (a+b) < a || (a+b) < b {
			return math.MaxUint32
		}
		return a + b
	}

	if hdpath.InternalChildNum != 0 {
		nextIndex := uint32(0)
		addressInfo := make([]*unlockDeriveInfo, 0, hdpath.InternalChildNum)

		for i := uint32(0); i < safeUint32Add(nextIndex, addressGapLimit) || i < safeUint32Add(hdpath.InternalChildNum, addressGapLimit); i++ {
			var nextKey *hdkeychain.ExtendedKey
			for {
				indexKey, err := internalBranchPubKey.Child(i)
				if err != nil {
					if err == hdkeychain.ErrInvalidChild {
						continue
					}
					logging.CPrint(logging.ERROR, "new childKey failed",
						logging.LogFormat{
							"err": err,
						})
					return nil, err
				}
				indexKey.SetNet(net)
				nextKey = indexKey
				break
			}
			// Now that we know this key can be used, we'll create the
			// proper derivation path so this information can be available
			// to callers.
			derivationPath := DerivationPath{
				Account: hdpath.Account,
				Branch:  InternalBranch,
				Index:   i,
			}

			// create a new managed address based on the private key.
			// Also, zero the next key after creating the managed address
			// from it.
			managedAddr, err := newManagedAddressFromExtKey(accountID, derivationPath, nextKey, nRequiredDefault, massutil.AddressClassWitnessV0, net)
			if err != nil {
				logging.CPrint(logging.ERROR, "new managedAddress failed",
					logging.LogFormat{
						"err": err,
					})
				return nil, err
			}

			used, err := checkfunc(managedAddr.scriptHash)
			if err != nil {
				return nil, err
			}
			if used {
				nextIndex = i + 1
			}

			nextKey.Zero()

			info := unlockDeriveInfo{
				managedAddr: managedAddr,
				branch:      InternalBranch,
				index:       i,
			}
			addressInfo = append(addressInfo, &info)
		}

		if nextIndex < hdpath.InternalChildNum {
			nextIndex = hdpath.InternalChildNum
		}

		err = updateChildNum(accountBucket, true, nextIndex)
		if err != nil {
			logging.CPrint(logging.ERROR, "put db failed",
				logging.LogFormat{
					"err": err,
				})
			return nil, err
		}

		pkBucket, err := db.GetOrCreateBucket(accountBucket, pubKeyBucket)
		if err != nil {
			return nil, err
		}
		for _, info := range addressInfo[:nextIndex] {
			pubKeyBytes := info.managedAddr.pubKey.SerializeCompressed()
			pubKeyEnc, err := cryptoKeyPub.Encrypt(pubKeyBytes)
			if err != nil {
				return nil, err
			}
			err = putEncryptedPubKey(pkBucket, info.branch, info.index, pubKeyEnc)
			if err != nil {
				return nil, err
			}
		}
	}

	if hdpath.ExternalChildNum != 0 {
		nextIndex := uint32(0)
		addressInfo := make([]*unlockDeriveInfo, 0)
		for i := uint32(0); i < safeUint32Add(nextIndex, addressGapLimit) || i < safeUint32Add(hdpath.ExternalChildNum, addressGapLimit); i++ {
			var nextKey *hdkeychain.ExtendedKey
			for {
				indexKey, err := externalBranchPubKey.Child(i)
				if err != nil {
					if err == hdkeychain.ErrInvalidChild {
						continue
					}
					logging.CPrint(logging.ERROR, "new childKey failed",
						logging.LogFormat{
							"err": err,
						})
					return nil, err
				}
				indexKey.SetNet(net)
				nextKey = indexKey
				break
			}
			// Now that we know this key can be used, we'll create the
			// proper derivation path so this information can be available
			// to callers.
			derivationPath := DerivationPath{
				Account: hdpath.Account,
				Branch:  ExternalBranch,
				Index:   i,
			}

			// create a new managed address based on the private key.
			// Also, zero the next key after creating the managed address
			// from it.
			managedAddr, err := newManagedAddressFromExtKey(accountID, derivationPath, nextKey, nRequiredDefault, massutil.AddressClassWitnessV0, net)
			if err != nil {
				logging.CPrint(logging.ERROR, "new managedAddress failed",
					logging.LogFormat{
						"err": err,
					})
				return nil, err
			}

			used, err := checkfunc(managedAddr.scriptHash)
			if err != nil {
				return nil, err
			}
			if used {
				nextIndex = i + 1
			}

			nextKey.Zero()

			info := unlockDeriveInfo{
				managedAddr: managedAddr,
				branch:      ExternalBranch,
				index:       i,
			}
			addressInfo = append(addressInfo, &info)
		}

		if nextIndex < hdpath.ExternalChildNum {
			nextIndex = hdpath.ExternalChildNum
		}

		err = updateChildNum(accountBucket, false, nextIndex)
		if err != nil {
			logging.CPrint(logging.ERROR, "put db failed",
				logging.LogFormat{
					"err": err,
				})
			return nil, err
		}

		pkBucket, err := db.GetOrCreateBucket(accountBucket, pubKeyBucket)
		if err != nil {
			return nil, err
		}
		for _, info := range addressInfo[:nextIndex] {
			pubKeyBytes := info.managedAddr.pubKey.SerializeCompressed()
			pubKeyEnc, err := cryptoKeyPub.Encrypt(pubKeyBytes)
			if err != nil {
				return nil, err
			}
			err = putEncryptedPubKey(pkBucket, info.branch, info.index, pubKeyEnc)
			if err != nil {
				return nil, err
			}
		}
	}

	return accountBucket.GetBucketMeta(), nil
}

func initAcctBucket(dbTransaction db.DBTransaction, kmBucketMeta db.BucketMeta, remark string, net *config.Params, addressGapLimit uint32,
	scryptConfig *ScryptOptions, hdPath *hdPath, pubPassphrase, privPassphrase, entropy, seed []byte, checkfunc func([]byte) (bool, error)) (db.BucketMeta, error) {
	// get the km bucket created before
	kmBucket := dbTransaction.FetchBucket(kmBucketMeta)

	if kmBucket == nil {
		return nil, ErrBucketNotFound
	}

	if scryptConfig == nil {
		scryptConfig = &DefaultScryptOptions
	}

	// Generate new master keys.  These master keys are used to protect the
	// crypto keys that will be generated next.
	masterKeyPub, err := secretKeyGen(&pubPassphrase, scryptConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to generate master public key: %v", err)
	}
	masterKeyPriv, err := secretKeyGen(&privPassphrase, scryptConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to generate master private key: %v", err)
	}
	defer masterKeyPriv.Zero()

	// Generate new crypto public, private, and script keys.  These keys are
	// used to protect the actual public and private data such as addresses,
	// extended keys, and scripts.
	cryptoKeyPub, err := newCryptoKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate crypto public key: %v", err)
	}
	cryptoKeyPriv, err := newCryptoKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate crypto private key: %v", err)
	}
	defer cryptoKeyPriv.Zero()
	cryptoKeyEnt, err := newCryptoKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate crypto script key: %v", err)
	}
	defer cryptoKeyEnt.Zero()

	// Encrypt the crypto keys with the associated master keys.
	cryptoKeyPubEnc, err := masterKeyPub.Encrypt(cryptoKeyPub.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt crypto public key: %v", err)
	}
	cryptoKeyPrivEnc, err := masterKeyPriv.Encrypt(cryptoKeyPriv.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt crypto private key: %v", err)
	}
	cryptoKeyEntEnc, err := masterKeyPriv.Encrypt(cryptoKeyEnt.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt crypto script key: %v", err)
	}

	// Save the master key params to the database.
	pubParams := masterKeyPub.Marshal()
	privParams := masterKeyPriv.Marshal()

	// Generate the BIP0044 HD key structure to ensure the provided seed
	// can generate the required structure with no issues.

	// Derive the master extended key from the seed.
	rootKey, err := hdkeychain.NewMaster(seed, net)
	if err != nil {
		return nil, fmt.Errorf("failed to derive master extended key: %v", err)
	}
	defer rootKey.Zero()

	// Next, for each registers default manager scope, we'll create the
	// hardened cointype key for it, as well as the first default account.

	// Before we proceed, we'll also store the root master private key
	// within the database in an encrypted format. This is required as in
	// the future, we may need to create additional scoped key managers.
	entropyEnc, err := cryptoKeyEnt.Encrypt(entropy)
	if err != nil {
		return nil, err
	}
	zero.Bytes(entropy)

	acctBucketMeta, err := createManagerKeyScope(kmBucket, rootKey,
		cryptoKeyPub, cryptoKeyPriv, hdPath, checkfunc, net, addressGapLimit)
	if err != nil {
		return nil, err
	}

	acctBucket := dbTransaction.FetchBucket(acctBucketMeta)
	if acctBucket == nil {
		return nil, ErrUnexpecteDBError
	}

	if len(remark) > 0 {
		err = putRemark(acctBucket, []byte(remark))
		if err != nil {
			return nil, err
		}
	}

	err = putMasterKeyParams(acctBucket, pubParams, privParams)
	if err != nil {
		return nil, err
	}

	err = putEntropy(acctBucket, entropyEnc)
	if err != nil {
		return nil, err
	}

	// Save the encrypted crypto keys to the database.
	err = putCryptoKeys(acctBucket, cryptoKeyPubEnc, cryptoKeyPrivEnc,
		cryptoKeyEntEnc)
	if err != nil {
		return nil, err
	}

	return acctBucketMeta, nil
}

// create creates a new address manager in the given namespace.  The seed must
// conform to the standards described in hdkeychain.NewMaster and will be used
// to create the master root node from which all hierarchical deterministic
// addresses are derived.  This allows all chained addresses in the address
// manager to be recovered by using the same seed.
//
// All private and public keys and information are protected by secret keys
// derived from the provided private and public passphrases.  The public
// passphrase is required on subsequent opens of the address manager, and the
// private passphrase is required to unlock the address manager in order to
// gain access to any private keys and information.
//
// If a config structure is passed to the function, that configuration will
// override the defaults.
//
// A ManagerError with an error code of ErrAlreadyExists will be returned the
// address manager already exists in the specified namespace.
func create(dbTransaction db.DBTransaction, kmBucketMeta db.BucketMeta, bitSize int, pubPassphrase, privPassphrase []byte,
	usage AddrUse, remark string, net *config.Params, scryptConfig *ScryptOptions, addressGapLimit uint32) (db.BucketMeta, string, error) {
	if !ValidatePassphrase(privPassphrase) {
		return nil, "", ErrIllegalPassphrase
	}

	if !ValidatePassphrase(pubPassphrase) {
		return nil, "", ErrIllegalPassphrase
	}
	if bytes.Compare(privPassphrase, pubPassphrase) == 0 {
		return nil, "", ErrIllegalNewPrivPass
	}

	// set defaultBitSize if it wasn't set
	if bitSize == 0 {
		bitSize = defaultBitSize
	}

	entropy, mnemonic, seed, err := generateSeed(bitSize, privPassphrase)
	if err != nil {
		return nil, "", err
	}

	hdpath := &hdPath{
		Account:          uint32(usage),
		InternalChildNum: 0,
		ExternalChildNum: 0,
	}

	acctBucketMeta, err := initAcctBucket(dbTransaction, kmBucketMeta, remark, net, addressGapLimit, scryptConfig, hdpath, pubPassphrase,
		privPassphrase, entropy, seed, nil)
	if err != nil {
		return nil, "", err
	}

	return acctBucketMeta, mnemonic, nil
}

func loadAddrManager(amBucket db.Bucket, pubPassphrase []byte, net *config.Params) (*AddrManager, error) {
	amBucketMeta := amBucket.GetBucketMeta()
	// fetch the coin type
	coin, err := fetchCoinType(amBucket)
	if err != nil {
		return nil, err
	}
	keyScope, ok := Net2KeyScope[coin]
	if !ok {
		return nil, ErrKeyScopeNotFound
	}
	if coin != net.HDCoinType {
		str := "the coin type doesn't match the connected network"
		return nil, errors.New(str)
	}

	remarkBytes, err := fetchRemark(amBucket)
	if err != nil {
		return nil, err
	}

	// Load the master key params from the db.
	masterKeyPubParams, masterKeyPrivParams, err := fetchMasterKeyParams(amBucket)
	if err != nil {
		return nil, err
	}

	// Load the crypto keys from the db.
	cryptoKeyPubEnc, cryptoKeyPrivEnc, cryptoKeyEntropyEnc, err := fetchCryptoKeys(amBucket)
	if err != nil {
		return nil, err
	}

	var masterKeyPriv snacl.SecretKey
	err = masterKeyPriv.Unmarshal(masterKeyPrivParams)
	if err != nil {
		str := "failed to unmarshal master private key"
		return nil, errors.New(str)
	}

	// Derive the master public key using the serialized params and provided
	// passphrase.
	var masterKeyPub snacl.SecretKey
	if err := masterKeyPub.Unmarshal(masterKeyPubParams); err != nil {
		str := "failed to unmarshal master public key"
		return nil, errors.New(str)
	}
	if err := masterKeyPub.DeriveKey(&pubPassphrase); err != nil {
		str := "invalid passphrase for master public key"
		return nil, errors.New(str)
	}

	// Use the master public key to decrypt the crypto public key.
	cryptoKeyPub := &cryptoKey{snacl.CryptoKey{}}
	cryptoKeyPubCT, err := masterKeyPub.Decrypt(cryptoKeyPubEnc)
	if err != nil {
		str := "failed to decrypt crypto public key"
		return nil, errors.New(str)
	}
	cryptoKeyPub.CopyBytes(cryptoKeyPubCT)
	zero.Bytes(cryptoKeyPubCT)

	// Generate private passphrase salt.
	var privPassphraseSalt [saltSize]byte
	_, err = rand.Read(privPassphraseSalt[:])
	if err != nil {
		str := "failed to read random source for passphrase salt"
		return nil, errors.New(str)
	}

	// fetch the account usage
	account, err := fetchAccountUsage(amBucket)
	if err != nil {
		return nil, err
	}

	// fetch the account info
	rowInterface, err := fetchAccountInfo(amBucket, account)
	if err != nil {
		return nil, err
	}
	row, ok := rowInterface.(*dbHDAccountKey)
	if !ok {
		str := fmt.Sprintf("unsupported account type %T", row)
		return nil, errors.New(str)
	}

	// Use the crypto public key to decrypt the account public extended
	// key.
	serializedKeyPub, err := cryptoKeyPub.Decrypt(row.pubKeyEncrypted)
	if err != nil {
		str := fmt.Sprintf("failed to decrypt public key for account %d",
			account)
		return nil, errors.New(str)
	}
	acctKeyPub, err := hdkeychain.NewKeyFromString(string(serializedKeyPub))
	if err != nil {
		str := fmt.Sprintf("failed to create extended public key for "+
			"account %d", account)
		return nil, errors.New(str)
	}

	// create the new account info with the known information.  The rest of
	// the fields are filled out below.
	acctInfo := &accountInfo{
		acctType:         account,
		acctKeyEncrypted: row.privKeyEncrypted,
		acctKeyPub:       acctKeyPub,
	}

	internalBranchPubEnc, externalBranchPubEnc, err := fetchBranchPubKeys(amBucket)
	// Use the crypto public key to decrypt the external and internal branch public extended keys.
	serializedInBrKeyPub, err := cryptoKeyPub.Decrypt(internalBranchPubEnc)
	if err != nil {
		str := fmt.Sprintf("failed to decrypt internal branch public key for account %d",
			account)
		return nil, errors.New(str)
	}
	internalBranchPub, err := hdkeychain.NewKeyFromString(string(serializedInBrKeyPub))
	if err != nil {
		str := fmt.Sprintf("failed to create extended internal branch public key for "+
			"account %d", account)
		return nil, errors.New(str)
	}

	serializedExBrKeyPub, err := cryptoKeyPub.Decrypt(externalBranchPubEnc)
	if err != nil {
		str := fmt.Sprintf("failed to decrypt external branch public key for account %d",
			account)
		return nil, errors.New(str)
	}
	externalBranchPub, err := hdkeychain.NewKeyFromString(string(serializedExBrKeyPub))
	if err != nil {
		str := fmt.Sprintf("failed to create extended external branch public key for "+
			"account %d", account)
		return nil, errors.New(str)
	}

	// get child number
	internalChildNum, externalChildNum, err := fetchChildNum(amBucket)

	branchInfo := &branchInfo{
		internalBranchPub: internalBranchPub,
		externalBranchPub: externalBranchPub,
		nextInternalIndex: internalChildNum,
		nextExternalIndex: externalChildNum,
	}

	managedAddresses := make(map[string]*ManagedAddress)
	index := make(map[uint32]string)
	pubkeyBucket := amBucket.Bucket(pubKeyBucket)
	if pubkeyBucket != nil {
		pks, err := fetchEncryptedPubKey(pubkeyBucket)
		if err != nil {
			return nil, err
		}
		for _, pkp := range pks {
			path := DerivationPath{
				Account: account,
				Branch:  pkp.branch,
				Index:   pkp.index,
			}
			pubkeyBytes, err := cryptoKeyPub.Decrypt(pkp.pubkeyEnc)
			if err != nil {
				return nil, err
			}
			pubkey, err := btcec.ParsePubKey(pubkeyBytes, btcec.S256())
			if err != nil {
				return nil, err
			}
			managedAddress, err := newManagedAddressWithoutPrivKey(amBucketMeta.Name(), path, pubkey, nRequiredDefault, massutil.AddressClassWitnessStaking, net)
			if err != nil {
				return nil, err
			}
			managedAddresses[managedAddress.address] = managedAddress
			index[pkp.index] = managedAddress.address
		}
	}

	return &AddrManager{
		use:                       AddrUse(account),
		unlocked:                  false,
		index:                     index,
		addrs:                     managedAddresses,
		keystoreName:              amBucketMeta.Name(),
		remark:                    string(remarkBytes),
		masterKeyPub:              &masterKeyPub,
		masterKeyPriv:             &masterKeyPriv,
		cryptoKeyPub:              cryptoKeyPub,
		cryptoKeyPrivEncrypted:    cryptoKeyPrivEnc,
		cryptoKeyPriv:             &cryptoKey{},
		cryptoKeyEntropyEncrypted: cryptoKeyEntropyEnc,
		privPassphraseSalt:        privPassphraseSalt,
		hdScope:                   keyScope,
		acctInfo:                  acctInfo,
		branchInfo:                branchInfo,
		storage:                   amBucketMeta,
	}, nil

}

func (km *KeystoreManager) NewKeystore(dbTransaction db.DBTransaction, bitSize int, privPassphrase []byte, remark string,
	net *config.Params, scryptConfig *ScryptOptions, addressGapLimit uint32) (string, string, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	// create hd key chain and init the bucket
	acctBucketMeta, mnemonic, err := create(dbTransaction, km.ksMgrMeta, bitSize, km.pubPassphrase, privPassphrase, WalletUsage, remark, net, scryptConfig, addressGapLimit)
	if err != nil {
		return "", "", err
	}

	amBucket := dbTransaction.FetchBucket(acctBucketMeta)
	if amBucket == nil {
		return "", "", ErrUnexpecteDBError
	}

	addrManager, err := loadAddrManager(amBucket, km.pubPassphrase, net)
	if err != nil {
		return "", "", err
	}
	km.managedKeystores[addrManager.Name()] = addrManager
	return addrManager.Name(), mnemonic, nil
}

func (km *KeystoreManager) allocAddrMgrNamespace(dbTransaction db.DBTransaction, privPassphrase []byte, pubPassphrase []byte,
	kStore *Keystore, checkfunc func([]byte) (bool, error), net *config.Params, scryptConfig *ScryptOptions, addressGapLimit uint32) (db.BucketMeta, error) {
	masterKeyPrivParams, err := hex.DecodeString(kStore.Crypto.PrivParams)
	if err != nil {
		return nil, err
	}
	var masterKeyPriv snacl.SecretKey
	defer masterKeyPriv.Zero()
	err = unmarshalMasterPrivKey(&masterKeyPriv, privPassphrase, masterKeyPrivParams)
	if err != nil {
		logging.CPrint(logging.ERROR, "unmarshalMasterPrivKey failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	cryptoKeyEntropyEncOld, err := hex.DecodeString(kStore.Crypto.CryptoKeyEntropyEnc)
	if err != nil {
		return nil, err
	}

	entropyEnc, err := hex.DecodeString(kStore.Crypto.EntropyEnc)
	if err != nil {
		return nil, err
	}

	cEntropyBytes, err := masterKeyPriv.Decrypt(cryptoKeyEntropyEncOld)
	if err != nil {
		return nil, err
	}

	var cEntropyOld cryptoKey
	cEntropyOld.CopyBytes(cEntropyBytes)
	zero.Bytes(cEntropyBytes)
	defer cEntropyOld.Zero()

	entropy, err := cEntropyOld.Decrypt(entropyEnc)
	if err != nil {
		return nil, err
	}
	defer zero.Bytes(entropy)

	mnemonic, err := NewMnemonic(entropy)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to new mnemonic", logging.LogFormat{"error": err})
		return nil, err
	}

	seed := NewSeed(mnemonic, string(privPassphrase))

	rootKey, err := hdkeychain.NewMaster(seed, net)
	if err != nil {
		return nil, fmt.Errorf("failed to derive master extended key: %v", err)
	}
	defer rootKey.Zero()

	if scryptConfig == nil {
		scryptConfig = &DefaultScryptOptions
	}

	masterKeyPub, err := secretKeyGen(&pubPassphrase, scryptConfig)
	if err != nil {
		return nil, err
	}

	pubParams := masterKeyPub.Marshal()
	privParams := masterKeyPriv.Marshal()

	cryptoKeyPub, err := newCryptoKey()
	if err != nil {
		return nil, err
	}
	cryptoKeyPriv, err := newCryptoKey()
	if err != nil {
		return nil, err
	}
	defer cryptoKeyPriv.Zero()
	cryptoKeyEnt, err := newCryptoKey()
	if err != nil {
		return nil, err
	}
	defer cryptoKeyEnt.Zero()

	cryptoKeyPubEnc, err := masterKeyPub.Encrypt(cryptoKeyPub.Bytes())
	if err != nil {
		return nil, err
	}
	cryptoKeyPrivEnc, err := masterKeyPriv.Encrypt(cryptoKeyPriv.Bytes())
	if err != nil {
		return nil, err
	}
	cryptoKeyEntEnc, err := masterKeyPriv.Encrypt(cryptoKeyEnt.Bytes())
	if err != nil {
		return nil, err
	}

	entropyEncNew, err := cryptoKeyEnt.Encrypt(entropy)
	if err != nil {
		return nil, err
	}

	kmBucket := dbTransaction.FetchBucket(km.ksMgrMeta)
	if kmBucket == nil {
		return nil, ErrBucketNotFound
	}

	acctBucketMeta, err := createManagerKeyScope(kmBucket, rootKey,
		cryptoKeyPub, cryptoKeyPriv, &kStore.HDpath, checkfunc, net, addressGapLimit)
	if err != nil {
		return nil, err
	}

	acctBucket := dbTransaction.FetchBucket(acctBucketMeta)
	if acctBucket == nil {
		return nil, ErrUnexpecteDBError
	}

	if len(kStore.Remarks) > 0 {
		err = putRemark(acctBucket, []byte(kStore.Remarks))
		if err != nil {
			return nil, err
		}
	}

	err = putMasterKeyParams(acctBucket, pubParams, privParams)
	if err != nil {
		return nil, err
	}

	err = putEntropy(acctBucket, entropyEncNew)
	if err != nil {
		return nil, err
	}

	// Save the encrypted crypto keys to the database.
	err = putCryptoKeys(acctBucket, cryptoKeyPubEnc, cryptoKeyPrivEnc, cryptoKeyEntEnc)
	if err != nil {
		return nil, err
	}

	return acctBucketMeta, nil
}

func (km *KeystoreManager) ImportKeystore(dbTransaction db.DBTransaction, checkfunc func([]byte) (bool, error),
	keystoreJson []byte, oldPrivPass []byte, addressGapLimit uint32) (*AddrManager, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	// load keystore
	kStore, err := getKeystoreFromJson(keystoreJson)
	if err != nil {
		return nil, err
	}

	// check coinType
	if kStore.HDpath.Coin != km.params.HDCoinType {
		return nil, ErrCoinType
	}

	// check accountType
	if kStore.HDpath.Account != uint32(WalletUsage) {
		return nil, ErrAccountType
	}

	if kStore.HDpath.ExternalChildNum == 0 {
		kStore.HDpath.ExternalChildNum = 1
	}

	// storage update
	amBucketMeta, err := km.allocAddrMgrNamespace(dbTransaction, oldPrivPass, km.pubPassphrase, kStore, checkfunc, km.params, nil, addressGapLimit)
	if err != nil {
		return nil, err
	}

	// load addrManager
	amBucket := dbTransaction.FetchBucket(amBucketMeta)
	addrManager, err := loadAddrManager(amBucket, km.pubPassphrase, km.params)
	if err != nil {
		return nil, err
	}

	km.managedKeystores[addrManager.keystoreName] = addrManager
	return addrManager, nil
}

func (km *KeystoreManager) ImportKeystoreWithMnemonic(dbTransaction db.DBTransaction, checkfunc func([]byte) (bool, error),
	mnemonic, remark string, privPass []byte, externalIndex, internalIndex, addressGapLimit uint32) (*AddrManager, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	entropy, err := EntropyFromMnemonic(mnemonic)
	if err != nil {
		return nil, err
	}

	seed, err := NewSeedWithErrorChecking(mnemonic, string(privPass))
	if err != nil {
		return nil, err
	}

	hdpath := &hdPath{
		Account:          uint32(WalletUsage),
		InternalChildNum: internalIndex,
		ExternalChildNum: externalIndex,
	}
	if hdpath.ExternalChildNum == 0 {
		hdpath.ExternalChildNum = 1
	}

	acctBucketMeta, err := initAcctBucket(dbTransaction, km.ksMgrMeta, remark, km.params, addressGapLimit, &DefaultScryptOptions, hdpath,
		km.pubPassphrase, privPass, entropy, seed, checkfunc)
	if err != nil {
		return nil, err
	}

	acctBucket := dbTransaction.FetchBucket(acctBucketMeta)
	if acctBucket == nil {
		return nil, ErrUnexpecteDBError
	}

	addrManager, err := loadAddrManager(acctBucket, km.pubPassphrase, km.params)
	if err != nil {
		return nil, err
	}

	km.managedKeystores[addrManager.keystoreName] = addrManager
	return addrManager, nil
}

func (km *KeystoreManager) ExportKeystore(dbTransaction db.ReadTransaction, accountID string, privPassphrase []byte) ([]byte, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	addrManager, found := km.managedKeystores[accountID]
	if found {
		keystore, err := addrManager.exportKeystore(dbTransaction, privPassphrase)
		if err != nil {
			return nil, err
		}
		keystoreBytes := keystore.Bytes()

		return keystoreBytes, nil
	} else {
		logging.CPrint(logging.ERROR, "account not exists",
			logging.LogFormat{
				"err": ErrAccountNotFound,
			})
		return nil, ErrAccountNotFound
	}
}

func (km *KeystoreManager) DeleteKeystore(dbTransaction db.DBTransaction, accountID string, privPassphrase []byte) (bool, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	addrManager, found := km.managedKeystores[accountID]
	if found {
		// check private pass
		err := addrManager.safelyCheckPassword(privPassphrase)
		if err != nil {
			logging.CPrint(logging.ERROR, "wrong passphrase",
				logging.LogFormat{
					"err": err,
				})
			return false, err
		}

		if addrManager.destroy(dbTransaction) != nil {
			logging.CPrint(logging.ERROR, "delete account failed",
				logging.LogFormat{
					"err": err,
				})
			return false, err
		}

		kmBucket := dbTransaction.FetchBucket(km.ksMgrMeta)
		if kmBucket == nil {
			logging.CPrint(logging.ERROR, "failed to fetch keystore manager bucket", logging.LogFormat{})
			return false, ErrBucketNotFound
		}

		err = kmBucket.DeleteBucket(addrManager.keystoreName)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to delete bucket under keystore manager", logging.LogFormat{"bucket name": addrManager.keystoreName})
			return false, err
		}

		accountIDBucket := dbTransaction.FetchBucket(km.accountIDMeta)
		err = deleteAccountID(accountIDBucket, []byte(accountID))
		if err != nil {
			return false, err
		}
		addrManager.clearPrivKeys()
		delete(km.managedKeystores, accountID)
		if km.currentKeystore != nil && km.currentKeystore.accountName == accountID {
			km.currentKeystore = nil
		}
		return true, nil

	} else {
		logging.CPrint(logging.ERROR, "account not exists",
			logging.LogFormat{
				"err": ErrAccountNotFound,
			})
		return false, ErrAccountNotFound
	}
}

func (km *KeystoreManager) UseKeystoreForWallet(name string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	_, found := km.managedKeystores[name]
	if found {
		// same as current keystore, just return
		if km.currentKeystore != nil && name == km.currentKeystore.accountName {
			return nil
		}

		km.currentKeystore = &currentKeystore{accountName: name}

	} else {
		logging.CPrint(logging.ERROR, "account not exists",
			logging.LogFormat{
				"err": ErrAccountNotFound,
			})
		return ErrAccountNotFound
	}

	return nil
}

func (km *KeystoreManager) updateManagedKeystore(dbTransaction db.ReadTransaction, accountID string) error {
	addrManager, ok := km.managedKeystores[accountID]
	kmBucket := dbTransaction.FetchBucket(km.ksMgrMeta)
	if kmBucket == nil {
		return ErrBucketNotFound
	}
	amBucket := kmBucket.Bucket(accountID)
	if amBucket != nil && !ok {
		addrManager, err := loadAddrManager(amBucket, km.pubPassphrase, km.params)
		if err != nil {
			return err
		}
		km.managedKeystores[addrManager.keystoreName] = addrManager
	}
	if amBucket == nil && ok {
		addrManager.clearPrivKeys()
		delete(km.managedKeystores, accountID)
	}
	return nil
}

func (km *KeystoreManager) UpdateManagedKeystores(dbTransaction db.ReadTransaction, accountID string) {
	err := km.updateManagedKeystore(dbTransaction, accountID)
	if err != nil {
		logging.CPrint(logging.FATAL, "failed to update managed keystore", logging.LogFormat{"error": err})
	}
}

func (km *KeystoreManager) RemoveCachedKeystore(accountID string) {
	km.mu.Lock()
	defer km.mu.Unlock()
	addrManager, ok := km.managedKeystores[accountID]
	if ok {
		addrManager.clearPrivKeys()
		delete(km.managedKeystores, accountID)
	}
}

func (km *KeystoreManager) NextAddresses(dbTransaction db.DBTransaction, checkfunc func([]byte) (bool, error), internal bool, numAddresses, addressGapLimit uint32, addressClass uint16) ([]*ManagedAddress, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	if km.currentKeystore == nil {
		return nil, ErrCurrentKeystoreNotFound
	}
	accountID := km.currentKeystore.accountName
	addrManager := km.managedKeystores[accountID]
	managedAddresses, err := addrManager.nextAddresses(dbTransaction, checkfunc, internal, numAddresses, addressGapLimit, km.params, nRequiredDefault, addressClass)
	if err != nil {
		logging.CPrint(logging.ERROR, "new address failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	err = addrManager.updateManagedAddress(dbTransaction, managedAddresses)
	if err != nil {
		return nil, err
	}

	return managedAddresses, nil
}

func (km *KeystoreManager) CheckPrivPassphrase(acctId string, pass []byte) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	addrManager, err := km.getAddrManagerByAccountID(acctId)
	if err != nil {
		return err
	}

	return addrManager.safelyCheckPassword(pass)
}

func (km *KeystoreManager) SignHash(pubKey *btcec.PublicKey, hash []byte, password []byte) (*btcec.Signature, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	pks := make([]*btcec.PublicKey, 0)
	pks = append(pks, pubKey)
	_, addr, err := newWitnessScriptAddressForBtcec(pks, nRequiredDefault, massutil.AddressClassWitnessV0, km.params)
	if err != nil {
		logging.CPrint(logging.ERROR, "newWitnessScriptAddressForBtcec error",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	addrManager, err := km.getAddrManager(addr.EncodeAddress())
	if err != nil {
		return nil, err
	}

	var sig *btcec.Signature
	sig, err = addrManager.signBtcec(hash, addr.EncodeAddress(), password)
	if err != nil {
		logging.CPrint(logging.ERROR, "sign failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}

	return sig, nil
}

func (km *KeystoreManager) ChangePubPassphrase(dbTransaction db.DBTransaction, oldPubPass, newPubPass []byte, scryptConfig *ScryptOptions) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	if !ValidatePassphrase(newPubPass) {
		return ErrIllegalPassphrase
	}

	if bytes.Compare(newPubPass, oldPubPass) == 0 {
		return ErrSamePubpass
	}

	if scryptConfig == nil {
		scryptConfig = &DefaultScryptOptions
	}

	for _, addrManager := range km.managedKeystores {
		amBucket := dbTransaction.FetchBucket(addrManager.storage)
		if amBucket == nil {
			return ErrUnexpecteDBError
		}

		err := addrManager.safelyCheckPassword(newPubPass)
		if err == nil {
			return ErrIllegalNewPubPass
		}

		// Load the old master pubkey params from the db.
		masterKeyPubParams, _, err := fetchMasterKeyParams(amBucket)
		if err != nil {
			return err
		}
		// Derive the master public key using the serialized params and provided
		// passphrase.
		var oldMasterKeyPub snacl.SecretKey
		if err := oldMasterKeyPub.Unmarshal(masterKeyPubParams); err != nil {
			return err
		}
		if err := oldMasterKeyPub.DeriveKey(&oldPubPass); err != nil {
			return err
		}

		// Load the crypto keys from the db.
		cryptoKeyPubEnc, _, _, err := fetchCryptoKeys(amBucket)
		if err != nil {
			return err
		}
		// Use the master public key to decrypt the crypto public key.
		cryptoKeyPubCT, err := oldMasterKeyPub.Decrypt(cryptoKeyPubEnc)
		if err != nil {
			return err
		}

		// Generate new master keys.  These master keys are used to protect the
		// crypto keys that will be generated next.
		newMasterKeyPub, err := secretKeyGen(&newPubPass, scryptConfig)
		if err != nil {
			return err
		}

		// Save the master key params to the database.
		newPubParams := newMasterKeyPub.Marshal()

		// Encrypt the crypto keys with the associated master keys.
		newCryptoKeyPubEnc, err := newMasterKeyPub.Encrypt(cryptoKeyPubCT)
		if err != nil {
			return err
		}

		err = putMasterKeyParams(amBucket, newPubParams, nil)
		if err != nil {
			return err
		}

		err = putCryptoKeys(amBucket, newCryptoKeyPubEnc, nil, nil)
		if err != nil {
			return err
		}

		addrManager.masterKeyPub = newMasterKeyPub
	}
	km.pubPassphrase = newPubPass
	return nil
}

func (km *KeystoreManager) ChangeRemark(dbTransaction db.DBTransaction, accountID, newRemark string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	addrManager, found := km.managedKeystores[accountID]
	if found {
		err := addrManager.changeRemark(dbTransaction, newRemark)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to change remark", logging.LogFormat{"error": err})
			return err
		}
		return nil
	} else {
		logging.CPrint(logging.ERROR, "account not exists",
			logging.LogFormat{
				"err": ErrAccountNotFound,
			})
		return ErrAccountNotFound
	}
}

func (km *KeystoreManager) ClearPrivKey() {
	km.mu.Lock()
	defer km.mu.Unlock()

	for _, addrManager := range km.managedKeystores {
		addrManager.clearPrivKeys()
	}
}

func (km *KeystoreManager) ListKeystoreNames() []string {
	km.mu.Lock()
	defer km.mu.Unlock()
	list := make([]string, 0)
	for id := range km.managedKeystores {
		list = append(list, id)
	}

	return list
}

func (km *KeystoreManager) ChainParams() *config.Params {
	km.mu.Lock()
	defer km.mu.Unlock()
	return km.params
}

func (km *KeystoreManager) CurrentKeystore() *AddrManager {
	km.mu.Lock()
	defer km.mu.Unlock()
	if km.currentKeystore == nil {
		return nil
	}
	return km.managedKeystores[km.currentKeystore.accountName]
}

func (km *KeystoreManager) GetMnemonic(dbTransaction db.ReadTransaction, accountID string, privpass []byte) (string, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	addrManager, ok := km.managedKeystores[accountID]
	if !ok {
		return "", ErrAccountNotFound
	}
	err := addrManager.checkPassword(privpass)
	if err != nil {
		return "", err
	}
	defer addrManager.masterKeyPriv.Zero()

	mnemonic, err := addrManager.getMnemonic(dbTransaction, privpass)
	if err != nil {
		return "", err
	}

	return mnemonic, nil
}

func (km *KeystoreManager) getAddrManager(addr string) (*AddrManager, error) {
	for acctID, addrManager := range km.managedKeystores {
		_, ok := addrManager.addrs[addr]
		if ok {
			return km.managedKeystores[acctID], nil
		}
	}

	return nil, ErrAccountNotFound
}

func (km *KeystoreManager) GetAddrManager(addr string) (*AddrManager, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	return km.getAddrManager(addr)
}

// get AddrManager by given acctID
func (km *KeystoreManager) getAddrManagerByAccountID(acctID string) (*AddrManager, error) {
	addrManager, ok := km.managedKeystores[acctID]
	if !ok {
		return nil, ErrAccountNotFound
	}
	return addrManager, nil
}

func (km *KeystoreManager) GetAddrManagerByAccountID(acctID string) (*AddrManager, error) {
	km.mu.Lock()
	defer km.mu.Unlock()
	return km.getAddrManagerByAccountID(acctID)
}

func (km *KeystoreManager) GetManagedAddrManager() []*AddrManager {
	ret := make([]*AddrManager, 0)
	for _, v := range km.managedKeystores {
		ret = append(ret, v)
	}
	return ret
}

func (km *KeystoreManager) GetAddrs(accountId string) ([]string, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	addrManager, ok := km.managedKeystores[accountId]
	if !ok {
		return nil, ErrAccountNotFound
	}

	return addrManager.ListAddresses(), nil
}

func (km *KeystoreManager) GetManagedAddressByScriptHash(scriptHash []byte) (*ManagedAddress, error) {
	km.mu.Lock()
	defer km.mu.Unlock()
	// create common address, because common address as index
	scriptHashStruct, err := massutil.NewAddressWitnessScriptHash(scriptHash, km.params)
	if err != nil {
		return nil, err
	}
	witAddress := scriptHashStruct.EncodeAddress()

	for _, addrManager := range km.managedKeystores {
		mAddr, ok := addrManager.addrs[witAddress]
		if ok {
			return mAddr, nil
		}
	}

	return nil, ErrScriptHashNotFound
}

func (km *KeystoreManager) GetManagedAddressByStdAddress(encoded string) (*ManagedAddress, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	for _, addrManager := range km.managedKeystores {
		mAddr, ok := addrManager.addrs[encoded]
		if ok {
			return mAddr, nil
		}
	}

	return nil, ErrAddressNotFound
}

func (km *KeystoreManager) GetManagedAddressByScriptHashInCurrent(scriptHash []byte) (*ManagedAddress, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	if km.currentKeystore == nil {
		return nil, ErrCurrentKeystoreNotFound
	}

	// create common address, because common address as index
	scriptHashStruct, err := massutil.NewAddressWitnessScriptHash(scriptHash, km.params)
	if err != nil {
		return nil, err
	}
	encoded := scriptHashStruct.EncodeAddress()

	mAddr, ok := km.managedKeystores[km.currentKeystore.accountName].addrs[encoded]
	if ok {
		return mAddr, nil
	}

	return nil, ErrAddressNotFound
}

func NewKeystoreManager(rootBucket db.Bucket, pubPassphrase []byte, net *config.Params) (*KeystoreManager, error) {
	if rootBucket == nil || net == nil || len(pubPassphrase) == 0 {
		return nil, ErrNilPointer
	}

	kmBucket, err := db.GetOrCreateBucket(rootBucket, ksMgrBucket)
	if err != nil {
		return nil, err
	}

	managed := make(map[string]*AddrManager)
	kmBucketMeta := kmBucket.GetBucketMeta()

	// Load the account from db
	accountIDBucket, err := db.GetOrCreateBucket(kmBucket, accountIDBucket)
	accountIDBucketMeta := accountIDBucket.GetBucketMeta()

	byteAccountIDs, err := fetchAccountID(accountIDBucket)
	if err != nil {
		return nil, err
	}
	for _, accountIDbyte := range byteAccountIDs {
		accountID := string(accountIDbyte)
		amBucket := kmBucket.Bucket(accountID)
		if err != nil {
			return nil, err
		}

		addrManager, err := loadAddrManager(amBucket, pubPassphrase, net)
		if err != nil {
			return nil, err
		}
		managed[accountID] = addrManager
	}
	return &KeystoreManager{
		managedKeystores: managed,
		params:           net,
		ksMgrMeta:        kmBucketMeta,
		accountIDMeta:    accountIDBucketMeta,
		pubPassphrase:    pubPassphrase,
	}, nil
}
