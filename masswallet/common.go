package masswallet

import (
	"fmt"
	"strconv"
	"strings"

	"massnet.org/mass-wallet/consensus"
	"massnet.org/mass-wallet/massutil/safetype"

	"massnet.org/mass-wallet/blockchain"

	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	mwdb "massnet.org/mass-wallet/masswallet/db"
	"massnet.org/mass-wallet/masswallet/txmgr"
	"massnet.org/mass-wallet/masswallet/utils"
	"massnet.org/mass-wallet/wire"
)

const (
	defaultChangeAddress = ""
)

// for current wallet
func (w *WalletManager) existsMsgTx(out *wire.OutPoint) (mtx *wire.MsgTx, err error) {
	err = mwdb.View(w.db, func(tx mwdb.ReadTransaction) error {
		mtx, err = w.txStore.ExistsTx(tx, out)
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

		prevTx, err := w.existsMsgTx(&txIn.PreviousOutPoint)
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
		if pks.IsStaking() {
			txIn.Sequence = pks.Maturity()
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
				return outAmounts, ErrInsufficient
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
