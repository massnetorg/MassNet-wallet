package keystore

import "math"

type KeystoreVersion uint8

func (k KeystoreVersion) Value() uint8 {
	return uint8(k)
}

const (
	// KeystoreVersion0
	// generates seed with non-empty passphrase and passphrase is immutable
	KeystoreVersion0 KeystoreVersion = iota

	KeystoreVersionLatest = KeystoreVersion0

	KeystoreVersionInvalid = KeystoreVersion(math.MaxUint8)
)

type WalletParams struct {
	Version           KeystoreVersion
	Mnemonic          string
	Remarks           string
	PrivatePassphrase []byte
	ExternalIndex     uint32
	InternalIndex     uint32
	AddressGapLimit   uint32
}
