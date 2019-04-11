// Copyright (c) 2017-2019 The massnet developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.
// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"massnet.org/mass-wallet/consensus"
	"sync"
	"sync/atomic"
	"time"

	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/wallet"

	"massnet.org/mass-wallet/api"
	"massnet.org/mass-wallet/blockchain"
	"massnet.org/mass-wallet/chain"
	"massnet.org/mass-wallet/chainindexer"
	"massnet.org/mass-wallet/database"
	"massnet.org/mass-wallet/netsync"
	"massnet.org/mass-wallet/txscript"
	"massnet.org/mass-wallet/wire"

	"massnet.org/mass-wallet/utxo"
)

type server struct {
	started       int32 // atomic
	shutdown      int32 // atomic
	shutdownSched int32 // atomic
	sigCache      *txscript.SigCache
	hashCache     *txscript.HashCache
	apiServer     *api.Server
	bc            *blockchain.BlockChain
	syncManager   *netsync.SyncManager
	Chain         *chain.Chain
	NewBlockCh    chan *wire.Hash
	addrIndexer   *chainindexer.AddrIndexer
	Wallet        *wallet.Wallet
	wg            sync.WaitGroup
	quit          chan struct{}
	db            database.Db
	timeSource    blockchain.MedianTimeSource
}

// Start begins accepting connections from peers.
func (s *server) Start() {
	// Already started?
	if atomic.AddInt32(&s.started, 1) != 1 {
		logging.CPrint(logging.INFO, "started exit", logging.LogFormat{"started": s.started})
		return
	}

	logging.CPrint(logging.TRACE, "starting server", logging.LogFormat{})

	logging.CPrint(logging.INFO, "begin to start any com", logging.LogFormat{})

	// Start SyncManager
	s.syncManager.Start()

	s.bc.Start()

	s.apiServer.Start()

	s.apiServer.RunGateway()

	s.wg.Add(1)

}

// Stop gracefully shuts down the server by stopping and disconnecting all
// peers and the main listener.
func (s *server) Stop() error {
	// Make sure this only happens once.
	if atomic.AddInt32(&s.shutdown, 1) != 1 {
		logging.CPrint(logging.INFO, "server is already in the process of shutting down", logging.LogFormat{})
		return nil
	}

	logging.CPrint(logging.WARN, "server shutting down", logging.LogFormat{})

	// Shutdown apiServer
	s.apiServer.Stop()

	s.bc.Stop()

	s.syncManager.Stop()

	// Shutdown walletdb for Tx
	s.Wallet.Close()

	// Signal the remaining goroutines to quit.
	close(s.quit)

	s.wg.Done()

	return nil
}

// WaitForShutdown blocks until the main listener and peer handlers are stopped.
func (s *server) WaitForShutdown() {
	s.wg.Wait()
}

// ScheduleShutdown schedules a server shutdown after the specified duration.
// It also dynamically adjusts how often to warn the server is going down based
// on remaining duration.
func (s *server) ScheduleShutdown(duration time.Duration) {
	// Don't schedule shutdown more than once.
	if atomic.AddInt32(&s.shutdownSched, 1) != 1 {
		return
	}
	logging.CPrint(logging.WARN, "server shutdown on schedule", logging.LogFormat{"duration": duration})
	go func() {
		remaining := duration
		tickDuration := dynamicTickDuration(remaining)
		done := time.After(remaining)
		ticker := time.NewTicker(tickDuration)
	out:
		for {
			select {
			case <-done:
				ticker.Stop()
				s.Stop()
				break out
			case <-ticker.C:
				remaining = remaining - tickDuration
				if remaining < time.Second {
					continue
				}

				// Change tick duration dynamically based on remaining time.
				newDuration := dynamicTickDuration(remaining)
				if tickDuration != newDuration {
					tickDuration = newDuration
					ticker.Stop()
					ticker = time.NewTicker(tickDuration)
				}
				logging.CPrint(logging.WARN, fmt.Sprintf("Server shutdown in %v", remaining), logging.LogFormat{})
			}
		}
	}()
}

func newServer(db database.Db) (*server, error) {
	s := server{
		quit:       make(chan struct{}),
		db:         db,
		timeSource: blockchain.NewMedianTime(),
		sigCache:   txscript.NewSigCache(config.SigCacheMaxSize),
		hashCache:  txscript.NewHashCache(config.SigCacheMaxSize),
	}

	// Create user wallet
	walletForTx, err := wallet.NewWallet(cfg.Wallet.WalletDir)
	if err != nil {
		return nil, err
	}
	s.Wallet = walletForTx

	utxoStrut, err := utxo.NewUtxo(db)
	if err != nil {
		return nil, err
	}

	// Create txMemPool instance.
	txMemPool := blockchain.NewTxMemPool(s.db, nil, s.sigCache, s.hashCache, s.timeSource)

	// Create addrIndexer instance.
	if config.AddrIndex {
		ai, err := chainindexer.NewAddrIndexer(s.db, &s)
		if err != nil {
			return nil, err
		}
		s.addrIndexer = ai
	}

	s.bc, err = blockchain.New(s.db, s.sigCache, s.hashCache, txMemPool, s.addrIndexer)
	if err != nil {
		return nil, err
	}

	//bm, err := blockmanager.NewBlockManager(s.spaceKeeper.Wallet, walletForTx, cfg, s.db, s.addrIndexer, s.txMemPool, s.sigCache, s.hashCache, s.timeSource)
	//if err != nil {
	//	return nil, err
	//}
	//s.blockManager = bm

	// New Chain
	chainX, err := chain.NewChain(s.bc, s.timeSource)
	s.Chain = chainX

	// New SyncManager
	newBlockCh := make(chan *wire.Hash, consensus.MaxNewBlockChSize)
	s.NewBlockCh = newBlockCh
	if err != nil {
		return nil, err
	}
	syncManager, err := netsync.NewSyncManager(cfg, chainX, s.bc.TxPool, newBlockCh)
	if err != nil {
		return nil, err
	}
	s.syncManager = syncManager

	s.apiServer, err = api.NewServer(s.db, s.bc.TxPool, utxoStrut, s.Wallet, s.syncManager, cfg)
	if err != nil {
		logging.CPrint(logging.ERROR, "new server", logging.LogFormat{"err": err})
		return nil, err
	}

	return &s, nil
}

func (s *server) addTimeSample(id string, timeVal time.Time) {
	s.timeSource.AddTimeSample(id, timeVal)
}

//func (s *server) isRpcServerValid() bool {
//	if s.rpcServer != nil {
//		return true
//	} else {
//		return false
//	}
//}

// dynamicTickDuration is a convenience function used to dynamically choose a
// tick duration based on remaining time.  It is primarily used during
// server shutdown to make shutdown warnings more frequent as the shutdown time
// approaches.
func dynamicTickDuration(remaining time.Duration) time.Duration {
	switch {
	case remaining <= time.Second*5:
		return time.Second
	case remaining <= time.Second*15:
		return time.Second * 5
	case remaining <= time.Minute:
		return time.Second * 15
	case remaining <= time.Minute*5:
		return time.Minute
	case remaining <= time.Minute*15:
		return time.Minute * 5
	case remaining <= time.Hour:
		return time.Minute * 15
	}
	return time.Hour
}
