// Modified for MassNet
// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"

	"github.com/massnetorg/MassNet-wallet/logging"
	"github.com/massnetorg/MassNet-wallet/massutil"
	"github.com/massnetorg/MassNet-wallet/wire"
)

// BehaviorFlags is a bitmask defining tweaks to the normal behavior when
// performing chain processing and consensus rules checks.
type BehaviorFlags uint32

const (
	BFFastAdd BehaviorFlags = 1 << iota

	BFDryRun

	BFNone BehaviorFlags = 0
)

// blockExists determines whether a block with the given hash exists either in
// the main chain or any side chains.
func (b *BlockChain) blockExists(hash *wire.Hash) (bool, error) {
	if _, ok := b.index[*hash]; ok {
		return true, nil
	}

	return b.db.ExistsSha(hash)
}

// processOrphans determines if there are any orphans which depend on the passed
// block hash (they are no longer orphans if true) and potentially accepts them.
// It repeats the process for the newly accepted blocks (to detect further
// orphans which may no longer be orphans) until there are no more.
//
// The flags do not modify the behavior of this function directly, however they
// are needed to pass along to maybeAcceptBlock.
func (b *BlockChain) processOrphans(hash *wire.Hash, flags BehaviorFlags) error {
	processHashes := make([]*wire.Hash, 0, 10)
	processHashes = append(processHashes, hash)
	for len(processHashes) > 0 {
		processHash := processHashes[0]
		processHashes[0] = nil // Prevent GC leak.
		processHashes = processHashes[1:]

		for i := 0; i < len(b.orphanBlockPool.prevOrphans[*processHash]); i++ {
			orphan := b.orphanBlockPool.prevOrphans[*processHash][i]
			if orphan == nil {
				logging.CPrint(logging.WARN, "Found a nil entry at index i in the "+
					"orphan dependency list for block", logging.LogFormat{
					"index": i,
					"block": processHash,
				})
				continue
			}

			orphanHash := orphan.block.Hash()
			b.orphanBlockPool.removeOrphanBlock(orphan)
			i--

			err := b.maybeAcceptBlock(orphan.block, flags)
			if err != nil {
				return err
			}

			processHashes = append(processHashes, orphanHash)
		}
	}
	return nil
}

// ProcessBlock is the main workhorse for handling insertion of new blocks into
// the block chain.  It includes functionality such as rejecting duplicate
// blocks, ensuring blocks follow all rules, orphan handling, and insertion into
// the block chain along with best chain selection and reorganization.
//
// It returns a bool which indicates whether or not the block is an orphan and
// any errors that occurred during processing.  The returned bool is only valid
// when the error is nil.
func (b *BlockChain) ProcessBlock(block *massutil.Block, timeSource MedianTimeSource, flags BehaviorFlags) (bool, error) {
	b.accessLock.Lock()
	defer b.accessLock.Unlock()

	dryRun := flags&BFDryRun == BFDryRun

	blockHash := block.Hash()
	logging.CPrint(logging.TRACE, "Processing block", logging.LogFormat{
		"block": blockHash,
	})

	exists, err := b.blockExists(blockHash)
	if err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}

	if _, exists := b.orphanBlockPool.orphans[*blockHash]; exists {
		return true, nil
	}

	err = checkBlockSanity(block, b.info.chainID)
	if err != nil {
		return false, err
	}

	blockHeader := &block.MsgBlock().Header
	checkpointBlock, err := b.findPreviousCheckpoint()
	if err != nil {
		return false, err
	}
	if checkpointBlock != nil {
		if valid, checkpointTime := ensureCheckPointTime(blockHeader, checkpointBlock); !valid {
			str := fmt.Sprintf("block %v has timestamp %v before "+
				"last checkpoint timestamp %v", blockHash,
				blockHeader.Timestamp, checkpointTime)
			return false, ruleError(ErrCheckpointTimeTooOld, str)
		}
	}

	prevHash := &blockHeader.Previous
	if !prevHash.IsEqual(zeroHash) {
		prevHashExists, err := b.blockExists(prevHash)
		if err != nil {
			return false, err
		}
		if !prevHashExists {
			if !dryRun {
				logging.CPrint(logging.INFO, "Adding orphan block with parent", logging.LogFormat{
					"orphan block": blockHash,
					"parent":       prevHash,
				})
				b.orphanBlockPool.addOrphanBlock(block)
			}

			return true, nil
		}
	}

	err = b.maybeAcceptBlock(block, flags)
	if err != nil {
		return false, err
	}

	if !dryRun {
		err := b.processOrphans(blockHash, flags)
		if err != nil {
			return false, err
		}
		logging.CPrint(logging.DEBUG, "Accepted block", logging.LogFormat{
			"block": blockHash,
		})
	}

	return false, nil
}
