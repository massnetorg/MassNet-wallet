package wirepb

import (
	"bytes"
	"encoding/binary"
	"math/big"

	"massnet.org/mass-wallet/btcec"
)

// BigIntToProto get proto BigInt from golang big.Int
func BigIntToProto(x *big.Int) *BigInt {
	pb := new(BigInt)
	pb.RawAbs = x.Bytes()
	return pb
}

// ProtoToBigInt get golang big.Int from proto BigInt
func ProtoToBigInt(pb *BigInt) *big.Int {
	return new(big.Int).SetBytes(pb.RawAbs)
}

func (m *BigInt) Bytes() []byte {
	buf := make([]byte, len(m.RawAbs), len(m.RawAbs))
	copy(buf, m.RawAbs)
	return buf
}

// NewEmptyPublicKey returns new empty initialized proto PublicKey
func NewEmptyPublicKey() *PublicKey {
	return &PublicKey{
		RawX: &BigInt{RawAbs: make([]byte, 0)},
		RawY: &BigInt{RawAbs: make([]byte, 0)},
	}
}

// PublicKeyToProto accespts a btcec PublicKey, returns a proto PublicKey
func PublicKeyToProto(pub interface{}) *PublicKey {
	pb := NewEmptyPublicKey()
	switch p := pub.(type) {
	case *btcec.PublicKey:
		pb.RawX.RawAbs = p.X.Bytes()
		pb.RawY.RawAbs = p.Y.Bytes()
	}
	return pb
}

// ProtoToPublicKey accepts a proto PublicKey and a btcec PublicKey,
// fills content into the latter
func ProtoToPublicKey(pb *PublicKey, pub interface{}) {
	switch p := pub.(type) {
	case *btcec.PublicKey:
		p.Curve = btcec.S256()
		p.X = ProtoToBigInt(pb.RawX)
		p.Y = ProtoToBigInt(pb.RawY)
	}
}

func (m *PublicKey) Bytes() []byte {
	return bytes.Join([][]byte{m.RawX.Bytes(), m.RawY.Bytes()}, []byte(""))
}

// NewEmptySignature returns new empty initialized proto Signature
func NewEmptySignature() *Signature {
	return &Signature{
		RawR: &BigInt{RawAbs: make([]byte, 0)},
		RawS: &BigInt{RawAbs: make([]byte, 0)},
	}
}

// SignatureToProto accepts a btcec Signature, returns a proto Signature
func SignatureToProto(sig interface{}) *Signature {
	pb := NewEmptySignature()
	switch s := sig.(type) {
	case *btcec.Signature:
		pb.RawR.RawAbs = s.R.Bytes()
		pb.RawS.RawAbs = s.S.Bytes()
	}
	return pb
}

// ProtoToSignature accepts a proto Signture and a btcec Signture,
// fills content into the latter
func ProtoToSignature(pb *Signature, sig interface{}) {
	switch s := sig.(type) {
	case *btcec.Signature:
		s.R = ProtoToBigInt(pb.RawR)
		s.S = ProtoToBigInt(pb.RawS)
	}
}

func (m *Signature) Bytes() []byte {
	return bytes.Join([][]byte{m.RawR.Bytes(), m.RawS.Bytes()}, []byte(""))
}

// NewEmptyPrivateKey returns new empty initialized proto PrivateKey
func NewEmptyPrivateKey() *PrivateKey {
	pub := NewEmptyPublicKey()
	return &PrivateKey{
		RawPub: pub,
		RawD:   &BigInt{RawAbs: make([]byte, 0)},
	}
}

func (m *Hash) Bytes() []byte {
	var s0, s1, s2, s3 [8]byte
	binary.LittleEndian.PutUint64(s0[:], m.S0)
	binary.LittleEndian.PutUint64(s1[:], m.S1)
	binary.LittleEndian.PutUint64(s2[:], m.S2)
	binary.LittleEndian.PutUint64(s3[:], m.S3)
	return bytes.Join([][]byte{s0[:], s1[:], s2[:], s3[:]}, []byte(""))
}
