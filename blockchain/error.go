// Modified for MassNet
// Copyright (c) 2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"
)

// ErrorCode identifies a kind of error.
type ErrorCode int

// These constants are used to identify a specific MpRuleError.
const (
	ErrDuplicateBlock ErrorCode = iota

	ErrBlockTooBig

	ErrBlockSizeTooHigh

	ErrBlockVersionTooOld

	ErrInvalidTime

	ErrTimeTooOld

	ErrTimeTooNew

	ErrDifficultyTooLow

	ErrUnexpectedDifficulty

	ErrHighHash

	ErrBadMerkleRoot

	ErrBadProposalRoot

	ErrBadCheckpoint

	ErrForkTooOld

	ErrCheckpointTimeTooOld

	ErrNoTransactions

	ErrTooManyTransactions

	ErrNoTxInputs

	ErrNoTxOutputs

	ErrTxTooBig

	ErrBadTxOutValue

	ErrDuplicateTxInputs

	ErrBadTxInput

	ErrMissingTx

	ErrUnfinalizedTx

	ErrDuplicateTx

	ErrOverwriteTx

	ErrImmatureSpend

	ErrDoubleSpend

	ErrSpendTooHigh

	ErrBadFees

	ErrTooManySigOps

	ErrFirstTxNotCoinbase

	ErrMultipleCoinbases

	ErrBadCoinbaseScriptLen

	ErrBadCoinbaseValue

	ErrMissingCoinbaseHeight

	ErrBadCoinbaseHeight

	ErrScriptMalformed

	ErrScriptValidation

	ErrUnexpectedWitness

	ErrInvalidWitnessCommitment

	ErrWitnessCommitmentMismatch

	ErrBadBlockHeight

	ErrHeaderTimestamp

	ErrChainID

	ErrCollateralAddress

	ErrTxOutNil

	ErrIndex

	ErrTxSpent

	ErrFoundationAddress

	ErrCoinbaseOutputValue

	ErrMissingTxOut
)

// Map of ErrorCode values back to their constant names for pretty printing.
var errorCodeStrings = map[ErrorCode]string{
	ErrDuplicateBlock:            "ErrDuplicateBlock",
	ErrBlockTooBig:               "ErrBlockTooBig",
	ErrBlockVersionTooOld:        "ErrBlockVersionTooOld",
	ErrBlockSizeTooHigh:          "ErrBlockSizeTooHigh",
	ErrInvalidTime:               "ErrInvalidTime",
	ErrTimeTooOld:                "ErrTimeTooOld",
	ErrTimeTooNew:                "ErrTimeTooNew",
	ErrDifficultyTooLow:          "ErrDifficultyTooLow",
	ErrUnexpectedDifficulty:      "ErrUnexpectedDifficulty",
	ErrHighHash:                  "ErrHighHash",
	ErrBadMerkleRoot:             "ErrBadMerkleRoot",
	ErrBadProposalRoot:           "ErrBadProposalRoot",
	ErrBadCheckpoint:             "ErrBadCheckpoint",
	ErrForkTooOld:                "ErrForkTooOld",
	ErrCheckpointTimeTooOld:      "ErrCheckpointTimeTooOld",
	ErrNoTransactions:            "ErrNoTransactions",
	ErrTooManyTransactions:       "ErrTooManyTransactions",
	ErrNoTxInputs:                "ErrNoTxInputs",
	ErrNoTxOutputs:               "ErrNoTxOutputs",
	ErrTxTooBig:                  "ErrTxTooBig",
	ErrBadTxOutValue:             "ErrBadTxOutValue",
	ErrDuplicateTxInputs:         "ErrDuplicateTxInputs",
	ErrBadTxInput:                "ErrBadTxInput",
	ErrMissingTx:                 "ErrMissingTx",
	ErrUnfinalizedTx:             "ErrUnfinalizedTx",
	ErrDuplicateTx:               "ErrDuplicateTx",
	ErrOverwriteTx:               "ErrOverwriteTx",
	ErrImmatureSpend:             "ErrImmatureSpend",
	ErrDoubleSpend:               "ErrDoubleSpend",
	ErrSpendTooHigh:              "ErrSpendTooHigh",
	ErrBadFees:                   "ErrBadFees",
	ErrTooManySigOps:             "ErrTooManySigOps",
	ErrFirstTxNotCoinbase:        "ErrFirstTxNotCoinbase",
	ErrMultipleCoinbases:         "ErrMultipleCoinbases",
	ErrBadCoinbaseScriptLen:      "ErrBadCoinbaseScriptLen",
	ErrBadCoinbaseValue:          "ErrBadCoinbaseValue",
	ErrMissingCoinbaseHeight:     "ErrMissingCoinbaseHeight",
	ErrBadCoinbaseHeight:         "ErrBadCoinbaseHeight",
	ErrScriptMalformed:           "ErrScriptMalformed",
	ErrScriptValidation:          "ErrScriptValidation",
	ErrUnexpectedWitness:         "ErrUnexpectedWitness",
	ErrInvalidWitnessCommitment:  "ErrInvalidWitnessCommitment",
	ErrWitnessCommitmentMismatch: "ErrWitnessCommitmentMismatch",
	ErrBadBlockHeight:            "ErrBadBlockHeight",
	ErrChainID:                   "ErrChainID",
	ErrHeaderTimestamp:           "ErrHeaderTimestamp",
	ErrTxOutNil:                  "ErrTxOutNil",
	ErrIndex:                     "ErrIndex",
	ErrTxSpent:                   "ErrTxSpent",
	ErrFoundationAddress:         "ErrFoundationAddress",
	ErrCollateralAddress:         "ErrCollateralAddress",
	ErrCoinbaseOutputValue:       "ErrCoinbaseOutputValue",
}

// String returns the ErrorCode as a human-readable name.
func (e ErrorCode) String() string {
	if s := errorCodeStrings[e]; s != "" {
		return s
	}
	return fmt.Sprintf("Unknown ErrorCode (%d)", int(e))
}

// MpRuleError identifies a rule violation.  It is used to indicate that
// processing of a block or transaction failed due to one of the many validation
// rules.  The caller can use type assertions to determine if a failure was
// specifically due to a rule violation and access the ErrorCode field to
// ascertain the specific reason for the rule violation.
type RuleError struct {
	Err         error
	ErrorCode   ErrorCode // Describes the kind of error
	Description string    // Human readable description of the issue
}

// Error satisfies the error interface and prints human-readable errors.
func (e RuleError) Error() string {
	return e.Description
}

// ruleError creates an MpRuleError given a set of arguments.
func ruleError(c ErrorCode, desc string) RuleError {
	return RuleError{ErrorCode: c, Description: desc}
}
