package api

import (
	"strings"

	pb "github.com/massnetorg/MassNet-wallet/api/proto"
)

type addrAndBalanceList []*pb.AddressAndBalance

func (a addrAndBalanceList) Len() int {
	return len(a)
}

func (a addrAndBalanceList) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a addrAndBalanceList) Less(i, j int) bool {
	if a[i].Balance < a[j].Balance {
		return true
	} else if a[i].Balance > a[j].Balance {
		return false
	}

	return strings.Compare(a[i].Address, a[j].Address) == -1
}
