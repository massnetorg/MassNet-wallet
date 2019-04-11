package blockchain

import (
	"sync"
	"time"

	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/wire"
)

type orphanBlock struct {
	block      *massutil.Block
	expiration time.Time
}

type OrphanBlockPool struct {
	orphans      map[wire.Hash]*orphanBlock
	prevOrphans  map[wire.Hash][]*orphanBlock
	oldestOrphan *orphanBlock
	orphanLock   sync.RWMutex
}

func newOrphanBlockPool() *OrphanBlockPool {
	return &OrphanBlockPool{
		orphans:      make(map[wire.Hash]*orphanBlock),
		prevOrphans:  make(map[wire.Hash][]*orphanBlock),
		oldestOrphan: nil,
	}
}

// IsKnownOrphan returns whether the passed hash is currently a known orphan.
// Keep in mind that only a limited number of orphans are held onto for a
// limited amount of time, so this function must not be used as an absolute
// way to test if a block is an orphan block.  A full block (as opposed to just
// its hash) must be passed to ProcessBlock for that purpose.  However, calling
// ProcessBlock with an orphan that already exists results in an error, so this
// function provides a mechanism for a caller to intelligently detect *recent*
// duplicate orphans and react accordingly.
//
// This function is safe for concurrent access.
func (ors *OrphanBlockPool) IsKnownOrphan(hash *wire.Hash) bool {
	ors.orphanLock.RLock()
	defer ors.orphanLock.RUnlock()

	if _, exists := ors.orphans[*hash]; exists {
		return true
	}

	return false
}

// GetOrphanRoot returns the head of the chain for the provided hash from the
// map of orphan blocks.
//
// This function is safe for concurrent access.
func (ors *OrphanBlockPool) GetOrphanRoot(hash *wire.Hash) *wire.Hash {
	ors.orphanLock.RLock()
	defer ors.orphanLock.RUnlock()

	orphanRoot := hash
	prevHash := hash
	for {
		orphan, exists := ors.orphans[*prevHash]
		if !exists {
			break
		}
		orphanRoot = prevHash
		prevHash = &orphan.block.MsgBlock().Header.Previous
	}

	return orphanRoot
}

// removeOrphanBlock removes the passed orphan block from the orphan pool and
// previous orphan index.
func (ors *OrphanBlockPool) removeOrphanBlock(orphan *orphanBlock) {
	ors.orphanLock.Lock()
	defer ors.orphanLock.Unlock()

	orphanHash := orphan.block.Hash()
	delete(ors.orphans, *orphanHash)

	prevHash := &orphan.block.MsgBlock().Header.Previous
	orphans := ors.prevOrphans[*prevHash]
	for i := 0; i < len(orphans); i++ {
		hash := orphans[i].block.Hash()
		if hash.IsEqual(orphanHash) {
			copy(orphans[i:], orphans[i+1:])
			orphans[len(orphans)-1] = nil
			orphans = orphans[:len(orphans)-1]
			i--
		}
	}
	ors.prevOrphans[*prevHash] = orphans

	if len(ors.prevOrphans[*prevHash]) == 0 {
		delete(ors.prevOrphans, *prevHash)
	}
}

// addOrphanBlock adds the passed block (which is already determined to be
// an orphan prior calling this function) to the orphan pool.  It lazily cleans
// up any expired blocks so a separate cleanup poller doesn't need to be run.
// It also imposes a maximum limit on the number of outstanding orphan
// blocks and will remove the oldest received orphan block if the limit is
// exceeded.
func (ors *OrphanBlockPool) addOrphanBlock(block *massutil.Block) {
	for _, oBlock := range ors.orphans {
		if time.Now().After(oBlock.expiration) {
			ors.removeOrphanBlock(oBlock)
			continue
		}

		if ors.oldestOrphan == nil || oBlock.expiration.Before(ors.oldestOrphan.expiration) {
			ors.oldestOrphan = oBlock
		}
	}

	if len(ors.orphans)+1 > maxOrphanBlocks {
		ors.removeOrphanBlock(ors.oldestOrphan)
		ors.oldestOrphan = nil
	}

	ors.orphanLock.Lock()
	defer ors.orphanLock.Unlock()

	expiration := time.Now().Add(time.Hour)
	oBlock := &orphanBlock{
		block:      block,
		expiration: expiration,
	}
	ors.orphans[*block.Hash()] = oBlock

	prevHash := &block.MsgBlock().Header.Previous
	ors.prevOrphans[*prevHash] = append(ors.prevOrphans[*prevHash], oBlock)

	return
}
