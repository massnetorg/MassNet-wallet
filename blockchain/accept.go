// Modified for MassNet
// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"massnet.org/mass-wallet/logging"
	"massnet.org/mass-wallet/massutil"
)

// maybeAcceptBlock potentially accepts a block into the memory block chain.
// It performs several validation checks which depend on its position within
// the block chain before adding it.  The block is expected to have already gone
// through ProcessBlock before calling this function with it.
func (b *BlockChain) maybeAcceptBlock(block *massutil.Block, flags BehaviorFlags) error {
	dryRun := flags&BFDryRun == BFDryRun

	prevNode, err := b.getPrevNodeFromBlock(block)
	if err != nil {
		logging.CPrint(logging.ERROR, "getPrevNodeFromBlock", logging.LogFormat{"err": err})
		return err
	}

	blockHeight := int32(0)
	if prevNode != nil {
		blockHeight = prevNode.height + 1
	}
	block.SetHeight(blockHeight)

	err = b.checkBlockContext(block, prevNode, flags)
	if err != nil {
		return err
	}

	if !dryRun {
		err = b.pruneBlockNodes()
		if err != nil {
			return err
		}
	}

	blockHeader := &block.MsgBlock().Header
	newNode := newBlockNode(blockHeader, block.Hash(), blockHeight)
	if prevNode != nil {
		newNode.parent = prevNode
		newNode.height = blockHeight
		newNode.capSum.Add(prevNode.capSum, newNode.capSum)
	}

	err = b.connectBestChain(newNode, block, flags)
	if err != nil {
		return err
	}

	return nil
}
