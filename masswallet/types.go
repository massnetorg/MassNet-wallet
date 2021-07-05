package masswallet

import (
	"errors"

	"github.com/massnetorg/mass-core/blockchain"
	"github.com/massnetorg/mass-core/database"
	"github.com/massnetorg/mass-core/massutil"
	"github.com/massnetorg/mass-core/netsync"
	"massnet.org/mass-wallet/masswallet/txmgr"
)

var (
	ErrInvalidTx         = errors.New("invalid transaction")
	ErrBothBinding       = errors.New("both input and output are binding")
	ErrMaybeChainRevoked = errors.New("maybe chain revoked")
)

type WalletBalance struct {
	Total               massutil.Amount
	Spendable           massutil.Amount
	WithdrawableStaking massutil.Amount
	WithdrawableBinding massutil.Amount
	WalletID            string
}

type AddressBalance struct {
	Address             string
	Total               massutil.Amount
	Spendable           massutil.Amount
	WithdrawableStaking massutil.Amount
	WithdrawableBinding massutil.Amount
}

type WalletSummary struct {
	WalletID string
	Type     uint32
	Version  uint8
	Remarks  string
	Status   *txmgr.WalletStatus
}

type WalletInfo struct {
	ChainID          string          `json:"chain_id"`
	WalletID         string          `json:"wallet_id"`
	Type             uint32          `json:"type"`
	Version          uint8           `json:"version"`
	TotalBalance     massutil.Amount `json:"total_balance"`
	ExternalKeyCount int32           `json:"external_key_count"`
	InternalKeyCount int32           `json:"internal_key_count"`
	Remarks          string          `json:"remarks"`
}

type UnspentDetail struct {
	TxId           string          `json:"tx_id"`
	Vout           uint32          `json:"vout"`
	Amount         massutil.Amount `json:"amount"`
	BlockHeight    uint64          `json:"block_height"`
	Maturity       uint32          `json:"maturity"`
	Confirmations  uint32          `json:"confirmations"`
	SpentByUnmined bool            `json:"spent_by_unmined"`
	// IsCoinbase bool `json"is_coinbase"`
}

type Server interface {
	Blockchain() *blockchain.Blockchain
	ChainDB() database.Db
	TxMemPool() *blockchain.TxPool
	SyncManager() *netsync.SyncManager
}
