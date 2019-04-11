package api_test

import (
	"bytes"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"

	pb "massnet.org/mass-wallet/api/proto"
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
	if a[i].Balance < a[j].Balance {
		return true
	} else if a[i].Balance > a[j].Balance {
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
		tests[i] = &pb.AddressAndBalance{
			Address: strArr[i],
			Balance: float64(rand.Intn(maxValue) + 1),
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
