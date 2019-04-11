// Modified for MassNet
// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"fmt"
	"io"

	"massnet.org/mass-wallet/logging"

	"github.com/golang/protobuf/proto"

	wirepb "massnet.org/mass-wallet/wire/pb"
)

// defaultTransactionAlloc is the default size used for the backing array
// for transactions.  The transaction array will dynamically grow as needed, but
// this figure is intended to provide enough space for the number of
// transactions in the vast majority of blocks without needing to grow the
// backing array multiple times.
const defaultTransactionAlloc = 2048

// MaxBlockPayload is the maximum bytes a block message can be in bytes.
// After Segregated Witness, the max block payload has been raised to 4MB.
const MaxBlockPayload = 4000000

// maxTxPerBlock is the maximum number of transactions that could
// possibly fit into a block.
const MaxTxPerBlock = (MaxBlockPayload / minTxPayload) + 1

// TxLoc holds locator data for the offset and length of where a transaction is
// located within a MsgBlock data buffer.
type TxLoc struct {
	TxStart int
	TxLen   int
}

type MsgBlock struct {
	Header       BlockHeader
	Proposals    ProposalArea
	Transactions []*MsgTx
}

func NewEmptyMsgBlock() *MsgBlock {
	return &MsgBlock{
		Header:       *NewEmptyBlockHeader(),
		Proposals:    *newEmptyProposalArea(),
		Transactions: make([]*MsgTx, 0),
	}
}

// AddTransaction adds a transaction to the message.
func (msg *MsgBlock) AddTransaction(tx *MsgTx) error {
	msg.Transactions = append(msg.Transactions, tx)
	return nil

}

// ClearTransactions removes all transactions from the message.
func (msg *MsgBlock) ClearTransactions() {
	msg.Transactions = make([]*MsgTx, 0, defaultTransactionAlloc)
}

// MassDecode decodes r using the given protocol encoding into the receiver.
func (msg *MsgBlock) MassDecode(r io.Reader, mode CodecMode) error {
	switch mode {
	case DB:
		// Read BLockBase length
		blockBaseLength, err := ReadVarInt(r, 0)
		if err != nil {
			return err
		}

		// Read BlockBase
		baseData := make([]byte, blockBaseLength, blockBaseLength)
		_, err = r.Read(baseData)
		if err != nil {
			return err
		}
		basePb := new(wirepb.BlockBase)
		err = proto.Unmarshal(baseData, basePb)
		if err != nil {
			return err
		}
		base, err := NewBlockBaseFromProto(basePb)
		if err != nil {
			return err
		}

		// Read txCount
		txCount, err := ReadVarInt(r, 0)
		if err != nil {
			return err
		}

		// Read Transactions
		txs := make([]*MsgTx, txCount, txCount)
		for i := 0; i < len(txs); i++ {
			// Read tx length
			txLen, err := ReadVarInt(r, 0)
			if err != nil {
				return err
			}
			// Read tx
			tx := new(MsgTx)
			buf := make([]byte, txLen, txLen)
			_, err = r.Read(buf)
			if err != nil {
				return err
			}
			err = tx.Deserialize(bytes.NewReader(buf), DB)
			if err != nil {
				return err
			}
			txs[i] = tx
		}

		// fill element
		msg.Header = base.Header
		msg.Proposals = base.Proposals
		msg.Transactions = txs

	case Packet:
		var buf bytes.Buffer
		_, err := buf.ReadFrom(r)
		if err != nil {
			return err
		}

		pb := new(wirepb.Block)
		err = proto.Unmarshal(buf.Bytes(), pb)
		if err != nil {
			return err
		}

		err = msg.FromProto(pb)
		if err != nil {
			return err
		}

	default:
		logging.CPrint(logging.FATAL, "MsgTx.MassDecode: invalid CodecMode", logging.LogFormat{"mode": mode})
		panic(nil) // unreachable
	}

	// Prevent more transactions than could possibly fit into a block.
	// It would be possible to cause memory exhaustion and panics without
	// a sane upper bound on this count.
	txCount := len(msg.Transactions)
	if txCount > MaxTxPerBlock {
		str := fmt.Sprintf("too many transactions to fit into a block "+
			"[count %d, max %d]", txCount, MaxTxPerBlock)
		return messageError("MsgBlock.MassDecode", str)
	}

	return nil
}

// Deserialize decodes a block from r into the receiver
func (msg *MsgBlock) Deserialize(r io.Reader, mode CodecMode) error {
	return msg.MassDecode(r, mode)
}

// DeserializeTxLoc decodes r in the same manner Deserialize does, but it takes
// a byte buffer instead of a generic reader and returns a slice containing the
// start and length of each transaction within the raw data that is being
// deserialized.
func (msg *MsgBlock) DeserializeTxLoc(r *bytes.Buffer) ([]TxLoc, error) {
	fullLen := r.Len()

	// Read BLockBase length
	blockBaseLength, err := ReadVarInt(r, 0)
	if err != nil {
		return nil, err
	}

	// Read blockBase
	baseData := make([]byte, blockBaseLength, blockBaseLength)
	if n, err := r.Read(baseData); uint64(n) != blockBaseLength || err != nil {
		return nil, err
	}

	// Read txCount
	txCount, err := ReadVarInt(r, 0)
	if err != nil {
		return nil, err
	}

	// Prevent more transactions than could possibly fit into a block.
	// It would be possible to cause memory exhaustion and panics without
	// a sane upper bound on this count.
	if txCount > MaxTxPerBlock {
		str := fmt.Sprintf("too many transactions to fit into a block "+
			"[count %d, max %d]", txCount, MaxTxPerBlock)
		return nil, messageError("MsgBlock.DeserializeTxLoc", str)
	}

	// Deserialize each transaction while keeping track of its location
	// within the byte stream.
	msg.Transactions = make([]*MsgTx, 0, txCount)
	txLocs := make([]TxLoc, txCount)
	for i := uint64(0); i < txCount; i++ {
		// Read tx length
		txLen, err := ReadVarInt(r, 0)
		if err != nil {
			return nil, err
		}
		// Set txLoc
		txLocs[i].TxStart = fullLen - r.Len()
		txLocs[i].TxLen = int(txLen)
		// Read tx
		tx := new(MsgTx)
		buf := make([]byte, txLen, txLen)
		_, err = r.Read(buf)
		if err != nil {
			return nil, err
		}
		err = tx.Deserialize(bytes.NewReader(buf), DB)
		if err != nil {
			return nil, err
		}
		msg.Transactions = append(msg.Transactions, tx)
	}

	return txLocs, nil
}

// MassEncode encodes the receiver to w using the given protocol encoding.
func (msg *MsgBlock) MassEncode(w io.Writer, mode CodecMode) error {
	switch mode {
	case DB:
		base := BlockBase{Header: msg.Header, Proposals: msg.Proposals}
		pb, err := base.ToProto()
		if err != nil {
			return err
		}
		baseData, err := proto.Marshal(pb)
		if err != nil {
			return err
		}

		// Write BlockBase length & data
		err = writeVarInt(w, 0, uint64(len(baseData)))
		if err != nil {
			return err
		}
		_, err = w.Write(baseData)
		if err != nil {
			return err
		}

		// Write txCount
		err = writeVarInt(w, 0, uint64(len(msg.Transactions)))
		if err != nil {
			return err
		}

		// Write Transactions
		for i := 0; i < len(msg.Transactions); i++ {
			txPb := msg.Transactions[i].ToProto()
			txData, err := proto.Marshal(txPb)
			if err != nil {
				return err
			}
			err = writeVarInt(w, 0, uint64(len(txData)))
			if err != nil {
				return err
			}
			_, err = w.Write(txData)
			if err != nil {
				return err
			}
		}
		return nil

	case Packet:
		pb, err := msg.ToProto()
		if err != nil {
			return err
		}

		buf, err := proto.Marshal(pb)
		if err != nil {
			return err
		}

		_, err = w.Write(buf)
		return err

	case Plain:
		pb, err := msg.ToProto()
		if err != nil {
			return err
		}
		_, err = w.Write(pb.Bytes())
		return err

	default:
		logging.CPrint(logging.FATAL, "MsgTx.MassEncode: invalid CodecMode", logging.LogFormat{"mode": mode})
		panic(nil) // unreachable
	}

}

// Serialize encodes the block to w
func (msg *MsgBlock) Serialize(w io.Writer, mode CodecMode) error {
	return msg.MassEncode(w, mode)
}

// SerializeSize returns the number of bytes it would take to serialize the
// the block.
func (msg *MsgBlock) SerializeSize() int {
	var buf bytes.Buffer
	err := msg.Serialize(&buf, Plain)
	if err != nil {
		return 0
	}

	return len(buf.Bytes())
}

// Command returns the protocol command string for the message.  This is part
// of the Message interface implementation.
func (msg *MsgBlock) Command() string {
	return CmdBlock
}

// MaxPayloadLength returns the maximum length the payload can be for the
// receiver.  This is part of the Message interface implementation.
func (msg *MsgBlock) MaxPayloadLength(pver uint32) uint32 {
	// Block header at 80 bytes + transaction count + max transactions
	// which can vary up to the MaxBlockPayload (including the block header
	// and transaction count).
	return MaxBlockPayload
}

// BlockHash computes the block identifier hash for this block.
func (msg *MsgBlock) BlockHash() Hash {
	return msg.Header.BlockHash()
}

// TxHashes returns a slice of hashes of all of transactions in this block.
func (msg *MsgBlock) TxHashes() ([]Hash, error) {
	hashList := make([]Hash, 0, len(msg.Transactions))
	for _, tx := range msg.Transactions {
		hashList = append(hashList, tx.TxHash())
	}
	return hashList, nil
}

func NewMsgBlock(blockHeader *BlockHeader) *MsgBlock {
	return &MsgBlock{
		Header:       *blockHeader,
		Proposals:    *newEmptyProposalArea(),
		Transactions: make([]*MsgTx, 0, defaultTransactionAlloc),
	}
}

// ToProto get proto Block from wire Block
func (msg *MsgBlock) ToProto() (*wirepb.Block, error) {
	pa, err := msg.Proposals.ToProto()
	if err != nil {
		return nil, err
	}
	h := msg.Header.ToProto()
	txs := make([]*wirepb.Tx, len(msg.Transactions), len(msg.Transactions))
	for i, tx := range msg.Transactions {
		txs[i] = tx.ToProto()
	}

	return &wirepb.Block{
		Header:       h,
		Proposals:    pa,
		Transactions: txs,
	}, nil
}

// FromProto load proto Block into wire Block,
// if error happens, old content is still immutable
func (msg *MsgBlock) FromProto(pb *wirepb.Block) error {
	pa := ProposalArea{}
	err := pa.FromProto(pb.Proposals)
	if err != nil {
		return err
	}
	h := BlockHeader{}
	h.FromProto(pb.Header)
	txs := make([]*MsgTx, len(pb.Transactions), len(pb.Transactions))
	for i, v := range pb.Transactions {
		tx := new(MsgTx)
		tx.FromProto(v)
		txs[i] = tx
	}

	msg.Header = h
	msg.Proposals = pa
	msg.Transactions = txs

	return nil
}

// NewBlockFromProto get wire Block from proto Block
func NewBlockFromProto(pb *wirepb.Block) (*MsgBlock, error) {
	block := new(MsgBlock)
	err := block.FromProto(pb)
	if err != nil {
		return nil, err
	}
	return block, nil
}

type BlockBase struct {
	Header    BlockHeader
	Proposals ProposalArea
}

func (base *BlockBase) ToProto() (*wirepb.BlockBase, error) {
	pa, err := base.Proposals.ToProto()
	if err != nil {
		return nil, err
	}
	h := base.Header.ToProto()

	return &wirepb.BlockBase{
		Header:    h,
		Proposals: pa,
	}, nil
}

func (base *BlockBase) FromProto(pb *wirepb.BlockBase) error {
	pa := ProposalArea{}
	err := pa.FromProto(pb.Proposals)
	if err != nil {
		return err
	}
	h := BlockHeader{}
	h.FromProto(pb.Header)

	base.Header = h
	base.Proposals = pa

	return nil
}

func NewBlockBaseFromProto(pb *wirepb.BlockBase) (*BlockBase, error) {
	base := new(BlockBase)
	err := base.FromProto(pb)
	if err != nil {
		return nil, err
	}
	return base, nil
}
