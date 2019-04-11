// Modified for MassNet
// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package massutil

import (
	"bytes"
	"io"
	"massnet.org/mass-wallet/wire"
)

// TxIndexUnknown is the value returned for a transaction index that is unknown.
// This is typically because the transaction has not been inserted into a block
// yet.
const TxIndexUnknown = -1

// Tx defines a mass transaction that provides easier and more efficient
// manipulation of raw transactions.  It also memoizes the hash for the
// transaction on its first access so subsequent accesses don't have to repeat
// the relatively expensive hashing operations.
type Tx struct {
	msgTx         *wire.MsgTx // Underlying MsgTx
	txHash        *wire.Hash  // Cached transaction hash
	txHashWitness *wire.Hash  // Cached transaction witness hash
	txHasWitness  *bool       // If the transaction has witness data
	txIndex       int         // Position within a block or TxIndexUnknown
}

// MsgTx returns the underlying wire.MsgTx for the transaction.
func (t *Tx) MsgTx() *wire.MsgTx {
	// Return the cached transaction.
	return t.msgTx
}

// Hash returns the hash of the transaction.  This is equivalent to
// calling TxHash on the underlying wire.MsgTx, however it caches the
// result so subsequent calls are more efficient.
func (t *Tx) Hash() *wire.Hash {
	// Return the cached hash if it has already been generated.
	if t.txHash != nil {
		return t.txHash
	}

	// Cache the hash and return it.
	hash := t.msgTx.TxHash()
	t.txHash = &hash
	return &hash
}

// WitnessHash returns the witness hash (wtxid) of the transaction.  This is
// equivalent to calling WitnessHash on the underlying wire.MsgTx, however it
// caches the result so subsequent calls are more efficient.
func (t *Tx) WitnessHash() *wire.Hash {
	// Return the cached hash if it has already been generated.
	if t.txHashWitness != nil {
		return t.txHashWitness
	}

	// Cache the hash and return it.
	hash := t.msgTx.WitnessHash()
	t.txHashWitness = &hash
	return &hash
}

func (t *Tx) Bytes(mode wire.CodecMode) ([]byte, error) {
	// Serialize the MsgBlock.
	var w bytes.Buffer
	err := t.msgTx.Serialize(&w, mode)
	if err != nil {
		return nil, err
	}
	serializedTx := w.Bytes()

	return serializedTx, nil
}

// Index returns the saved index of the transaction within a block.  This value
// will be TxIndexUnknown if it hasn't already explicitly been set.
func (t *Tx) Index() int {
	return t.txIndex
}

// SetIndex sets the index of the transaction in within a block.
func (t *Tx) SetIndex(index int) {
	t.txIndex = index
}

// NewTx returns a new instance of a mass transaction given an underlying
// wire.MsgTx.  See Tx.
func NewTx(msgTx *wire.MsgTx) *Tx {
	return &Tx{
		msgTx:   msgTx,
		txIndex: TxIndexUnknown,
	}
}

// NewTxFromBytes returns a new instance of a mass transaction given the
// serialized bytes.  See Tx.
func NewTxFromBytes(serializedTx []byte, mode wire.CodecMode) (*Tx, error) {
	br := bytes.NewReader(serializedTx)
	return NewTxFromReader(br, mode)
}

// NewTxFromReader returns a new instance of a mass transaction given a
// Reader to deserialize the transaction.  See Tx.
func NewTxFromReader(r io.Reader, mode wire.CodecMode) (*Tx, error) {
	// Deserialize the bytes into a MsgTx.
	var msgTx wire.MsgTx
	err := msgTx.Deserialize(r, mode)
	if err != nil {
		return nil, err
	}

	t := Tx{
		msgTx:   &msgTx,
		txIndex: TxIndexUnknown,
	}
	return &t, nil
}
