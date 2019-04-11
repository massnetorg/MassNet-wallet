package netsync

import (
	"encoding/hex"
	"errors"

	"github.com/massnetorg/MassNet-wallet/logging"

	//log "github.com/sirupsen/logrus"
	"net"
	"path"
	"reflect"
	"strconv"
	"strings"

	"github.com/tendermint/go-crypto"
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/massnetorg/MassNet-wallet/blockchain"
	"github.com/massnetorg/MassNet-wallet/config"
	cfg "github.com/massnetorg/MassNet-wallet/config"
	"github.com/massnetorg/MassNet-wallet/consensus"
	"github.com/massnetorg/MassNet-wallet/massutil"
	"github.com/massnetorg/MassNet-wallet/p2p"
	"github.com/massnetorg/MassNet-wallet/p2p/discover"
	"github.com/massnetorg/MassNet-wallet/version"
	"github.com/massnetorg/MassNet-wallet/wire"
)

const (
	maxTxChanSize         = 10000
	maxFilterAddressSize  = 50
	maxFilterAddressCount = 1000
)

type Chain interface {
	BestBlockHeader() *wire.BlockHeader
	BestBlockHeight() uint64
	GetBlockByHash(*wire.Hash) (*massutil.Block, error)
	GetBlockByHeight(uint64) (*massutil.Block, error)
	GetHeaderByHash(*wire.Hash) (*wire.BlockHeader, error)
	GetHeaderByHeight(uint64) (*wire.BlockHeader, error)
	// GetTransactionStatus(*wire.Hash) (*bc.TransactionStatus, error)
	InMainChain(wire.Hash) bool
	ProcessBlock(*massutil.Block) (bool, error)
	ValidateTx(*massutil.Tx) (bool, error)
	ChainID() *wire.Hash
}

//SyncManager Sync Manager is responsible for the business layer information synchronization
type SyncManager struct {
	sw          *p2p.Switch
	genesisHash wire.Hash

	privKey      crypto.PrivKeyEd25519 // local node's p2p key
	Chain        Chain
	txPool       *blockchain.TxPool
	blockFetcher *blockFetcher
	blockKeeper  *blockKeeper
	peers        *peerSet

	newTxCh    chan *massutil.Tx
	newBlockCh chan *wire.Hash
	txSyncCh   chan *txSyncMsg
	quitSync   chan struct{}
	config     *config.Config
}

//NewSyncManager create a sync manager
func NewSyncManager(config *config.Config, chain Chain, txPool *blockchain.TxPool, newBlockCh chan *wire.Hash) (*SyncManager, error) {
	genesisHeader, err := chain.GetHeaderByHeight(0)
	if err != nil {
		return nil, err
	}

	sw := p2p.NewSwitch(config)
	peers := newPeerSet(sw)
	manager := &SyncManager{
		sw:           sw,
		genesisHash:  genesisHeader.BlockHash(),
		txPool:       txPool,
		Chain:        chain,
		privKey:      crypto.GenPrivKeyEd25519(),
		blockFetcher: newBlockFetcher(chain, peers),
		blockKeeper:  newBlockKeeper(chain, peers),
		peers:        peers,
		newTxCh:      make(chan *massutil.Tx, maxTxChanSize),
		newBlockCh:   newBlockCh,
		txSyncCh:     make(chan *txSyncMsg),
		quitSync:     make(chan struct{}),
		config:       config,
	}

	manager.txPool.NewTxCh = manager.newTxCh

	protocolReactor := NewProtocolReactor(manager, manager.peers)
	manager.sw.AddReactor("PROTOCOL", protocolReactor)

	// Create & add listener
	var listenerStatus bool
	var l p2p.Listener
	if !config.Network.P2P.VaultMode {
		p, address := protocolAndAddress(config.Network.P2P.ListenAddress)
		l, listenerStatus = p2p.NewDefaultListener(p, address, config.Network.P2P.SkipUpnp)
		manager.sw.AddListener(l)

		discv, err := initDiscover(config, &manager.privKey, l.ExternalAddress().Port)
		if err != nil {
			return nil, err
		}
		manager.sw.SetDiscv(discv)
	}
	manager.sw.SetNodeInfo(manager.makeNodeInfo(listenerStatus))
	manager.sw.SetNodePrivKey(manager.privKey)
	return manager, nil
}

//BestPeer return the highest p2p peerInfo
func (sm *SyncManager) BestPeer() *PeerInfo {
	bestPeer := sm.peers.bestPeer(consensus.SFFullNode)
	if bestPeer != nil {
		return bestPeer.getPeerInfo()
	}
	return nil
}

// GetNewTxCh return a unconfirmed transaction feed channel
func (sm *SyncManager) GetNewTxCh() chan *massutil.Tx {
	return sm.newTxCh
}

//GetPeerInfos return peer info of all peers
func (sm *SyncManager) GetPeerInfos() []*PeerInfo {
	return sm.peers.getPeerInfos()
}

//IsCaughtUp check wheather the peer finish the sync
func (sm *SyncManager) IsCaughtUp() bool {
	peer := sm.peers.bestPeer(consensus.SFFullNode)
	return peer == nil || peer.Height() <= sm.Chain.BestBlockHeight()
}

//NodeInfo get P2P peer node info
func (sm *SyncManager) NodeInfo() *p2p.NodeInfo {
	return sm.sw.NodeInfo()
}

// NodePubKeyS get P2P peer node pubKey string
func (sm *SyncManager) NodePubKeyS() string {
	return sm.NodeInfo().PubKey.KeyString()
}

//StopPeer try to stop peer by given ID
func (sm *SyncManager) StopPeer(peerID string) error {
	if peer := sm.peers.getPeer(peerID); peer == nil {
		return errors.New("peerId not exist")
	}
	sm.peers.removePeer(peerID)
	return nil
}

//Switch get sync manager switch
func (sm *SyncManager) Switch() *p2p.Switch {
	return sm.sw
}

func (sm *SyncManager) handleBlockMsg(peer *peer, msg *BlockMessage) {
	block, err := msg.GetBlock()
	if err != nil {
		return
	}

	sm.blockKeeper.processBlock(peer.ID(), block)
}

func (sm *SyncManager) handleBlocksMsg(peer *peer, msg *BlocksMessage) {
	blocks, err := msg.GetBlocks()
	if err != nil {
		logging.CPrint(logging.DEBUG, "fail on handleBlocksMsg GetBlocks", logging.LogFormat{"err": err})
		return
	}

	sm.blockKeeper.processBlocks(peer.ID(), blocks)
}

func (sm *SyncManager) handleFilterAddMsg(peer *peer, msg *FilterAddMessage) {
	peer.addFilterAddress(msg.Address)
}

func (sm *SyncManager) handleFilterClearMsg(peer *peer) {
	peer.filterAdds.Clear()
}

func (sm *SyncManager) handleFilterLoadMsg(peer *peer, msg *FilterLoadMessage) {
	peer.addFilterAddresses(msg.Addresses)
}

func (sm *SyncManager) handleGetBlockMsg(peer *peer, msg *GetBlockMessage) {
	var block *massutil.Block
	var err error
	if msg.Height != 0 {
		block, err = sm.Chain.GetBlockByHeight(msg.Height)
	} else {
		block, err = sm.Chain.GetBlockByHash(msg.GetHash())
	}
	if err != nil {
		logging.CPrint(logging.WARN, "fail on handleGetBlockMsg get block from chain", logging.LogFormat{"err": err})
		return
	}

	ok, err := peer.sendBlock(block)
	if !ok {
		sm.peers.removePeer(peer.ID())
	}
	if err != nil {
		logging.CPrint(logging.ERROR, "fail on handleGetBlockMsg sentBlock", logging.LogFormat{"err": err})
	}
}

func (sm *SyncManager) handleGetBlocksMsg(peer *peer, msg *GetBlocksMessage) {
	blocks, err := sm.blockKeeper.locateBlocks(msg.GetBlockLocator(), msg.GetStopHash())
	if err != nil || len(blocks) == 0 {
		return
	}

	totalSize := 0
	sendBlocks := []*massutil.Block{}
	for _, block := range blocks {
		rawData, err := block.Bytes(wire.Packet)
		if err != nil {
			logging.CPrint(logging.ERROR, "fail on handleGetBlocksMsg marshal block", logging.LogFormat{"err": err})
			continue
		}

		if totalSize+len(rawData) > maxBlockchainResponseSize/2 {
			break
		}
		totalSize += len(rawData)
		sendBlocks = append(sendBlocks, block)
	}

	ok, err := peer.sendBlocks(sendBlocks)
	if !ok {
		sm.peers.removePeer(peer.ID())
	}
	if err != nil {
		logging.CPrint(logging.ERROR, "fail on handleGetBlocksMsg sentBlock", logging.LogFormat{"err": err})
	}
}

func (sm *SyncManager) handleGetHeadersMsg(peer *peer, msg *GetHeadersMessage) {
	headers, err := sm.blockKeeper.locateHeaders(msg.GetBlockLocator(), msg.GetStopHash())
	if err != nil || len(headers) == 0 {
		logging.CPrint(logging.DEBUG, "fail on handleGetHeadersMsg locateHeaders", logging.LogFormat{"err": err})
		return
	}

	ok, err := peer.sendHeaders(headers)
	if !ok {
		sm.peers.removePeer(peer.ID())
	}
	if err != nil {
		logging.CPrint(logging.ERROR, "fail on handleGetHeadersMsg sentBlock", logging.LogFormat{"err": err})
	}
}

func (sm *SyncManager) handleHeadersMsg(peer *peer, msg *HeadersMessage) {
	headers, err := msg.GetHeaders()
	if err != nil {
		logging.CPrint(logging.DEBUG, "fail on handleHeadersMsg GetHeaders", logging.LogFormat{"err": err})
		return
	}

	sm.blockKeeper.processHeaders(peer.ID(), headers)
}

func (sm *SyncManager) handleMineBlockMsg(peer *peer, msg *MineBlockMessage) {
	block, err := msg.GetMineBlock()
	if err != nil {
		logging.CPrint(logging.WARN, "fail on handleMineBlockMsg GetMineBlock", logging.LogFormat{"err": err})
		return
	}

	hash := block.Hash()
	peer.markBlock(hash)
	sm.blockFetcher.processNewBlock(&blockMsg{peerID: peer.ID(), block: block})
	peer.setStatus(block.MsgBlock().Header.Height, hash)
}

func (sm *SyncManager) handleStatusRequestMsg(peer BasePeer) {
	bestHeader := sm.Chain.BestBlockHeader()
	genesisBlock, err := sm.Chain.GetBlockByHeight(0)
	if err != nil {
		logging.CPrint(logging.ERROR, "fail on handleStatusRequestMsg get genesis", logging.LogFormat{"err": err})
	}

	genesisHash := genesisBlock.Hash()
	msg := NewStatusResponseMessage(bestHeader, genesisHash)
	if ok := peer.TrySend(BlockchainChannel, struct{ BlockchainMessage }{msg}); !ok {
		sm.peers.removePeer(peer.ID())
	}
}

func (sm *SyncManager) handleStatusResponseMsg(basePeer BasePeer, msg *StatusResponseMessage) {
	if peer := sm.peers.getPeer(basePeer.ID()); peer != nil {
		peer.setStatus(msg.Height, msg.GetHash())
		return
	}

	if genesisHash := msg.GetGenesisHash(); sm.genesisHash != *genesisHash {
		logging.CPrint(logging.WARN, "fail hand shake due to different genesis",
			logging.LogFormat{"remote_genesis": genesisHash.String(), "local_genesis": sm.genesisHash.String(),
				"peer_ip": basePeer.Addr(), "peer_id": basePeer.ID(), "outbound": basePeer.IsOutbound()})
		return
	}

	sm.peers.addPeer(basePeer, msg.Height, msg.GetHash())
}

func (sm *SyncManager) handleTransactionMsg(peer *peer, msg *TransactionMessage) {
	tx, err := msg.GetTransaction()
	if err != nil {
		sm.peers.addBanScore(peer.ID(), 0, 10, "fail on get tx from message")
		return
	}

	if isOrphan, err := sm.Chain.ValidateTx(tx); err != nil && isOrphan == false {
		if merr, ok := err.(blockchain.MpRuleError); ok {
			if rerr, ok := merr.Err.(blockchain.TxRuleError); ok {
				if rerr.RejectCode == wire.RejectAlreadyExists {
					return
				}
			}
		}
		logging.CPrint(logging.ERROR, "validate tx fail", logging.LogFormat{"err": err, "txid": tx.Hash().String()})
		sm.peers.addBanScore(peer.ID(), 10, 0, "fail on validate tx transaction")
	}
}

func (sm *SyncManager) processMsg(basePeer BasePeer, msgType byte, msg BlockchainMessage) {
	peer := sm.peers.getPeer(basePeer.ID())
	if peer == nil && msgType != StatusResponseByte && msgType != StatusRequestByte {
		return
	}

	switch msg := msg.(type) {
	case *GetBlockMessage:
		sm.handleGetBlockMsg(peer, msg)

	case *BlockMessage:
		sm.handleBlockMsg(peer, msg)

	case *StatusRequestMessage:
		sm.handleStatusRequestMsg(basePeer)

	case *StatusResponseMessage:
		sm.handleStatusResponseMsg(basePeer, msg)

	case *TransactionMessage:
		sm.handleTransactionMsg(peer, msg)

	case *MineBlockMessage:
		sm.handleMineBlockMsg(peer, msg)

	case *GetHeadersMessage:
		sm.handleGetHeadersMsg(peer, msg)

	case *HeadersMessage:
		sm.handleHeadersMsg(peer, msg)

	case *GetBlocksMessage:
		sm.handleGetBlocksMsg(peer, msg)

	case *BlocksMessage:
		sm.handleBlocksMsg(peer, msg)

	case *FilterLoadMessage:
		sm.handleFilterLoadMsg(peer, msg)

	case *FilterAddMessage:
		sm.handleFilterAddMsg(peer, msg)

	case *FilterClearMessage:
		sm.handleFilterClearMsg(peer)

	default:
		logging.CPrint(logging.ERROR, "unknown message type", logging.LogFormat{"typ": reflect.TypeOf(msg)})
	}
}

// Defaults to tcp
func protocolAndAddress(listenAddr string) (string, string) {
	p, address := "tcp", listenAddr
	parts := strings.SplitN(address, "://", 2)
	if len(parts) == 2 {
		p, address = parts[0], parts[1]
	}
	return p, address
}

func (sm *SyncManager) makeNodeInfo(listenerStatus bool) *p2p.NodeInfo {
	nodeInfo := &p2p.NodeInfo{
		PubKey:  sm.privKey.PubKey().Unwrap().(crypto.PubKeyEd25519),
		Moniker: config.Moniker,
		Network: cfg.ChainTag,
		Version: version.Version,
		Other:   []string{strconv.FormatUint(uint64(consensus.DefaultServices), 10)},
	}

	if !sm.sw.IsListening() {
		return nodeInfo
	}

	p2pListener := sm.sw.Listeners()[0]

	// We assume that the rpcListener has the same ExternalAddress.
	// This is probably true because both P2P and RPC listeners use UPnP,
	// except of course if the api is only bound to localhost
	if listenerStatus {
		nodeInfo.ListenAddr = cmn.Fmt("%v:%v", p2pListener.ExternalAddress().IP.String(), p2pListener.ExternalAddress().Port)
	} else {
		nodeInfo.ListenAddr = cmn.Fmt("%v:%v", p2pListener.InternalAddress().IP.String(), p2pListener.InternalAddress().Port)
	}
	return nodeInfo
}

//Start start sync manager service
func (sm *SyncManager) Start() {
	if _, err := sm.sw.Start(); err != nil {
		cmn.Exit(cmn.Fmt("fail on start SyncManager: %v", err))
	}
	// broadcast transactions
	go sm.txBroadcastLoop()
	go sm.minedBroadcastLoop()
	go sm.txSyncLoop()
}

//Stop stop sync manager
func (sm *SyncManager) Stop() {
	close(sm.quitSync)
	sm.sw.Stop()
}

func initDiscover(config *config.Config, priv *crypto.PrivKeyEd25519, port uint16) (*discover.Network, error) {
	addr, err := net.ResolveUDPAddr("udp", net.JoinHostPort("0.0.0.0", strconv.FormatUint(uint64(port), 10)))
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, err
	}

	realaddr := conn.LocalAddr().(*net.UDPAddr)
	ntab, err := discover.ListenUDP(priv, conn, realaddr, path.Join(config.Db.DataDir, "discover.db"), nil)
	if err != nil {
		return nil, err
	}

	// add the seeds node to the discover table
	if config.Network.P2P.Seeds == "" {
		return ntab, nil
	}
	nodes := []*discover.Node{}
	for _, seed := range strings.Split(config.Network.P2P.Seeds, ",") {
		version.Status.AddSeed(seed)
		url := "enode://" + hex.EncodeToString(crypto.Sha256([]byte(seed))) + "@" + seed
		nodes = append(nodes, discover.MustParseNode(url))
	}
	if err = ntab.SetFallbackNodes(nodes); err != nil {
		return nil, err
	}
	return ntab, nil
}

func (sm *SyncManager) minedBroadcastLoop() {
	for {
		select {
		case blockHash := <-sm.newBlockCh:
			block, err := sm.Chain.GetBlockByHash(blockHash)
			if err != nil {
				logging.CPrint(logging.ERROR, "fail on get block by hash",
					logging.LogFormat{"hash": blockHash})
				continue
			}
			if err := sm.peers.broadcastMinedBlock(block); err != nil {
				logging.CPrint(logging.ERROR, "fail on broadcast mined block",
					logging.LogFormat{"err": err})
				continue
			}
		case <-sm.quitSync:
			return
		}
	}
}
