package api

import (
	"fmt"
	"net"

	"github.com/massnetorg/MassNet-wallet/config"
	"github.com/massnetorg/MassNet-wallet/netsync"

	"google.golang.org/grpc"

	pb "github.com/massnetorg/MassNet-wallet/api/proto"
	"github.com/massnetorg/MassNet-wallet/blockchain"
	"github.com/massnetorg/MassNet-wallet/database"
	"github.com/massnetorg/MassNet-wallet/utxo"
	"github.com/massnetorg/MassNet-wallet/wallet"

	"github.com/massnetorg/MassNet-wallet/logging"

	"google.golang.org/grpc/reflection"
)

const (
	// maxProtocolVersion is the max protocol version the server supports.
	//maxProtocolVersion = 70002
	maxMsgSize        = 1024 * 1024 * 64
	GRPCListenAddress = "127.0.0.1"
)

type Server struct {
	rpcServer   *grpc.Server
	db          database.Db
	config      *config.Config
	txMemPool   *blockchain.TxPool
	wallet      *wallet.Wallet
	utxo        *utxo.Utxo
	syncManager *netsync.SyncManager
}

func NewServer(db database.Db, txMemPool *blockchain.TxPool, utxo *utxo.Utxo, wallet *wallet.Wallet, sm *netsync.SyncManager, config *config.Config) (*Server, error) {
	// set the size for receive Msg
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
	}
	s := grpc.NewServer(opts...)
	srv := &Server{
		rpcServer:   s,
		db:          db,
		config:      config,
		txMemPool:   txMemPool,
		wallet:      wallet,
		utxo:        utxo,
		syncManager: sm,
	}
	pb.RegisterApiServiceServer(s, srv)
	// Register reflection service on gRPC server.
	reflection.Register(s)

	logging.CPrint(logging.INFO, "new gRPC server", logging.LogFormat{})
	return srv, nil
}

func (s *Server) Start() error {
	address := fmt.Sprintf("%s%s%s", GRPCListenAddress, ":", s.config.Network.API.APIPortGRPC)
	listen, err := net.Listen("tcp", address)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to start tcp listener", logging.LogFormat{"port": s.config.Network.API.APIPortGRPC, "error": err})
		return err
	}
	go s.rpcServer.Serve(listen)
	logging.CPrint(logging.INFO, "gRPC server start", logging.LogFormat{"port": s.config.Network.API.APIPortGRPC})
	return nil
}

func (s *Server) Stop() {
	s.rpcServer.Stop()
	logging.CPrint(logging.INFO, "gRPC server stop", logging.LogFormat{})
}

func (s *Server) RunGateway() {
	go func() {
		if err := Run(s.config.Network.API.APIPortHttp, s.config.Network.API.APIPortGRPC); err != nil {
			logging.CPrint(logging.ERROR, "failed to start gateway", logging.LogFormat{"port": s.config.Network.API.APIPortHttp, "error": err})
		}
	}()
	logging.CPrint(logging.INFO, "gRPC-gateway start", logging.LogFormat{"port": s.config.Network.API.APIPortHttp})
}
