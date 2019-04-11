package consensus

import (
	"math"
	"os"
	"testing"
)

const (
	MinHighPriority          = 1e8 * 144.0 / 250
	MaxNewBlockChSize        = 1024
	DefaultBlockPrioritySize = 50000
	MaxOrphanTransactions    = 1000
)

var (
	DefaultMinRelayTxFee = float64(1000) / math.Pow10(int(8))
)

var (
	// flag for building project
	UserTestNet    = false
	UserTestNetStr = ""
)

const osUseCI = "MASS_CI"

func SkipCI(t *testing.T) {
	if os.Getenv(osUseCI) == "" {
		t.Skip("Skipping certain test for CI environment")
	}
}

func init() {
	if UserTestNetStr == "true" {
		UserTestNet = true
	}
}
