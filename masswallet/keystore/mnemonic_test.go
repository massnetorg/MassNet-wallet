package keystore

import (
	"massnet.org/mass-wallet/masswallet/keystore/wordlists"
	"math"
	"strings"
	"testing"
)

func TestNewMnemonic(t *testing.T) {
	SetWordList(wordlists.English)

	var maxLen int
	minLen := math.MaxInt8
	for _, word := range wordList {
		if len(word) < minLen {
			minLen = len(word)
		}
		if len(word) > maxLen {
			maxLen = len(word)
		}
	}
	t.Logf("maxLen: %v, minLen: %v", maxLen, minLen)
}

func TestSplit(t *testing.T) {
	str := "1 2 3 4 5"
	strs := strings.Split(str, " ")
	for _, s := range strs {
		t.Logf("after: %v", s)
	}
	t.Logf("")
	strs = strings.Fields(strings.TrimSpace(str))
	for _, s := range strs {
		t.Logf("after: %v", s)
	}

}
