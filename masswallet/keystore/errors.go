package keystore

import "massnet.org/mass-wallet/errors"

var (
	ErrKMPubKeyNotSet = errors.New("required KeystoreManager public key parameters not stored in database")

	ErrIllegalPassphrase  = errors.New("illegal passphrase")
	ErrIllegalNewPrivPass = errors.New("new private passphrase same as public passphrase")
	ErrIllegalNewPubPass  = errors.New("new public passphrase same as private passphrase")
	ErrSamePubpass        = errors.New("new public passphrase same as the original one")
	ErrIllegalSeed        = errors.New("illegal seed")
	ErrIllegalRemarks     = errors.New("illegal remarks")

	ErrUnexpectError = errors.New("unexpected error")

	ErrGapLimit                      = errors.New("too many unused address")
	ErrExceedAllowedNumberPerAccount = errors.New("exceed the maximum allowed number of addresses per account")

	ErrAddressNotFound         = errors.New("address not found")
	ErrAccountNotFound         = errors.New("account not found")
	ErrCurrentKeystoreNotFound = errors.New("current keystore not found")
	ErrUnexpecteDBError        = errors.New("unexpected error occurred in DB")
	ErrKeyScopeNotFound        = errors.New("KeyScope definition not found")
	ErrScriptHashNotFound      = errors.New("scriptHash not found")
	ErrPubKeyNotFound          = errors.New("pubKey not found")
	ErrNilPointer              = errors.New("the pointer is nil")
	ErrBucketNotFound          = errors.New("bucket not found")
	ErrInvalidPassphrase       = errors.New("invalid passphrase for master private key")
	ErrDeriveMasterPrivKey     = errors.New("failed to derive master private key")
	ErrCoinType                = errors.New("invalid coinType")
	ErrAccountType             = errors.New("invalid accountType")

	ErrNoKeystoreActivated = errors.New("no keystore activated")
	ErrDuplicateSeed       = errors.New("duplicate seed in the wallet")

	ErrAddressVersion      = errors.New("unexpected address version")
	ErrInvalidKeystoreJson = errors.New("invalid keystore json")
	ErrInvalidDataHash     = errors.New("invalid hash, length is not 32")

	ErrUnexpectedPubKeyToSign = errors.New("unexpected pubkey to sign")
	ErrBuildWitnessScript     = errors.New("failed to build witness script/address")
)
