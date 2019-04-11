// Copyright (c) 2017-2019 The massnet developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
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
	"runtime"
	"runtime/pprof"

	"github.com/massnetorg/MassNet-wallet/logging"
	"github.com/massnetorg/MassNet-wallet/version"

	"path/filepath"

	"github.com/massnetorg/MassNet-wallet/config"
	"github.com/massnetorg/MassNet-wallet/database"
	"github.com/massnetorg/MassNet-wallet/limits"
	"github.com/massnetorg/MassNet-wallet/massutil"
)

var (
	cfg             *config.Config
	shutdownChannel = make(chan struct{})
)

var winServiceMain func() (bool, error)

func massMain(serverChan chan<- *server) error {
	tcfg, _, err := config.ParseConfig()
	if err != nil {
		return err
	}

	tcfg, err = config.LoadConfig(tcfg)
	if err != nil {
		return err
	}

	cfg, err = config.CheckConfig(tcfg)
	if err != nil {
		return err
	}

	// Show version at startup.
	logging.CPrint(logging.INFO, fmt.Sprintf("version %s", version.GetVersion()), logging.LogFormat{})

	// Enable http profiling server if requested.
	if cfg.App.Profile != "" {
		go func() {
			listenAddr := net.JoinHostPort("", cfg.App.Profile)
			logging.CPrint(logging.INFO, fmt.Sprintf("profile server listening on %s", listenAddr), logging.LogFormat{})
			profileRedirect := http.RedirectHandler("/debug/pprof",
				http.StatusSeeOther)
			http.Handle("/", profileRedirect)
			logging.CPrint(logging.ERROR, fmt.Sprintf("%v", http.ListenAndServe(listenAddr, nil)), logging.LogFormat{})
		}()
	}

	// Write cpu profile if requested.
	if cfg.App.CPUProfile != "" {
		f, err := os.Create(cfg.App.CPUProfile)
		if err != nil {
			logging.CPrint(logging.ERROR, "unable to create cpu profile", logging.LogFormat{"err": err})
			return err
		}
		pprof.StartCPUProfile(f)
		defer f.Close()
		defer pprof.StopCPUProfile()
	}

	// Load the block database.
	db, err := loadBlockDB()
	if err != nil {
		logging.CPrint(logging.ERROR, "loadBlockDB error", logging.LogFormat{"err": err})
		return err
	}
	defer db.Close()

	if config.DropAddrIndex {
		logging.CPrint(logging.INFO, "deleting entire addrindex", logging.LogFormat{})
		err := db.DeleteAddrIndex()
		if err != nil {
			logging.CPrint(logging.ERROR, "unable to delete the addrindex", logging.LogFormat{"err": err})
			return err
		}
		logging.CPrint(logging.INFO, "successfully deleted addrindex, exiting", logging.LogFormat{})
		return nil
	}

	// Ensure the database is sync'd and closed on Ctrl+C.
	addInterruptHandler(func() {
		logging.CPrint(logging.INFO, "gracefully shutting down the database", logging.LogFormat{})
		db.RollbackClose()
	})

	// Create server and start it.
	server, err := newServer(db)
	if err != nil {
		logging.CPrint(logging.ERROR, "unable to start server on address", logging.LogFormat{"addr": cfg.Network.P2P.ListenAddress, "err": err})
		return err
	}
	addInterruptHandler(func() {
		logging.CPrint(logging.INFO, "gracefully shutting down the server", logging.LogFormat{})
		server.Stop()
		server.WaitForShutdown()
	})
	server.Start()
	if serverChan != nil {
		serverChan <- server
	}

	go func() {
		server.WaitForShutdown()
		logging.CPrint(logging.INFO, "server shutdown complete", logging.LogFormat{})
		shutdownChannel <- struct{}{}
	}()

	<-shutdownChannel
	logging.CPrint(logging.INFO, "shutdown complete", logging.LogFormat{})
	return nil
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	if err := limits.SetLimits(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set limits: %v\n", err)
		os.Exit(1)
	}

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

	if err := massMain(nil); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

const blockDbNamePrefix = "blocks"

func blockDbPath(dbType string) string {
	dbName := blockDbNamePrefix + ".db"
	if dbType == "sqlite" {
		dbName = dbName + ".db"
	}
	dbPath := filepath.Join(cfg.Db.DataDir, dbName)
	return dbPath
}

func warnMultipeDBs() {
	dbTypes := []string{"leveldb", "sqlite"}
	duplicateDbPaths := make([]string, 0, len(dbTypes)-1)
	for _, dbType := range dbTypes {
		if dbType == cfg.Db.DbType {
			continue
		}

		dbPath := blockDbPath(dbType)
		if FileExists(dbPath) {
			duplicateDbPaths = append(duplicateDbPaths, dbPath)
		}
	}

	if len(duplicateDbPaths) > 0 {
		selectedDbPath := blockDbPath(cfg.Db.DbType)
		str := fmt.Sprintf("WARNING: There are multiple block chain databases "+
			"using different database types.\nYou probably don't "+
			"want to waste disk space by having more than one.\n"+
			"Your current database is located at [%v].\nThe "+
			"additional database is located at %v", selectedDbPath,
			duplicateDbPaths)
		logging.CPrint(logging.WARN, str, logging.LogFormat{})
	}
}

func setupBlockDB() (database.Db, error) {
	if cfg.Db.DbType == "memdb" {
		logging.CPrint(logging.INFO, "creating block database in memory", logging.LogFormat{})
		db, err := database.CreateDB(cfg.Db.DbType)
		if err != nil {
			return nil, err
		}
		return db, nil
	}

	warnMultipeDBs()

	dbPath := blockDbPath(cfg.Db.DbType)

	logging.CPrint(logging.INFO, "loading block database", logging.LogFormat{"from": dbPath})
	db, err := database.OpenDB(cfg.Db.DbType, dbPath)
	if err != nil {
		// Return the error if it's not because the database
		// doesn't exist.
		if err != database.ErrDbDoesNotExist {
			return nil, err
		}

		// Create the db if it does not exist.
		err = os.MkdirAll(cfg.Db.DataDir, 0700)
		if err != nil {
			return nil, err
		}
		db, err = database.CreateDB(cfg.Db.DbType, dbPath)
		if err != nil {
			return nil, err
		}
	}

	return db, nil
}

// loadBlockDB opens the block database and returns a handle to it.
func loadBlockDB() (database.Db, error) {
	db, err := setupBlockDB()
	if err != nil {
		return nil, err
	}

	_, height, err := db.NewestSha()
	if err != nil {
		db.Close()
		return nil, err
	}

	if height == -1 {
		genesis := massutil.NewBlock(config.ChainParams.GenesisBlock)
		_, err := db.InsertBlock(genesis)
		if err != nil {
			db.Close()
			return nil, err
		}
		logging.CPrint(logging.INFO, "inserted genesis block", logging.LogFormat{"hash": config.ChainParams.GenesisHash})
		height = 0
	}

	logging.CPrint(logging.INFO, "block database loaded with block height", logging.LogFormat{"height": height})
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
