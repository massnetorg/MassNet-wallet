package api

import (
	"encoding/hex"
	"strconv"

	"github.com/golang/protobuf/ptypes/empty"
	"golang.org/x/net/context"
	"google.golang.org/grpc/status"

	"github.com/massnetorg/mass-core/logging"
	"github.com/massnetorg/mass-core/massutil"
	"github.com/massnetorg/mass-core/wire"

	pb "massnet.org/mass-wallet/api/proto"
	"massnet.org/mass-wallet/masswallet/keystore"
)

// no error for return
func (s *APIServer) GetClientStatus(ctx context.Context, in *empty.Empty) (*pb.GetClientStatusResponse, error) {
	logging.CPrint(logging.INFO, "api: GetClientStatus", logging.LogFormat{})

	height, err := s.massWallet.SyncedTo()
	if err != nil {
		return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
	}

	resp := &pb.GetClientStatusResponse{
		PeerListening:    s.node.SyncManager().Switch().IsListening(),
		Syncing:          !s.node.SyncManager().IsCaughtUp(),
		LocalBestHeight:  s.node.Blockchain().BestBlockHeight(),
		ChainId:          s.node.Blockchain().ChainID().String(),
		WalletSyncHeight: height,
	}

	if bestPeer := s.node.SyncManager().BestPeer(); bestPeer != nil {
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
	for _, info := range s.node.SyncManager().GetPeerInfos() {
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

	logging.CPrint(logging.INFO, "api: GetClientStatus completed", logging.LogFormat{})
	return resp, nil
}

func (s *APIServer) QuitClient(ctx context.Context, in *empty.Empty) (*pb.QuitClientResponse, error) {
	logging.CPrint(logging.INFO, "api: QuitClient completed", logging.LogFormat{})
	defer func() {
		go s.quitClient()
	}()
	return &pb.QuitClientResponse{
		Code: 200,
		Msg:  "wait for client quitting process",
	}, nil
}

func (s *APIServer) SignRawTransaction(ctx context.Context, in *pb.SignRawTransactionRequest) (*pb.SignRawTransactionResponse, error) {
	logging.CPrint(logging.INFO, "api: SignRawTransaction", logging.LogFormat{})

	if len(in.RawTx) == 0 {
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidTxHex], logging.LogFormat{
			"err": in.RawTx})
		st := status.New(ErrAPIInvalidTxHex, ErrCode[ErrAPIInvalidTxHex])
		return nil, st.Err()
	}
	serializedTx, err := decodeHexStr(in.RawTx)
	if err != nil {
		logging.CPrint(logging.ERROR, "Error in decodeHexStr", logging.LogFormat{
			"err": err,
		})
		st := status.New(ErrAPIInvalidTxHex, ErrCode[ErrAPIInvalidTxHex])
		return nil, st.Err()
	}
	var tx wire.MsgTx
	err = tx.SetBytes(serializedTx, wire.Packet)
	if err != nil {
		logging.CPrint(logging.ERROR, "Failed to decode tx", logging.LogFormat{"err": err.Error()})
		st := status.New(ErrAPIInvalidTxHex, ErrCode[ErrAPIInvalidTxHex])
		return nil, st.Err()
	}

	err = checkPassLen(in.Passphrase)
	if err != nil {
		s.massWallet.ClearUsedUTXOMark(&tx)
		return nil, err
	}

	//check in.Flags
	var flag string
	if len(in.Flags) == 0 {
		flag = "ALL"
	} else {
		flag = in.Flags
	}

	logging.CPrint(logging.INFO, "get tx", logging.LogFormat{"tx input count": len(tx.TxIn), "tx output count": len(tx.TxOut)})

	bufBytes, err := s.massWallet.SignRawTx([]byte(in.Passphrase), flag, &tx)
	if err != nil {
		s.massWallet.ClearUsedUTXOMark(&tx)
		return nil, convertResponseError(err)
	}

	logging.CPrint(logging.INFO, "api: SignRawTransaction completed", logging.LogFormat{})

	return &pb.SignRawTransactionResponse{
		Hex:      hex.EncodeToString(bufBytes),
		Complete: true,
	}, nil
}

func decodeHexStr(hexStr string) ([]byte, error) {
	if len(hexStr)%2 != 0 {
		hexStr = "0" + hexStr
	}
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		logging.CPrint(logging.ERROR, "Hex string decode failed", logging.LogFormat{"err": err.Error()})
		return nil, err
	}
	return decoded, nil
}

func (s *APIServer) CreateAddress(ctx context.Context, in *pb.CreateAddressRequest) (*pb.CreateAddressResponse, error) {
	logging.CPrint(logging.INFO, "api: CreateAddress", logging.LogFormat{"version": in.Version})

	addressClass := uint16(in.Version)
	if !massutil.IsValidAddressClass(addressClass) {
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidVersion], logging.LogFormat{
			"version": in.Version,
		})
		st := status.New(ErrAPIInvalidVersion, ErrCode[ErrAPIInvalidVersion])
		return nil, st.Err()
	}

	ads, err := s.massWallet.GetAddresses(addressClass)
	if err != nil {
		logging.CPrint(logging.ERROR, "GetAddresses failed", logging.LogFormat{"err": err})
		return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
	}

	count := 0
	for _, ad := range ads {
		if !ad.Used {
			count++
		}
	}
	if addressClass == massutil.AddressClassWitnessStaking &&
		count >= int(s.config.Wallet.Settings.MaxUnusedStakingAddress) {
		return nil, status.New(ErrAPIUnusedAddressLimit, ErrCode[ErrAPIUnusedAddressLimit]).Err()
	}
	if addressClass == massutil.AddressClassWitnessV0 &&
		count >= int(s.config.Wallet.Settings.AddressGapLimit-s.config.Wallet.Settings.MaxUnusedStakingAddress) {
		return nil, status.New(ErrAPIUnusedAddressLimit, ErrCode[ErrAPIUnusedAddressLimit]).Err()
	}

	address, err := s.massWallet.NewAddress(addressClass)
	if err != nil {
		logging.CPrint(logging.ERROR, "new address error", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}

	logging.CPrint(logging.INFO, "api: CreateAddress completed", logging.LogFormat{})
	return &pb.CreateAddressResponse{Address: address}, nil
}

func (s *APIServer) GetAddresses(ctx context.Context, in *pb.GetAddressesRequest) (*pb.GetAddressesResponse, error) {
	logging.CPrint(logging.INFO, "api: GetAddresses", logging.LogFormat{"version": in.Version})
	addressClass := uint16(in.Version)
	if !massutil.IsValidAddressClass(addressClass) {
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidVersion], logging.LogFormat{
			"version": in.Version,
		})
		return nil, status.New(ErrAPIInvalidVersion, ErrCode[ErrAPIInvalidVersion]).Err()
	}

	ads, err := s.massWallet.GetAddresses(addressClass)
	if err != nil {
		logging.CPrint(logging.ERROR, "GetAddresses failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
		}
		return nil, cvtErr
	}

	details := make([]*pb.GetAddressesResponse_AddressDetail, 0)
	for _, ad := range ads {
		pbAd := &pb.GetAddressesResponse_AddressDetail{
			Address:    ad.Address,
			Version:    int32(ad.AddressClass),
			Used:       ad.Used,
			StdAddress: ad.StdAddress,
		}
		details = append(details, pbAd)
	}
	reps := &pb.GetAddressesResponse{
		Details: details,
	}

	logging.CPrint(logging.INFO, "api: GetAddress completed", logging.LogFormat{
		"addressNumber": len(details),
	})
	return reps, nil
}

func (s *APIServer) ValidateAddress(ctx context.Context, in *pb.ValidateAddressRequest) (*pb.ValidateAddressResponse, error) {
	logging.CPrint(logging.INFO, "api: ValidateAddress", logging.LogFormat{"address": in.Address})
	err := checkAddressLen(in.Address)
	if err != nil {
		return nil, err
	}

	witAddr, isMine, err := s.massWallet.IsAddressInCurrent(in.Address)
	if err != nil {
		logging.CPrint(logging.ERROR, "failed to check validate address", logging.LogFormat{
			"err":     err,
			"address": in.Address,
		})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIInvalidAddress, ErrCode[ErrAPIInvalidAddress]).Err()
		}
		return nil, cvtErr
	}

	var ver uint16
	if massutil.IsWitnessV0Address(witAddr) {
		ver = massutil.AddressClassWitnessV0
	} else if massutil.IsWitnessStakingAddress(witAddr) {
		ver = massutil.AddressClassWitnessStaking
	} else {
		return &pb.ValidateAddressResponse{}, nil
	}
	logging.CPrint(logging.INFO, "api: ValidateAddress completed", logging.LogFormat{"address": in.Address})
	return &pb.ValidateAddressResponse{
		IsValid: true,
		IsMine:  isMine,
		Address: witAddr.EncodeAddress(),
		Version: int32(ver),
	}, nil
}

func (s *APIServer) GetWalletBalance(ctx context.Context, in *pb.GetWalletBalanceRequest) (*pb.GetWalletBalanceResponse, error) {
	logging.CPrint(logging.INFO, "api: GetWalletBalance", logging.LogFormat{"params": in})

	// uint32 is the set of all unsigned 32-bit integers.
	// Range: 0 through 4294967295. (in.RequiredConfirmations int32)
	// int32 is the set of all signed 32-bit integers.
	// Range: -2147483648 through 2147483647.
	if in.RequiredConfirmations < 0 {
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidParameter], logging.LogFormat{
			"confs": in.RequiredConfirmations,
		})
		return nil, status.New(ErrAPIInvalidParameter, ErrCode[ErrAPIInvalidParameter]).Err()
	}

	bal, err := s.massWallet.WalletBalance(uint32(in.RequiredConfirmations), in.Detail)
	if err != nil {
		logging.CPrint(logging.ERROR, "WalletBalance failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
		}
		return nil, cvtErr
	}

	total, err := checkFormatAmount(bal.Total)
	if err != nil {
		return nil, err
	}
	spendable, err := checkFormatAmount(bal.Spendable)
	if err != nil {
		return nil, err
	}
	withdrawableStaking, err := checkFormatAmount(bal.WithdrawableStaking)
	if err != nil {
		return nil, err
	}
	withdrawableBinding, err := checkFormatAmount(bal.WithdrawableBinding)
	if err != nil {
		return nil, err
	}
	logging.CPrint(logging.INFO, "api: GetWalletBalance completed", logging.LogFormat{"walletId": bal.WalletID})
	return &pb.GetWalletBalanceResponse{
		WalletId: bal.WalletID,
		Total:    total,
		Detail: &pb.GetWalletBalanceResponse_Detail{
			Spendable:           spendable,
			WithdrawableStaking: withdrawableStaking,
			WithdrawableBinding: withdrawableBinding,
		},
	}, nil
}

func (s *APIServer) GetAddressBalance(ctx context.Context, in *pb.GetAddressBalanceRequest) (*pb.GetAddressBalanceResponse, error) {
	logging.CPrint(logging.INFO, "api: GetAddressBalance", logging.LogFormat{"params": in})

	if in.RequiredConfirmations < 0 {
		logging.CPrint(logging.ERROR, ErrCode[ErrAPIInvalidParameter], logging.LogFormat{
			"confs": in.RequiredConfirmations,
		})
		return nil, status.New(ErrAPIInvalidParameter, ErrCode[ErrAPIInvalidParameter]).Err()
	}

	for _, addr := range in.Addresses {
		err := checkAddressLen(addr)
		if err != nil {
			return nil, err
		}
	}

	bals, err := s.massWallet.AddressBalance(uint32(in.RequiredConfirmations), in.Addresses)
	if err != nil {
		logging.CPrint(logging.ERROR, "AddressBalance failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
		}
		return nil, cvtErr
	}
	balances := make([]*pb.AddressAndBalance, 0)
	for _, bal := range bals {
		total, err := checkFormatAmount(bal.Total)
		if err != nil {
			return nil, err
		}
		spendable, err := checkFormatAmount(bal.Spendable)
		if err != nil {
			return nil, err
		}
		withdrawableStaking, err := checkFormatAmount(bal.WithdrawableStaking)
		if err != nil {
			return nil, err
		}
		withdrawableBinding, err := checkFormatAmount(bal.WithdrawableBinding)
		if err != nil {
			return nil, err
		}
		balances = append(balances, &pb.AddressAndBalance{
			Address:             bal.Address,
			Total:               total,
			Spendable:           spendable,
			WithdrawableStaking: withdrawableStaking,
			WithdrawableBinding: withdrawableBinding,
		})
	}

	logging.CPrint(logging.INFO, "api: GetAddressBalance completed", logging.LogFormat{})
	return &pb.GetAddressBalanceResponse{
		Balances: balances,
	}, nil
}

func (s *APIServer) UseWallet(ctx context.Context, in *pb.UseWalletRequest) (*pb.UseWalletResponse, error) {
	logging.CPrint(logging.INFO, "api: UseWallet", logging.LogFormat{"walletId": in.WalletId})

	err := checkWalletIdLen(in.WalletId)
	if err != nil {
		return nil, err
	}

	info, err := s.massWallet.UseWallet(in.WalletId)
	if err != nil {
		logging.CPrint(logging.ERROR, "UseWallet failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}
	totalBalance, err := checkFormatAmount(info.TotalBalance)
	if err != nil {
		return nil, err
	}

	logging.CPrint(logging.INFO, "api: UseWallet completed", logging.LogFormat{})
	return &pb.UseWalletResponse{
		ChainId:          info.ChainID,
		WalletId:         info.WalletID,
		Type:             info.Type,
		Version:          uint32(info.Version),
		TotalBalance:     totalBalance,
		ExternalKeyCount: info.ExternalKeyCount,
		InternalKeyCount: info.InternalKeyCount,
		Remarks:          info.Remarks,
	}, nil
}

func (s *APIServer) Wallets(ctx context.Context, in *empty.Empty) (*pb.WalletsResponse, error) {
	logging.CPrint(logging.INFO, "api: Wallets", logging.LogFormat{})

	wallets := make([]*pb.WalletsResponse_WalletSummary, 0)
	summaries, err := s.massWallet.Wallets()
	if err != nil {
		logging.CPrint(logging.ERROR, "Wallets failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
		}
		return nil, cvtErr
	}
	for _, summary := range summaries {
		ws := &pb.WalletsResponse_WalletSummary{
			WalletId: summary.WalletID,
			Type:     summary.Type,
			Version:  uint32(summary.Version),
			Remarks:  summary.Remarks,
		}
		switch {
		case summary.Status.IsRemoved():
			ws.Status = walletStatusRemoving
			ws.StatusMsg = walletStatusMsg[walletStatusRemoving]
		case summary.Status.Ready():
			ws.Status = walletStatusReady
			ws.StatusMsg = walletStatusMsg[walletStatusReady]
		default:
			ws.Status = walletStatusImporting
			ws.StatusMsg = strconv.Itoa(int(summary.Status.SyncedHeight))
		}
		wallets = append(wallets, ws)
	}

	logging.CPrint(logging.INFO, "api: Wallets completed", logging.LogFormat{})
	return &pb.WalletsResponse{
		Wallets: wallets,
	}, nil
}

func (s *APIServer) GetUtxo(ctx context.Context, in *pb.GetUtxoRequest) (*pb.GetUtxoResponse, error) {
	logging.CPrint(logging.INFO, "api: GetUtxo", logging.LogFormat{"addresses": in.Addresses})

	for _, addr := range in.Addresses {
		err := checkAddressLen(addr)
		if err != nil {
			return nil, err
		}
	}

	m, err := s.massWallet.GetUtxo(in.Addresses)
	if err != nil {
		logging.CPrint(logging.ERROR, "GetUtxo failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
		}
		return nil, cvtErr
	}
	list := make([]*pb.AddressUTXO, 0)
	for k, v := range m {
		utxos := make([]*pb.UTXO, 0)
		for _, item := range v {
			amt, err := checkFormatAmount(item.Amount)
			if err != nil {
				return nil, err
			}
			utxos = append(utxos, &pb.UTXO{
				TxId:           item.TxId,
				Vout:           item.Vout,
				Amount:         amt,
				BlockHeight:    uint64(item.BlockHeight),
				Maturity:       item.Maturity,
				Confirmations:  item.Confirmations,
				SpentByUnmined: item.SpentByUnmined,
			})
		}
		list = append(list, &pb.AddressUTXO{
			Address: k,
			Utxos:   utxos,
		})
	}

	logging.CPrint(logging.INFO, "api: GetUtxo completed", logging.LogFormat{})
	return &pb.GetUtxoResponse{
		AddressUtxos: list,
	}, nil
}

func (s *APIServer) ImportWallet(ctx context.Context, in *pb.ImportWalletRequest) (*pb.ImportWalletResponse, error) {
	logging.CPrint(logging.INFO, "api: ImportWallet", logging.LogFormat{})
	err := checkPassLen(in.Passphrase)
	if err != nil {
		return nil, err
	}

	ws, err := s.massWallet.ImportWallet(in.Keystore, in.Passphrase)
	if err != nil {
		logging.CPrint(logging.ERROR, "ImportWallet failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}

	logging.CPrint(logging.INFO, "api: ImportWallet completed",
		logging.LogFormat{
			"walletId": ws.WalletID,
			"remarks":  ws.Remarks,
		})
	return &pb.ImportWalletResponse{
		Ok:       true,
		WalletId: ws.WalletID,
		Type:     ws.Type,
		Version:  uint32(ws.Version),
		Remarks:  ws.Remarks,
	}, nil
}

func (s *APIServer) ImportMnemonic(ctx context.Context, in *pb.ImportMnemonicRequest) (*pb.ImportWalletResponse, error) {
	logging.CPrint(logging.INFO, "api: ImportMnemonic", logging.LogFormat{"remarks": in.Remarks})

	err := checkMnemonicLen(in.Mnemonic)
	if err != nil {
		return nil, err
	}

	remarks := checkRemarksLen(in.Remarks)

	err = checkPassLen(in.Passphrase)
	if err != nil {
		return nil, err
	}

	params := &keystore.WalletParams{
		Mnemonic:          in.Mnemonic,
		PrivatePassphrase: []byte(in.Passphrase),
		Remarks:           remarks,
		ExternalIndex:     in.ExternalIndex,
		InternalIndex:     in.InternalIndex,
		AddressGapLimit:   s.config.Wallet.Settings.AddressGapLimit,
	}
	ws, err := s.massWallet.ImportWalletWithMnemonic(params)
	if err != nil {
		logging.CPrint(logging.ERROR, "ImportWalletWithMnemonic failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}

	logging.CPrint(logging.INFO, "api: ImportWalletWithMnemonic completed",
		logging.LogFormat{
			"wallet id": ws.WalletID,
		})
	return &pb.ImportWalletResponse{
		Ok:       true,
		WalletId: ws.WalletID,
		Type:     ws.Type,
		Version:  uint32(ws.Version),
		Remarks:  ws.Remarks,
	}, nil
}

func (s *APIServer) CreateWallet(ctx context.Context, in *pb.CreateWalletRequest) (*pb.CreateWalletResponse, error) {
	logging.CPrint(logging.INFO, "api: CreateWallet",
		logging.LogFormat{
			"bit size": in.BitSize,
			"remarks":  in.Remarks,
		})

	err := checkPassLen(in.Passphrase)
	if err != nil {
		return nil, err
	}

	remarks := checkRemarksLen(in.Remarks)

	name, mnemonic, version, err := s.massWallet.CreateWallet(in.Passphrase, remarks, int(in.BitSize))
	if err != nil {
		logging.CPrint(logging.ERROR, "CreateWallet failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}

	logging.CPrint(logging.INFO, "api: CreateWallet completed", logging.LogFormat{})
	return &pb.CreateWalletResponse{
		WalletId: name,
		Mnemonic: mnemonic,
		Version:  uint32(version),
	}, nil
}

func (s *APIServer) ExportWallet(ctx context.Context, in *pb.ExportWalletRequest) (*pb.ExportWalletResponse, error) {
	logging.CPrint(logging.INFO, "api: ExportWallet", logging.LogFormat{"walletId": in.WalletId})

	err := checkWalletIdLen(in.WalletId)
	if err != nil {
		return nil, err
	}

	err = checkPassLen(in.Passphrase)
	if err != nil {
		return nil, err
	}

	keystoreJSON, err := s.massWallet.ExportWallet(in.WalletId, in.Passphrase)
	if err != nil {
		logging.CPrint(logging.ERROR, "ExportWallet failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
		}
		return nil, cvtErr
	}

	logging.CPrint(logging.INFO, "api: ExportWallet completed", logging.LogFormat{})
	return &pb.ExportWalletResponse{
		Keystore: keystoreJSON,
	}, nil
}

func (s *APIServer) RemoveWallet(ctx context.Context, in *pb.RemoveWalletRequest) (*pb.RemoveWalletResponse, error) {
	logging.CPrint(logging.INFO, "api: RemoveWallet", logging.LogFormat{"walletId": in.WalletId})

	err := checkWalletIdLen(in.WalletId)
	if err != nil {
		return nil, err
	}

	err = checkPassLen(in.Passphrase)
	if err != nil {
		return nil, err
	}

	err = s.massWallet.RemoveWallet(in.WalletId, in.Passphrase)
	if err != nil {
		logging.CPrint(logging.ERROR, "RemoveWallet failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}

	logging.CPrint(logging.INFO, "api: RemoveWallet completed", logging.LogFormat{})
	return &pb.RemoveWalletResponse{
		Ok: true,
	}, nil
}

/* func (s *APIServer) ChangePrivPassphrase(ctx context.Context, in *pb.ChangePrivPassphraseRequest) (*pb.ChangePrivPassphraseResponse, error) {
	logging.CPrint(logging.INFO, "api: ChangePrivPassphrase")

	err := checkPassLen(in.OldPassphrase)
	if err != nil {
		return nil, err
	}

	err = checkPassLen(in.NewPassphrase)
	if err != nil {
		return nil, status.New(ErrAPIInvalidNewPassphrase, ErrCode[ErrAPIInvalidNewPassphrase]).Err()
	}

	err = s.massWallet.ChangePrivPassphrase(in.OldPassphrase, in.NewPassphrase)
	if err != nil {
		logging.CPrint(logging.ERROR, "ChangePrivPassphrase failed", logging.LogFormat{
			"err": err,
		})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIAbnormalData, ErrCode[ErrAPIAbnormalData]).Err()
		}
		return nil, cvtErr
	}
	return &pb.ChangePrivPassphraseResponse{Ok: true}, nil
} */

func (s *APIServer) GetWalletMnemonic(ctw context.Context, in *pb.GetWalletMnemonicRequest) (*pb.GetWalletMnemonicResponse, error) {
	logging.CPrint(logging.INFO, "api: GetWalletMnemonic", logging.LogFormat{"walletId": in.WalletId})

	err := checkWalletIdLen(in.WalletId)
	if err != nil {
		return nil, err
	}

	err = checkPassLen(in.Passphrase)
	if err != nil {
		return nil, err
	}

	mnemonic, version, err := s.massWallet.GetMnemonic(in.WalletId, in.Passphrase)
	if err != nil {
		logging.CPrint(logging.ERROR, "GetMnemonic failed", logging.LogFormat{"err": err})
		cvtErr := convertResponseError(err)
		if cvtErr == apiUnknownError {
			return nil, status.New(ErrAPIQueryDataFailed, ErrCode[ErrAPIQueryDataFailed]).Err()
		}
		return nil, cvtErr
	}
	logging.CPrint(logging.INFO, "api: GetWalletMnemonic completed", logging.LogFormat{})
	return &pb.GetWalletMnemonicResponse{
		Mnemonic: mnemonic,
		Version:  uint32(version),
	}, nil
}
