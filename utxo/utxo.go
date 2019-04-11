package utxo

import (
	"container/list"
	"sort"

	"massnet.org/mass-wallet/blockchain"
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/database"
	"massnet.org/mass-wallet/errors"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/txscript"
)

var (
	Params           = &config.ChainParams
	minconfig        = int32(blockchain.TransactionMaturity)
	coinbaseMaturity = int32(blockchain.CoinbaseMaturity)
)

type Utxo struct {
	DB database.Db
}

func NewUtxo(db database.Db) (*Utxo, error) {
	if db == nil {
		err := errors.New("input parameter is empty")
		logging.CPrint(logging.ERROR, "input pointer is nil",
			logging.LogFormat{
				"err": err,
			})
		return nil, err
	}
	utxoStrcut := Utxo{
		DB: db,
	}
	return &utxoStrcut, nil
}

func (u *Utxo) GetUtxos(witnessAddr []string) ([]*database.UtxoListReply, map[string][]*database.UtxoListReply, error) {
	if witnessAddr == nil {
		err := errors.New("inputParams can not be nil")
		logging.CPrint(logging.ERROR, "inputParams can not be nil",
			logging.LogFormat{
				"err": err,
			})
		return nil, nil, err
	}

	var (
		utxosListReturn = make([]*database.UtxoListReply, 0)
		utxosMapReturn  = make(map[string][]*database.UtxoListReply)

		witAddrs []massutil.Address

		lockList = make([]*database.UtxoListReply, 0)
	)
	for _, v := range witnessAddr {
		kAddr, err := massutil.DecodeAddress(v, Params)
		if err != nil {
			logging.CPrint(logging.ERROR, "decode address failed",
				logging.LogFormat{
					"err": err,
				})
			return nil, nil, err
		}
		witAddrs = append(witAddrs, kAddr)
	}

	utxolist, err := u.DB.FetchUtxosForAddrs(witAddrs, Params)
	if err != nil {
		logging.CPrint(logging.ERROR, "fetch db failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, nil, err
	}

	_, bestHeight, err := u.DB.NewestSha()
	if err != nil {
		logging.CPrint(logging.ERROR, "get bestHeight",
			logging.LogFormat{
				"err": err,
			})
		return nil, nil, err
	}
	for addrIndex, utxos := range utxolist {
		uts := make([]*database.UtxoListReply, 0)
		//	utxo := make([]*database.UtxoListReply, 0)

		switch addr := witAddrs[addrIndex].(type) {
		case *massutil.AddressWitnessScriptHash:
			if addr.WitnessVersion() == 0 {
				for _, v := range utxos {
					if !v.Coinbase {
						if confirmed(minconfig, v.Height, bestHeight) {
							uts = append(uts, v)
							utxosListReturn = append(utxosListReturn, v)
						}
					} else {
						if confirmed(coinbaseMaturity, v.Height, bestHeight) {
							uts = append(uts, v)
							utxosListReturn = append(utxosListReturn, v)
						}
					}
				}
			} else if addr.WitnessVersion() == 10 {
				for _, entry := range utxos {
					txList, err := u.DB.FetchTxBySha(entry.TxSha)
					if err != nil || len(txList) == 0 {
						logging.CPrint(logging.ERROR, "No information available about transaction", logging.LogFormat{"txid": entry.TxSha.String(), "index": entry.Index})
						return nil, nil, err
					}
					lastTx := txList[len(txList)-1]
					mtx := lastTx.Tx
					class, pops := txscript.GetScriptInfo(mtx.TxOut[entry.Index].PkScript)
					height, _, err := txscript.GetParsedOpcode(pops, class)
					if err != nil {
						return nil, nil, err
					}
					if int64(height)+int64(entry.Height) <= int64(bestHeight) {
						if !entry.Coinbase {
							if confirmed(minconfig, entry.Height, bestHeight) {
								uts = append(uts, entry)
								utxosListReturn = append(utxosListReturn, entry)
							}
						} else {
							if confirmed(coinbaseMaturity, entry.Height, bestHeight) {
								uts = append(uts, entry)
								utxosListReturn = append(utxosListReturn, entry)
							}
						}
					} else {
						lockList = append(lockList, entry)
					}
				}

			}
		default:
			return nil, nil, errors.New("invalid address type")
		}

		utxosMapReturn[witnessAddr[addrIndex]] = uts
	}

	return utxosListReturn, utxosMapReturn, nil
}

func (u *Utxo) GetBalance(witnessAddr []string) (massutil.Amount, map[string]massutil.Amount, error) {
	if witnessAddr == nil {
		err := errors.New("inputParams can not be nil")
		logging.CPrint(logging.ERROR, "inputParams can not be nil",
			logging.LogFormat{
				"err": err,
			})
		return 0, nil, err
	}

	var (
		addrToBal = make(map[string]massutil.Amount)
		totalBal  massutil.Amount
	)

	_, addrToUtxos, err := u.GetUtxos(witnessAddr)
	if err != nil {
		logging.CPrint(logging.ERROR, "get utxos failed",
			logging.LogFormat{
				"err": err,
			})
		return 0, nil, err
	}

	for addr, utxos := range addrToUtxos {
		var allAmount massutil.Amount
		for _, utxo := range utxos {
			allAmount += utxo.Value
		}
		addrToBal[addr] = allAmount
		totalBal += allAmount

	}

	return totalBal, addrToBal, nil

}

func (u *Utxo) FindEligibleUtxos(amount massutil.Amount, witnessAddr []string) ([]*database.UtxoListReply, massutil.Amount, massutil.Amount, error) {
	if witnessAddr == nil {
		err := errors.New("inputParams can not be nil")
		logging.CPrint(logging.ERROR, "inputParams can not be nil",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, 0, err
	}
	if amount < 0 {
		err := errors.New("amount can not < 0")
		logging.CPrint(logging.ERROR, "amount can not < 0",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, 0, err
	}

	utxos, _, err := u.GetUtxos(witnessAddr)
	if err != nil {
		logging.CPrint(logging.ERROR, "get utxos failed",
			logging.LogFormat{
				"err": err,
			})
		return nil, 0, 0, err
	}

	utxosReturn, utBal, totalBal := optOutputs(amount, utxos)

	return utxosReturn, utBal, totalBal, nil
}

func optOutputs(amount massutil.Amount, utxos []*database.UtxoListReply) ([]*database.UtxoListReply, massutil.Amount, massutil.Amount) {
	var (
		optAmount, totalAmount massutil.Amount
		uts                    = make([]*database.UtxoListReply, 0)
		utsReturn              = make([]*database.UtxoListReply, 0)
	)

	sort.Slice(utxos, func(i, j int) bool {
		return utxos[i].Value > utxos[j].Value
	})
	utxoList := list.New()
	for _, u0 := range utxos {
		utxoList.PushBack(u0)
		totalAmount += u0.Value
		if u0.Value >= amount {
			uts = append(uts, u0)
		}
	}

	if len(uts) == 0 {

		for node := utxoList.Back(); node != nil; node = node.Prev() {
			if optAmount < amount {
				u1 := node.Value.(*database.UtxoListReply)
				utsReturn = append(utsReturn, u1)
				optAmount += u1.Value
				continue
			}
			break
		}
		return utsReturn, optAmount, totalAmount
	}

	if len(uts) == len(utxos) {
		lastUt := utxos[len(utxos)-1]
		utsReturn = append(utsReturn, lastUt)
		return utsReturn, lastUt.Value, totalAmount
	}

	index0 := len(uts)
	for i := index0; i < len(utxos); i++ {
		optAmount += utxos[i].Value
	}
	if optAmount < amount {
		utsReturn = append(utsReturn, utxos[index0-1])
		return utsReturn, utxos[index0-1].Value, totalAmount
	}

	utsReturn = append(utsReturn, utxos[index0])
	optAmount = utxos[index0].Value

	for node := utxoList.Back(); node != nil; node = node.Prev() {
		if optAmount < amount {
			utsReturn = append(utsReturn, node.Value.(*database.UtxoListReply))
			optAmount += node.Value.(*database.UtxoListReply).Value
			continue
		}
		break
	}

	return utsReturn, optAmount, totalAmount
}

func confirmed(minconf, txHeight, curHeight int32) bool {
	return confirms(txHeight, curHeight) >= minconf
}

// confirms returns the number of confirmations for a transaction in a block at
// height txHeight (or -1 for an unconfirmed tx) given the chain height
// curHeight.
func confirms(txHeight, curHeight int32) int32 {
	switch {
	case txHeight == -1, txHeight > curHeight:
		return 0
	default:
		return curHeight - txHeight + 1
	}
}
