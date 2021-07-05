package masswallet

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/massnetorg/mass-core/blockchain"
	"github.com/massnetorg/mass-core/consensus"
	"github.com/massnetorg/mass-core/consensus/forks"
	"github.com/massnetorg/mass-core/logging"
	"github.com/massnetorg/mass-core/massutil"
	"github.com/massnetorg/mass-core/massutil/safetype"
	"github.com/massnetorg/mass-core/txscript"
	"github.com/massnetorg/mass-core/wire"

	"massnet.org/mass-wallet/config"
	mwdb "massnet.org/mass-wallet/masswallet/db"
	"massnet.org/mass-wallet/masswallet/txmgr"
	"massnet.org/mass-wallet/masswallet/utils"
)

// for current wallet
func (w *WalletManager) existsMsgTx(out *wire.OutPoint) (mtx *wire.MsgTx, meta *txmgr.BlockMeta, err error) {
	err = mwdb.View(w.db, func(tx mwdb.ReadTransaction) error {
		mtx, meta, err = w.txStore.ExistsTx(tx, out)
		return err
	})
	return
}

func (w *WalletManager) existsUnminedTx(hash *wire.Hash) (mtx *wire.MsgTx, err error) {
	err = mwdb.View(w.db, func(tx mwdb.ReadTransaction) error {
		mtx, err = w.txStore.ExistUnminedTx(tx, hash)
		return err
	})
	return
}

// for current wallet
func (w *WalletManager) existsOutPoint(out *wire.OutPoint) (utxoFlags *txmgr.UtxoFlags, err error) {
	err = mwdb.View(w.db, func(tx mwdb.ReadTransaction) error {
		utxoFlags, err = w.txStore.ExistsUtxo(tx, out)
		return err
	})
	return
}

func (w *WalletManager) addTxIn(msgTx *wire.MsgTx, LockTime uint64, inputUtxos []*txmgr.Credit) error {
	for _, utx := range inputUtxos {
		txIn := wire.NewTxIn(&utx.OutPoint, nil)
		if LockTime != 0 {
			txIn.Sequence = wire.MaxTxInSequenceNum - 1
		}

		prevTx, block, err := w.existsMsgTx(&txIn.PreviousOutPoint)
		if err != nil {
			return err
		}

		txOut := prevTx.TxOut[txIn.PreviousOutPoint.Index]
		pks, err := utils.ParsePkScript(txOut.PkScript, w.chainParams)
		if err != nil {
			logging.CPrint(logging.ERROR, "ParsePkScript error",
				logging.LogFormat{
					"err": err,
				})
			return err
		}
		switch {
		case pks.IsStaking():
			txIn.Sequence = pks.Maturity()
		case pks.IsBinding() && forks.EnforceMASSIP0002WarmUp(block.Height):
			txIn.Sequence = consensus.MASSIP0002BindingLockedPeriod
		default:
		}
		msgTx.AddTxIn(txIn)
	}
	return nil
}

func (w *WalletManager) autoConstructTxInAndChangeTxOut(msgTx *wire.MsgTx, LockTime uint64,
	addrs []string, userTxFee massutil.Amount, changeAddr string) (fee massutil.Amount, err error) {

	targetTxFee := massutil.MinRelayTxFee()
	if !userTxFee.IsZero() {
		targetTxFee = userTxFee
	}

	outAmounts := massutil.ZeroAmount()
	// construct output
	for _, txout := range msgTx.TxOut {
		outAmounts, err = outAmounts.AddInt(txout.Value)
		if err != nil {
			logging.CPrint(logging.ERROR, "output amount error", logging.LogFormat{"err": err})
			return outAmounts, ErrInvalidAmount
		}
	}

	// construct input and change output
	for {
		var (
			adj       = massutil.ZeroAmount()
			changeOut *wire.TxOut
			txOutLen  int
			utxos     []*txmgr.Credit
		)
		for {
			want, err := targetTxFee.Add(outAmounts)
			if err != nil {
				return outAmounts, err
			}

			// ensure no dust change output
			wantAdj, err := want.Add(adj)
			if err != nil {
				return outAmounts, err
			}

			var (
				firstAddr string
				found     massutil.Amount
				overfull  bool
			)

			utxos, firstAddr, found, overfull, err = w.findEligibleUtxos(wantAdj, addrs)
			if err != nil {
				return outAmounts, err
			}

			if found.Cmp(wantAdj) < 0 {
				logging.CPrint(logging.WARN, "no enough mass to pay",
					logging.LogFormat{
						"found":    found,
						"want":     wantAdj,
						"overfull": overfull,
					})
				if overfull {
					return outAmounts, ErrOverfullUtxo
				}
				return outAmounts, ErrInsufficientFunds
			}

			txOutLen = len(msgTx.TxOut)
			// construct change output
			changeAmount, _ := found.Sub(want)
			if !changeAmount.IsZero() {
				if changeAmount.Cmp(massutil.MinRelayTxFee()) < 0 {
					adj = massutil.MinRelayTxFee()
					continue
				}
				if len(changeAddr) > 0 {
					changeOut, err = amountToTxOut(changeAddr, changeAmount)
				} else {
					changeOut, err = amountToTxOut(firstAddr, changeAmount)
				}
				if err != nil {
					return outAmounts, err
				}
				txOutLen++
			}
			break
		}

		// calc acutal fee
		signedTxSize, err := w.estimateSignedSize(utxos, txOutLen)
		if err != nil {
			logging.CPrint(logging.ERROR, "estimate signedSize failed", logging.LogFormat{"err": err})
			return outAmounts, ErrInvalidParameter
		}
		signedTxSize += int64(len(msgTx.Payload))
		requiredFee, err := blockchain.CalcMinRequiredTxRelayFee(signedTxSize, massutil.MinRelayTxFee())
		if err != nil {
			return outAmounts, err
		}
		if targetTxFee.Cmp(requiredFee) >= 0 {
			err = w.addTxIn(msgTx, LockTime, utxos)
			if err != nil {
				return outAmounts, err
			}
			if changeOut != nil {
				msgTx.AddTxOut(changeOut)
			}
			return targetTxFee, nil
		}

		msgTx.TxIn = make([]*wire.TxIn, 0)
		targetTxFee = requiredFee
	}
}

func (w *WalletManager) prepareFromAddresses(from string) (addrs []string, err error) {
	ks := w.ksmgr.CurrentKeystore()
	if ks == nil {
		return nil, ErrNoWalletInUse
	}

	if len(from) > 0 {
		ma, err := ks.Address(from)
		if err != nil {
			logging.CPrint(logging.ERROR, "address not found",
				logging.LogFormat{
					"from": from,
					"err":  err,
				})
			return nil, err
		}
		addrs = append(addrs, ma.String())
	} else {
		addrs = ks.ListAddresses()
	}

	if len(addrs) == 0 {
		return nil, ErrNoAddressInWallet
	}
	return
}

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

// Calculates new amounts after subtracting fee from amounts.
// If selectedAddresses not specified, nothing will be changed, means that the sender pays the fee.
func maybeSubtractFeeFromAmounts(
	amounts map[string]massutil.Amount,
	selectedAddresses map[string]struct{},
	requiredFee massutil.Amount,
) (
	newAmounts map[string]massutil.Amount,
	totalAndFee massutil.Amount,
	err error,
) {

	for addr := range selectedAddresses {
		if _, ok := amounts[addr]; !ok {
			return nil, massutil.ZeroAmount(), ErrUnknownSubfeefrom
		}
	}

	totalAndFee = massutil.ZeroAmount()

	// sender pays the fee
	numOfSelected := int64(len(selectedAddresses))
	if numOfSelected == 0 {
		totalAndFee, err = totalAndFee.Add(requiredFee)
		if err != nil {
			return nil, massutil.ZeroAmount(), err
		}
		for _, amount := range amounts {
			totalAndFee, err = totalAndFee.Add(amount)
			if err != nil {
				return nil, massutil.ZeroAmount(), err
			}
		}
		return amounts, totalAndFee, nil
	}

	// subtract fee from amounts
	newAmounts = make(map[string]massutil.Amount)
	eachSub := (requiredFee.IntValue() + numOfSelected - 1) / numOfSelected
	eachSubAmt, err := massutil.NewAmountFromInt(eachSub)
	if err != nil {
		return nil, massutil.ZeroAmount(), err
	}

	actualFee, err := massutil.NewAmountFromInt(eachSub * numOfSelected)
	if err != nil {
		return nil, massutil.ZeroAmount(), err
	}
	totalAndFee, err = totalAndFee.Add(actualFee)
	if err != nil {
		return nil, massutil.ZeroAmount(), err
	}
	for addr, amount := range amounts {
		newAmount := amount
		if _, ok := selectedAddresses[addr]; ok {
			newAmount, err = amount.Sub(eachSubAmt)
			if err != nil {
				return nil, massutil.ZeroAmount(), err
			}
		}
		newAmounts[addr] = newAmount
		totalAndFee, err = totalAndFee.Add(newAmount)
		if err != nil {
			return nil, massutil.ZeroAmount(), err
		}
	}
	return newAmounts, totalAndFee, nil
}

// PayToWitnessV0Address builds TxOut script after performing
// some validity checks.
func PayToWitnessV0Address(encodedAddr string, netParams *config.Params) (pkScript []byte, err error) {
	// Decode the provided wallet.
	addr, err := massutil.DecodeAddress(encodedAddr, netParams)
	if err != nil {
		logging.CPrint(logging.WARN, "Failed to decode address", logging.LogFormat{
			"err":     err,
			"address": encodedAddr,
		})
		return nil, ErrFailedDecodeAddress
	}

	// Ensure the wallet is one of the supported types and that
	// the network encoded with the wallet matches the network the
	// server is currently on.
	if !massutil.IsWitnessV0Address(addr) {
		logging.CPrint(logging.WARN, "Invalid witness address", logging.LogFormat{"address": encodedAddr})
		return nil, ErrInvalidAddress
	}
	if !addr.IsForNet(netParams) {
		return nil, ErrNet
	}

	// create a new script which pays to the provided wallet.
	pkScript, err = txscript.PayToAddrScript(addr)
	if err != nil {
		logging.CPrint(logging.WARN, "Failed to create pkScript", logging.LogFormat{
			"err":     err,
			"address": encodedAddr,
		})
		return nil, ErrCreatePkScript
	}
	return pkScript, nil
}

func amountToTxOut(encodedAddr string, amount massutil.Amount) (*wire.TxOut, error) {
	if amount.IsZero() {
		logging.CPrint(logging.ERROR, "invalid output amount",
			logging.LogFormat{
				"err": ErrInvalidAmount,
			})
		return nil, ErrInvalidAmount
	}

	pkScript, err := PayToWitnessV0Address(encodedAddr, config.ChainParams)
	if err != nil {
		return nil, err
	}
	return wire.NewTxOut(amount.IntValue(), pkScript), nil
}
