package api_test

import (
	"bytes"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"massnet.org/mass-wallet/massutil/safetype"

	"github.com/stretchr/testify/assert"
	"massnet.org/mass-wallet/api"
	pb "massnet.org/mass-wallet/api/proto"
	"massnet.org/mass-wallet/consensus"
	"massnet.org/mass-wallet/massutil"
)

var (
	str1   = "ms1q7h9485g0p07qvfspwthr62dqurjc508tg6ku86"
	str2   = "ms1qznpwnat8nrl4ndm8pt3axns0nvrfgv5lm2uzyy"
	str3   = "ms1qn2j4j2qd9lzzpq8w6jecgqw52s9xdnedtpduvy"
	str4   = "ms1q8jvtd44ndyd7v3yu8kt3dj0s0jl9sxnafna2k6"
	str5   = "ms1qjxvt092kefe04ask4k98hqrkscthyqes85l4l8"
	strArr = []string{str1, str2, str3, str4, str5}
)

func BenchmarkCmpString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		strings.Compare(str1, str2)
	}
}

func BenchmarkCmpBytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		bytes.Compare([]byte(str1), []byte(str2))
	}
}

type addrAndBalanceList []*pb.AddressAndBalance

func (a addrAndBalanceList) Len() int {
	return len(a)
}

func (a addrAndBalanceList) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a addrAndBalanceList) Less(i, j int) bool {
	if a[i].Total < a[j].Total {
		return true
	} else if a[i].Total > a[j].Total {
		return false
	}

	return strings.Compare(a[i].Address, a[j].Address) == -1
}

func shuffle(arr addrAndBalanceList) {
	var size = len(arr)
	var shuffleRound = size / 2
	for i := 0; i < shuffleRound; i++ {
		x, y := rand.Intn(size), rand.Intn(size)
		arr[x], arr[y] = arr[y], arr[x]
	}
}

func TestSortAddrAndBalanceList(t *testing.T) {
	rand.Seed(time.Now().UnixNano())
	var strCount = len(strArr)
	var maxValue = strCount / 2
	tests := make(addrAndBalanceList, 5, 5)
	for i := range tests {
		amt, _ := api.AmountToString(int64(rand.Intn(maxValue) + 1))
		tests[i] = &pb.AddressAndBalance{
			Address: strArr[i],
			Total:   amt,
		}
	}
	t.Log(tests)
	sort.Sort(sort.Reverse(tests))
	var answer = fmt.Sprintf("%s", tests)
	var testRound = 10000

	for i := 0; i < testRound; i++ {
		shuffle(tests)
		sort.Sort(sort.Reverse(tests))
		if s := fmt.Sprintf("%s", tests); s != answer {
			t.Errorf("expect %s, got %s", answer, s)
		}
	}
}

func TestAmountToString(t *testing.T) {
	s, err := api.AmountToString(12345678)
	assert.Nil(t, err)
	assert.Equal(t, "0.12345678", s)

	s, err = api.AmountToString(1)
	assert.Nil(t, err)
	assert.Equal(t, "0.00000001", s)

	s, err = api.AmountToString(123456789)
	assert.Nil(t, err)
	assert.Equal(t, "1.23456789", s)

	s, err = api.AmountToString(123456700)
	assert.Nil(t, err)
	assert.Equal(t, "1.234567", s)

	s, err = api.AmountToString(1234500)
	assert.Nil(t, err)
	assert.Equal(t, "0.012345", s)

	s, err = api.AmountToString(123450)
	assert.Nil(t, err)
	assert.Equal(t, "0.0012345", s)

	s, err = api.AmountToString(0)
	assert.Nil(t, err)
	assert.Equal(t, "0", s)

	_, err = api.AmountToString(massutil.MaxAmount().IntValue())
	assert.Nil(t, err)

	s, err = api.AmountToString(-1)
	assert.Equal(t, safetype.ErrUint128Underflow, err)

	s, err = api.AmountToString(massutil.MaxAmount().IntValue() + 1)
	assert.NotNil(t, err)
	assert.True(t, strings.HasPrefix(err.Error(), "amount is out of range:"))
}

func TestStringToAmount(t *testing.T) {
	n, err := api.StringToAmount("123456")
	assert.Nil(t, err)
	assert.Equal(t, uint64(12345600000000), n.UintValue())

	n, err = api.StringToAmount("123456.")
	assert.Nil(t, err)
	assert.Equal(t, uint64(12345600000000), n.UintValue())

	n, err = api.StringToAmount(".12345678")
	assert.Nil(t, err)
	assert.Equal(t, uint64(12345678), n.UintValue())

	n, err = api.StringToAmount("0.")
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), n.UintValue())

	n, err = api.StringToAmount("1.")
	assert.Nil(t, err)
	assert.Equal(t, uint64(100000000), n.UintValue())

	n, err = api.StringToAmount(".0")
	assert.Nil(t, err)
	assert.Equal(t, uint64(0), n.UintValue())

	n, err = api.StringToAmount(".1")
	assert.Nil(t, err)
	assert.Equal(t, uint64(10000000), n.UintValue())

	n, err = api.StringToAmount(".01")
	assert.Nil(t, err)
	assert.Equal(t, uint64(1000000), n.UintValue())

	n, err = api.StringToAmount(".00000001")
	assert.Nil(t, err)
	assert.Equal(t, uint64(1), n.UintValue())

	n, err = api.StringToAmount(".99999999")
	assert.Nil(t, err)
	assert.Equal(t, uint64(99999999), n.UintValue())

	n, err = api.StringToAmount(".000000001")
	assert.NotNil(t, err)
	assert.Equal(t, "precision is too high", err.Error())

	sInt := strconv.FormatInt(int64(consensus.MaxMass+1), 10)
	n, err = api.StringToAmount(sInt + ".00000000")
	assert.NotNil(t, err)
	assert.Equal(t, "integral part is out of range", err.Error())

	sInt = strconv.FormatInt(int64(consensus.MaxMass), 10)
	n, err = api.StringToAmount(sInt + ".00000000")
	assert.Nil(t, err)
	assert.Equal(t, consensus.MaxMass*consensus.MaxwellPerMass, n.UintValue())

	sInt = strconv.FormatInt(int64(consensus.MaxMass), 10)
	n, err = api.StringToAmount(sInt + ".00000001")
	assert.Equal(t, massutil.ErrMaxAmount, err)

	n, err = api.StringToAmount("-1.")
	assert.NotNil(t, err)
	assert.Equal(t, "integral part is out of range", err.Error())

	n, err = api.StringToAmount("1.-1000001")
	assert.NotNil(t, err)
	assert.Equal(t, "illegal number format", err.Error())

	n, err = api.StringToAmount("1.100.")
	assert.NotNil(t, err)
	assert.Equal(t, "illegal number format", err.Error())

	n, err = api.StringToAmount(".100.")
	assert.NotNil(t, err)
	assert.Equal(t, "illegal number format", err.Error())

}
