// Modified for MassNet
// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"massnet.org/mass-wallet/config"

	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/wire"
)

// LatestCheckpoint returns the most recent checkpoint (regardless of whether it
// is already known).  When checkpoints are disabled or there are no checkpoints
// for the active network, it will return nil.
func (b *BlockChain) LatestCheckpoint() *config.Checkpoint {
	if b.noCheckpoints || len(config.ChainParams.Checkpoints) == 0 {
		return nil
	}

	checkpoints := config.ChainParams.Checkpoints
	return &checkpoints[len(checkpoints)-1]
}

// verifyCheckpoint returns whether the passed block height and hash combination
// match the hard-coded checkpoint data.  It also returns true if there is no
// checkpoint data for the passed block height.
func (b *BlockChain) verifyCheckpoint(height int32, hash *wire.Hash) bool {
	if b.noCheckpoints || len(config.ChainParams.Checkpoints) == 0 {
		return true
	}

	checkpoint, exists := b.checkpointsByHeight[height]
	if !exists {
		return true
	}

	if !checkpoint.Hash.IsEqual(hash) {
		return false
	}

	logging.CPrint(logging.INFO, "verified checkpoint", logging.LogFormat{
		"height": checkpoint.Height,
		"block":  checkpoint.Hash,
	})
	return true
}

// findPreviousCheckpoint finds the most recent checkpoint that is already
// available in the downloaded portion of the block chain and returns the
// associated block.  It returns nil if a checkpoint can't be found (this should
// really only happen for blocks before the first checkpoint).
func (b *BlockChain) findPreviousCheckpoint() (*massutil.Block, error) {
	if b.noCheckpoints || len(config.ChainParams.Checkpoints) == 0 {
		return nil, nil
	}

	checkpoints := config.ChainParams.Checkpoints
	numCheckpoints := len(checkpoints)
	if numCheckpoints == 0 {
		return nil, nil
	}

	if b.bestChain == nil || (b.checkpointBlock == nil && b.nextCheckpoint == nil) {
		checkpointIndex := -1
		for i := numCheckpoints - 1; i >= 0; i-- {
			exists, err := b.db.ExistsSha(checkpoints[i].Hash)
			if err != nil {
				return nil, err
			}

			if exists {
				checkpointIndex = i
				break
			}
		}

		if checkpointIndex == -1 {
			b.nextCheckpoint = &checkpoints[0]
			return nil, nil
		}

		checkpoint := checkpoints[checkpointIndex]
		block, err := b.db.FetchBlockBySha(checkpoint.Hash)
		if err != nil {
			return nil, err
		}
		b.checkpointBlock = block

		b.nextCheckpoint = nil
		if checkpointIndex < numCheckpoints-1 {
			b.nextCheckpoint = &checkpoints[checkpointIndex+1]
		}

		return block, nil
	}

	if b.nextCheckpoint == nil {
		return b.checkpointBlock, nil
	}

	if uint64(b.bestChain.height) < b.nextCheckpoint.Height {
		return b.checkpointBlock, nil
	}

	block, err := b.db.FetchBlockBySha(b.nextCheckpoint.Hash)
	if err != nil {
		return nil, err
	}
	b.checkpointBlock = block

	checkpointIndex := -1
	for i := numCheckpoints - 1; i >= 0; i-- {
		if checkpoints[i].Hash.IsEqual(b.nextCheckpoint.Hash) {
			checkpointIndex = i
			break
		}
	}
	b.nextCheckpoint = nil
	if checkpointIndex != -1 && checkpointIndex < numCheckpoints-1 {
		b.nextCheckpoint = &checkpoints[checkpointIndex+1]
	}

	return b.checkpointBlock, nil
}
