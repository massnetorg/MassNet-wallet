package api

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"

	"google.golang.org/grpc/status"

	"github.com/massnetorg/mass-core/blockchain"
	"github.com/massnetorg/mass-core/consensus"
	"github.com/massnetorg/mass-core/logging"
	"github.com/massnetorg/mass-core/massutil"
	"github.com/massnetorg/mass-core/massutil/safetype"
	"github.com/massnetorg/mass-core/txscript"
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/errors"
	"massnet.org/mass-wallet/masswallet"
	"massnet.org/mass-wallet/masswallet/keystore"
)

const (
	LenWalletId   = 42
	AddressMaxLen = 100
	LenTxId       = 64
	LenPassMax    = 40
	LenPassMin    = 6
	LenRemarksMax = 20
	// estimated value
	LenMnemonicMax = 256
	LenMnemonicMin = 38
)

var (
	apiUnknownError = status.New(ErrAPIUnknownErr, ErrCode[ErrAPIUnknownErr]).Err()
)

// AmountToString converts m(in Maxwell) to the string representation(float, in Mass)
func AmountToString(m int64) (string, error) {
	if m > massutil.MaxAmount().IntValue() {
		return "", fmt.Errorf("amount is out of range: %d", m)
	}
	u, err := safetype.NewUint128FromInt(m)
	if err != nil {
		return "", err
	}
	u, err = u.AddUint(consensus.MaxwellPerMass)
	if err != nil {
		return "", err
	}
	s := u.String()
	sInt, sFrac := s[:len(s)-8], s[len(s)-8:]
	sFrac = strings.TrimRight(sFrac, "0")
	i, err := strconv.Atoi(sInt)
	if err != nil {
		return "", err
	}
	sInt = strconv.Itoa(i - 1)
	if len(sFrac) > 0 {
		return sInt + "." + sFrac, nil
	}
	return sInt, nil
}

// StringToAmount converts s(float, in Mass) to amount(in Maxwell)
func StringToAmount(s string) (massutil.Amount, error) {
	s1 := strings.Split(s, ".")
	if len(s1) > 2 {
		return massutil.ZeroAmount(), fmt.Errorf("illegal number format")
	}
	var sInt, sFrac string
	// preproccess integral part
	sInt = strings.TrimLeft(s1[0], "0")
	if len(sInt) == 0 {
		sInt = "0"
	}
	// preproccess fractional part
	if len(s1) == 2 {
		sFrac = strings.TrimRight(s1[1], "0")
		if len(sFrac) > 8 {
			return massutil.ZeroAmount(), fmt.Errorf("precision is too high")
		}
	}
	sFrac += strings.Repeat("0", 8-len(sFrac))

	// convert
	i, err := strconv.ParseInt(sInt, 10, 64)
	if err != nil {
		return massutil.ZeroAmount(), err
	}
	if i < 0 || uint64(i) > consensus.MaxMass {
		return massutil.ZeroAmount(), fmt.Errorf("integral part is out of range")
	}

	f, err := strconv.ParseInt(sFrac, 10, 64)
	if err != nil {
		return massutil.ZeroAmount(), err
	}
	if f < 0 {
		return massutil.ZeroAmount(), fmt.Errorf("illegal number format")
	}

	u := safetype.NewUint128FromUint(consensus.MaxwellPerMass)
	u, err = u.MulInt(i)
	if err != nil {
		return massutil.ZeroAmount(), err
	}
	u, err = u.AddInt(f)
	if err != nil {
		return massutil.ZeroAmount(), err
	}
	total, err := massutil.NewAmount(u)
	if err != nil {
		return massutil.ZeroAmount(), err
	}
	return total, nil
}

func checkLocktime(locktime uint64) error {
	if locktime > math.MaxInt64 {
		logging.CPrint(logging.ERROR, "Invalid lockTime", logging.LogFormat{
			"lockTime": locktime,
		})
		return status.New(ErrAPIInvalidLockTime, ErrCode[ErrAPIInvalidLockTime]).Err()
	}
	return nil
}

func checkNotEmpty(param interface{}) error {
	if isEmpty(param) {
		logging.CPrint(logging.ERROR, "empty param", logging.LogFormat{})
		st := status.New(ErrAPIInvalidParameter, ErrCode[ErrAPIInvalidParameter])
		return st.Err()
	}
	return nil
}

func isEmpty(obj interface{}) bool {
	if obj == nil {
		return true
	}
	objValue := reflect.ValueOf(obj)
	switch objValue.Kind() {
	case reflect.Array,
		reflect.Slice,
		reflect.Map,
		reflect.Chan:
		return objValue.Len() == 0
	case reflect.Ptr:
		if objValue.IsNil() {
			return true
		}
		return isEmpty(objValue.Elem().Interface())
	default:
		zero := reflect.Zero(objValue.Type())
		return reflect.DeepEqual(obj, zero.Interface())
	}
}

func checkParseAmount(s string) (massutil.Amount, error) {
	val, err := StringToAmount(s)
	if err != nil {
		logging.CPrint(logging.ERROR, "Invalid amount string", logging.LogFormat{
			"str": s,
			"err": err,
		})
		st := status.New(ErrAPIInvalidAmount, ErrCode[ErrAPIInvalidAmount])
		return massutil.ZeroAmount(), st.Err()
	}
	return val, nil
}

func checkFormatAmount(amt massutil.Amount) (string, error) {
	s, err := AmountToString(amt.IntValue())
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to format amount", logging.LogFormat{
			"err": err,
		})
		return "", status.New(ErrAPIInvalidAmount, ErrCode[ErrAPIInvalidAmount]).Err()
	}
	return s, nil
}

func checkWitnessAddress(address string, expectStaking bool, net *config.Params) (massutil.Address, error) {
	addr, err := massutil.DecodeAddress(address, net)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to decode address", logging.LogFormat{
			"address": addr,
			"err":     err,
		})
		st := status.New(ErrAPIInvalidAddress, ErrCode[ErrAPIInvalidAddress])
		return nil, st.Err()
	}
	witAddr, ok := addr.(*massutil.AddressWitnessScriptHash)
	if !ok || witAddr.WitnessVersion() != 0 ||
		(expectStaking && witAddr.WitnessExtendVersion() != 1) ||
		(!expectStaking && witAddr.WitnessExtendVersion() != 0) {
		st := status.New(ErrAPIInvalidAddress, ErrCode[ErrAPIInvalidAddress])
		return nil, st.Err()
	}
	return witAddr, nil
}

func parseBindingTarget(address string, net *config.Params) (massutil.Address, error) {
	target, err := massutil.DecodeAddress(address, net)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to decode binding target", logging.LogFormat{
			"address": address,
			"err":     err,
		})
		st := status.New(ErrAPIInvalidAddress, ErrCode[ErrAPIInvalidAddress])
		return nil, st.Err()
	}
	if !massutil.IsValidBindingTarget(target) {
		st := status.New(ErrAPIInvalidAddress, ErrCode[ErrAPIInvalidAddress])
		return nil, st.Err()
	}
	return target, nil
}

func checkTxFeeLimit(cfg *config.Config, fee massutil.Amount) error {
	max, err := checkParseAmount(cfg.Wallet.Settings.MaxTxFee)
	if err != nil {
		logging.CPrint(logging.WARN, "invalid max_tx_fee", logging.LogFormat{
			"err": err,
		})
		max, _ = checkParseAmount(config.DefaultMaxTxFee)
	}
	if max.Cmp(fee) < 0 {
		logging.CPrint(logging.ERROR, "big transaction fee", logging.LogFormat{
			"fee": fee,
			"max": max,
		})
		st := status.New(ErrAPIBigTransactionFee, ErrCode[ErrAPIBigTransactionFee])
		return st.Err()
	}
	return nil
}

func convertResponseError(err error) error {
	if err == nil {
		logging.CPrint(logging.ERROR, "nil err", logging.LogFormat{})
		return nil
	}
	switch err {
	case masswallet.ErrNoWalletInUse,
		keystore.ErrCurrentKeystoreNotFound:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPINoWalletInUse], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPINoWalletInUse, ErrCode[ErrAPINoWalletInUse]).Err()
	case masswallet.ErrNoAddressInWallet:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPINoAddressInWallet], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPINoAddressInWallet, ErrCode[ErrAPINoAddressInWallet]).Err()
	case masswallet.ErrInsufficientFunds:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInsufficientWalletBalance], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIInsufficientWalletBalance, ErrCode[ErrAPIInsufficientWalletBalance]).Err()
	case masswallet.ErrNotEnoughInputs:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPINotEnoughInputs], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPINotEnoughInputs, ErrCode[ErrAPINotEnoughInputs]).Err()
	case masswallet.ErrOverfullUtxo:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIOverfullInputs], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIOverfullInputs, ErrCode[ErrAPIOverfullInputs]).Err()
	case masswallet.ErrInvalidFlag:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidFlag], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIInvalidFlag, ErrCode[ErrAPIInvalidFlag]).Err()
	case keystore.ErrUnexpectedPubKeyToSign,
		masswallet.ErrUTXONotExists:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIOutputNotExist], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIOutputNotExist, ErrCode[ErrAPIOutputNotExist]).Err()
	case masswallet.ErrSignWitnessTx,
		keystore.ErrBuildWitnessScript:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPISignRawTx], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPISignRawTx, ErrCode[ErrAPISignRawTx]).Err()
	case masswallet.ErrWalletUnready:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIWalletUnready], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIWalletUnready, ErrCode[ErrAPIWalletUnready]).Err()
	case masswallet.ErrInvalidAmount:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidAmount], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIInvalidAmount, ErrCode[ErrAPIInvalidAmount]).Err()
	case masswallet.ErrDustChange:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIDustChange], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIDustChange, ErrCode[ErrAPIDustChange]).Err()
	case masswallet.ErrDustAmount:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIDustAmount], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIDustAmount, ErrCode[ErrAPIDustAmount]).Err()
	case masswallet.ErrUnknownSubfeefrom:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIUnknownSubfeefrom], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIUnknownSubfeefrom, ErrCode[ErrAPIUnknownSubfeefrom]).Err()
	case masswallet.ErrShaHashFromStr:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidTxId], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIInvalidTxId, ErrCode[ErrAPIInvalidTxId]).Err()
	case masswallet.ErrInvalidParameter,
		txscript.ErrFrozenPeriod,
		blockchain.ErrInvalidStakingTxValue:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidParameter], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIInvalidParameter, ErrCode[ErrAPIInvalidParameter]).Err()
	case keystore.ErrInvalidPassphrase:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidPassphrase], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIInvalidPassphrase, ErrCode[ErrAPIInvalidPassphrase]).Err()
	case keystore.ErrAddressNotFound,
		keystore.ErrAccountNotFound,
		keystore.ErrAddressVersion,
		masswallet.ErrFailedDecodeAddress,
		masswallet.ErrInvalidAddress,
		masswallet.ErrInvalidStakingAddress,
		masswallet.ErrCreatePkScript:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidAddress], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIInvalidAddress, ErrCode[ErrAPIInvalidAddress]).Err()
	case keystore.ErrIllegalPassphrase,
		keystore.ErrSamePrivpass:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidNewPassphrase], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIInvalidNewPassphrase, ErrCode[ErrAPIInvalidNewPassphrase]).Err()
	case keystore.ErrInvalidKeystoreJson:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidKeystoreJson], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIInvalidKeystoreJson, ErrCode[ErrAPIInvalidKeystoreJson]).Err()
	case keystore.ErrKeystoreVersion:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidKeystoreVersion], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIInvalidKeystoreVersion, ErrCode[ErrAPIInvalidKeystoreVersion]).Err()
	case keystore.ErrChangePassNotAllowed:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIChangePassUnsupported], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIChangePassUnsupported, ErrCode[ErrAPIChangePassUnsupported]).Err()
	case keystore.ErrBadTimingForChangingPass:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIWalletUnlocked], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIWalletUnlocked, ErrCode[ErrAPIWalletUnlocked]).Err()
	case keystore.ErrCoinType:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIMismatchedKeystoreJson], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIMismatchedKeystoreJson, ErrCode[ErrAPIMismatchedKeystoreJson]).Err()
	case keystore.ErrAccountType:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIMismatchedKeystoreJson], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIMismatchedKeystoreJson, ErrCode[ErrAPIMismatchedKeystoreJson]).Err()
	case keystore.ErrDuplicateSeed:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIDuplicateSeed], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIDuplicateSeed, ErrCode[ErrAPIDuplicateSeed]).Err()
	case keystore.ErrIllegalNewPrivPass:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIPrivPassSameAsPubPass], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIPrivPassSameAsPubPass, ErrCode[ErrAPIPrivPassSameAsPubPass]).Err()
	case keystore.ErrIllegalSeed:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidSeed], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIInvalidSeed, ErrCode[ErrAPIInvalidSeed]).Err()
	case keystore.ErrInvalidMnemonic,
		keystore.ErrChecksumIncorrect,
		keystore.ErrInvalidMnemonicWord:
		logging.CPrint(logging.ERROR, "Mnemonic error", logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIInvalidMnemonic, ErrCode[ErrAPIInvalidMnemonic]).Err()
	case keystore.ErrEntropyLengthInvalid:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidBitSize], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIInvalidBitSize, ErrCode[ErrAPIInvalidBitSize]).Err()
	case keystore.ErrGapLimit:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIGapLimit], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIGapLimit, ErrCode[ErrAPIGapLimit]).Err()
	case errors.ErrTxAlreadyExists:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPITxAlreadyExists], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPITxAlreadyExists, ErrCode[ErrAPITxAlreadyExists]).Err()
	case blockchain.ErrNonStandardTxSize,
		blockchain.ErrTxMsgPayloadSize,
		blockchain.ErrTxTooBig:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPINonStandardTxSize], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPINonStandardTxSize, ErrCode[ErrAPINonStandardTxSize]).Err()
	case blockchain.ErrImmatureSpend,
		blockchain.ErrSequenceNotSatisfied:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIUnspendable], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIUnspendable, ErrCode[ErrAPIUnspendable]).Err()
	case blockchain.ErrInsufficientFee:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIUserTxFee], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIUserTxFee, ErrCode[ErrAPIUserTxFee]).Err()
	case blockchain.ErrDoubleSpend,
		masswallet.ErrDoubleSpend:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIDoubleSpend], logging.LogFormat{
			"err": err,
		})
		return status.New(ErrAPIDoubleSpend, ErrCode[ErrAPIDoubleSpend]).Err()
	default:
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIUnknownErr], logging.LogFormat{
			"err": err,
		})
		return apiUnknownError
	}
}

func checkAddressLen(addr string) error {
	if len(addr) == 0 || len(addr) > AddressMaxLen {
		logging.CPrint(logging.ERROR, "illegal address length",
			logging.LogFormat{
				"address": addr,
			})
		st := status.New(ErrAPIInvalidAddress, ErrCode[ErrAPIInvalidAddress])
		return st.Err()
	}
	return nil
}

func checkWalletIdLen(walletId string) error {
	if len(walletId) != LenWalletId {
		logging.CPrint(logging.ERROR, "The length of the wallet id is incorrect", logging.LogFormat{
			"wallet_id":       walletId,
			"length":          len(walletId),
			"expected length": LenWalletId,
		})
		st := status.New(ErrAPIInvalidWalletId, ErrCode[ErrAPIInvalidWalletId])
		return st.Err()
	}
	return nil
}

func checkTransactionIdLen(txId string) error {
	if len(txId) != LenTxId {
		logging.CPrint(logging.ERROR, "", logging.LogFormat{
			"tx_id":           txId,
			"length":          len(txId),
			"expected length": LenTxId,
		})
		st := status.New(ErrAPIInvalidTxId, ErrCode[ErrAPIInvalidTxId])
		return st.Err()
	}
	return nil
}

func checkMnemonicLen(mnemonic string) error {
	if len(mnemonic) > LenMnemonicMax || len(mnemonic) < LenMnemonicMin {
		logging.CPrint(logging.ERROR, "The length of the pass is out of range", logging.LogFormat{
			"mnemonic":        mnemonic,
			"length":          len(mnemonic),
			"allowable range": fmt.Sprintf("[%v, %v]", LenMnemonicMin, LenMnemonicMax),
		})
		st := status.New(ErrAPIInvalidMnemonic, ErrCode[ErrAPIInvalidMnemonic])
		return st.Err()
	}
	return nil
}

func checkPassLen(pass string) error {
	if len(pass) > LenPassMax || len(pass) < LenPassMin {
		logging.CPrint(logging.ERROR, "The length of the pass is out of range", logging.LogFormat{
			"length":          len(pass),
			"allowable range": fmt.Sprintf("[%v, %v]", LenPassMin, LenPassMax),
		})
		st := status.New(ErrAPIInvalidPassphrase, ErrCode[ErrAPIInvalidPassphrase])
		return st.Err()
	}
	return nil
}

func checkRemarksLen(remarks string) string {
	r := []rune(remarks)
	if len(r) > LenRemarksMax {
		return string(r[:LenRemarksMax])
	}
	return remarks
}

func extractAddressInfos(pkScript []byte) (scriptClass txscript.ScriptClass, recipient, staking, binding string, reqSigs int, err error) {
	scriptClass, addrs, _, reqSigs, err := txscript.ExtractPkScriptAddrs(pkScript, config.ChainParams)
	if err != nil {
		return 0, "", "", "", 0, err
	}
	if len(addrs) == 0 {
		return 0, "", "", "", 0, fmt.Errorf("no address parsed from output script")
	}

	switch scriptClass {
	case txscript.StakingScriptHashTy:
		std, err := massutil.NewAddressWitnessScriptHash(addrs[0].ScriptAddress(), config.ChainParams)
		if err != nil {
			return 0, "", "", "", 0, err
		}
		recipient = std.EncodeAddress()
		staking = addrs[0].EncodeAddress()
	case txscript.BindingScriptHashTy:
		targetType := "MASS"
		targetSize := 0
		if len(addrs[1].ScriptAddress()) == 22 {
			if addrs[1].ScriptAddress()[20] == 1 {
				targetType = "Chia"
			}
			targetSize = int(addrs[1].ScriptAddress()[21])
		}
		binding = fmt.Sprintf("%s:%s:%d", addrs[1].EncodeAddress(), targetType, targetSize)
		recipient = addrs[0].EncodeAddress()
	case txscript.WitnessV0ScriptHashTy:
		recipient = addrs[0].EncodeAddress()
	}
	return
}
