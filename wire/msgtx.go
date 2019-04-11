// Modified for MassNet
// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"io"
	"math"
	"strconv"

	"github.com/massnetorg/MassNet-wallet/logging"
	wirepb "github.com/massnetorg/MassNet-wallet/wire/pb"

	"github.com/golang/protobuf/proto"
)

const (
	// TxVersion is the current latest supported transaction version.
	TxVersion = 1

	// MaxTxInSequenceNum is the maximum sequence number the sequence field
	// of a transaction input can be.
	MaxTxInSequenceNum uint32 = 0xffffffff

	// MaxPrevOutIndex is the maximum index the index field of a previous
	// outpoint can be.
	MaxPrevOutIndex uint32 = 0xffffffff

	// SequenceLockTimeDisabled is a flag that if set on a transaction
	// input's sequence number, the sequence number will not be interpreted
	// as a relative locktime.
	SequenceLockTimeDisabled = 1 << 31

	// SequenceLockTimeIsSeconds is a flag thabt if set on a transaction
	// input's sequence number, the relative locktime has units of 512
	// seconds.
	SequenceLockTimeIsSeconds = 1 << 22

	// SequenceLockTimeMask is a mask that extracts the relative locktime
	// when masked against the transaction input sequence number.
	SequenceLockTimeMask = 0x0000ffff

	// SequenceLockTimeGranularity is the defined time based granularity
	// for seconds-based relative time locks. When converting from seconds
	// to a sequence number, the value is right shifted by this amount,
	// therefore the granularity of relative time locks in 512 or 2^9
	// seconds. Enforced relative lock times are multiples of 512 seconds.
	SequenceLockTimeGranularity = 9

	//Min Locked Heigt in a LocktimeScriptHash output
	MinLockHeight = 40
)

const defaultTxInOutAlloc = 15

const (
	minTxPayload = 10
)

// zeroHash is the zero value for a wire.Hash and is defined as
// a package level variable to avoid the need to create a new instance
// every time a check is needed.
var zeroHash = &Hash{}

type OutPoint struct {
	Hash  Hash
	Index uint32
}

func NewOutPoint(hash *Hash, index uint32) *OutPoint {
	return &OutPoint{
		Hash:  *hash,
		Index: index,
	}
}

// String returns the OutPoint in the human-readable form "hash:index".
func (o OutPoint) String() string {
	buf := make([]byte, 2*HashSize+1, 2*HashSize+1+10)
	copy(buf, o.Hash.String())
	buf[2*HashSize] = ':'
	buf = strconv.AppendUint(buf, uint64(o.Index), 10)
	return string(buf)
}

type TxIn struct {
	PreviousOutPoint OutPoint
	Witness          TxWitness
	Sequence         uint32
}

// SerializeSize returns the number of bytes it would take to serialize the
// the transaction input.
func (t *TxIn) SerializeSize() int {
	var n = 0
	n += 4 + 32                    // OutPoint
	n += t.Witness.SerializeSize() // Witness
	n += 4                         // Sequence
	return n
}

func NewTxIn(prevOut *OutPoint, witness [][]byte) *TxIn {
	return &TxIn{
		PreviousOutPoint: *prevOut,
		Witness:          witness,
		Sequence:         MaxTxInSequenceNum,
	}
}

// TxWitness defines the witness for a TxIn. A witness is to be interpreted as
// a slice of byte slices, or a stack with one or many elements.
type TxWitness [][]byte

// SerializeSize returns the number of bytes it would take to serialize the the
// transaction input's witness.
func (t TxWitness) SerializeSize() int {
	var n = 0

	// For each element in the witness, we'll need a varint to signal the
	// size of the element, then finally the number of bytes the element
	// itself comprises.
	for _, witItem := range t {
		n += len(witItem)
	}

	return n
}

type TxOut struct {
	Value    int64
	PkScript []byte
}

// SerializeSize returns the number of bytes it would take to serialize the
// the transaction output.
func (t *TxOut) SerializeSize() int {
	// Value 8 bytes + serialized varint size for the length of PkScript +
	// PkScript bytes.
	return 8 + len(t.PkScript)
}

func NewTxOut(value int64, pkScript []byte) *TxOut {
	return &TxOut{
		Value:    value,
		PkScript: pkScript,
	}
}

// Use the AddTxIn and AddTxOut functions to build up the list of transaction
// inputs and outputs.
type MsgTx struct {
	Version  int32
	TxIn     []*TxIn
	TxOut    []*TxOut
	LockTime uint32
	Payload  []byte
}

// AddTxIn adds a transaction input to the message.
func (msg *MsgTx) AddTxIn(ti *TxIn) {
	msg.TxIn = append(msg.TxIn, ti)
}

// AddTxOut adds a transaction output to the message.
func (msg *MsgTx) AddTxOut(to *TxOut) {
	msg.TxOut = append(msg.TxOut, to)
}

func (msg *MsgTx) AddPayload(payload []byte) {
	msg.Payload = payload
}

// IsCoinBaseTx determines whether or not a transaction is a coinbase.  A coinbase
// is a special transaction created by miners that has no inputs.  This is
// represented in the block chain by a transaction with a single input that has
// a previous output transaction index set to the maximum value along with a
// zero hash.
//
// This function only differs from IsCoinBase in that it works with a raw wire
// transaction as opposed to a higher level util transaction.
func (msg *MsgTx) IsCoinBaseTx() bool {
	prevOut := &msg.TxIn[0].PreviousOutPoint
	if prevOut.Index != math.MaxUint32 || !prevOut.Hash.IsEqual(zeroHash) {
		return false
	}

	return true
}

// TxHash generates the Hash for the transaction.
func (msg *MsgTx) TxHash() Hash {
	var buf bytes.Buffer
	err := msg.Serialize(&buf, ID)
	if err != nil {
		return Hash{}
	}
	return DoubleHashH(buf.Bytes())
}

func (msg *MsgTx) WitnessHash() Hash {
	var buf bytes.Buffer
	err := msg.Serialize(&buf, WitnessID)
	if err != nil {
		return Hash{}
	}
	return DoubleHashH(buf.Bytes())
}

// Copy creates a deep copy of a transaction so that the original does not get
// modified when the copy is manipulated.
func (msg *MsgTx) Copy() *MsgTx {
	// Create new tx and start by copying primitive values and making space
	// for the transaction inputs and outputs.
	newTx := MsgTx{
		Version:  msg.Version,
		TxIn:     make([]*TxIn, 0, len(msg.TxIn)),
		TxOut:    make([]*TxOut, 0, len(msg.TxOut)),
		LockTime: msg.LockTime,
		Payload:  make([]byte, 0, defaultTxInOutAlloc),
	}

	// Deep copy the old TxIn data.
	for _, oldTxIn := range msg.TxIn {
		// Deep copy the old previous outpoint.
		oldOutPoint := oldTxIn.PreviousOutPoint
		newOutPoint := OutPoint{}
		newOutPoint.Hash.SetBytes(oldOutPoint.Hash[:])
		newOutPoint.Index = oldOutPoint.Index

		// Create new txIn with the deep copied data and append it to
		// new Tx.
		newTxIn := TxIn{
			PreviousOutPoint: newOutPoint,
			Sequence:         oldTxIn.Sequence,
		}
		newTx.TxIn = append(newTx.TxIn, &newTxIn)
		// If the transaction is witnessy, then also copy the
		// witnesses.
		if len(oldTxIn.Witness) != 0 {
			// Deep copy the old witness data.
			newTxIn.Witness = make([][]byte, len(oldTxIn.Witness))
			for i, oldItem := range oldTxIn.Witness {
				newItem := make([]byte, len(oldItem))
				copy(newItem, oldItem)
				newTxIn.Witness[i] = newItem
			}
		}
		// Finally, append this fully copied txin.
		newTx.TxIn = append(newTx.TxIn, &newTxIn)
	}

	// Deep copy the old TxOut data.
	for _, oldTxOut := range msg.TxOut {
		// Deep copy the old PkScript
		var newScript []byte
		oldScript := oldTxOut.PkScript
		oldScriptLen := len(oldScript)
		if oldScriptLen > 0 {
			newScript = make([]byte, oldScriptLen, oldScriptLen)
			copy(newScript, oldScript[:oldScriptLen])
		}

		// Create new txOut with the deep copied data and append it to
		// new Tx.
		newTxOut := TxOut{
			Value:    oldTxOut.Value,
			PkScript: newScript,
		}
		newTx.TxOut = append(newTx.TxOut, &newTxOut)
	}

	return &newTx
}

func (msg *MsgTx) MassDecode(r io.Reader, mode CodecMode) error {
	var buf bytes.Buffer
	buf.ReadFrom(r)

	switch mode {
	case DB:
		pb := new(wirepb.Tx)
		err := proto.Unmarshal(buf.Bytes(), pb)
		if err != nil {
			return err
		}
		msg.FromProto(pb)
		return nil

	case Packet:
		pb := new(wirepb.Tx)
		err := proto.Unmarshal(buf.Bytes(), pb)
		if err != nil {
			return err
		}
		msg.FromProto(pb)
		return nil

	default:
		logging.CPrint(logging.FATAL, "MsgTx.MassDecode: invalid CodecMode", logging.LogFormat{"mode": mode})
		panic(nil) // unreachable
	}

}

// Deserialize decodes a transaction from r.
func (msg *MsgTx) Deserialize(r io.Reader, mode CodecMode) error {
	return msg.MassDecode(r, mode)
}

func (msg *MsgTx) MassEncode(w io.Writer, mode CodecMode) error {
	pb := msg.ToProto()

	var writeAll = func() error {
		_, err := w.Write(pb.Bytes())
		return err
	}

	switch mode {
	case DB:
		content, err := proto.Marshal(pb)
		if err != nil {
			return err
		}
		_, err = w.Write(content)
		return err

	case Packet:
		content, err := proto.Marshal(pb)
		if err != nil {
			return err
		}
		_, err = w.Write(content)
		return err

	case Plain:
		// Write all elements of a transaction
		return writeAll()

	case ID:
		_, err := w.Write(pb.BytesNoWitness())
		return err

	case WitnessID:
		// Write all elements of a transaction
		return writeAll()

	default:
		logging.CPrint(logging.FATAL, "MsgTx.MassEncode: invalid CodecMode", logging.LogFormat{"mode": mode})
		panic(nil) // unreachable
	}

}

// Serialize encodes the transaction to w.
func (msg *MsgTx) Serialize(w io.Writer, mode CodecMode) error {
	return msg.MassEncode(w, mode)
}

// SerializeSize returns the number of bytes it would take to serialize the
// the transaction.
func (msg *MsgTx) SerializeSize() int {
	var n = 0

	// Version
	n += 4

	// TxIns
	for _, ti := range msg.TxIn {
		n += ti.SerializeSize()
	}

	// TxOuts
	for _, to := range msg.TxOut {
		n += to.SerializeSize()
	}

	// LockTime
	n += 4

	//Payload
	n += len(msg.Payload)

	return n
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgTx) Command() string {
	return CmdTx
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgTx) MaxPayloadLength(pver uint32) uint32 {
	return MaxBlockPayload
}

func NewMsgTx() *MsgTx {
	return &MsgTx{
		Version: TxVersion,
		TxIn:    make([]*TxIn, 0, defaultTxInOutAlloc),
		TxOut:   make([]*TxOut, 0, defaultTxInOutAlloc),
		Payload: make([]byte, 0, defaultTxInOutAlloc),
	}
}

func WriteTxOut(w io.Writer, pver uint32, version int32, to *TxOut) error {
	err := binarySerializer.PutUint64(w, littleEndian, uint64(to.Value))
	if err != nil {
		return err
	}

	return WriteVarBytes(w, pver, to.PkScript)
}

// ToProto get proto Tx from wire Tx
func (msg *MsgTx) ToProto() *wirepb.Tx {
	txIns := make([]*wirepb.TxIn, len(msg.TxIn), len(msg.TxIn))
	for i, ti := range msg.TxIn {
		witness := make([][]byte, len(ti.Witness), len(ti.Witness))
		for j, data := range ti.Witness {
			witness[j] = data
		}
		txIn := &wirepb.TxIn{
			PreviousOutPoint: &wirepb.OutPoint{
				Hash:  ti.PreviousOutPoint.Hash.ToProto(),
				Index: ti.PreviousOutPoint.Index,
			},
			Witness:  witness,
			Sequence: ti.Sequence,
		}
		txIns[i] = txIn
	}

	txOuts := make([]*wirepb.TxOut, len(msg.TxOut), len(msg.TxOut))
	for i, to := range msg.TxOut {
		txOut := &wirepb.TxOut{
			Value:    to.Value,
			PkScript: to.PkScript,
		}
		txOuts[i] = txOut
	}

	return &wirepb.Tx{
		Version:  msg.Version,
		TxIn:     txIns,
		TxOut:    txOuts,
		LockTime: msg.LockTime,
		Payload:  msg.Payload,
	}
}

// FromProto load proto TX into wire Tx
func (msg *MsgTx) FromProto(pb *wirepb.Tx) {
	txIns := make([]*TxIn, len(pb.TxIn), len(pb.TxIn))
	for i, ti := range pb.TxIn {
		witness := make([][]byte, len(ti.Witness), len(ti.Witness))
		for j, data := range ti.Witness {
			witness[j] = data
		}
		txIn := &TxIn{
			PreviousOutPoint: OutPoint{
				Hash:  *NewHashFromProto(ti.PreviousOutPoint.Hash),
				Index: ti.PreviousOutPoint.Index,
			},
			Witness:  witness,
			Sequence: ti.Sequence,
		}
		txIns[i] = txIn
	}

	txOuts := make([]*TxOut, len(pb.TxOut), len(pb.TxOut))
	for i, to := range pb.TxOut {
		txOut := &TxOut{
			Value:    to.Value,
			PkScript: to.PkScript,
		}
		txOuts[i] = txOut
	}

	msg.Version = pb.Version
	msg.TxIn = txIns
	msg.TxOut = txOuts
	msg.LockTime = pb.LockTime
	msg.Payload = pb.Payload
}

// NewTxFromProto get wire Tx from proto Tx
func NewTxFromProto(pb *wirepb.Tx) *MsgTx {
	msg := new(MsgTx)
	msg.FromProto(pb)
	return msg
}
