package chain

import (
	"sync"

	"math/big"

	"github.com/massnetorg/MassNet-wallet/blockchain"
	"github.com/massnetorg/MassNet-wallet/wire"
)

const maxProcessBlockChSize = 1024

type Chain struct {
	blockChain     *blockchain.BlockChain
	processBlockCh chan *processBlockMsg
	timeSource     blockchain.MedianTimeSource
	cond           sync.Cond
}

// NewChain returns a new Chain using store as the underlying storage.
func NewChain(bc *blockchain.BlockChain, timeSource blockchain.MedianTimeSource) (*Chain, error) {
	c := &Chain{
		blockChain:     bc,
		processBlockCh: make(chan *processBlockMsg, maxProcessBlockChSize),
		timeSource:     timeSource,
	}
	c.cond.L = new(sync.Mutex)

	go c.blockProcesser()
	return c, nil
}

// BestBlockHeight returns the current height of the blockchain.
func (c *Chain) BestBlockHeight() uint64 {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	return c.blockChain.GetBestChainHeight()
}

// BestBlockHeader returns the chain tail block
func (c *Chain) BestBlockHeader() *wire.BlockHeader {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	return c.blockChain.GetBestChainHeader()
}

func (c *Chain) BestBlockHash() *wire.Hash {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	return c.blockChain.GetBestChainHash()
}

// InMainChain checks wheather a block is in the main chain
func (c *Chain) InMainChain(hash wire.Hash) bool {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	return c.blockChain.InMainChain(hash)
}

// CalcNextBits return the seed for the given block
func (c *Chain) CalcNextRequiredDifficulty() (*big.Int, error) {
	return nil, nil
}

func (c *Chain) CalcNextChallenge() (*big.Int, error) {
	return nil, nil
}

// GetTxPool return chain txpool.
func (c *Chain) GetTxPool() *blockchain.TxPool {
	return c.blockChain.TxPool
}

func (c *Chain) ChainID() *wire.Hash {
	return c.blockChain.ChainID()
}
