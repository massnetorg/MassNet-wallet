package ifc

import (
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/wire"
)

type BlockListener interface {
	OnBlockConnected(newBlock *wire.MsgBlock) error
}

type TransactionListener interface {
	OnTransactionReceived(tx *wire.MsgTx) error
}

type HeightSortedRelatedTx struct {
	Descending    bool
	SortedHeights []uint64
	Data          map[uint64][]*wire.TxLoc
}

func (h *HeightSortedRelatedTx) Heights() []uint64 {
	return h.SortedHeights[:]
}

func (h *HeightSortedRelatedTx) Get(height uint64) []*wire.TxLoc {
	return h.Data[height]
}

type ChainFetcher interface {
	FetchLastTxUntilHeight(txsha *wire.Hash, height uint64) (*wire.MsgTx, error)

	FetchTxBySha(txsha *wire.Hash) (*wire.MsgTx, error)

	FetchTxByLoc(height uint64, loc *wire.TxLoc) (*wire.MsgTx, error)

	FetchBlockBySha(sha *wire.Hash) (blk *wire.MsgBlock, err error)

	FetchBlockByHeight(uint64) (blk *wire.MsgBlock, err error)

	FetchBlockHeaderByHeight(uint64) (*wire.BlockHeader, error)

	FetchBlockHeaderBySha(sha *wire.Hash) (bh *wire.BlockHeader, err error)

	FetchScriptHashRelatedTx(scriptHashes [][]byte, start, stop uint64, chainParams *config.Params) (*HeightSortedRelatedTx, error)

	CheckScriptHashUsed(scriptHash []byte) (bool, error)

	NewestSha() (sha *wire.Hash, height uint64, err error)
}
