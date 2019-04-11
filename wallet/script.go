package wallet

import (
	"errors"

	"github.com/massnetorg/MassNet-wallet/btcec"
	"github.com/massnetorg/MassNet-wallet/config"
	"github.com/massnetorg/MassNet-wallet/database"
	"github.com/massnetorg/MassNet-wallet/logging"
	"github.com/massnetorg/MassNet-wallet/massutil"
	"github.com/massnetorg/MassNet-wallet/txscript"
	"github.com/massnetorg/MassNet-wallet/wire"
)

func NewWitnessProgram(pubkeys []*btcec.PublicKey, nrequired int) (string, error) {
	//verify nil pointer,avoid panic error
	if pubkeys == nil {
		logging.CPrint(logging.ERROR, "input pointer is nil",
			logging.LogFormat{
				"err": ErrNilPointer,
			})
		return "", ErrNilPointer
	}
	//check n
	if nrequired <= 0 {
		err := errors.New("nrequired can not be <= 0")
		logging.CPrint(logging.ERROR, "nrequired can not be <= 0",
			logging.LogFormat{
				"err": err,
			})
		return "", err
	}

	var addressPubKeyStructs []*massutil.AddressPubKey
	for i := 0; i < len(pubkeys); i++ {
		pubKeySerial := pubkeys[i].SerializeCompressed()
		addressPubKeyStruct, err := massutil.NewAddressPubKey(pubKeySerial, Params)
		if err != nil {
			logging.CPrint(logging.ERROR, "create addressPubKey failed",
				logging.LogFormat{
					"err": err,
				})
			return "", err
		}
		addressPubKeyStructs = append(addressPubKeyStructs, addressPubKeyStruct)

	}
	redeemScript, err := txscript.MultiSigScript(addressPubKeyStructs, nrequired)
	if err != nil {
		logging.CPrint(logging.ERROR, "create redeemScript failed",
			logging.LogFormat{
				"err": err,
			})
		return "", err
	}

	scriptHashH := wire.HashH(redeemScript)

	scriptString := scriptHashH.String()

	return scriptString, nil

}

// non-persisitent
func NewNonPersistentWitSAddr(pubkeys []*btcec.PublicKey, nrequired int) (string, error) {

	//verify nil pointer,avoid panic error
	if pubkeys == nil {
		logging.CPrint(logging.ERROR, "input pointer is nil",
			logging.LogFormat{
				"err": ErrNilPointer,
			})
		return "", ErrNilPointer
	}
	// check n
	if nrequired <= 0 {
		err := errors.New("nrequired can not be <= 0")
		logging.CPrint(logging.ERROR, "nrequired can not be <= 0",
			logging.LogFormat{
				"err": err,
			})
		return "", err
	}
	_, witAddress, err := newWitnessScriptAddress(pubkeys, nrequired, 0, Params)
	if err != nil {
		logging.CPrint(logging.ERROR, "create witnessScriptAddress failed",
			logging.LogFormat{
				"err": err,
			})
		return "", err
	}

	return witAddress, nil

}

func newWitnessScriptAddress(pubkeys []*btcec.PublicKey, nrequired int, version int, net *config.Params) ([]byte, string, error) {

	var addressPubKeyStructs []*massutil.AddressPubKey
	for i := 0; i < len(pubkeys); i++ {
		pubKeySerial := pubkeys[i].SerializeCompressed()
		addressPubKeyStruct, err := massutil.NewAddressPubKey(pubKeySerial, net)
		if err != nil {
			logging.CPrint(logging.ERROR, "create addressPubKey failed",
				logging.LogFormat{
					"err": err,
				})
			return nil, "", err
		}
		addressPubKeyStructs = append(addressPubKeyStructs, addressPubKeyStruct)
	}

	//
	redeemScript, err := txscript.MultiSigScript(addressPubKeyStructs, nrequired)
	if err != nil {
		logging.CPrint(logging.ERROR, "create redeemScript failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, "", err
	}

	// scriptHash=witnessProgram
	scriptHash := massutil.Hash160(redeemScript)
	switch version {
	case 10:
		scriptHashStruct, err := massutil.NewAddressLocktimeScriptHash(scriptHash, net)
		if err != nil {
			logging.CPrint(logging.ERROR, "create addressLocktimeScriptHash failed",
				logging.LogFormat{
					"err": err,
				})
			return nil, "", err
		}
		//witness wallet
		witAddress := scriptHashStruct.EncodeAddress()
		return redeemScript, witAddress, nil
	default:
		scriptHashStruct, err := massutil.NewAddressWitnessScriptHash(scriptHash, net)
		if err != nil {
			logging.CPrint(logging.ERROR, "create addressLocktimeScriptHash failed",
				logging.LogFormat{
					"err": err,
				})
			return nil, "", err
		}
		//witness wallet bech32
		witAddress := scriptHashStruct.EncodeAddress()
		return redeemScript, witAddress, nil
	}
}

func getWitnessMap(db database.Db) (map[string][]byte, error) {
	witnessMap, err := db.FetchWitnessAddrToRedeem()
	if err != nil {
		return nil, err
	}

	return witnessMap, nil
}
