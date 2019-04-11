package wirepb

import (
	"github.com/golang/protobuf/proto"

	"testing"
	"time"
)

// TestEncodeTxTimeUsage tests encode 2000 mocked txs.
func TestEncodeTxTimeUsage(t *testing.T) {
	var txCount = 2000

	start := time.Now()
	for i := 0; i < txCount; i++ {
		tx := mockTx()
		_, err := proto.Marshal(tx)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	}

	t.Log(time.Since(start))
}

// BenchmarkEncodeTx benchmarks tx encode.
func BenchmarkEncodeTx(b *testing.B) {
	b.StopTimer()

	tx := mockTx()

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := proto.Marshal(tx)
		if err != nil {
			b.Error(err)
			b.FailNow()
		}
	}
}

// TestDecodeTxTimeUsage tests decode 2000 mocked txs.
func TestDecodeTxTimeUsage(t *testing.T) {
	var txCount = 2000

	txs := make([][]byte, txCount, txCount)
	for i := 0; i < txCount; i++ {
		tx := mockTx()
		buf, err := proto.Marshal(tx)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		txs[i] = buf
	}

	start := time.Now()
	for i := 0; i < txCount; i++ {
		newPb := new(Tx)
		err := proto.Unmarshal(txs[i], newPb)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	}

	t.Log(time.Since(start))
}

// BenchmarkDecodeTx benchmarks tx decode.
func BenchmarkDecodeTx(b *testing.B) {
	b.StopTimer()

	tx := mockTx()
	buf, err := proto.Marshal(tx)
	if err != nil {
		b.Error(err)
		b.FailNow()
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		newPb := new(Tx)
		err := proto.Unmarshal(buf, newPb)
		if err != nil {
			b.Error(err)
			b.FailNow()
		}
	}
}

// TestEncodeBlockTimeUsage tests encode 500 mocked blocks with 3000 txs.
func TestEncodeBlockTimeUsage(t *testing.T) {
	var blockCount = 500
	block := mockBlock(3000)

	start := time.Now()
	for i := 0; i < blockCount; i++ {
		_, err := proto.Marshal(block)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	}

	t.Log(time.Since(start))
}

// BenchmarkEncodeBlock benchmarks block encode.
func BenchmarkEncodeBlock(b *testing.B) {
	b.StopTimer()

	block := mockBlock(3000)

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		_, err := proto.Marshal(block)
		if err != nil {
			b.Error(err)
			b.FailNow()
		}
	}
}

// TestDecodeBlockTimeUsage tests decode 500 mocked block with 3000 txs.
func TestDecodeBlockTimeUsage(t *testing.T) {
	var blockCount = 500
	block := mockBlock(3000)
	buf, err := proto.Marshal(block)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	start := time.Now()
	for i := 0; i < blockCount; i++ {
		newPb := new(Block)
		err := proto.Unmarshal(buf, newPb)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
	}

	t.Log(time.Since(start))
}

// BenchmarkDecodeBlock benchmarks block decode.
func BenchmarkDecodeBlock(b *testing.B) {
	b.StopTimer()

	block := mockBlock(3000)
	buf, err := proto.Marshal(block)
	if err != nil {
		b.Error(err)
		b.FailNow()
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		newPb := new(Block)
		err := proto.Unmarshal(buf, newPb)
		if err != nil {
			b.Error(err)
			b.FailNow()
		}
	}
}
