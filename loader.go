// Copyright (c) 2015-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"errors"
	"os"
	"path/filepath"
	"sync"

	"massnet.org/mass-wallet/api"
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/masswallet"
	mwdb "massnet.org/mass-wallet/masswallet/db"
)

const (
	walletDbName = "wallet.db"
)

var (
	// ErrLoaded describes the error condition of attempting to load or
	// create a wallet when the loader has already done so.
	ErrLoaded = errors.New("wallet already loaded")

	// // ErrNotLoaded describes the error condition of attempting to close a
	// // loaded wallet when a wallet has not been loaded.
	// ErrNotLoaded = errors.New("wallet is not loaded")

	// ErrExists describes the error condition of attempting to create a new
	// wallet when one exists already.
	ErrExists = errors.New("wallet already exists")
)

// Loader implements the creating of new and opening of existing wallets, while
// providing a callback system for other subsystems to handle the loading of a
// wallet.  This is primarily intended for use by the RPC servers, to enable
// methods and services which require the wallet when the wallet is loaded by
// another subsystem.
//
// Loader is safe for concurrent access.
type Loader struct {
	massServer  *server
	apiServer   *api.APIServer
	callbacks   []func(*masswallet.WalletManager)
	chainParams *config.Params
	cfg         *config.Config
	walletMgr   *masswallet.WalletManager
	mu          sync.Mutex
}

// NewLoader constructs a Loader with an optional recovery window. If the
// recovery window is non-zero, the wallet will attempt to recovery addresses
// starting from the last SyncedTo height.
func NewLoader(massServer *server, chainParams *config.Params, cfg *config.Config) *Loader {

	return &Loader{
		massServer:  massServer,
		chainParams: chainParams,
		cfg:         cfg,
	}
}

// onLoaded executes each added callback and prevents loader from loading any
// additional wallets.  Requires mutex to be locked.
func (l *Loader) onLoaded(w *masswallet.WalletManager) (err error) {
	for _, fn := range l.callbacks {
		fn(w)
	}

	l.walletMgr = w
	l.callbacks = nil // not needed anymore
	l.apiServer, err = api.NewAPIServer(l.massServer, w, func() { interruptChannel <- os.Interrupt }, l.cfg)
	if err != nil {
		return
	}
	if err = l.apiServer.Start(); err != nil {
		return
	}
	l.apiServer.RunGateway()
	return
}

// RunAfterLoad adds a function to be executed when the loader creates or opens
// a wallet.  Functions are executed in a single goroutine in the order they are
// added.
func (l *Loader) RunAfterLoad(fn func(*masswallet.WalletManager)) {
	l.mu.Lock()
	if l.walletMgr != nil {
		w := l.walletMgr
		l.mu.Unlock()
		fn(w)
	} else {
		l.callbacks = append(l.callbacks, fn)
		l.mu.Unlock()
	}
}

func (l *Loader) createWallet(dbPath string) (*masswallet.WalletManager, error) {

	defer l.mu.Unlock()
	l.mu.Lock()

	if l.walletMgr != nil {
		return nil, ErrLoaded
	}

	db, err := mwdb.CreateDB(l.cfg.Data.DbType, dbPath)
	if err != nil {
		logging.CPrint(logging.ERROR, "Error opening database", logging.LogFormat{"err": err})
		return nil, err
	}
	w, err := masswallet.NewWalletManager(l.massServer, db, l.cfg, l.chainParams, l.cfg.Data.WalletPubPass)
	if err != nil {
		if e := db.Close(); e != nil {
			logging.CPrint(logging.WARN, "Error closing database", logging.LogFormat{"dbPath": dbPath, "err": err})
		}
		return nil, err
	}

	err = w.Start()
	if err != nil {
		return nil, err
	}

	err = l.onLoaded(w)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (l *Loader) openWallet(dbPath string) (*masswallet.WalletManager, error) {
	defer l.mu.Unlock()
	l.mu.Lock()

	if l.walletMgr != nil {
		return nil, ErrLoaded
	}

	db, err := mwdb.OpenDB(l.cfg.Data.DbType, dbPath)
	if err != nil {
		return nil, err
	}
	w, err := masswallet.NewWalletManager(l.massServer, db, l.cfg, l.chainParams, l.cfg.Data.WalletPubPass)
	if err != nil {
		if e := db.Close(); e != nil {
			logging.CPrint(logging.WARN, "Error closing database", logging.LogFormat{"dbPath": dbPath, "err": err})
		}
		return nil, err
	}
	err = w.Start()
	if err != nil {
		return nil, err
	}

	err = l.onLoaded(w)
	if err != nil {
		return nil, err
	}
	return w, nil
}

// LoadedWalletManager returns the loaded wallet, if any, and a bool for whether the
// wallet has been loaded or not.  If true, the wallet pointer should be safe to
// dereference.
func (l *Loader) LoadedWalletManager() (*masswallet.WalletManager, bool) {
	l.mu.Lock()
	w := l.walletMgr
	l.mu.Unlock()
	return w, w != nil
}

func (l *Loader) LoadWallet() error {
	dbPath := filepath.Join(l.cfg.Data.DbDir, walletDbName)
	fi, _ := os.Stat(dbPath)
	if fi != nil {
		if _, err := l.openWallet(dbPath); err != nil {
			logging.CPrint(logging.ERROR, "Error opening wallet", logging.LogFormat{
				"DbType": l.cfg.Data.DbType,
				"path":   dbPath,
				"err":    err,
			})
			return err
		}
	} else {
		if _, err := l.createWallet(dbPath); err != nil {
			logging.CPrint(logging.ERROR, "Error creating wallet", logging.LogFormat{
				"DbType": l.cfg.Data.DbType,
				"path":   dbPath,
				"err":    err,
			})
			return err
		}
	}

	logging.CPrint(logging.INFO, "load wallet done", logging.LogFormat{
		"DbType": l.cfg.Data.DbType,
		"path":   dbPath,
	})
	return nil
}

// UnloadWallet stops the loaded wallet, if any, and closes the wallet database.
// This returns ErrNotLoaded if the wallet has not been loaded with
// CreateNewWallet or LoadExistingWallet.  The Loader may be reused if this
// function returns without error.
func (l *Loader) UnloadWallet() error {
	defer l.mu.Unlock()
	l.mu.Lock()

	if l.walletMgr == nil {
		return nil
	}

	l.apiServer.Stop()
	l.walletMgr.Stop()

	l.massServer = nil
	l.apiServer = nil
	l.callbacks = nil
	l.walletMgr = nil
	return nil
}
