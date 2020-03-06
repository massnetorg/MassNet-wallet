package masswallet

import (
	"massnet.org/mass-wallet/blockchain"
	"massnet.org/mass-wallet/database"
	"massnet.org/mass-wallet/errors"
	"massnet.org/mass-wallet/massutil"
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
	WalletID     string
	Type         uint32
	Remarks      string
	Ready        bool
	SyncedHeight uint64
}

type WalletInfo struct {
	ChainID          string          `json:"chain_id"`
	WalletID         string          `json:"wallet_id"`
	Type             uint32          `json:"type"`
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
}
