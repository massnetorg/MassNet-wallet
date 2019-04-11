package api

import (
	"context"

	pb "github.com/massnetorg/MassNet-wallet/api/proto"
	"github.com/massnetorg/MassNet-wallet/logging"
)

func (s *Server) GetClientStatus(ctx context.Context, in *pb.GetClientStatusRequest) (*pb.GetClientStatusResponse, error) {
	logging.CPrint(logging.INFO, "a request is received to query the status of client", logging.LogFormat{})

	resp := &pb.GetClientStatusResponse{
		PeerListening:   s.syncManager.Switch().IsListening(),
		Syncing:         !s.syncManager.IsCaughtUp(),
		LocalBestHeight: s.syncManager.Chain.BestBlockHeight(),
		ChainID:         s.syncManager.Chain.ChainID().String(),
	}

	if bestPeer := s.syncManager.BestPeer(); bestPeer != nil {
		resp.KnownBestHeight = bestPeer.Height
	}
	if resp.LocalBestHeight > resp.KnownBestHeight {
		resp.KnownBestHeight = resp.LocalBestHeight
	}

	var outCount, inCount uint32
	resp.Peers = &pb.GetClientStatusResponsePeerList{
		Outbound: make([]*pb.GetClientStatusResponsePeerInfo, 0),
		Inbound:  make([]*pb.GetClientStatusResponsePeerInfo, 0),
		Other:    make([]*pb.GetClientStatusResponsePeerInfo, 0),
	}
	for _, info := range s.syncManager.GetPeerInfos() {
		peer := &pb.GetClientStatusResponsePeerInfo{
			Id:      info.ID,
			Address: info.RemoteAddr,
		}
		if info.IsOutbound {
			outCount++
			peer.Direction = "outbound"
			resp.Peers.Outbound = append(resp.Peers.Outbound, peer)
			continue
		}
		inCount++
		peer.Direction = "inbound"
		resp.Peers.Inbound = append(resp.Peers.Inbound, peer)
	}
	resp.PeerCount = &pb.GetClientStatusResponsePeerCountInfo{Total: outCount + inCount, Outbound: outCount, Inbound: inCount}

	logging.CPrint(logging.INFO, "GetClientStatus completed", logging.LogFormat{})
	return resp, nil
}
