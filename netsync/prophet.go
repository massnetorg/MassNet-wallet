package netsync

import (
	"time"

	"github.com/massnetorg/MassNet-wallet/errors"
	"github.com/massnetorg/MassNet-wallet/massutil"
)

// Reject block from far future (12 seconds for now)
func preventBlockFromFuture(block *massutil.Block) error {
	if time.Now().Add(12 * time.Second).Before(block.MsgBlock().Header.Timestamp) {
		return errors.Wrap(errPeerMisbehave, "preventBlockFromFuture")
	}
	return nil
}

// Reject blocks from far future (12 seconds for now)
func preventBlocksFromFuture(blocks []*massutil.Block) error {
	for _, block := range blocks {
		if preventBlockFromFuture(block) != nil {
			return errors.Wrap(errPeerMisbehave, "preventBlocksFromFuture")
		}
	}
	return nil
}
