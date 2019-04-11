// Modified for MassNet
// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/big"
	"time"

	"github.com/massnetorg/MassNet-wallet/btcec"
	"github.com/massnetorg/MassNet-wallet/logging"
	"github.com/massnetorg/MassNet-wallet/poc"
	"github.com/massnetorg/MassNet-wallet/wire/pb"

	"github.com/golang/protobuf/proto"
)

// BlockVersion is the current latest supported block version.
const BlockVersion = 1

const MinBlockHeaderPayload = blockHeaderMinLen

type BlockHeader struct {
	ChainID         Hash
	Version         uint64
	Height          uint64
	Timestamp       time.Time
	Previous        Hash
	TransactionRoot Hash
	ProposalRoot    Hash
	Target          *big.Int
	Challenge       *big.Int
	PubKey          *btcec.PublicKey
	Proof           *poc.Proof
	SigQ            *btcec.Signature
	Sig2            *btcec.Signature
	BanList         []*btcec.PublicKey
}

const blockHeaderMinLen = 429

// BlockHash computes the block identifier hash for the given block header.
func (h *BlockHeader) BlockHash() Hash {
	var buf bytes.Buffer
	_ = writeBlockHeader(&buf, h, ID)

	return DoubleHashH(buf.Bytes())
}

// MassDecode decodes r using the given protocol encoding into the receiver.
func (h *BlockHeader) MassDecode(r io.Reader, mode CodecMode) error {
	return h.Deserialize(r, mode)
}

// Deserialize decodes a block header from r into the receiver
func (h *BlockHeader) Deserialize(r io.Reader, mode CodecMode) error {
	return readBlockHeader(r, h, mode)
}

// readBlockHeader reads a block header from r.
func readBlockHeader(r io.Reader, bh *BlockHeader, mode CodecMode) error {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	if err != nil {
		return err
	}

	switch mode {
	case DB:
		pb := new(wirepb.BlockHeader)
		err = proto.Unmarshal(buf.Bytes(), pb)
		if err != nil {
			return err
		}
		bh.FromProto(pb)
		return nil

	case Packet:
		pb := new(wirepb.BlockHeader)
		err = proto.Unmarshal(buf.Bytes(), pb)
		if err != nil {
			return err
		}
		bh.FromProto(pb)
		return nil

	default:
		logging.CPrint(logging.FATAL, "readBlockHeader invalid CodecMode", logging.LogFormat{"mode": mode})
		panic(nil) // unreachable
	}

}

// MassEncode encodes the receiver to w using the given protocol encoding.
func (h *BlockHeader) MassEncode(w io.Writer, mode CodecMode) error {
	return h.Serialize(w, mode)
}

// Serialize encodes a block header from r into the receiver
func (h *BlockHeader) Serialize(w io.Writer, mode CodecMode) error {
	return writeBlockHeader(w, h, mode)
}

// writeBlockHeader writes a block header to w.
func writeBlockHeader(w io.Writer, bh *BlockHeader, mode CodecMode) error {
	pb := bh.ToProto()

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
		// Write every elements of blockHeader
		return writeAll()

	case ID:
		// Write every elements of blockHeader
		return writeAll()

	case PoCID:
		// Write elements excepts for Sig2
		_, err := w.Write(pb.BytesPoC())
		return err

	case ChainID:
		if pb.Height > 0 {
			logging.CPrint(logging.FATAL, "ChainID only be calc for genesis block", logging.LogFormat{"height": pb.Height})
			panic(nil) // unreachable
		}
		// Write elements excepts for ChainID
		_, err := w.Write(pb.BytesChainID())
		return err

	default:
		logging.CPrint(logging.FATAL, "writeBlockHeader invalid CodecMode", logging.LogFormat{"mode": mode})
		panic(nil) // unreachable
	}

}

func (h *BlockHeader) Bytes(mode CodecMode) ([]byte, error) {
	var w bytes.Buffer
	err := h.Serialize(&w, mode)
	if err != nil {
		return nil, err
	}
	serializedBlockHeader := w.Bytes()

	return serializedBlockHeader, nil
}

// GetChainID calc chainID, only block with 0 height can be calc
func (h *BlockHeader) GetChainID() (Hash, error) {
	if h.Height > 0 {
		return Hash{}, errors.New(fmt.Sprintf("invalid height %d to calc chainID", h.Height))
	}
	buf, err := h.Bytes(ChainID)
	if err != nil {
		return Hash{}, err
	}
	return DoubleHashH(buf), nil
}

func NewBlockHeader(prevHash *Hash, merkleRootHash *Hash, target *big.Int,
	proof *poc.Proof, pubKey *btcec.PublicKey, sigQ *btcec.Signature,
	sig2 *btcec.Signature, height uint64, challenge *big.Int) *BlockHeader {

	return &BlockHeader{
		Version:         BlockVersion,
		Previous:        *prevHash,
		TransactionRoot: *merkleRootHash,
		Timestamp:       time.Unix(time.Now().Unix(), 0),
		Target:          target,
		Proof:           proof,
		SigQ:            sigQ,
		PubKey:          pubKey,
		Sig2:            sig2,
		Height:          height,
		Challenge:       challenge,
	}
}

func NewBlockHeaderFromBytes(bhBytes []byte, mode CodecMode) (*BlockHeader, error) {
	bh := NewEmptyBlockHeader()
	bhr := bytes.NewReader(bhBytes)

	err := bh.Deserialize(bhr, mode)
	if err != nil {
		return nil, err
	}

	return bh, nil
}

func NewEmptyBigInt() *big.Int {
	return new(big.Int).SetUint64(0)
}

func NewEmptyPoCSignature() *btcec.Signature {
	return &btcec.Signature{
		R: NewEmptyBigInt(),
		S: NewEmptyBigInt(),
	}
}

func NewEmptyPoCPublicKey() *btcec.PublicKey {
	return &btcec.PublicKey{
		X: NewEmptyBigInt(),
		Y: NewEmptyBigInt(),
	}
}

func NewEmptyBlockHeader() *BlockHeader {
	return &BlockHeader{
		Timestamp: time.Unix(0, 0),
		Target:    NewEmptyBigInt(),
		Challenge: NewEmptyBigInt(),
		Proof:     poc.NewEmptyProof(),
		PubKey:    NewEmptyPoCPublicKey(),
		SigQ:      NewEmptyPoCSignature(),
		Sig2:      NewEmptyPoCSignature(),
		BanList:   make([]*btcec.PublicKey, 0),
	}
}

// PoCHash generate hash of all PoC needed elements in block header
func (h *BlockHeader) PoCHash() (Hash, error) {
	buf, err := h.Bytes(PoCID)
	if err != nil {
		return Hash{}, err
	}

	return DoubleHashH(buf), nil
}

// ToProto get proto BlockHeader from wire BlockHeader
func (h *BlockHeader) ToProto() *wirepb.BlockHeader {
	proof := &wirepb.Proof{
		X:         h.Proof.X,
		XPrime:    h.Proof.X_prime,
		BitLength: int32(h.Proof.BitLength),
	}

	banList := make([]*wirepb.PublicKey, len(h.BanList), len(h.BanList))
	for i, pub := range h.BanList {
		banList[i] = wirepb.PublicKeyToProto(pub)
	}

	return &wirepb.BlockHeader{
		ChainID:         h.ChainID.ToProto(),
		Version:         h.Version,
		Height:          h.Height,
		Timestamp:       h.Timestamp.Unix(),
		Previous:        h.Previous.ToProto(),
		TransactionRoot: h.TransactionRoot.ToProto(),
		ProposalRoot:    h.ProposalRoot.ToProto(),
		Target:          wirepb.BigIntToProto(h.Target),
		Challenge:       wirepb.BigIntToProto(h.Challenge),
		PubKey:          wirepb.PublicKeyToProto(h.PubKey),
		Proof:           proof,
		SigQ:            wirepb.SignatureToProto(h.SigQ),
		Sig2:            wirepb.SignatureToProto(h.Sig2),
		BanList:         banList,
	}
}

// FromProto load proto BlockHeader into wire BlockHeader
func (h *BlockHeader) FromProto(pb *wirepb.BlockHeader) {
	pub := new(btcec.PublicKey)
	wirepb.ProtoToPublicKey(pb.PubKey, pub)
	proof := &poc.Proof{
		X:         pb.Proof.X,
		X_prime:   pb.Proof.XPrime,
		BitLength: int(pb.Proof.BitLength),
	}
	sigQ := new(btcec.Signature)
	wirepb.ProtoToSignature(pb.SigQ, sigQ)
	sig2 := new(btcec.Signature)
	wirepb.ProtoToSignature(pb.Sig2, sig2)
	banList := make([]*btcec.PublicKey, len(pb.BanList), len(pb.BanList))
	for i, pk := range pb.BanList {
		pub := new(btcec.PublicKey)
		wirepb.ProtoToPublicKey(pk, pub)
		banList[i] = pub
	}

	h.ChainID = *NewHashFromProto(pb.ChainID)
	h.Version = pb.Version
	h.Height = pb.Height
	h.Timestamp = time.Unix(pb.Timestamp, 0)
	h.Previous = *NewHashFromProto(pb.Previous)
	h.TransactionRoot = *NewHashFromProto(pb.TransactionRoot)
	h.ProposalRoot = *NewHashFromProto(pb.ProposalRoot)
	h.Target = wirepb.ProtoToBigInt(pb.Target)
	h.Challenge = wirepb.ProtoToBigInt(pb.Challenge)
	h.PubKey = pub
	h.Proof = proof
	h.SigQ = sigQ
	h.Sig2 = sig2
	h.BanList = banList
}

// NewBlockHeaderFromProto get wire BlockHeader from proto BlockHeader
func NewBlockHeaderFromProto(pb *wirepb.BlockHeader) *BlockHeader {
	h := new(BlockHeader)
	h.FromProto(pb)
	return h
}
