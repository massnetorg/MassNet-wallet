package txmgr

import "errors"

var (
	ErrChainReorg = errors.New("chain reorganization")
)
