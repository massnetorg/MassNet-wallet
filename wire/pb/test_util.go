package wirepb

import (
	crand "crypto/rand"
	"math/rand"
	"time"
)

// mockBlock mocks a block with given txCount.
func mockBlock(txCount int) *Block {
	txs := make([]*Tx, txCount, txCount)
	for i := 0; i < txCount; i++ {
		txs[i] = mockTx()
	}

	block := &Block{
		Header:       mockHeader(),
		Proposals:    mockProposalArea(),
		Transactions: txs,
	}
	for i := 0; i < len(block.Proposals.Punishments); i++ {
		block.Header.BanList = append(block.Header.BanList, mockPublicKey())
	}

	return block
}

// mockHeader mocks a blockHeader.
func mockHeader() *BlockHeader {
	return &BlockHeader{
		ChainID:         mockHash(),
		Version:         1,
		Height:          rand.Uint64(),
		Timestamp:       rand.Int63(),
		Previous:        mockHash(),
		TransactionRoot: mockHash(),
		ProposalRoot:    mockHash(),
		Target:          mockBigInt(),
		Challenge:       mockBigInt(),
		PubKey:          mockPublicKey(),
		Proof: &Proof{
			X:         mockLenBytes(3),
			XPrime:    mockLenBytes(3),
			BitLength: rand.Int31n(20)*2 + 20,
		},
		SigQ:    mockSignature(),
		Sig2:    mockSignature(),
		BanList: make([]*PublicKey, 0),
	}
}

func mockBigInt() *BigInt {
	return &BigInt{
		RawAbs: mockLenBytes(32),
	}
}

func mockPublicKey() *PublicKey {
	return &PublicKey{
		RawX: mockBigInt(),
		RawY: mockBigInt(),
	}
}

func mockSignature() *Signature {
	return &Signature{
		RawR: mockBigInt(),
		RawS: mockBigInt(),
	}
}

func mockProposalArea() *ProposalArea {
	punishmentCount := rand.Intn(10)
	punishments := make([]*Punishment, punishmentCount, punishmentCount)
	for i := range punishments {
		punishments[i] = mockPunishment()
	}

	proposalCount := rand.Intn(5)
	proposals := make([]*Proposal, proposalCount, proposalCount)
	for i := range proposals {
		proposals[i] = mockProposal()
	}

	pa := new(ProposalArea)
	pa.Punishments = punishments
	if punishmentCount == 0 {
		pa.PlaceHolder = mockPlaceHolder()
	}
	pa.OtherProposals = proposals

	return pa
}

func mockPunishment() *Punishment {
	return &Punishment{
		Version:    1,
		Type:       0,
		TestimonyA: mockHeader(),
		TestimonyB: mockHeader(),
	}
}

func mockPlaceHolder() *Proposal {
	return &Proposal{
		Version: 1,
		Type:    1,
		Content: mockLenBytes(429 * 2),
	}
}

func mockProposal() *Proposal {
	return &Proposal{
		Version: 1,
		Type:    1 + rand.Int31n(10),
		Content: mockLenBytes(10 + rand.Intn(20)),
	}
}

// mockTx mocks a tx (scripts are random bytes).
func mockTx() *Tx {
	rand.Seed(time.Now().Unix())
	return &Tx{
		Version: 1,
		TxIn: []*TxIn{
			{
				PreviousOutPoint: &OutPoint{
					Hash:  mockHash(),
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
func mockHash() *Hash {
	pb := new(Hash)
	pb.S0 = rand.Uint64()
	pb.S1 = rand.Uint64()
	pb.S2 = rand.Uint64()
	pb.S3 = rand.Uint64()
	return pb
}

// mockLenBytes mocks bytes with given length.
func mockLenBytes(len int) []byte {
	buf := make([]byte, len, len)
	crand.Read(buf)
	return buf
}
