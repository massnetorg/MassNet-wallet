package wire

import (
	crand "crypto/rand"
	"math/big"
	"math/rand"
	"time"

	"github.com/massnetorg/MassNet-wallet/poc"
	"github.com/massnetorg/MassNet-wallet/btcec"
	wirepb "github.com/massnetorg/MassNet-wallet/wire/pb"
)

func mockBlock(txCount int) *MsgBlock {
	txs := make([]*MsgTx, txCount, txCount)
	txs[0] = mockCoinbaseTx()
	for i := 1; i < txCount; i++ {
		txs[i] = mockTx()
	}

	blk := &MsgBlock{
		Header:       *mockHeader(),
		Proposals:    *mockProposalArea(),
		Transactions: txs,
	}

	for _, fpk := range blk.Proposals.PunishmentArea {
		blk.Header.BanList = append(blk.Header.BanList, fpk.content.(*FaultPubKey).Pk)
	}

	return blk
}

// mockHeader mocks a blockHeader.
func mockHeader() *BlockHeader {
	return &BlockHeader{
		ChainID:         mockHash(),
		Version:         1,
		Height:          rand.Uint64(),
		Timestamp:       time.Unix(rand.Int63(), 0),
		Previous:        mockHash(),
		TransactionRoot: mockHash(),
		ProposalRoot:    mockHash(),
		Target:          mockBigInt(),
		Challenge:       mockBigInt(),
		PubKey:          mockPublicKey(),
		Proof: &poc.Proof{
			X:         mockLenBytes(3),
			X_prime:   mockLenBytes(3),
			BitLength: rand.Intn(20)*2 + 20,
		},
		SigQ:    mockSignature(),
		Sig2:    mockSignature(),
		BanList: make([]*btcec.PublicKey, 0),
	}
}

// mockBigInt mocks *big.Int with 32 bytes.
func mockBigInt() *big.Int {
	return new(big.Int).SetBytes(mockLenBytes(32))
}

// mockPublicKey mocks *btcec.PublicKey.
func mockPublicKey() *btcec.PublicKey {
	priv, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		panic(err)
	}
	return priv.PubKey()
}

// mockSignature mocks *btcec.Signature
func mockSignature() *btcec.Signature {
	priv, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		panic(err)
	}
	hash := mockHash()
	sig, err := priv.Sign(hash[:])
	if err != nil {
		panic(err)
	}
	return sig
}

// mockProposalArea mocks ProposalArea.
func mockProposalArea() *ProposalArea {
	punishmentCount := rand.Intn(10)
	punishments := make([]*Proposal, punishmentCount, punishmentCount)
	for i := range punishments {
		punishments[i] = mockPunishment()
	}

	proposalCount := rand.Intn(5)
	proposals := make([]*Proposal, proposalCount, proposalCount)
	for i := range proposals {
		proposals[i] = mockProposal()
	}

	pa := new(ProposalArea)
	pa.PunishmentArea = punishments
	pa.PunishmentCount = uint16(punishmentCount)
	pa.OtherArea = proposals
	pa.AllCount = uint16(punishmentCount + proposalCount)

	return pa
}

// mockPunishment mocks proposal in punishmentArea.
func mockPunishment() *Proposal {
	fpk := &FaultPubKey{
		Pk:        mockPublicKey(),
		Testimony: [2]*BlockHeader{mockHeader(), mockHeader()},
	}
	fpk.Testimony[0].PubKey = fpk.Pk
	fpk.Testimony[1].PubKey = fpk.Pk

	return &Proposal{
		version:      ProposalVersion,
		proposalType: typeFaultPubKey,
		content:      fpk,
	}
}

// mockProposal mocks normal proposal.
func mockProposal() *Proposal {
	var content proposalContent

	length := rand.Intn(30) + 10
	content = &AnyMessage{
		Data:   mockLenBytes(length),
		Length: uint16(length),
	}

	return &Proposal{
		version:      ProposalVersion,
		proposalType: typeAnyMessage,
		content:      content,
	}
}

// mockTx mocks a tx (scripts are random bytes).
func mockTx() *MsgTx {
	rand.Seed(time.Now().Unix())
	return &MsgTx{
		Version: 1,
		TxIn: []*TxIn{
			{
				PreviousOutPoint: OutPoint{
					Hash:  mockHash(),
					Index: rand.Uint32() % 20,
				},
				Witness:  [][]byte{mockLenBytes(rand.Intn(50) + 100), mockLenBytes(rand.Intn(50) + 100)},
				Sequence: 0xffffffff,
			},
		},
		TxOut: []*TxOut{
			{
				Value:    rand.Int63(),
				PkScript: mockLenBytes(rand.Intn(10) + 20),
			},
		},
		LockTime: 0,
		Payload:  mockLenBytes(rand.Intn(20)),
	}
}

// mockCoinbaseTx mocks a coinbase tx.
func mockCoinbaseTx() *MsgTx {
	rand.Seed(time.Now().Unix())
	return &MsgTx{
		Version: 1,
		TxIn: []*TxIn{
			{
				PreviousOutPoint: OutPoint{
					Hash:  Hash{},
					Index: 0xffffffff,
				},
				Witness:  [][]byte{mockLenBytes(rand.Intn(50) + 100), mockLenBytes(rand.Intn(50) + 100)},
				Sequence: 0xffffffff,
			},
		},
		TxOut: []*TxOut{
			{
				Value:    rand.Int63(),
				PkScript: mockLenBytes(rand.Intn(10) + 20),
			},
		},
		LockTime: 0,
		Payload:  mockLenBytes(rand.Intn(20)),
	}
}

// mockHash mocks a hash.
func mockHash() Hash {
	pb := new(wirepb.Hash)
	pb.S0 = rand.Uint64()
	pb.S1 = rand.Uint64()
	pb.S2 = rand.Uint64()
	pb.S3 = rand.Uint64()
	return *NewHashFromProto(pb)
}

// mockLenBytes mocks bytes with given length.
func mockLenBytes(len int) []byte {
	buf := make([]byte, len, len)
	crand.Read(buf)
	return buf
}
