package masswallet

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	mwdb "massnet.org/mass-wallet/masswallet/db"
	"massnet.org/mass-wallet/masswallet/ifc"
	"massnet.org/mass-wallet/masswallet/keystore"
	"massnet.org/mass-wallet/masswallet/txmgr"
	"massnet.org/mass-wallet/txscript"
	"massnet.org/mass-wallet/wire"

	cache "github.com/patrickmn/go-cache"
)

const (
	keystoreBucket = "k"
	utxoBucket     = "u"
	txBucket       = "t"
	syncBucket     = "s"
)

type WalletManager struct {
	config       *config.Config
	db           mwdb.DB //walletdb
	chainParams  *config.Params
	chainFetcher ifc.ChainFetcher
	ksmgr        *keystore.KeystoreManager

	bucketMeta *txmgr.StoreBucketMeta
	utxoStore  *txmgr.UtxoStore
	txStore    *txmgr.TxStore
	syncStore  *txmgr.SyncStore

	ntfnsHandler *NtfnsHandler

	server Server

	usedCache *cache.Cache

	mu sync.RWMutex
	wg sync.WaitGroup
}

func NewWalletManager(server Server, db mwdb.DB, config *config.Config,
	chainParams *config.Params, pubpass string) (*WalletManager, error) {
	if db == nil {
		logging.CPrint(logging.ERROR, "db is nil", logging.LogFormat{
			"err": ErrNilDB,
		})
		return nil, ErrNilDB
	}

	w := &WalletManager{
		config:       config,
		db:           db,
		chainParams:  chainParams,
		chainFetcher: ifc.NewChainFetcher(server.ChainDB()),
		bucketMeta:   &txmgr.StoreBucketMeta{},
		server:       server,
		usedCache:    cache.New(5*time.Minute, 10*time.Minute),
	}

	err := mwdb.Update(db, func(tx mwdb.DBTransaction) error {
		// init KeystoreManager
		bucket, err := mwdb.GetOrCreateTopLevelBucket(tx, keystoreBucket)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to get bucket", logging.LogFormat{
				"err": err,
			})
			return err
		}
		w.ksmgr, err = keystore.NewKeystoreManager(bucket, []byte(pubpass), chainParams)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to new keystore", logging.LogFormat{
				"err": err,
			})
			return err
		}

		// init UtxoStore
		bucket, err = mwdb.GetOrCreateTopLevelBucket(tx, utxoBucket)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to get bucket", logging.LogFormat{
				"err": err,
			})
			return err
		}
		w.utxoStore, err = txmgr.NewUtxoStore(bucket, w.ksmgr, w.bucketMeta, chainParams)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to new UtxoStore", logging.LogFormat{
				"err": err,
			})
			return err
		}

		// init SyncStore
		bucket, err = mwdb.GetOrCreateTopLevelBucket(tx, syncBucket)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to get bucket", logging.LogFormat{
				"err": err,
			})
			return err
		}
		w.syncStore, err = txmgr.NewSyncStore(bucket, w.bucketMeta, chainParams)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to new SyncStore", logging.LogFormat{
				"err": err,
			})
			return err
		}

		// init TxStore
		bucket, err = mwdb.GetOrCreateTopLevelBucket(tx, txBucket)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to get bucket", logging.LogFormat{
				"err": err,
			})
			return err
		}
		w.txStore, err = txmgr.NewTxStore(w.chainFetcher, bucket,
			w.utxoStore, w.syncStore, w.ksmgr, w.bucketMeta, chainParams)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to new TxStore", logging.LogFormat{
				"err": err,
			})
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// init NtfnsHandler
	h, err := NewNtfnsHandler(w)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to create NtfnsHandler", logging.LogFormat{
			"err": err,
		})
		return nil, err
	}

	w.ntfnsHandler = h

	err = checkInit(w)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to checkInit", logging.LogFormat{
			"err": err,
		})
		return nil, err
	}

	return w, nil
}

func checkInit(w *WalletManager) error {
	err := w.bucketMeta.CheckInit()
	if err != nil {
		return err
	}
	// ...
	return nil
}

func (w *WalletManager) CheckReady(name string) (bool, error) {
	var ws *txmgr.WalletStatus
	err := mwdb.View(w.db, func(dbtx mwdb.ReadTransaction) (err error) {
		ws, err = w.syncStore.GetWalletStatus(dbtx, name)
		return err
	})
	if err != nil {
		return false, err
	}
	return ws.Ready(), nil
}

func (w *WalletManager) UseWallet(name string) (*WalletInfo, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	ready, err := w.CheckReady(name)
	if err != nil {
		return nil, err
	}
	if !ready {
		return nil, ErrWalletUnready
	}

	err = w.ksmgr.UseKeystoreForWallet(name)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to useKeystore", logging.LogFormat{
			"err": err,
		})
		return nil, err
	}
	am := w.ksmgr.CurrentKeystore()
	if am == nil || am.Name() != name {
		logging.CPrint(logging.ERROR, "failed to change in-use wallet", logging.LogFormat{
			"err": ErrChangeInUseWallet,
		})
		return nil, ErrChangeInUseWallet
	}
	var balance massutil.Amount
	err = mwdb.View(w.db, func(tx mwdb.ReadTransaction) error {
		var err error
		balance, err = w.utxoStore.GrossBalance(tx, name)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to get gross balance", logging.LogFormat{
				"err":    err,
				"wallet": am.Name(),
			})
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	external, internal := am.CountAddresses()
	wi := &WalletInfo{
		ChainID:          w.ksmgr.ChainParams().ChainID.String(),
		WalletID:         am.Name(),
		Type:             uint32(am.AddrUse()),
		TotalBalance:     balance,
		ExternalKeyCount: int32(external),
		InternalKeyCount: int32(internal),
		Remarks:          am.Remarks(),
	}
	return wi, nil
}

func (w *WalletManager) Wallets() ([]*WalletSummary, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	ret := make([]*WalletSummary, 0)
	err := mwdb.View(w.db, func(dbtx mwdb.ReadTransaction) error {
		list, err := w.syncStore.GetAllWalletStatus(dbtx)
		if err != nil {
			return err
		}
		for _, status := range list {
			mgr, err := w.ksmgr.GetAddrManagerByAccountID(status.WalletID)
			if err != nil {
				return fmt.Errorf("%s: %v", status.WalletID, err)
			}
			summary := &WalletSummary{
				WalletID: mgr.Name(),
				Type:     uint32(mgr.AddrUse()),
				Remarks:  mgr.Remarks(),
				Ready:    status.Ready(),
			}
			if !summary.Ready {
				summary.SyncedHeight = status.SyncedHeight
			}
			ret = append(ret, summary)
		}
		return nil
	})
	return ret, err
}

func (w *WalletManager) CreateWallet(passphrase, remark string, bitSize int) (string, string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	var walletId, mnemonic string
	err := mwdb.Update(w.db, func(tx mwdb.DBTransaction) error {
		var err error
		walletId, mnemonic, err = w.ksmgr.NewKeystore(tx, bitSize, []byte(passphrase), remark, w.chainParams, &keystore.DefaultScryptOptions, w.config.Advanced.AddressGapLimit)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to new keystore", logging.LogFormat{
				"err": err,
			})
			return err
		}
		am, _ := w.ksmgr.GetAddrManagerByAccountID(walletId)
		if err = w.utxoStore.InitNewWallet(tx, am); err != nil {
			return err
		}

		return w.syncStore.PutWalletStatus(tx, &txmgr.WalletStatus{
			WalletID:     walletId,
			SyncedHeight: txmgr.WalletSyncedDone,
		})
	})
	if err != nil {
		w.ksmgr.RemoveCachedKeystore(walletId)
		return "", "", err
	}
	return walletId, mnemonic, nil
}

func (w *WalletManager) ImportWallet(keystoreJSON, pass string) (*WalletSummary, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.ntfnsHandler.exceedImportingLimit() {
		return nil, ErrExceedMaxImportingTask
	}
	var am *keystore.AddrManager
	var ws *txmgr.WalletStatus
	err := mwdb.Update(w.db, func(tx mwdb.DBTransaction) error {
		var err error
		am, err = w.ksmgr.ImportKeystore(tx, w.chainFetcher.CheckScriptHashUsed, []byte(keystoreJSON), []byte(pass), w.config.Advanced.AddressGapLimit)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to import keystore", logging.LogFormat{
				"err": err,
			})
			return err
		}
		if err = w.utxoStore.InitNewWallet(tx, am); err != nil {
			return err
		}
		ws = &txmgr.WalletStatus{
			WalletID: am.Name(),
		}
		addrs := am.ManagedAddresses()
		if len(addrs) == 0 {
			ws.SyncedHeight = txmgr.WalletSyncedDone
		}
		err = w.syncStore.PutWalletStatus(tx, ws)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to put wallet status", logging.LogFormat{
				"err": err,
			})
			return err
		}

		for _, managedAddr := range addrs {
			err = w.utxoStore.PutNewAddress(tx, am.Name(), managedAddr.String(), massutil.AddressClassWitnessV0)
			if err != nil {
				logging.CPrint(logging.ERROR, "failed to put new address", logging.LogFormat{
					"err": err,
				})
				return err
			}
		}
		return nil
	})
	if err != nil {
		if am != nil {
			w.ksmgr.RemoveCachedKeystore(am.Name())
		}
		return nil, err
	}

	if !ws.Ready() {
		w.ntfnsHandler.pushImportingTask(am)
	}
	return &WalletSummary{
		WalletID: am.Name(),
		Type:     uint32(am.AddrUse()),
		Remarks:  am.Remarks(),
	}, nil
}

func (w *WalletManager) ImportWalletWithMnemonic(mnemonic, pass, remark string, externalIndex, internalIndex uint32) (*WalletSummary, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.ntfnsHandler.exceedImportingLimit() {
		return nil, ErrExceedMaxImportingTask
	}
	var am *keystore.AddrManager
	var ws *txmgr.WalletStatus
	err := mwdb.Update(w.db, func(tx mwdb.DBTransaction) error {
		var err error
		am, err = w.ksmgr.ImportKeystoreWithMnemonic(tx, w.chainFetcher.CheckScriptHashUsed, mnemonic, remark, []byte(pass), externalIndex, internalIndex, w.config.Advanced.AddressGapLimit)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to import keystore", logging.LogFormat{
				"err": err,
			})
			return err
		}
		if err = w.utxoStore.InitNewWallet(tx, am); err != nil {
			return err
		}
		ws = &txmgr.WalletStatus{
			WalletID: am.Name(),
		}
		addrs := am.ManagedAddresses()
		if len(addrs) == 0 {
			ws.SyncedHeight = txmgr.WalletSyncedDone
		}
		err = w.syncStore.PutWalletStatus(tx, ws)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to put wallet status", logging.LogFormat{
				"err": err,
			})
			return err
		}

		for _, managedAddr := range addrs {
			err = w.utxoStore.PutNewAddress(tx, am.Name(), managedAddr.String(), massutil.AddressClassWitnessV0)
			if err != nil {
				logging.CPrint(logging.ERROR, "failed to put new address", logging.LogFormat{
					"err": err,
				})
				return err
			}
		}
		return nil
	})
	if err != nil {
		if am != nil {
			w.ksmgr.RemoveCachedKeystore(am.Name())
		}
		return nil, err
	}

	if !ws.Ready() {
		w.ntfnsHandler.pushImportingTask(am)
	}
	return &WalletSummary{
		WalletID: am.Name(),
		Type:     uint32(am.AddrUse()),
		Remarks:  am.Remarks(),
	}, nil
}

func (w *WalletManager) ExportWallet(name, pass string) (string, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var ret string
	err := mwdb.View(w.db, func(tx mwdb.ReadTransaction) error {
		buf, err := w.ksmgr.ExportKeystore(tx, name, []byte(pass))
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to export wallet", logging.LogFormat{
				"err": err,
			})
			return err
		}
		ret = string(buf)
		return nil
	})
	if err != nil {
		return "", err
	}

	return ret, err
}

func (w *WalletManager) RemoveWallet(name, pass string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	ready, err := w.CheckReady(name)
	if err != nil {
		return err
	}
	if !ready {
		return ErrWalletUnready
	}

	w.ntfnsHandler.suspend("start removing wallet", logging.LogFormat{"walletId": name})
	defer w.ntfnsHandler.resume("done removing wallet", logging.LogFormat{"walletId": name})

	var deletedHash []*wire.Hash
	err = mwdb.Update(w.db, func(tx mwdb.DBTransaction) error {
		am, err := w.ksmgr.GetAddrManagerByAccountID(name)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to get addrMananger", logging.LogFormat{
				"err": err,
			})
			return err
		}

		deletedHash, err = w.ntfnsHandler.onKeystoreRemove(tx, am)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to notice onKeystoreRemove", logging.LogFormat{
				"err": err,
			})
			return err
		}

		ok, err := w.ksmgr.DeleteKeystore(tx, name, []byte(pass))
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to delete keystore", logging.LogFormat{
				"err": err,
			})
			return err
		}
		if !ok {
			err := fmt.Errorf("failed to delete keystore %s", name)
			logging.CPrint(logging.ERROR, "failed to delete keystore", logging.LogFormat{
				"err": err,
			})
			return err
		}

		err = w.syncStore.DeleteWalletStatus(tx, name)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		mwdb.View(w.db, func(tx mwdb.ReadTransaction) error {
			w.ksmgr.UpdateManagedKeystores(tx, name)
			return nil
		})
		return err
	}
	w.ntfnsHandler.RemoveMempoolTx(deletedHash)
	return nil
}

func (w *WalletManager) GetMnemonic(name, pass string) (string, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var mnemonic string
	err := mwdb.View(w.db, func(tx mwdb.ReadTransaction) error {
		var err error
		mnemonic, err = w.ksmgr.GetMnemonic(tx, name, []byte(pass))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return mnemonic, nil
}

//WalletBalance returns total balance of current wallet
func (w *WalletManager) WalletBalance(confs uint32, queryDetail bool) (*WalletBalance, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	am := w.ksmgr.CurrentKeystore()
	if am == nil {
		logging.CPrint(logging.ERROR, "no wallet in use", logging.LogFormat{
			"err": ErrNoWalletInUse,
		})
		return nil, ErrNoWalletInUse
	}
	wb := &WalletBalance{
		WalletID:            am.Name(),
		Total:               massutil.ZeroAmount(),
		Spendable:           massutil.ZeroAmount(),
		WithdrawableBinding: massutil.ZeroAmount(),
		WithdrawableStaking: massutil.ZeroAmount(),
	}
	err := mwdb.View(w.db, func(tx mwdb.ReadTransaction) error {
		if queryDetail {
			syncedTo, err := w.syncStore.SyncedTo(tx)
			if err != nil {
				return err
			}
			bal, err := w.utxoStore.WalletBalance(tx, am, confs, syncedTo.Height)
			if err != nil {
				logging.VPrint(logging.ERROR, "failed to get wallet balance", logging.LogFormat{
					"err":    err,
					"wallet": am.Name(),
				})
				return err
			}
			wb.Total = bal.Total
			wb.Spendable = bal.Spendable
			wb.WithdrawableBinding = bal.WithdrawableBinding
			wb.WithdrawableStaking = bal.WithdrawableStaking
		}
		gross, err := w.utxoStore.GrossBalance(tx, am.Name())
		if err != nil {
			logging.VPrint(logging.ERROR, "failed to get gross balance", logging.LogFormat{
				"err":    err,
				"wallet": am.Name(),
			})
			return err
		}
		if queryDetail && gross.Cmp(wb.Total) != 0 {
			// possible
			logging.VPrint(logging.WARN, "unequal wallet balance", logging.LogFormat{
				"grossBalance": gross,
				"sumBalance":   wb.Total,
				"wallet":       am.Name(),
			})
		}
		wb.Total = gross
		return nil
	})

	if err != nil {
		return nil, err
	}
	return wb, nil
}

//AddressBalance if addrs is empty, balances of all addresses of current wallet will be returned
func (w *WalletManager) AddressBalance(confs uint32, addrs []string) ([]*AddressBalance, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	am := w.ksmgr.CurrentKeystore()
	if am == nil {
		logging.CPrint(logging.ERROR, "no wallet in use", logging.LogFormat{
			"err": ErrNoWalletInUse,
		})
		return nil, ErrNoWalletInUse
	}
	if len(addrs) == 0 {
		addrs = am.ListAddresses()
	}
	// mas := make([]*keystore.ManagedAddress, 0)
	scriptToAddrs := make(map[string][]string)
	scriptSet := make(map[string]struct{})
	for _, addr := range addrs {
		ma, err := am.Address(addr)
		if err != nil {
			return nil, err
		}

		addrs, ok := scriptToAddrs[string(ma.ScriptAddress())]
		if !ok {
			addrs = make([]string, 0)
		}
		scriptToAddrs[string(ma.ScriptAddress())] = append(addrs, addr)
		scriptSet[string(ma.ScriptAddress())] = struct{}{}
	}

	ret := make([]*AddressBalance, 0)
	if len(scriptSet) > 0 {
		err := mwdb.View(w.db, func(tx mwdb.ReadTransaction) error {
			syncedTo, err := w.syncStore.SyncedTo(tx)
			if err != nil {
				return err
			}
			m, err := w.utxoStore.ScriptAddressBalance(tx, scriptSet, confs, syncedTo.Height)
			if err != nil {
				logging.CPrint(logging.ERROR, "failed to get scriptAddress balance", logging.LogFormat{
					"err": err,
				})
				return err
			}
			for script, bal := range m {
				for _, addr := range scriptToAddrs[script] {
					ret = append(ret, &AddressBalance{
						Address:             addr,
						Total:               bal.Total,
						Spendable:           bal.Spendable,
						WithdrawableBinding: bal.WithdrawableBinding,
						WithdrawableStaking: bal.WithdrawableStaking,
					})
				}
			}
			return nil
		})

		if err != nil {
			return nil, err
		}
	}
	return ret, nil
}

func (w *WalletManager) GetUtxo(addrs []string) (map[string][]*UnspentDetail, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	am := w.ksmgr.CurrentKeystore()
	if am == nil {
		return nil, ErrNoWalletInUse
	}
	ret := make(map[string][]*UnspentDetail)
	if len(addrs) == 0 {
		addrs = am.ListAddresses()
	}
	creditsMap, _, err := w.getUtxos(addrs)
	if err != nil {
		logging.CPrint(logging.ERROR, "get utxos failed", logging.LogFormat{
			"err": err,
		})
		return nil, err
	}
	for addr, cres := range creditsMap {
		unspents := make([]*UnspentDetail, 0)
		for _, credit := range cres {
			unspents = append(unspents, &UnspentDetail{
				TxId:           credit.OutPoint.Hash.String(),
				Vout:           credit.OutPoint.Index,
				Amount:         credit.Amount,
				BlockHeight:    credit.BlockMeta.Height,
				Maturity:       credit.Maturity,
				Confirmations:  credit.Confirmations,
				SpentByUnmined: credit.Flags.SpentByUnmined,
			})
		}
		ret[addr] = unspents
	}

	return ret, nil
}

// TODO: only generate external address default
func (w *WalletManager) NewAddress(addrClass uint16) (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	am := w.ksmgr.CurrentKeystore()
	if am == nil {
		logging.CPrint(logging.ERROR, "no wallet in use", logging.LogFormat{
			"err": ErrNoWalletInUse,
		})
		return "", ErrNoWalletInUse
	}

	var address string
	err := mwdb.Update(w.db, func(tx mwdb.DBTransaction) error {
		mas, err := w.ksmgr.NextAddresses(tx, w.chainFetcher.CheckScriptHashUsed, false, 1, w.config.Advanced.AddressGapLimit, addrClass)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to nextAddress", logging.LogFormat{
				"err": err,
			})
			return err
		}
		switch addrClass {
		case massutil.AddressClassWitnessV0:
			address = mas[0].String()
		case massutil.AddressClassWitnessStaking:
			address = mas[0].StakingAddress()
		default:
			logging.CPrint(logging.ERROR, "unknown version", logging.LogFormat{
				"err": ErrInvalidVersion,
			})
			return ErrInvalidVersion
		}
		err = w.utxoStore.PutNewAddress(tx, mas[0].Account(), address, addrClass)
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to put new address in utxoStore", logging.LogFormat{
				"err": err,
			})
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return address, nil
}

// for testing purpose
func (w *WalletManager) GetAllAddressesWithPubkey() ([]*txmgr.AddressDetail, error) {
	list, err := w.GetAddresses(math.MaxUint16)
	if err != nil {
		return nil, err
	}

	m0 := make(map[string]*txmgr.AddressDetail)
	m1 := make(map[string]*txmgr.AddressDetail)
	for _, addr := range list {
		if addr.AddressClass == massutil.AddressClassWitnessStaking {
			m1[addr.StdAddress] = addr
		} else {
			m0[addr.Address] = addr
		}
	}

	for _, ma := range w.ksmgr.CurrentKeystore().ManagedAddresses() {
		if addr, ok := m0[ma.String()]; ok {
			addr.PubKey = ma.PubKey()
			continue
		}
		if addr, ok := m1[ma.String()]; ok {
			addr.PubKey = ma.PubKey()
		}
	}

	return list, nil
}

func (w *WalletManager) GetAddresses(addressClass uint16) ([]*txmgr.AddressDetail, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	acct := w.ksmgr.CurrentKeystore()
	if acct == nil {
		logging.CPrint(logging.ERROR, "no wallet in use", logging.LogFormat{
			"err": ErrNoWalletInUse,
		})
		return nil, ErrNoWalletInUse
	}

	var result, all []*txmgr.AddressDetail
	err := mwdb.View(w.db, func(tx mwdb.ReadTransaction) (err error) {
		all, err = w.utxoStore.GetAddresses(tx, acct.Name())
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to get address from utxoStore",
				logging.LogFormat{"err": err})
		}
		return err
	})
	if err != nil {
		return nil, err
	}

	mStd := make(map[string]*txmgr.AddressDetail)
	var stakingAddrs []*txmgr.AddressDetail

	for _, addr := range all {
		if addr.AddressClass == massutil.AddressClassWitnessStaking {
			address, err := massutil.DecodeAddress(addr.Address, w.chainParams)
			if err != nil {
				return nil, err
			}

			std, err := massutil.NewAddressWitnessScriptHash(address.ScriptAddress(), w.chainParams)
			if err != nil {
				return nil, err
			}
			addr.StdAddress = std.EncodeAddress()
			stakingAddrs = append(stakingAddrs, addr)
		} else {
			mStd[addr.Address] = addr
		}
	}

	// filter staking address
	for _, stakingAddr := range stakingAddrs {
		if stdAddr, ok := mStd[stakingAddr.StdAddress]; ok {
			if stakingAddr.Used {
				stdAddr.Used = stakingAddr.Used
				continue
			}
			if !stdAddr.Used {
				delete(mStd, stakingAddr.StdAddress)
			}
		} else {
			if stakingAddr.Used {
				mStd[stakingAddr.StdAddress] = &txmgr.AddressDetail{
					Address:      stakingAddr.StdAddress,
					AddressClass: massutil.AddressClassWitnessV0,
					Used:         stakingAddr.Used,
				}
			}
		}
	}

	switch addressClass {
	case massutil.AddressClassWitnessV0:
		for _, v := range mStd {
			result = append(result, v)
		}
	case massutil.AddressClassWitnessStaking:
		result = stakingAddrs
	case math.MaxUint16:
		for _, v := range mStd {
			result = append(result, v)
		}
		result = append(result, stakingAddrs...)
	default:
		// impossible
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].AddressClass != result[j].AddressClass {
			return result[i].AddressClass < result[j].AddressClass
		}
		return strings.Compare(result[i].Address, result[j].Address) < 0
	})

	return result, nil
}

func (w *WalletManager) CreateRawTransaction(Inputs []*TxIn, Outputs map[string]massutil.Amount, lockTime uint64) (string, error) {
	// Add all transaction inputs to a new transaction after performing
	// some validity checks.
	mtx, err := w.constructTxIn(Inputs, lockTime)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to constructTxIn", logging.LogFormat{
			"err": err,
		})
		return "", err
	}

	mtx, err = constructTxOut(Outputs, mtx)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to constructTxOut", logging.LogFormat{
			"err": err,
		})
		return "", err
	}
	// Set the Locktime, if given.
	mtx.LockTime = lockTime
	mtx.Version = wire.TxVersion

	// Return the serialized and hex-encoded transaction.  Note that this
	// is intentionally not directly returning because the first return
	// value is a string and it would result in returning an empty string to
	// the client instead of nothing (nil) in the case of an error.
	mtxHex, err := messageToHex(mtx)
	if err != nil {
		logging.CPrint(logging.ERROR, "Error in messageToHex(mtx)", logging.LogFormat{
			"err": err,
		})
		return "", err
	}
	w.MarkUsedUTXO(mtx)
	return mtxHex, nil
}

func (w *WalletManager) AutoCreateRawTransaction(Amounts map[string]massutil.Amount, LockTime uint64,
	userTxFee massutil.Amount, from string) (string, massutil.Amount, error) {

	w.mu.RLock()
	defer w.mu.RUnlock()

	mtx, txFee, err := w.EstimateTxFee(Amounts, LockTime, userTxFee, from)
	if err != nil {
		logging.CPrint(logging.ERROR, "estimate txFee failed", logging.LogFormat{
			"err": err,
		})
		return "", massutil.ZeroAmount(), err
	}

	mtx.LockTime = LockTime
	mtx.Version = wire.TxVersion
	mtxHex, err := messageToHex(mtx)
	if err != nil {
		logging.CPrint(logging.ERROR, "Error in messageToHex(mtx)", logging.LogFormat{
			"err": err,
		})
		return "", massutil.ZeroAmount(), err
	}
	w.MarkUsedUTXO(mtx)
	return mtxHex, txFee, nil
}

func (w *WalletManager) CreateStakingTransaction(fromAddr string, Outputs []*StakingTxOut, lockTime uint64, userTxFee massutil.Amount) (string, error) {

	w.mu.RLock()
	defer w.mu.RUnlock()

	msgTx, _, err := w.EstimateStakingTxFee(Outputs, lockTime, userTxFee, fromAddr, defaultChangeAddress)
	if err != nil {
		return "", err
	}

	msgTx.LockTime = lockTime
	msgTx.Version = wire.TxVersion
	mtxHex, err := messageToHex(msgTx)
	if err != nil {
		logging.CPrint(logging.ERROR, "Error in messageToHex(mtx)", logging.LogFormat{
			"err": err,
		})
		return "", err
	}
	w.MarkUsedUTXO(msgTx)
	return mtxHex, nil
}

func (w *WalletManager) CreateBindingTransaction(fromAddress string, txFee massutil.Amount, output []*BindingOutput) (string, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	locktime := uint64(0)
	msgTx, _, err := w.EstimateBindingTxFee(output, locktime, txFee, fromAddress, defaultChangeAddress)
	if err != nil {
		return "", err
	}

	msgTx.LockTime = locktime
	msgTx.Version = wire.TxVersion
	mtxHex, err := messageToHex(msgTx)
	if err != nil {
		logging.CPrint(logging.ERROR, "Error in messageToHex(mtx)", logging.LogFormat{
			"err": err,
		})
		return "", err
	}
	w.MarkUsedUTXO(msgTx)
	return mtxHex, nil
}

// MarkUsed marks utxo used in cache
func (w *WalletManager) MarkUsedUTXO(msgTx *wire.MsgTx) {
	for _, txIn := range msgTx.TxIn {
		w.usedCache.Set(txIn.PreviousOutPoint.String(), nil, cache.DefaultExpiration)
	}
}

func (w *WalletManager) UTXOUsed(op *wire.OutPoint) bool {
	_, exist := w.usedCache.Get(op.String())
	return exist
}

// ClearUsedUTXOMark ...
func (w *WalletManager) ClearUsedUTXOMark(msgTx *wire.MsgTx) {
	for _, txIn := range msgTx.TxIn {
		w.usedCache.Delete(txIn.PreviousOutPoint.String())
	}
}

func (w *WalletManager) SignRawTx(password []byte, flag string, tx *wire.MsgTx) ([]byte, error) {
	ks := w.ksmgr.CurrentKeystore()
	if ks == nil {
		logging.CPrint(logging.ERROR, "no wallet in use", logging.LogFormat{
			"err": ErrNoWalletInUse,
		})
		return nil, ErrNoWalletInUse
	}

	var hashType txscript.SigHashType
	switch flag {
	case "ALL":
		hashType = txscript.SigHashAll
	case "NONE":
		hashType = txscript.SigHashNone
	case "SINGLE":
		hashType = txscript.SigHashSingle
	case "ALL|ANYONECANPAY":
		hashType = txscript.SigHashAll | txscript.SigHashAnyOneCanPay
	case "NONE|ANYONECANPAY":
		hashType = txscript.SigHashNone | txscript.SigHashAnyOneCanPay
	case "SINGLE|ANYONECANPAY":
		hashType = txscript.SigHashSingle | txscript.SigHashAnyOneCanPay
	default:
		logging.CPrint(logging.ERROR, "Invalid sighash parameter", logging.LogFormat{"flag": flag})
		return nil, ErrInvalidFlag
	}

	// All args collected. Now we can sign all the inputs that we can.
	// `complete' denotes that we successfully signed all outputs and that
	// all scripts will run to completion. This is returned as part of the
	// reply.
	err := w.signWitnessTx(password, tx, hashType, w.chainParams)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to sign the transaction", logging.LogFormat{
			"err": err,
		})
		return nil, err
	}

	// All returned errors (not OOM, which panics) encounted during
	// bytes.Buffer writes are unexpected.
	bs, err := tx.Bytes(wire.Packet)
	if err != nil {
		logging.CPrint(logging.FATAL, "err in api tx serialize", logging.LogFormat{
			"err":  err,
			"hash": tx.TxHash().String(),
		})
		return nil, ErrSignWitnessTx
	}
	return bs, nil
}

func (w *WalletManager) GetStakingHistory() ([]*txmgr.StakingHistoryDetail, error) {
	acct := w.ksmgr.CurrentKeystore()
	if acct == nil {
		logging.CPrint(logging.ERROR, "no wallet in use", logging.LogFormat{})
		return nil, ErrNoWalletInUse
	}

	ret := make([]*txmgr.StakingHistoryDetail, 0)
	err := mwdb.View(w.db, func(tx mwdb.ReadTransaction) error {
		unmined, err := w.utxoStore.GetUnminedStakingHistoryDetail(tx, acct)
		if err != nil {
			return err
		}
		ret = append(ret, unmined...)
		mined, err := w.utxoStore.GetStakingHistoryDetail(tx, acct)
		if err != nil {
			return err
		}
		ret = append(ret, mined...)
		return err
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (w *WalletManager) GetBindingHistory() ([]*txmgr.BindingHistoryDetail, error) {
	ks := w.ksmgr.CurrentKeystore()
	if ks == nil {
		return nil, ErrNoWalletInUse
	}
	ret := make([]*txmgr.BindingHistoryDetail, 0)
	err := mwdb.View(w.db, func(tx mwdb.ReadTransaction) error {
		unmined, err := w.utxoStore.GetUnminedBindingHistoryDetail(tx, ks)
		if err != nil {
			return err
		}
		ret = append(ret, unmined...)

		mined, err := w.utxoStore.GetBindingHistoryDetail(tx, ks)
		if err != nil {
			return err
		}
		ret = append(ret, mined...)
		return err
	})
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (w *WalletManager) Start() error {
	w.server.Blockchain().RegisterListener(w.ntfnsHandler)
	err := w.ntfnsHandler.Start()
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to start ntfnsHandler", logging.LogFormat{
			"err": err,
		})
		return err
	}
	return nil
}

func (w *WalletManager) Stop() {
	w.server.Blockchain().UnregisterListener(w.ntfnsHandler)
	w.wg.Add(1)
	w.ntfnsHandler.Stop()
	w.wg.Wait()
	logging.CPrint(logging.INFO, "WalletManager stopped", logging.LogFormat{})
	return
}

func (w *WalletManager) CloseDB() {
	err := w.db.Close()
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to close wallet db", logging.LogFormat{
			"err": err,
		})
	}
	w.wg.Done()
}

func (w *WalletManager) ChainIndexerSyncedHeight() uint64 {
	return w.server.Blockchain().BestBlockHeight()
}

func (w *WalletManager) CurrentWallet() string {
	if w.ksmgr.CurrentKeystore() != nil {
		return w.ksmgr.CurrentKeystore().Name()
	}
	return ""
}

func (w *WalletManager) SyncedTo() (uint64, error) {
	var height uint64
	err := mwdb.View(w.db, func(tx mwdb.ReadTransaction) error {
		syncedTo, err := w.syncStore.SyncedTo(tx)
		if err != nil {
			return err
		}
		height = syncedTo.Height
		return nil
	})
	if err != nil {
		return 0, err
	}
	return height, nil
}

func (w *WalletManager) IsAddressInCurrent(addr string) (massutil.Address, bool, error) {
	address, err := massutil.DecodeAddress(addr, w.chainParams)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to decode address", logging.LogFormat{
			"err":     err,
			"address": addr,
		})
		return nil, false, nil
	}

	_, err = w.ksmgr.GetManagedAddressByScriptHashInCurrent(address.ScriptAddress())
	if err != nil {
		if err == keystore.ErrAddressNotFound {
			return address, false, nil
		}
		return nil, false, err
	}
	return address, true, nil
}
