package api

import "math"

const (
	txStatusConfirmed int32 = iota + 1
	txStatusMissing
	txStatusPacking
	txStatusConfirming

	txStatusUndefined = math.MaxInt32
)

// staking tx status
const (
	stakingStatusPending uint32 = iota
	stakingStatusImmature
	stakingStatusMature
	stakingStatusExpired
	stakingStatusWithdrawing
	stakingStatusWithdrawn

	stakingStatusUndefined uint32 = math.MaxUint32
)

// binding tx status
const (
	bindingStatusPending uint32 = iota
	bindingStatusConfirmed
	bindingStatusWithdrawing
	bindingStatusWithdrawn

	bindingStatusUndefined uint32 = math.MaxUint32
)

// wallet status
const (
	walletStatusReady uint32 = iota
	walletStatusImporting
	walletStatusRemoving
)

var (
	txStatusDesc = map[int32]string{
		txStatusConfirmed:  "confirmed",
		txStatusMissing:    "missing",
		txStatusPacking:    "packing",
		txStatusConfirming: "confirming",
	}
)

var (
	walletStatusMsg = map[uint32]string{
		walletStatusReady:    "ready",
		walletStatusRemoving: "removing",
		// walletStatusImporting: "{synced_height}",
	}
)
