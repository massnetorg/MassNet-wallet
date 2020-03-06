package api

import "math"

const (
	txStatusConfirmed  int32 = 1
	txStatusMissing    int32 = 2
	txStatusPacking    int32 = 3
	txStatusConfirming int32 = 4

	txStatusUndefined = math.MaxInt32
)

// staking tx status
const (
	stakingStatusPending     uint32 = 0
	stakingStatusImmature    uint32 = 1
	stakingStatusMature      uint32 = 2
	stakingStatusExpired     uint32 = 3
	stakingStatusWithdrawing uint32 = 4
	stakingStatusWithdrawn   uint32 = 5

	stakingStatusUndefined uint32 = math.MaxUint32
)

// binding tx status
const (
	bindingStatusPending     uint32 = 0
	bindingStatusConfirmed   uint32 = 1
	bindingStatusWithdrawing uint32 = 2
	bindingStatusWithdrawn   uint32 = 3

	bindingStatusUndefined uint32 = math.MaxUint32
)

var (
	txStatusDesc = map[int32]string{
		txStatusConfirmed:  "confirmed",
		txStatusMissing:    "missing",
		txStatusPacking:    "packing",
		txStatusConfirming: "confirming",
	}
)
