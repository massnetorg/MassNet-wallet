package keystore

import (
	"crypto/sha256"
	"errors"

	"github.com/btcsuite/btcd/btcec"
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/pocec"
	"massnet.org/mass-wallet/txscript"
)

func NewPoCAddress(pubKey *pocec.PublicKey, net *config.Params) ([]byte, massutil.Address, error) {

	//verify nil pointer,avoid panic error
	if pubKey == nil {
		return nil, nil, ErrNilPointer
	}
	scriptHash, pocAddress, err := newPoCAddress(pubKey, net)
	if err != nil {
		logging.CPrint(logging.ERROR, "newPoCAddress failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, nil, err
	}

	return scriptHash, pocAddress, nil
}

func NewNonPersistentWitSAddrForBtcec(pubkeys []*btcec.PublicKey, nrequired int,
	addressClass uint16, net *config.Params) ([]byte, massutil.Address, error) {

	//verify nil pointer,avoid panic error
	if len(pubkeys) == 0 {
		logging.CPrint(logging.ERROR, "input pointer is nil",
			logging.LogFormat{
				"err": ErrNilPointer,
			})
		return nil, nil, ErrNilPointer
	}

	if nrequired <= 0 {
		err := errors.New("nrequired can not be <= 0")
		logging.CPrint(logging.ERROR, "nrequired can not be <= 0",
			logging.LogFormat{
				"err": err,
			})
		return nil, nil, err
	}
	redeemScript, witAddress, err := newWitnessScriptAddressForBtcec(pubkeys, nrequired, addressClass, net)
	if err != nil {
		logging.CPrint(logging.ERROR, "create witnessScriptAddress failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, nil, err
	}

	return redeemScript, witAddress, nil
}

func newPoCAddress(pubKey *pocec.PublicKey, net *config.Params) ([]byte, massutil.Address, error) {

	scriptHash := massutil.Hash160(pubKey.SerializeCompressed())

	addressPubKeyHash, err := massutil.NewAddressPubKeyHash(scriptHash, net)
	if err != nil {
		return nil, nil, err
	}

	return scriptHash, addressPubKeyHash, nil
}

func newWitnessScriptAddressForBtcec(pubkeys []*btcec.PublicKey, nrequired int,
	addressClass uint16, net *config.Params) ([]byte, massutil.Address, error) {

	var addressPubKeyStructs []*massutil.AddressPubKey
	for i := 0; i < len(pubkeys); i++ {
		pubKeySerial := pubkeys[i].SerializeCompressed()
		addressPubKeyStruct, err := massutil.NewAddressPubKey(pubKeySerial, net)
		if err != nil {
			logging.CPrint(logging.ERROR, "create addressPubKey failed",
				logging.LogFormat{
					"err":       err,
					"version":   addressClass,
					"nrequired": nrequired,
				})
			return nil, nil, ErrBuildWitnessScript
		}
		addressPubKeyStructs = append(addressPubKeyStructs, addressPubKeyStruct)
	}

	redeemScript, err := txscript.MultiSigScript(addressPubKeyStructs, nrequired)
	if err != nil {
		logging.CPrint(logging.ERROR, "create redeemScript failed",
			logging.LogFormat{
				"err":       err,
				"version":   addressClass,
				"nrequired": nrequired,
			})
		return nil, nil, ErrBuildWitnessScript
	}
	var witAddress massutil.Address
	scriptHash := sha256.Sum256(redeemScript)
	switch addressClass {
	case massutil.AddressClassWitnessStaking:
		witAddress, err = massutil.NewAddressStakingScriptHash(scriptHash[:], net)
	case massutil.AddressClassWitnessV0:
		witAddress, err = massutil.NewAddressWitnessScriptHash(scriptHash[:], net)
	default:
		return nil, nil, ErrAddressVersion
	}

	if err != nil {
		logging.CPrint(logging.ERROR, "create witness address failed",
			logging.LogFormat{
				"err":       err,
				"version":   addressClass,
				"nrequired": nrequired,
			})
		return nil, nil, ErrBuildWitnessScript
	}

	return redeemScript, witAddress, nil
}
