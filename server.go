// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"sync"
	"sync/atomic"
	"time"

	"massnet.org/mass-wallet/logging"

	"massnet.org/mass-wallet/blockchain"
	"massnet.org/mass-wallet/database"
	"massnet.org/mass-wallet/netsync"
	"massnet.org/mass-wallet/wire"

	"massnet.org/mass-wallet/consensus"
)

// server provides a bitcoin server for handling communications to and from
// bitcoin peers.
type server struct {
	started     int32 // atomic
	shutdown    int32 // atomic
	db          database.Db
	chain       *blockchain.Blockchain
	syncManager *netsync.SyncManager
	wg          sync.WaitGroup
	quit        chan struct{}
}

// Start begins accepting connections from peers.
func (s *server) Start() {
	// Already started?
	if atomic.AddInt32(&s.started, 1) != 1 {
		logging.CPrint(logging.INFO, "started exit", logging.LogFormat{"started": s.started})
		return
	}

	logging.CPrint(logging.TRACE, "starting server", logging.LogFormat{})

	// srvrLog.Trace("Starting server")
	logging.CPrint(logging.INFO, "begin to start any com", logging.LogFormat{})

	// Start SyncManager
	s.syncManager.Start()

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

	s.syncManager.Stop()

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
//func (s *server) ScheduleShutdown(duration time.Duration) {
//	// Don't schedule shutdown more than once.
//	if atomic.AddInt32(&s.shutdownSched, 1) != 1 {
//		return
//	}
//	logging.CPrint(logging.WARN, "server shutdown on schedule", logging.LogFormat{"duration": duration})
//	go func() {
//		remaining := duration
//		tickDuration := dynamicTickDuration(remaining)
//		done := time.After(remaining)
//		ticker := time.NewTicker(tickDuration)
//	out:
//		for {
//			select {
//			case <-done:
//				ticker.Stop()
//				s.Stop()
//				break out
//			case <-ticker.C:
//				remaining = remaining - tickDuration
//				if remaining < time.Second {
//					continue
//				}
//
//				// Change tick duration dynamically based on remaining time.
//				newDuration := dynamicTickDuration(remaining)
//				if tickDuration != newDuration {
//					tickDuration = newDuration
//					ticker.Stop()
//					ticker = time.NewTicker(tickDuration)
//				}
//				logging.CPrint(logging.WARN, fmt.Sprintf("Server shutdown in %v", remaining), logging.LogFormat{})
//			}
//		}
//	}()
//}

//// parseListeners splits the list of listen addresses passed in addrs into
//// IPv4 and IPv6 slices and returns them.  This allows easy creation of the
//// listeners on the correct interface "tcp4" and "tcp6".  It also properly
//// detects addresses which apply to "all interfaces" and adds the address to
//// both slices.
//func parseListeners(addrs []string) ([]string, []string, bool, error) {
//	ipv4ListenAddrs := make([]string, 0, len(addrs)*2)
//	ipv6ListenAddrs := make([]string, 0, len(addrs)*2)
//	haveWildcard := false
//
//	for _, addr := range addrs {
//		host, _, err := net.SplitHostPort(addr)
//		if err != nil {
//			// Shouldn't happen due to already being normalized.
//			return nil, nil, false, err
//		}
//
//		// Empty host or host of * on plan9 is both IPv4 and IPv6.
//		if host == "" || (host == "*" && runtime.GOOS == "plan9") {
//			ipv4ListenAddrs = append(ipv4ListenAddrs, addr)
//			ipv6ListenAddrs = append(ipv6ListenAddrs, addr)
//			haveWildcard = true
//			continue
//		}
//
//		// Strip IPv6 zone id if present since net.ParseIP does not
//		// handle it.
//		zoneIndex := strings.LastIndex(host, "%")
//		if zoneIndex > 0 {
//			host = host[:zoneIndex]
//		}
//
//		// Parse the IP.
//		ip := net.ParseIP(host)
//		if ip == nil {
//			return nil, nil, false, fmt.Errorf("'%s' is not a "+
//				"valid IP address", host)
//		}
//
//		// To4 returns nil when the IP is not an IPv4 address, so use
//		// this determine the address type.
//		if ip.To4() == nil {
//			ipv6ListenAddrs = append(ipv6ListenAddrs, addr)
//		} else {
//			ipv4ListenAddrs = append(ipv4ListenAddrs, addr)
//		}
//	}
//	return ipv4ListenAddrs, ipv6ListenAddrs, haveWildcard, nil
//}

// newServer returns a new mass server configured to listen on addr for the
// bitcoin network type specified by chainParams.  Use start to begin accepting
// connections from peers.
func newServer(db database.Db) (*server, error) {
	s := &server{
		quit: make(chan struct{}),
		db:   db,
	}

	var err error
	// Create Blockchain
	s.chain, err = blockchain.NewBlockchain(db, cfg.Data.DbDir, s)
	if err != nil {
		logging.CPrint(logging.ERROR, "fail on new BlockChain", logging.LogFormat{"err": err})
		return nil, err
	}

	// New SyncManager
	newBlockCh := make(chan *wire.Hash, consensus.MaxNewBlockChSize)
	syncManager, err := netsync.NewSyncManager(cfg, s.chain, s.chain.GetTxPool(), newBlockCh)
	if err != nil {
		return nil, err
	}
	s.syncManager = syncManager
	return s, nil
}

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

func (s *server) Blockchain() *blockchain.Blockchain {
	return s.chain
}

func (s *server) ChainDB() database.Db {
	return s.db
}

func (s *server) TxMemPool() *blockchain.TxPool {
	return s.chain.GetTxPool()
}

func (s *server) SyncManager() *netsync.SyncManager {
	return s.syncManager
}
