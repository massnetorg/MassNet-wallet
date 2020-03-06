package ifc

import (
	"sort"

	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/database"
	"massnet.org/mass-wallet/database/storage"
	"massnet.org/mass-wallet/wire"
)

type chainFetcher struct {
	db database.Db
}

func NewChainFetcher(db database.Db) ChainFetcher {
	return &chainFetcher{
		db: db,
	}
}

// FetchLastTxUntilHeight returns the last tx before a height, inclusive, of specified hash
func (c *chainFetcher) FetchLastTxUntilHeight(txsha *wire.Hash, height uint64) (*wire.MsgTx, error) {
	rep, err := c.db.FetchTxBySha(txsha)
	if err != nil && err != storage.ErrNotFound && err != database.ErrTxShaMissing {
		return nil, err
	}
	for i := 1; i <= len(rep); i++ {
		if rep[len(rep)-i].Height <= height {
			return rep[len(rep)-i].Tx, nil
		}
	}
	return nil, nil
}

// FetchTxBySha returns the latest tx on best chain of specified hash
func (c *chainFetcher) FetchTxBySha(txsha *wire.Hash) (*wire.MsgTx, error) {
	rep, err := c.db.FetchTxBySha(txsha)
	if err != nil && err != storage.ErrNotFound && err != database.ErrTxShaMissing {
		return nil, err
	}
	if len(rep) == 0 {
		return nil, nil
	}
	return rep[len(rep)-1].Tx, nil
}

// FetchBlockBySha will not returns error if not exists
func (c *chainFetcher) FetchBlockBySha(sha *wire.Hash) (blk *wire.MsgBlock, err error) {
	ret, err := c.db.FetchBlockBySha(sha)
	if err != nil && err != storage.ErrNotFound && err != database.ErrBlockShaMissing {
		return nil, err
	}
	if ret == nil {
		return nil, nil
	}
	return ret.MsgBlock(), nil
}

func (c *chainFetcher) FetchBlockByHeight(height uint64) (blk *wire.MsgBlock, err error) {
	sha, err := c.db.FetchBlockShaByHeight(height)
	if err != nil && err != storage.ErrNotFound {
		return nil, err
	}
	return c.FetchBlockBySha(sha)
}

// FetchBlockHeaderBySha will not returns error if not exists
func (c *chainFetcher) FetchBlockHeaderBySha(sha *wire.Hash) (*wire.BlockHeader, error) {
	header, err := c.db.FetchBlockHeaderBySha(sha)
	if err != nil && err != storage.ErrNotFound && err != database.ErrBlockShaMissing {
		return nil, err
	}
	if header == nil {
		return nil, nil
	}
	return header, nil
}

// FetchBlockHeaderByHeight will not returns error if not exists
func (c *chainFetcher) FetchBlockHeaderByHeight(height uint64) (*wire.BlockHeader, error) {
	sha, err := c.db.FetchBlockShaByHeight(height)
	if err != nil && err != storage.ErrNotFound {
		return nil, err
	}
	if sha == nil {
		return nil, nil
	}
	return c.FetchBlockHeaderBySha(sha)
}

func (c *chainFetcher) FetchScriptHashRelatedTx(scriptHashes [][]byte, start, stop uint64, chainParams *config.Params) (*HeightSortedRelatedTx, error) {
	m, err := c.db.FetchScriptHashRelatedTx(scriptHashes, start, stop, chainParams)
	if err != nil {
		return nil, err
	}
	heights := make([]uint64, 0, len(m))
	for k := range m {
		heights = append(heights, k)
	}
	sort.Slice(heights, func(i, j int) bool {
		return heights[i] < heights[j]
	})
	return &HeightSortedRelatedTx{
		Descending:    false,
		SortedHeights: heights,
		Data:          m,
	}, nil
}

func (c *chainFetcher) CheckScriptHashUsed(scriptHash []byte) (bool, error) {
	return c.db.CheckScriptHashUsed(scriptHash)
}

// FetchTxByLoc will not returns error if not exists
func (c *chainFetcher) FetchTxByLoc(height uint64, loc *wire.TxLoc) (*wire.MsgTx, error) {
	mtx, err := c.db.FetchTxByLoc(height, loc.TxStart, loc.TxLen)
	if err != nil && err != storage.ErrNotFound && err != database.ErrTxShaMissing {
		return nil, err
	}
	if mtx == nil {
		return nil, nil
	}
	return mtx, nil
}

func (c *chainFetcher) NewestSha() (sha *wire.Hash, height uint64, err error) {
	return c.db.NewestSha()
}
