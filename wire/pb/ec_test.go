package wirepb

import (
	"math/big"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"massnet.org/mass-wallet/btcec"
)

// TestBigInt tests encode/decode BigInt.
func TestBigInt(t *testing.T) {
	x := new(big.Int).SetUint64(uint64(0xffffffffffffffff))
	x = x.Mul(x, x)
	pb := BigIntToProto(x)
	y := ProtoToBigInt(pb)
	if !reflect.DeepEqual(x, y) {
		t.Error("obj BigInt not equal")
	}
}

// TestPublicKey test encode/decode PublicKey.
func TestPublicKey(t *testing.T) {
	btcPriv, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	btcPub := btcPriv.PubKey()
	btcProto := PublicKeyToProto(btcPub)
	btcPubNew := new(btcec.PublicKey)
	ProtoToPublicKey(btcProto, btcPubNew)
	if !reflect.DeepEqual(btcPub, btcPubNew) {
		t.Error("obj btcec PublicKey not equal")
	}
}

// TestSignature tests encode/decode Signature.
func TestSignature(t *testing.T) {
	rand.Seed(time.Now().Unix())

	btcSig := &btcec.Signature{
		R: new(big.Int).SetUint64(rand.Uint64()),
		S: new(big.Int).SetUint64(rand.Uint64()),
	}
	btcProto := SignatureToProto(btcSig)
	btcSigNew := new(btcec.Signature)
	ProtoToSignature(btcProto, btcSigNew)
	if !reflect.DeepEqual(btcSig, btcSigNew) {
		t.Error("obj btcec Signature not equal")
	}
}
