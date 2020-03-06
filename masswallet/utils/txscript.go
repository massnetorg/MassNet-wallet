package utils

import (
	"errors"

	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/txscript"
)

var (
	ErrUnsupportedScript = errors.New("unsupported script")
)

type PkScript interface {
	Maturity() uint64
	AddressClass() uint16
	StdScriptAddress() []byte
	StdEncodeAddress() string
	SecondScriptAddress() []byte
	SecondEncodeAddress() string
	IsStaking() bool
	IsBinding() bool
	ScriptClass() txscript.ScriptClass
}

type pkScriptInfo struct {
	maturity uint64

	stdAddress    massutil.Address
	stdScriptAddr []byte
	stdEncodeAddr string

	secondAddress    massutil.Address
	secondScriptAddr []byte
	secondEncodeAddr string

	addressClass uint16
	scriptClass  txscript.ScriptClass
}

// Maturity maturity of staking = frozen period + 1
func (s *pkScriptInfo) Maturity() uint64 {
	return s.maturity
}

func (s *pkScriptInfo) AddressClass() uint16 {
	return s.addressClass
}

func (s *pkScriptInfo) IsStaking() bool {
	return s.scriptClass == txscript.StakingScriptHashTy
}

func (s *pkScriptInfo) IsBinding() bool {
	return s.scriptClass == txscript.BindingScriptHashTy
}

// StdScriptAddress:
// returns withdraw address when scriptclass is staking/binding
func (s *pkScriptInfo) StdScriptAddress() []byte {
	if s.stdScriptAddr == nil {
		s.stdScriptAddr = s.stdAddress.ScriptAddress()
	}
	return s.stdScriptAddr
}

// StdEncodeAddress:
// returns withdraw address when scriptclass is staking/binding
func (s *pkScriptInfo) StdEncodeAddress() string {
	if s.stdEncodeAddr == "" {
		s.stdEncodeAddr = s.stdAddress.EncodeAddress()
	}
	return s.stdEncodeAddr
}

// SecondScriptAddress:
// 1. returns staking script address when scriptclass is staking
// 2. returns binding script address when scriptcalss is binding
// 3. panic(no meaning for standard script)
func (s *pkScriptInfo) SecondScriptAddress() []byte {
	if s.secondScriptAddr == nil {
		s.secondScriptAddr = s.secondAddress.ScriptAddress()
	}
	return s.secondScriptAddr
}

// SecondEncodeAddress:
// 1. returns staking encode address when scriptclass is staking
// 2. returns binding encode address when scriptcalss is binding
// 3. panic(no meaning for standard script)
func (s *pkScriptInfo) SecondEncodeAddress() string {
	if s.secondEncodeAddr == "" {
		s.secondEncodeAddr = s.secondAddress.EncodeAddress()
	}
	return s.secondEncodeAddr
}

func (s *pkScriptInfo) ScriptClass() txscript.ScriptClass {
	return s.scriptClass
}

func ParsePkScript(pkScript []byte, chainParams *config.Params) (PkScript, error) {

	scriptClass, pops := txscript.GetScriptInfo(pkScript)
	height, scriptHash, err := txscript.GetParsedOpcode(pops, scriptClass)
	if err != nil {
		return nil, err
	}

	ret := &pkScriptInfo{
		addressClass: massutil.AddressClassWitnessV0,
		scriptClass:  scriptClass,
	}

	switch scriptClass {
	case txscript.WitnessV0ScriptHashTy:
		ret.stdAddress, err = massutil.NewAddressWitnessScriptHash(scriptHash[:], chainParams)
	case txscript.StakingScriptHashTy:
		ret.stdAddress, err = massutil.NewAddressWitnessScriptHash(scriptHash[:], chainParams)
		if err != nil {
			return nil, err
		}
		ret.secondAddress, err = massutil.NewAddressStakingScriptHash(scriptHash[:], chainParams)
		ret.addressClass = massutil.AddressClassWitnessStaking
		ret.maturity = height + 1
	case txscript.BindingScriptHashTy:
		var s1, s2 []byte
		s1, s2, err = txscript.GetParsedBindingOpcode(pops)
		if err != nil {
			return nil, err
		}
		ret.stdAddress, err = massutil.NewAddressWitnessScriptHash(s1, chainParams)
		if err != nil {
			return nil, err
		}
		ret.secondAddress, err = massutil.NewAddressPubKeyHash(s2, chainParams)
	default:
		return nil, ErrUnsupportedScript
	}
	return ret, nil
}
