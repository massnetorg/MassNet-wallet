package masswallet

import (
	"fmt"

	"massnet.org/mass-wallet/masswallet/txmgr"

	"massnet.org/mass-wallet/massutil"
)

func ExampleSubmit() {
	seq := []int{200, 1, 150, 20, 101, 100, 40, 30, 9, 1000}

	items := make([]*txmgr.Credit, 0, len(seq))
	for _, n := range seq {
		amt, _ := massutil.NewAmountFromInt(int64(n))
		items = append(items, &txmgr.Credit{
			Amount: amt,
		})
	}

	req, _ := massutil.NewAmountFromUint(100)

	for k := 0; k <= 7; k++ {
		selector := &topKSelector{
			k:          k,
			base:       make([]*txmgr.Credit, 0, k),
			requireAmt: req,
		}
		for _, item := range items {
			selector.submit(item)
		}
		var out []int64
		for _, item := range selector.Items() {
			out = append(out, item.Amount.IntValue())
		}
		fmt.Println(out, selector.guard.Amount.IntValue())
	}

	// Output:
	// [101] 101
	// [100 101] 101
	// [40 100 101] 101
	// [30 40 100 101] 101
	// [20 30 100 40 101] 101
	// [9 20 100 40 30 101] 101
	// [1 20 9 40 30 100 101] 101
	// [1 20 100 40 30 9 101] 101
}
