package keystore

import (
	"github.com/btcsuite/btcd/btcec"
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/masswallet/keystore/hdkeychain"
)

type ManagedAddress struct {
	pubKey         *btcec.PublicKey
	privKey        *btcec.PrivateKey
	scriptHash     []byte
	derivationPath DerivationPath
	address        string
	stakingAddress string
	keystoreName   string
}

func newManagedAddressWithoutPrivKey(keystoreName string, derivationPath DerivationPath,
	pubKey *btcec.PublicKey, nRequired int, addressClass uint16, net *config.Params) (*ManagedAddress, error) {
	var pubkeys []*btcec.PublicKey
	pubkeys = append(pubkeys, pubKey)
	_, witAddress, err := newWitnessScriptAddressForBtcec(pubkeys, nRequired, addressClass, net)
	if err != nil {
		logging.CPrint(logging.ERROR, "newWitnessScriptAddressForBtcec error",
			logging.LogFormat{
				"error": err,
			})
		return nil, err
	}
	switch addressClass {
	case massutil.AddressClassWitnessV0:
		return &ManagedAddress{
			address:        witAddress.EncodeAddress(),
			derivationPath: derivationPath,
			pubKey:         pubKey,
			keystoreName:   keystoreName,
			scriptHash:     witAddress.ScriptAddress(),
		}, nil
	case massutil.AddressClassWitnessStaking:
		witV0Addr, err := massutil.NewAddressWitnessScriptHash(witAddress.ScriptAddress(), net)
		if err != nil {
			logging.CPrint(logging.ERROR, "NewAddressWitnessScriptHash error",
				logging.LogFormat{
					"error": err,
				})
			return nil, err
		}
		return &ManagedAddress{
			address:        witV0Addr.EncodeAddress(),
			stakingAddress: witAddress.EncodeAddress(),
			derivationPath: derivationPath,
			pubKey:         pubKey,
			keystoreName:   keystoreName,
			scriptHash:     witV0Addr.ScriptAddress(),
		}, nil
	default:
		return nil, ErrAddressVersion
	}

}

func newManagedAddress(keystoreName string, derivationPath DerivationPath, privKey *btcec.PrivateKey, nRquired int, addressClass uint16, net *config.Params) (*ManagedAddress, error) {
	ecPubKey := (*btcec.PublicKey)(&privKey.PublicKey)
	managedAddress, err := newManagedAddressWithoutPrivKey(keystoreName, derivationPath, ecPubKey, nRquired, addressClass, net)
	if err != nil {
		logging.CPrint(logging.ERROR, "create address failed", logging.LogFormat{"error": err})
		return nil, err
	}
	managedAddress.privKey = privKey
	return managedAddress, nil
}

func newManagedAddressFromExtKey(keystoreName string, derivationPath DerivationPath,
	extKey *hdkeychain.ExtendedKey, nRequired int, addressClass uint16, net *config.Params) (*ManagedAddress, error) {
	// create a new managed address based on the public or private key
	// depending on whether the generated key is private.
	var managedAddr *ManagedAddress
	if extKey.IsPrivate() {
		privKey, err := extKey.ECPrivKey()
		if err != nil {
			return nil, err
		}

		// Ensure the temp private key big integer is cleared after
		// use.
		managedAddr, err = newManagedAddress(
			keystoreName, derivationPath, privKey, nRequired, addressClass, net,
		)
		if err != nil {
			return nil, err
		}
	} else {
		pubKey, err := extKey.ECPubKey()
		if err != nil {
			return nil, err
		}

		managedAddr, err = newManagedAddressWithoutPrivKey(
			keystoreName, derivationPath, pubKey, nRequired, addressClass, net,
		)
		if err != nil {
			return nil, err
		}
	}
	return managedAddr, nil
}

func (mAddr *ManagedAddress) Account() string {
	return mAddr.keystoreName
}

func (mAddr *ManagedAddress) IsChangeAddr() bool {
	return mAddr.derivationPath.Branch == InternalBranch
}

// return address
func (mAddr *ManagedAddress) String() string {
	return mAddr.address
}

func (mAddr *ManagedAddress) StakingAddress() string {
	return mAddr.stakingAddress
}

// scriptHash of 1-1 witness
func (mAddr *ManagedAddress) ScriptAddress() []byte {
	return mAddr.scriptHash
}

func (mAddr *ManagedAddress) PubKey() *btcec.PublicKey {
	return mAddr.pubKey
}

func (mAddr *ManagedAddress) PrivKey() *btcec.PrivateKey {
	return mAddr.privKey
}

func (mAddr *ManagedAddress) RedeemScript(chainParams *config.Params) ([]byte, error) {
	var pubkeys []*btcec.PublicKey
	pubkeys = append(pubkeys, mAddr.PubKey())
	script, _, err := NewNonPersistentWitSAddrForBtcec(pubkeys, nRequiredDefault, massutil.AddressClassWitnessV0, chainParams)
	if err != nil {
		return nil, err
	}
	return script, nil
}
