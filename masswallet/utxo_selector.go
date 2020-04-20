package masswallet

import (
	"massnet.org/mass-wallet/blockchain"
	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/masswallet/txmgr"
)

type topKSelector struct {
	k          int
	base       []*txmgr.Credit
	guard      *txmgr.Credit
	requireAmt massutil.Amount
}

func newTopKSelector(requireAmt massutil.Amount) *topKSelector {
	// See policy.go isDust(...)
	k := blockchain.GetMaxStandardTxSize() / 154
	return &topKSelector{
		k:          k,
		base:       make([]*txmgr.Credit, 0, k),
		requireAmt: requireAmt,
	}
}

func (s *topKSelector) adjust(i int) {
	cur := i
	child := 2*cur + 1
	for cur < s.k/2 {
		if child+1 < s.k && s.base[child].Amount.Cmp(s.base[child+1].Amount) > 0 {
			child++
		}
		if s.base[cur].Amount.Cmp(s.base[child].Amount) > 0 {
			s.base[cur], s.base[child] = s.base[child], s.base[cur]
			cur = child
			child = 2*cur + 1
		} else {
			break
		}
	}
}

func (s *topKSelector) submit(item *txmgr.Credit) {
	if item.Amount.Cmp(s.requireAmt) > 0 {
		if s.guard == nil || item.Amount.Cmp(s.guard.Amount) < 0 {
			s.guard = item
		}
		return
	}
	// s.base is not full
	if len(s.base) < s.k {
		s.base = append(s.base, item)
		if len(s.base) == s.k {
			for i := s.k/2 - 1; i >= 0; i-- {
				s.adjust(i)
			}
		}
		return
	}

	// s.base is full
	if s.k > 0 && item.Amount.Cmp(s.base[0].Amount) > 0 {
		s.base[0] = item
		s.adjust(0)
	}
}

func (s *topKSelector) Items() []*txmgr.Credit {
	result := make([]*txmgr.Credit, 0, len(s.base)+1)
	result = append(result, s.base...)
	if s.guard != nil {
		result = append(result, s.guard)
	}
	return result
}

func (s *topKSelector) K() int {
	return s.k
}
