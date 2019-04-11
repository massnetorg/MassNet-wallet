package config

import (
	"encoding/hex"
	"time"

	"math/big"

	"github.com/massnetorg/MassNet-wallet/poc"
	"github.com/massnetorg/MassNet-wallet/btcec"
	"github.com/massnetorg/MassNet-wallet/wire"
)

// genesisCoinbaseTx is the coinbase transaction for genesis block
var genesisCoinbaseTx = wire.MsgTx{
	Version: 1,
	TxIn: []*wire.TxIn{
		{
			PreviousOutPoint: wire.OutPoint{
				Hash:  mustDecodeHash("0000000000000000000000000000000000000000000000000000000000000000"),
				Index: 0xffffffff,
			},
			Sequence: 0xffffffff,
			Witness:  wire.TxWitness{mustDecodeString("0000000000000000000000000000000000000000000000000000000000000000")},
		},
	},
	TxOut: []*wire.TxOut{
		{
			Value:    0x12a05f200,
			PkScript: mustDecodeString("0014c1514fcc0de63e0558524e0c5669c0821424bb2a"),
		},
	},
	LockTime: 0,
	Payload:  mustDecodeString("000c4d41535320544553544e4554"),
}

var genesisHeader = wire.BlockHeader{
	ChainID:         mustDecodeHash("f03d58e3ac59d98d51e4c8d101f4aeaaaf170d8074f5f97c8d76f94100e080cd"),
	Version:         1,
	Height:          0,
	Timestamp:       time.Unix(0x5c9c7f00, 0), // 2019-03-28 16:00:00 +0800 CST Beijing, 1553760000
	Previous:        mustDecodeHash("0000000000000000000000000000000000000000000000000000000000000000"),
	TransactionRoot: mustDecodeHash("db0c4b2eecef52f1681b53413b2f2d4226a9033067b07b8758f4ca7254bb29b8"),
	ProposalRoot:    mustDecodeHash("32c3867350eec5797cfcc9cd9024f6a9e2dfdc2f5b02b6c6332a1d8a55b6c4a1"),
	Target:          hexToBigInt("20000000"),
	Challenge:       hexToBigInt("6841135c02ed85e5a175ba0fa25e44c4bdf08a6abb588237781544bbdc5ee528"),
	PubKey: &btcec.PublicKey{
		Curve: btcec.S256(),
		X:     hexToBigInt("510ac8ee4dda12e2ea6b351e98d2e9b4a092db1be492aaedfdbe5da7195d7bb1"),
		Y:     hexToBigInt("2e601752a3eda47ea7dc33275d173055846b3f14f24551877ce9df586f9d2af1"),
	},
	Proof: &poc.Proof{
		X:         mustDecodeString("ab5d05"),
		X_prime:   mustDecodeString("bda502"),
		BitLength: 20,
	},
	SigQ: &btcec.Signature{
		R: hexToBigInt("ab825f340cc228783e1c082437a3c725c232ae798c9e530a26df4cdc32f76551"),
		S: hexToBigInt("594ef8f7dcf79655bc3ef39fa9153761290575419362abb76d856c6d27811f87"),
	},
	Sig2: &btcec.Signature{
		R: hexToBigInt("e35d417c8e9fb91e7426c93092eae8187c3d99c80d086785d3eb8836d4fd46f7"),
		S: hexToBigInt("3378e12eb92b5a314d284ed11219786a1366d7fac56f5547652bdbe52f251ec9"),
	},
	BanList: make([]*btcec.PublicKey, 0),
}

// genesisBlock defines the genesis block of the block chain which serves as the
// public transaction ledger.
var genesisBlock = wire.MsgBlock{
	Header: genesisHeader,
	Proposals: wire.ProposalArea{
		AllCount:        0,
		PunishmentCount: 0,
		PunishmentArea:  make([]*wire.Proposal, 0),
		OtherArea:       make([]*wire.Proposal, 0),
	},
	Transactions: []*wire.MsgTx{&genesisCoinbaseTx},
}

var genesisHash = mustDecodeHash("fbe44dbc9605e10edb03a6f860ded0bcdabe8e6f60c8de79f1e6f9d4961afd72")

var genesisChainID = mustDecodeHash("f03d58e3ac59d98d51e4c8d101f4aeaaaf170d8074f5f97c8d76f94100e080cd")

func hexToBigInt(str string) *big.Int {
	return new(big.Int).SetBytes(mustDecodeString(str))
}

func mustDecodeString(str string) []byte {
	buf, err := hex.DecodeString(str)
	if err != nil {
		panic(err)
	}
	return buf
}

func mustDecodeHash(str string) wire.Hash {
	h, err := wire.NewHashFromStr(str)
	if err != nil {
		panic(err)
	}
	return *h
}
