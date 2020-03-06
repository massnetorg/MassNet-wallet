package keystore

import (
	"fmt"
	"strings"
	"testing"

	"encoding/hex"
	"massnet.org/mass-wallet/massutil/base58"
	"math"
	"math/big"
	"sync"
)

func TestBase58(t *testing.T) {
	testBytes := []byte("abcd")
	base58String := base58.Encode(testBytes)
	testString := string(testBytes)
	if strings.Compare(base58String, testString) != 0 {
		fmt.Println("base58 string: ", base58String)
		fmt.Println("string: ", testString)
		fmt.Println("not same")
	} else {
		fmt.Println("same")
	}
}

func TestJson(t *testing.T) {
	k := &Keystore{
		Crypto: cryptoJSON{
			Cipher: "a",
		},
		HDpath: hdPath{
			Purpose:          44,
			Coin:             1,
			Account:          1,
			ExternalChildNum: 2,
			InternalChildNum: 0,
		},
	}
	kbytes := k.Bytes()
	getKeystoreFromJson(kbytes)
}

type A struct {
	mu   sync.Mutex
	name map[int]string
}

type B struct {
	A
	total int
}

func newA() *A {
	m := make(map[int]string)
	m[1] = "One"
	m[2] = "Two"
	m[3] = "Three"
	return &A{
		name: m,
	}
}

func newB(a *A) *B {
	return &B{
		A:     *a,
		total: len(a.name),
	}
}

func (a *A) add(key int, value string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.name[key] = value
}

func (b *B) add(key int, value string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.A.add(key, value)
}

//func TestLock(t *testing.T) {
//	a := newA()
//	b := newB(a)
//
//	b.add(4, "Four")
//}

func TestPubKeyToAccountID(t *testing.T) {
	maxBytes := [32]byte{
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
		255, 255, 255, 255, 255, 255, 255, 255,
	}
	minBytes := [32]byte{
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	}

	maxString := hex.EncodeToString(maxBytes[:])
	minString := hex.EncodeToString(minBytes[:])
	t.Logf("max: %v, len: %v", maxString, len([]byte(maxString)))
	t.Logf("min: %v, len: %v", minString, len([]byte(minString)))
}

func TestBigInt(t *testing.T) {
	sn := big.NewInt(4000000000)
	v := big.NewInt(100000000000)
	tv := big.NewInt(300000000000)
	temp := big.NewInt(1)
	temp.Mul(sn, v)
	res := big.NewInt(1)
	res.Div(temp, tv)
	t.Logf("result: %v", res.Int64())
}

type txout struct {
	v int
	a string
}

type tx struct {
	tos []*txout
}

func (t *tx) add(to *txout) {
	t.tos = append(t.tos, to)
}

func (t *tx) remove(index int) {
	t.tos = append(t.tos[:index], t.tos[index+1:]...)
}

func TestRemove(t *testing.T) {
	va := &tx{
		tos: make([]*txout, 0),
	}
	as := []string{
		"one", "two", "three", "four", "five",
	}
	for i := 0; i < 5; i++ {
		va.add(&txout{
			v: i + 1,
			a: as[i],
		})
	}
	fmt.Println(va)
	va.remove(0)
	fmt.Println(va)
	va.remove(2)
	fmt.Println(va)

}

func TestSafeAdd(t *testing.T) {
	var safeUint32Add = func(a, b uint32) uint32 {
		if (a+b) < a || (a+b) < b {
			return math.MaxUint32
		}
		return a + b
	}
	data := []struct {
		a      uint32
		b      uint32
		result uint32
	}{
		{1, 1, 2},
		{math.MaxUint32 - 10, 10, math.MaxUint32},
		{1, math.MaxUint32, math.MaxUint32},
		{math.MaxUint32 - 5, 10, math.MaxUint32},
		{math.MaxUint32, math.MaxUint32, math.MaxUint32},
	}

	for _, d := range data {
		res := safeUint32Add(d.a, d.b)
		if res != d.result {
			t.Errorf("error, a: %v, b: %v, expected: %v, res: %v", d.a, d.b, d.result, res)
		}
	}
}

func TestLogic(t *testing.T) {
	countMap := make(map[uint64]int)
	countMap[1] = 2
	countMap[2] = 3
	testdata := []uint64{0, 1, 2, 3}
	for _, idx := range testdata {
		if _, exist := countMap[idx]; !exist {
			countMap[idx] = 1
		} else {
			countMap[idx]++
		}
	}
	t.Log(countMap)
}
