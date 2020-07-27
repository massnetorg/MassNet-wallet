package txmgr

import "errors"

var (
	ErrChainReorg               = errors.New("chain reorganization")
	ErrUnexpectedCreditNotFound = errors.New("unexpected credit not found")
)
