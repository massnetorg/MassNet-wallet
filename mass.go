// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"runtime"

	"github.com/massnetorg/mass-core/blockchain/state"
	"github.com/massnetorg/mass-core/database"
	_ "github.com/massnetorg/mass-core/database/ldb"
	"github.com/massnetorg/mass-core/database/storage"
	_ "github.com/massnetorg/mass-core/database/storage/ldbstorage"
	_ "github.com/massnetorg/mass-core/database/storage/rdbstorage"
	"github.com/massnetorg/mass-core/limits"
	"github.com/massnetorg/mass-core/logging"
	"github.com/massnetorg/mass-core/trie/massdb"
	"github.com/massnetorg/mass-core/trie/rawdb"
	"github.com/massnetorg/mass-core/version"
	"massnet.org/mass-wallet/config"
	_ "massnet.org/mass-wallet/masswallet/db/ldb"
	_ "massnet.org/mass-wallet/masswallet/db/rdb"
)

var (
	cfg             *config.Config
	closeDbChannel  = make(chan struct{})
	shutdownChannel = make(chan struct{})
)

// winServiceMain is only invoked on Windows.  It detects when mass is running
// as a service and reacts accordingly.
var winServiceMain func() (bool, error)

// massMain is the real main function for mass.  It is necessary to work around
// the fact that deferred functions do not run when os.Exit() is called.  The
// optional serverChan parameter is mainly used by the service code to be
// notified with the server once it is setup so it can gracefully stop it when
// requested from the service control manager.
func massMain(serverChan chan<- *server) error {
	// Load configuration and parse command line.  This function also
	// initializes logging and configures it accordingly.
	tempCfg, _, err := config.ParseConfig()
	if err != nil {
		return err
	}

	config.LoadConfig(tempCfg)
	cfg = config.CheckConfig(tempCfg)

	logging.Init(cfg.Core.Log.LogDir, config.DefaultLoggingFilename, cfg.Core.Log.LogLevel, 1, false)

	// Show version at startup.
	logging.CPrint(logging.INFO, fmt.Sprintf("version %s", version.GetVersion()), logging.LogFormat{})

	// Enable http profiling server if requested.
	if cfg.Core.Metrics.ProfilePort != "" {
		go func() {
			listenAddr := net.JoinHostPort("", cfg.Core.Metrics.ProfilePort)
			logging.CPrint(logging.INFO, fmt.Sprintf("profile server listening on %s", listenAddr), logging.LogFormat{})
			profileRedirect := http.RedirectHandler("/debug/pprof",
				http.StatusSeeOther)
			http.Handle("/", profileRedirect)
			logging.CPrint(logging.ERROR, fmt.Sprintf("%v", http.ListenAndServe(listenAddr, nil)), logging.LogFormat{})
		}()
	}

	bindingDb, err := openStateDatabase(cfg.Core.Datastore.Dir, "bindingstate", 0, 0, "", false)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to load binding database", logging.LogFormat{"err": err})
		return err
	}

	// Load the block database.
	db, err := setupBlockDB()
	if err != nil {
		bindingDb.Close()
		logging.CPrint(logging.ERROR, "setupBlockDB error", logging.LogFormat{"err": err})
		return err
	}

	// Create server
	server, err := newServer(db, state.NewDatabase(bindingDb), config.ChainParams)
	if err != nil {
		bindingDb.Close()
		db.Close()
		logging.CPrint(logging.ERROR, "unable to start server on address", logging.LogFormat{"addr": cfg.Core.P2P.ListenAddress, "err": err})
		return err
	}

	// Load wallet
	loader := NewLoader(server, config.ChainParams, cfg)
	if err = loader.LoadWallet(); err != nil {
		bindingDb.Close()
		db.Close()
		return err
	}

	addInterruptHandler(func() {
		server.Stop()
		err := loader.UnloadWallet()
		if err != nil {
			logging.CPrint(logging.ERROR, "failed to unload wallet", logging.LogFormat{"err": err})
		}

		server.WaitForShutdown()
		err0 := bindingDb.Close()
		err1 := db.Close()
		logging.CPrint(logging.INFO, "db closed", logging.LogFormat{
			"binding error": err0,
			"chain error":   err1,
		})
		closeDbChannel <- struct{}{}
	})

	server.Start()
	if serverChan != nil {
		serverChan <- server
	}

	// Monitor for graceful server shutdown and signal the main goroutine
	// when done.  This is done in a separate goroutine rather than waiting
	// directly so the main goroutine can be signaled for shutdown by either
	// a graceful shutdown or from the main interrupt handler.  This is
	// necessary since the main goroutine must be kept running long enough
	// for the interrupt handler goroutine to finish.
	go func() {
		shutdownChannel <- (<-closeDbChannel)
	}()

	// Wait for shutdown signal from either a graceful server stop or from
	// the interrupt handler.
	<-shutdownChannel
	return nil
}

func main() {
	// Use all processor cores.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Up some limits.
	if err := limits.SetLimits(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set limits: %v\n", err)
		os.Exit(1)
	}

	// Call serviceMain on Windows to handle running as a service.  When
	// the return isService flag is true, exit now since we ran as a
	// service.  Otherwise, just fall through to normal operation.
	if runtime.GOOS == "windows" {
		isService, err := winServiceMain()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if isService {
			os.Exit(0)
		}
	}

	// Work around defer not working after os.Exit()
	if err := massMain(nil); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// blockDbNamePrefix is the prefix for the block database name.  The
// database type is appended to this value to form the full block
// database name.
const blockDbNamePrefix = "blocks"

// dbPath returns the path to the block database given a database type.
func blockDbPath(dbType string) string {
	// The database name is based on the database type.
	dbName := blockDbNamePrefix + ".db"
	if dbType == "sqlite" {
		dbName = dbName + ".db"
	}
	dbPath := filepath.Join(cfg.Core.Datastore.Dir, dbName)
	return dbPath
}

// setupBlockDB loads (or creates when needed) the block database taking into
// account the selected database backend.  It also contains additional logic
// such warning the user if there are multiple databases which consume space on
// the file system and ensuring the regression test database is clean when in
// regression test mode.
func setupBlockDB() (database.Db, error) {
	// The memdb backend does not have a file path associated with it, so
	// handle it uniquely.  We also don't want to worry about the multiple
	// database type warnings when running with the memory database.
	if cfg.Core.Datastore.DBType == "memdb" {
		logging.CPrint(logging.INFO, "creating block database in memory", logging.LogFormat{})
		db, err := database.CreateDB("memdb")
		if err != nil {
			return nil, err
		}
		return db, nil
	}

	// Create the new path if needed.
	err := os.MkdirAll(cfg.Core.Datastore.Dir, 0700)
	if err != nil {
		return nil, err
	}

	if err = storage.CheckCompatibility(cfg.Core.Datastore.DBType, cfg.Core.Datastore.Dir); err != nil {
		logging.CPrint(logging.ERROR, "check storage compatibility failed", logging.LogFormat{"err": err})
		return nil, err
	}

	dbPath := blockDbPath(cfg.Core.Datastore.DBType)
	db, err := database.OpenDB(cfg.Core.Datastore.DBType, dbPath, false)
	if err != nil {
		logging.CPrint(logging.WARN, "open db failed", logging.LogFormat{"err": err, "path": dbPath})
		db, err = database.CreateDB(cfg.Core.Datastore.DBType, dbPath)
		if err != nil {
			logging.CPrint(logging.ERROR, "create db failed", logging.LogFormat{"err": err, "path": dbPath})
			return nil, err
		}
	}

	return db, nil
}

// filesExists reports whether the named file or directory exists.
func FileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func openStateDatabase(dataDir, name string, cache, handles int, namespace string, readonly bool) (massdb.Database, error) {
	if dataDir == "" {
		return rawdb.NewMemoryDatabase(), nil
	} else {
		path := filepath.Join(dataDir, name)
		return rawdb.NewLevelDBDatabase(path, cache, handles, namespace, readonly)
	}
}
