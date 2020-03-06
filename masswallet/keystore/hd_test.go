package keystore

import (
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/masswallet/keystore/hdkeychain"
	"testing"
)

var (
	bitSize = 128
)

func TestHD(t *testing.T) {
	_, _, seed, err := generateSeed(bitSize, privPassphrase)
	if err != nil {
		t.Fatalf("failed to new seed, %v", err)
	}

	rootKey, err := hdkeychain.NewMaster(seed, &config.ChainParams)
	if err != nil {
		t.Fatalf("failed to derive master extended key: %v", err)
	}

	coinTypeKeyPriv, err := deriveCoinTypeKey(rootKey, KeyScope{Purpose: 44, Coin: hdkeychain.HardenedKeyStart})
	if err != ErrInvalidCoinType {
		t.Fatalf("failed to catch err, %v", err)
	}

	_, err = deriveAccountKey(coinTypeKeyPriv, hdkeychain.HardenedKeyStart-1)
	if err != ErrInvalidAccountNumber {
		t.Fatalf("failed to catch err, %v", err)
	}
}
