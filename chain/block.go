package chain

import (
	"github.com/massnetorg/MassNet-wallet/blockchain"
	"github.com/massnetorg/MassNet-wallet/errors"
	"github.com/massnetorg/MassNet-wallet/massutil"
	"github.com/massnetorg/MassNet-wallet/wire"
)

var (
	// ErrBadBlock is returned when a block is invalid.
	ErrBadBlock = errors.New("invalid block")
	// ErrBadStateRoot is returned when the computed assets merkle root
	// disagrees with the one declared in a block header.
	ErrBadStateRoot = errors.New("invalid state merkle root")
)

// GetBlockByHash return a block by given hash
func (c *Chain) GetBlockByHash(hash *wire.Hash) (*massutil.Block, error) {
	return c.blockChain.GetBlockByHash(hash)
}

// GetBlockByHeight return a block header by given height
func (c *Chain) GetBlockByHeight(height uint64) (*massutil.Block, error) {
	return c.blockChain.GetBlockByHeight(height)
}

// GetHeaderByHash return a block header by given hash
func (c *Chain) GetHeaderByHash(hash *wire.Hash) (*wire.BlockHeader, error) {
	blk, err := c.blockChain.GetBlockByHash(hash)
	if err != nil {
		return nil, err
	}
	return &blk.MsgBlock().Header, nil
}

// GetHeaderByHeight return a block header by given height
func (c *Chain) GetHeaderByHeight(height uint64) (*wire.BlockHeader, error) {
	blk, err := c.blockChain.GetBlockByHeight(height)
	if err != nil {
		return nil, err
	}
	return &blk.MsgBlock().Header, nil
}

type processBlockResponse struct {
	isOrphan bool
	err      error
}

type processBlockMsg struct {
	block *massutil.Block
	reply chan processBlockResponse
}

// ProcessBlock is the entry for chain update
func (c *Chain) ProcessBlock(block *massutil.Block) (bool, error) {
	reply := make(chan processBlockResponse, 1)
	c.processBlockCh <- &processBlockMsg{block: block, reply: reply}
	response := <-reply
	return response.isOrphan, response.err
}

func (c *Chain) blockProcesser() {
	for msg := range c.processBlockCh {
		isOrphan, err := c.processBlock(msg.block)
		msg.reply <- processBlockResponse{isOrphan: isOrphan, err: err}
	}
}

// ProcessBlock is the entry for handle block insert
func (c *Chain) processBlock(block *massutil.Block) (bool, error) {
	return c.blockChain.ProcessBlock(block, c.timeSource, blockchain.BFNone)
}
