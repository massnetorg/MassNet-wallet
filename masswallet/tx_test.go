package masswallet

import (
	"testing"

	"massnet.org/mass-wallet/massutil"
	"massnet.org/mass-wallet/masswallet/txmgr"
)

func mockCredits(nums int) []*txmgr.Credit {
	credits := make([]*txmgr.Credit, 0)
	for i := 1; i <= nums; i++ {
		a := 10e8
		a += a
		amt, _ := massutil.NewAmountFromUint(uint64(a))
		credits = append(credits, &txmgr.Credit{
			Amount: amt,
		})
	}
	return credits
}
func TestOptOutputs(t *testing.T) {
	//mockCredits:20mass--200mass
	amt1, _ := massutil.NewAmountFromUint(19e8)
	amt2, _ := massutil.NewAmountFromUint(21e8)
	amt3, _ := massutil.NewAmountFromUint(150e8)
	amt4, _ := massutil.NewAmountFromUint(300e8)
	targets := []massutil.Amount{
		amt1, amt2, amt3, amt4,
	}

	for index, target := range targets {
		uts, optAmount, totalAmount, _ := optOutputs(target, mockCredits(10))
		t.Log()
		t.Logf("test_%v", index)
		t.Log("target: ", target)
		t.Log("optAmount: ", optAmount)
		t.Log("totalAmount: ", totalAmount)
		for index, ut := range uts {
			t.Logf("%v: %v mass", index, ut.Amount)
		}
	}

}
