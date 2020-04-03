package keystore

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"massnet.org/mass-wallet/config"
	"massnet.org/mass-wallet/massutil"
	mwdb "massnet.org/mass-wallet/masswallet/db"
)

func Test_newWitnessScriptAddressForBtcec(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown()
	t.Log("/*keystoreMananger*/")
	// new keystore manager
	km := &KeystoreManager{}
	var accountID1 string
	var pks []*btcec.PublicKey
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		bucket, err := mwdb.GetOrCreateTopLevelBucket(tx, keystoreBucket)
		if err != nil {
			return fmt.Errorf("failed to get bucket, %v", err)
		}
		km, err = NewKeystoreManager(bucket, pubPassphrase, &config.ChainParams)
		if err != nil {
			return fmt.Errorf("failed to new keystore manager, %v", err)
		}

		//new keystore
		var mnemonic1 string
		accountID1, mnemonic1, err = km.NewKeystore(tx, 0, privPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}
		t.Logf("accountID: %v, mnemonic: %v", accountID1, mnemonic1)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to new keystore manager or new keystore, %v", err)
	}

	err = km.UseKeystoreForWallet(accountID1)
	if err != nil {
		t.Fatalf("failed to use keystore, %v", err)
	}

	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		externalAddresses, err := km.NextAddresses(tx, alwaysTrueCheck, false, 3, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return fmt.Errorf("failed to new external address, %v", err)
		}

		showAddrManagerDetails(t, km.managedKeystores[accountID1])

		for _, addr := range externalAddresses {
			pks = append(pks, addr.pubKey)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	redeemScript, witAddress, err := newWitnessScriptAddressForBtcec(pks, 1, massutil.AddressClassWitnessV0, &config.ChainParams)
	if err != nil {
		t.Fatalf("failed to new witness script address, %v", err)
	}
	t.Logf("redeem script: %v", hex.EncodeToString(redeemScript))
	t.Logf("witness address: %v", witAddress)

	return
}
