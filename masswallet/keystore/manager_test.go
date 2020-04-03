package keystore

import (
	//"bufio"
	"crypto/sha256"
	//"crypto/sha512"
	"encoding/hex"
	"fmt"

	//"io"
	//"os"
	"testing"
	//"time"

	"massnet.org/mass-wallet/config"
	//"massnet.org/mass-wallet/errors"
	"massnet.org/mass-wallet/massutil"
	mwdb "massnet.org/mass-wallet/masswallet/db"

	//"massnet.org/mass-wallet/masswallet/keystore/snacl"
	"crypto/sha512"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"massnet.org/mass-wallet/masswallet/keystore/zero"
)

const (
	testDbRoot      = "testDbs"
	keystoreBucket  = "k"
	utxoBucket      = "u"
	txBucket        = "t"
	syncBucket      = "s"
	addressGapLimit = 20
)

var (
	pubPassphrase   = []byte("@DJr@fL4H0O#$%0^n@V1")
	privPassphrase  = []byte("@#XXd7O9xyDIWIbXX$lj")
	pubPassphrase2  = []byte("@0NV4P@VSJBWbunw#%ZI")
	privPassphrase2 = []byte("@#$#08%68^f&5#4@%$&Y")
	invalidPass     = []byte("1234567890qwertyuiop!@#$%^&*()asdfghjklzx")

	// fastScrypt are parameters used throughout the tests to speed up the
	// scrypt operations.
	fastScrypt = &ScryptOptions{
		N: 65536,
		R: 8,
		P: 1,
	}

	alwaysTrueCheck = func(bytes []byte) (bool, error) { return true, nil }

	alwaysFalseCheck = func(bytes []byte) (bool, error) { return false, nil }
)

func TestGenerateSeed(t *testing.T) {
	var rightData = []struct {
		bitSize int
		pass    []byte
	}{
		{bitSize: 128, pass: privPassphrase},
		{bitSize: 160, pass: privPassphrase},
		{bitSize: 192, pass: privPassphrase},
		{bitSize: 224, pass: privPassphrase},
		{bitSize: 256, pass: privPassphrase},
	}
	for _, data := range rightData {
		entropy, mnemonic, hdSeed, err := generateSeed(data.bitSize, data.pass)
		if err != nil {
			t.Fatalf("failed to generate seed, error: %v", err)
		}
		t.Logf("entropy: %v", entropy)
		t.Logf("mnemonic: %v", mnemonic)
		t.Logf("hdSeed: %v", hdSeed)
	}
	var wrongData = []struct {
		bitSize int
		pass    []byte
		err     error
	}{
		{bitSize: 127, pass: privPassphrase, err: ErrEntropyLengthInvalid},
		{bitSize: 129, pass: privPassphrase, err: ErrEntropyLengthInvalid},
		{bitSize: 257, pass: privPassphrase, err: ErrEntropyLengthInvalid},
	}
	for _, data := range wrongData {
		_, _, _, err := generateSeed(data.bitSize, data.pass)
		if err != data.err {
			t.Fatalf("failed to catch error, expected: %v, actual: %v", data.err, err)
		}
	}

}

// time cost
//
// |   262144   |   65536    |   16384    |
// | ---------- | ---------- | ---------- |
// | 3.4308911s | 875.4617ms | 237.8524ms |
func TestKeystoreManager_NewKeystore(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown()
	t.Log("/*keystoreManager*/")
	// new keystore manager
	km := &KeystoreManager{}
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		bucket, err := mwdb.GetOrCreateTopLevelBucket(tx, keystoreBucket)
		if err != nil {
			return fmt.Errorf("failed to get bucket, %v", err)
		}
		km, err = NewKeystoreManager(bucket, pubPassphrase, &config.ChainParams)
		if err != nil {
			return fmt.Errorf("failed to new keystore manager, %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	testConfig := []*ScryptOptions{
		{
			N: 262144,
			R: 8,
			P: 1,
		},
		{
			N: 65536,
			R: 8,
			P: 1,
		},
		{
			N: 16384,
			R: 8,
			P: 1,
		},
	}
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		for _, scryptConfig := range testConfig {
			start := time.Now()
			_, _, err := km.NewKeystore(tx, 128, privPassphrase, "test", &config.ChainParams, scryptConfig, addressGapLimit)
			if err != nil {
				return err
			}
			t.Logf("new keystore time: %v", time.Since(start))
		}
		return nil

	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestKeystoreManager_NewKeystore_NextAddress(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown()
	t.Log("/*keystoreManager*/")
	// new keystore manager
	km := &KeystoreManager{}
	scripts := make(map[[20]byte]struct{})
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		bucket, err := mwdb.GetOrCreateTopLevelBucket(tx, keystoreBucket)
		if err != nil {
			return fmt.Errorf("failed to get bucket, %v", err)
		}
		km, err = NewKeystoreManager(bucket, pubPassphrase, &config.ChainParams)
		if err != nil {
			return fmt.Errorf("failed to new keystore manager, %v", err)
		}

		// invalid privpass
		_, _, err = km.NewKeystore(tx, defaultBitSize, invalidPass, "first", &config.ChainParams, nil, addressGapLimit)
		if err != ErrIllegalPassphrase {
			return fmt.Errorf("failed to catch error, %v", err)
		}

		// privpass same as pubpass
		_, _, err = km.NewKeystore(tx, defaultBitSize, pubPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != ErrIllegalNewPrivPass {
			return fmt.Errorf("failed to catch error, %v", err)
		}

		//new keystore
		accountID1, mnemonic1, err := km.NewKeystore(tx, 0, privPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}
		t.Logf("accountID: %v, mnemonic: %v", accountID1, mnemonic1)
		_, _, err = km.NewKeystore(tx, defaultBitSize, privPassphrase, "second", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}

		_, err = km.NextAddresses(tx, alwaysFalseCheck, false, 21, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != ErrCurrentKeystoreNotFound {
			return fmt.Errorf("failed to catch err, %v", err)
		}

		err = km.UseKeystoreForWallet(accountID1)
		if err != nil {
			return fmt.Errorf("failed to use keystore, %v", err)
		}

		_, err = km.NextAddresses(tx, alwaysFalseCheck, false, 21, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != ErrGapLimit {
			return fmt.Errorf("failed to catch err, %v", err)
		}

		_, err = km.NextAddresses(tx, alwaysFalseCheck, false, MaxAddressesPerAccount+1, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != ErrExceedAllowedNumberPerAccount {
			return fmt.Errorf("failed to catch err, %v", err)
		}

		externalAddresses, err := km.NextAddresses(tx, alwaysTrueCheck, false, 20, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return fmt.Errorf("failed to new external address, %v", err)
		}

		showAddrManagerDetails(t, km.managedKeystores[accountID1])

		used := make(map[uint32]struct{})
		used[0] = struct{}{}
		used[1] = struct{}{}
		for _, addr := range externalAddresses {
			if _, ok := used[addr.derivationPath.Index]; ok {
				var v [20]byte
				copy(v[:], addr.scriptHash[:])
				scripts[v] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var checkFunc = func(scriptHash []byte) (bool, error) {
		var v [20]byte
		copy(v[:], scriptHash[:])
		if _, ok := scripts[v]; ok {
			return true, nil
		} else {
			return false, nil
		}
	}

	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		_, err := km.NextAddresses(tx, checkFunc, false, 2, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return err
		}
		showAddrManagerDetails(t, km.managedKeystores[km.currentKeystore.accountName])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		_, err := km.NextAddresses(tx, checkFunc, false, 1, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return err
		}
		showAddrManagerDetails(t, km.managedKeystores[km.currentKeystore.accountName])
		return nil
	})
	if err != ErrGapLimit {
		t.Fatal(err)
	}
	t.Logf("pass test")

	return
}

func TestKeystoreManager_ExportKeystore_ImportKeystore(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown()
	t.Log("/*keystoreManager*/")

	// new keystore manager (not poc)
	km := &KeystoreManager{}
	scripts := make(map[[20]byte]struct{})
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
		accountID1, mnemonic1, err := km.NewKeystore(tx, defaultBitSize, privPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}
		t.Logf("accountID: %v, mnemonic: %v", accountID1, mnemonic1)
		_, _, err = km.NewKeystore(tx, defaultBitSize, privPassphrase, "second", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}

		err = km.UseKeystoreForWallet(accountID1)
		if err != nil {
			return fmt.Errorf("failed to use keystore, %v", err)
		}

		externalAddresses, err := km.NextAddresses(tx, alwaysTrueCheck, false, 2, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return fmt.Errorf("failed to new external address, %v", err)
		}

		used := make(map[uint32]struct{})
		used[0] = struct{}{}
		used[1] = struct{}{}
		for _, addr := range externalAddresses {
			if _, ok := used[addr.derivationPath.Index]; ok {
				var v [20]byte
				copy(v[:], addr.scriptHash[:])
				scripts[v] = struct{}{}
			}
		}

		internalAddresses, err := km.NextAddresses(tx, alwaysTrueCheck, true, 1, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return fmt.Errorf("failed to new external address, %v", err)
		}
		for _, addr := range internalAddresses {
			if _, ok := used[addr.derivationPath.Index]; ok {
				var v [20]byte
				copy(v[:], addr.scriptHash[:])
				scripts[v] = struct{}{}
			}
		}
		showAddrManagerDetails(t, km.managedKeystores[accountID1])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var checkFunc = func(scriptHash []byte) (bool, error) {
		var v [20]byte
		copy(v[:], scriptHash[:])
		if _, ok := scripts[v]; ok {
			return true, nil
		} else {
			return false, nil
		}
	}

	err = mwdb.View(ldb, func(tx mwdb.ReadTransaction) error {
		_, err := km.ExportKeystore(tx, "wrong account", privPassphrase)
		return err
	})
	if err != ErrAccountNotFound {
		t.Fatalf("failed to catch error, %v", err)
	}

	// export keystore
	var keystoreJson []byte
	err = mwdb.View(ldb, func(tx mwdb.ReadTransaction) error {
		accountIDs := km.ListKeystoreNames()
		accountID := accountIDs[0]
		var err error
		keystoreJson, err = km.ExportKeystore(tx, accountID, privPassphrase)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("keystore: %v", string(keystoreJson))

	kStore, err := getKeystoreFromJson(keystoreJson)
	if err != nil {
		t.Fatalf("failed to get keystore, %v", err)
	}
	kStore.HDpath.ExternalChildNum = 10
	keystoreJson = kStore.Bytes()
	t.Logf("keystore: %v", string(keystoreJson))

	// new keystore manager
	ldb1, tearDown1, err := GetDb("Tst_Manager1")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown1()
	t.Log("/*keystoreManager 1*/")

	km1 := &KeystoreManager{}
	err = mwdb.Update(ldb1, func(tx mwdb.DBTransaction) error {
		bucket, err := mwdb.GetOrCreateTopLevelBucket(tx, keystoreBucket)
		if err != nil {
			return fmt.Errorf("failed to get bucket, %v", err)
		}
		km1, err = NewKeystoreManager(bucket, pubPassphrase2, &config.ChainParams)
		if err != nil {
			return fmt.Errorf("failed to new keystore manager, %v", err)
		}

		// import with wrong pass
		_, err = km1.ImportKeystore(tx, checkFunc, keystoreJson, privPassphrase2, addressGapLimit)
		if err != ErrInvalidPassphrase {
			t.Fatalf("failed to catch err, %v", err)
		}

		// import keystore
		start := time.Now()
		addrManager, err := km1.ImportKeystore(tx, checkFunc, keystoreJson, privPassphrase, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to import keystore, %v", err)
		}
		t.Logf("time: %v", time.Since(start))
		showAddrManagerDetails(t, addrManager)

		accountIDs := km1.ListKeystoreNames()
		err = km1.UseKeystoreForWallet(accountIDs[0])
		if err != nil {
			return err
		}

		showAddrManagerDetails(t, addrManager)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = mwdb.View(ldb1, func(tx mwdb.ReadTransaction) error {
		accountIDs := km1.ListKeystoreNames()
		mnemonic, err := km1.GetMnemonic(tx, accountIDs[0], privPassphrase)
		if err != nil {
			return fmt.Errorf("failed to get mnemonic, %v", err)
		}
		t.Logf("mnemonic: %v", mnemonic)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return
}

func TestKeystoreManager_ImportKeystoreWithMnemonic(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown()
	t.Log("/*keystoreManager*/")

	// new keystore manager
	km := &KeystoreManager{}
	var mnemonic, remark string
	var internalIndex, externalIndex uint32
	scripts := make(map[[20]byte]struct{})
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
		accountID1, mnemonic1, err := km.NewKeystore(tx, defaultBitSize, privPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}
		t.Logf("accountID: %v, mnemonic: %v", accountID1, mnemonic1)
		mnemonic = mnemonic1
		remark = "first"

		err = km.UseKeystoreForWallet(accountID1)
		if err != nil {
			return fmt.Errorf("failed to use keystore, %v", err)
		}

		externalAddresses, err := km.NextAddresses(tx, alwaysTrueCheck, false, 10, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return fmt.Errorf("failed to new external address, %v", err)
		}
		showAddrManagerDetails(t, km.managedKeystores[accountID1])
		used := make(map[uint32]struct{})
		used[0] = struct{}{}
		used[1] = struct{}{}
		used[2] = struct{}{}
		used[3] = struct{}{}
		used[4] = struct{}{}
		for _, addr := range externalAddresses {
			if _, ok := used[addr.derivationPath.Index]; ok {
				var v [20]byte
				copy(v[:], addr.scriptHash[:])
				scripts[v] = struct{}{}
			}
		}
		internalIndex, externalIndex = km.managedKeystores[accountID1].branchInfo.nextInternalIndex, km.managedKeystores[accountID1].branchInfo.nextExternalIndex
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var checkFunc = func(scriptHash []byte) (bool, error) {
		var v [20]byte
		copy(v[:], scriptHash[:])
		if _, ok := scripts[v]; ok {
			return true, nil
		} else {
			return false, nil
		}
	}

	// new keystore manager
	ldb1, tearDown1, err := GetDb("Tst_Manager1")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown1()
	t.Log("/*keystoreManager 1*/")

	km1 := &KeystoreManager{}
	err = mwdb.Update(ldb1, func(tx mwdb.DBTransaction) error {
		bucket, err := mwdb.GetOrCreateTopLevelBucket(tx, keystoreBucket)
		if err != nil {
			return fmt.Errorf("failed to get bucket, %v", err)
		}
		km1, err = NewKeystoreManager(bucket, pubPassphrase2, &config.ChainParams)
		if err != nil {
			return fmt.Errorf("failed to new keystore manager, %v", err)
		}
		// import keystore
		addrManager, err := km1.ImportKeystoreWithMnemonic(tx, checkFunc, mnemonic, remark, privPassphrase, externalIndex, internalIndex, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to import keystore, %v", err)
		}
		showAddrManagerDetails(t, addrManager)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return
}

func TestKeystoreManager_ExportKeystore(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown()
	t.Log("/*keystoreManager*/")
	// new keystore manager
	kmw := &KeystoreManager{}
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		bucket, err := mwdb.GetOrCreateTopLevelBucket(tx, keystoreBucket)
		if err != nil {
			t.Fatalf("failed to get bucket, %v", err)
		}
		kmw, err = NewKeystoreManager(bucket, pubPassphrase, &config.ChainParams)
		if err != nil {
			t.Fatalf("failed to new keystore manager, %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to new keystore manager, %v", err)
	}
	var accountID string
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		var err error
		accountID, _, err = kmw.NewKeystore(tx, defaultBitSize, []byte("123456"), "first", &config.ChainParams, nil, addressGapLimit)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to new keystore, %v", err)
	}

	addrManager := kmw.managedKeystores[accountID]
	err = kmw.UseKeystoreForWallet(accountID)
	if err != nil {
		t.Fatalf("failed to use keystore, %v", err)
	}

	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		_, err := kmw.NextAddresses(tx, alwaysTrueCheck, false, 2, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to new address, %v", err)
	}

	var keystore []byte
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		var err error
		keystore, err = kmw.ExportKeystore(tx, accountID, []byte("123456"))
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to export keystore, %v", err)
	}
	t.Logf("keystore: %v", string(keystore))

	// wrong pass
	err = addrManager.checkPassword([]byte("1234567890"))
	if err != ErrInvalidPassphrase {
		t.Fatalf("failed to check password, %v", err)
	}

	// right pass
	err = addrManager.checkPassword([]byte("123456"))
	if err != nil {
		t.Fatalf("failed to check password, %v", err)
	}

	// update cache, mark unlocked
	saltPassphrase := append(addrManager.privPassphraseSalt[:], []byte("123456")...)
	addrManager.hashedPrivPassphrase = sha512.Sum512(saltPassphrase)
	zero.Bytes(saltPassphrase)
	addrManager.unlocked = true

	// check again, unlocked
	err = addrManager.checkPassword([]byte("123456"))
	if err != nil {
		t.Fatalf("failed to check password, %v", err)
	}

	// check again, wrong pass
	err = addrManager.checkPassword([]byte("1234567890"))
	if err != ErrInvalidPassphrase {
		t.Fatalf("failed to check password, %v", err)
	}

	err = addrManager.updatePrivKeys([]byte("123456"))
	if err != nil {
		t.Fatalf("failed to update privKey, %v", err)
	}

	showAddrManagerDetails(t, addrManager)

	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		_, err := kmw.NextAddresses(tx, alwaysTrueCheck, true, 2, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to new address, %v", err)
	}

	addrManager.clearPrivKeys()

	return
}

func TestKeystoreManager_DeleteKeystore(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown()
	t.Log("/*keystoreManager*/")

	// new keystore manager
	var accountID string
	km := &KeystoreManager{}
	scripts := make(map[[20]byte]struct{})
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
		accountID1, mnemonic1, err := km.NewKeystore(tx, defaultBitSize, privPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}
		accountID = accountID1
		t.Logf("accountID: %v, mnemonic: %v", accountID1, mnemonic1)
		_, _, err = km.NewKeystore(tx, defaultBitSize, privPassphrase, "second", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}

		err = km.UseKeystoreForWallet(accountID1)
		if err != nil {
			return fmt.Errorf("failed to use keystore, %v", err)
		}

		externalAddresses, err := km.NextAddresses(tx, alwaysTrueCheck, false, 2, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return fmt.Errorf("failed to new external address, %v", err)
		}

		used := make(map[uint32]struct{})
		used[0] = struct{}{}
		used[1] = struct{}{}
		for _, addr := range externalAddresses {
			if _, ok := used[addr.derivationPath.Index]; ok {
				var v [20]byte
				copy(v[:], addr.scriptHash[:])
				scripts[v] = struct{}{}
			}
		}

		internalAddresses, err := km.NextAddresses(tx, alwaysTrueCheck, true, 1, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return fmt.Errorf("failed to new external address, %v", err)
		}
		for _, addr := range internalAddresses {
			if _, ok := used[addr.derivationPath.Index]; ok {
				var v [20]byte
				copy(v[:], addr.scriptHash[:])
				scripts[v] = struct{}{}
			}
		}
		showAddrManagerDetails(t, km.managedKeystores[accountID1])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var checkFunc = func(scriptHash []byte) (bool, error) {
		var v [20]byte
		copy(v[:], scriptHash[:])
		if _, ok := scripts[v]; ok {
			return true, nil
		} else {
			return false, nil
		}
	}

	// export keystore
	var keystoreJson []byte
	err = mwdb.View(ldb, func(tx mwdb.ReadTransaction) error {
		var err error
		keystoreJson, err = km.ExportKeystore(tx, accountID, privPassphrase)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("keystore: %v", string(keystoreJson))

	kStore, err := getKeystoreFromJson(keystoreJson)
	if err != nil {
		t.Fatalf("failed to get keystore, %v", err)
	}
	kStore.HDpath.ExternalChildNum = 10
	kStore.HDpath.InternalChildNum = 5
	keystoreJson = kStore.Bytes()
	t.Logf("keystore: %v", string(keystoreJson))

	// wrong accountID
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		_, err := km.DeleteKeystore(tx, "wrong accountID", privPassphrase2)
		if err != nil {
			return err
		}
		return nil
	})
	if err != ErrAccountNotFound {
		t.Fatalf("failed to catch error, %v", err)
	}

	// wrong privpass
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		_, err := km.DeleteKeystore(tx, accountID, privPassphrase2)
		if err != nil {
			return err
		}
		return nil
	})
	if err != ErrInvalidPassphrase {
		t.Fatalf("failed to catch error, %v", err)
	}

	// delete keystore
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		_, err := km.DeleteKeystore(tx, accountID, privPassphrase)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to delete keystore, %v", err)
	}

	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		addrManager, err := km.ImportKeystore(tx, checkFunc, keystoreJson, privPassphrase, addressGapLimit)
		if err != nil {
			return err
		}
		showAddrManagerDetails(t, addrManager)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to import keystore, %v", err)
	}

}

func TestKeystoreManager_SignHash(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown()
	t.Log("/*keystoreManager*/")
	// new keystore manager
	km := &KeystoreManager{}
	var pk, wrongPk *btcec.PublicKey
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
		accountID1, mnemonic1, err := km.NewKeystore(tx, defaultBitSize, privPassphrase, "first", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}
		t.Logf("accountID: %v, mnemonic: %v", accountID1, mnemonic1)
		err = km.UseKeystoreForWallet(accountID1)
		if err != nil {
			return fmt.Errorf("failed to use keystore, %v", err)
		}
		addrs, err := km.NextAddresses(tx, alwaysTrueCheck, false, 2, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return fmt.Errorf("failed to new address, %v", err)
		}
		pk = addrs[0].pubKey
		wrongPk = addrs[1].pubKey
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	data := "test sign hash"
	hash := sha256.New()
	hash.Write([]byte(data))
	dataHash := hash.Sum(nil)

	defer km.ClearPrivKey()
	// wrong pass
	sign, err := km.SignHash(pk, dataHash, privPassphrase2)
	if err != ErrInvalidPassphrase {
		t.Fatalf("failed to catch error, %v", err)
	}

	// invalid hash
	wrongSizeHash := append(dataHash, byte(1))
	sign, err = km.SignHash(pk, wrongSizeHash, privPassphrase)
	if err != ErrInvalidDataHash {
		t.Fatalf("failed to catch error, %v", err)
	}

	// public key cannot be found
	pkbytes := []byte{2, 148, 88, 79, 120, 77, 199, 37, 242, 26, 48, 16, 187, 48, 235, 210, 54, 201, 161, 144, 139, 68,
		131, 119, 67, 15, 135, 67, 87, 127, 207, 36, 31}
	newpk, err := btcec.ParsePubKey(pkbytes, btcec.S256())
	if err != nil {
		t.Errorf("failed to unmarshal publik key, %v", err)
	} else {
		sign, err = km.SignHash(newpk, dataHash, privPassphrase)
		if err != ErrAccountNotFound {
			t.Fatalf("failed to sign hash, %v", err)
		}
	}

	// right pass
	start := time.Now()
	sign, err = km.SignHash(pk, dataHash, privPassphrase)
	if err != nil {
		t.Fatalf("failed to sign hash, %v", err)
	}
	t.Logf("sign time cost: %v", time.Since(start))
	success := sign.Verify(dataHash, pk)
	if !success {
		t.Fatal("failed to verify signature")
	}
	t.Log("successfully verify signature")

	// wrong dataHash
	wrongData := "wrong test data"
	hash2 := sha256.New()
	hash2.Write([]byte(wrongData))
	wrongDataHash := hash2.Sum(nil)
	success = sign.Verify(wrongDataHash, pk)
	if success {
		t.Fatalf("failed to find out that the signature does not match the data")
	}
	t.Log("successfully find out that the signature does not match the data")

	// wrong pk
	success = sign.Verify(dataHash, wrongPk)
	if success {
		t.Fatalf("failed to find out that the signature does not match the public key")
	}
	t.Log("successfully find out that the signature does not match the public key")
	return
}

func TestKeystoreManager_ChangePubPassphrase(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown()
	t.Log("/*keystoreManager*/")
	// new keystore manager
	km := &KeystoreManager{}
	scripts := make(map[[20]byte]struct{})
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		bucket, err := mwdb.GetOrCreateTopLevelBucket(tx, keystoreBucket)
		if err != nil {
			return fmt.Errorf("failed to get bucket, %v", err)
		}
		km, err = NewKeystoreManager(bucket, pubPassphrase, &config.ChainParams)
		if err != nil {
			return fmt.Errorf("failed to new keystore manager, %v", err)
		}

		// invalid privpass
		_, _, err = km.NewKeystore(tx, defaultBitSize, invalidPass, "first", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != ErrIllegalPassphrase {
			return fmt.Errorf("failed to catch error, %v", err)
		}

		// privpass same as pubpass
		_, _, err = km.NewKeystore(tx, defaultBitSize, pubPassphrase, "first", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != ErrIllegalNewPrivPass {
			return fmt.Errorf("failed to catch error, %v", err)
		}

		//new keystore
		accountID1, mnemonic1, err := km.NewKeystore(tx, 0, privPassphrase, "first", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}
		t.Logf("accountID: %v, mnemonic: %v", accountID1, mnemonic1)
		_, _, err = km.NewKeystore(tx, defaultBitSize, privPassphrase, "second", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}

		err = km.UseKeystoreForWallet(accountID1)
		if err != nil {
			return fmt.Errorf("failed to use keystore, %v", err)
		}

		_, err = km.NextAddresses(tx, alwaysFalseCheck, false, 21, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != ErrGapLimit {
			return fmt.Errorf("failed to catch err, %v", err)
		}

		externalAddresses, err := km.NextAddresses(tx, alwaysTrueCheck, false, 20, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return fmt.Errorf("failed to new external address, %v", err)
		}

		showAddrManagerDetails(t, km.managedKeystores[accountID1])

		used := make(map[uint32]struct{})
		used[0] = struct{}{}
		used[1] = struct{}{}
		for _, addr := range externalAddresses {
			if _, ok := used[addr.derivationPath.Index]; ok {
				var v [20]byte
				copy(v[:], addr.scriptHash[:])
				scripts[v] = struct{}{}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// invalid pass
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		return km.ChangePubPassphrase(tx, pubPassphrase, invalidPass, fastScrypt)
	})
	if err != ErrIllegalPassphrase {
		t.Fatalf("failed to catch error, %v", err)
	}

	// same as old one
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		return km.ChangePubPassphrase(tx, pubPassphrase, pubPassphrase, fastScrypt)
	})
	if err != ErrSamePubpass {
		t.Fatalf("failed to catch error, %v", err)
	}

	// same as privpass
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		return km.ChangePubPassphrase(tx, pubPassphrase, privPassphrase, fastScrypt)
	})
	if err != ErrIllegalNewPubPass {
		t.Fatalf("failed to catch error, %v", err)
	}

	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		return km.ChangePubPassphrase(tx, pubPassphrase, pubPassphrase2, nil)
	})
	if err != nil {
		t.Fatalf("failed to change public pass, %v", err)
	}
}

func TestKeystoreManager_ChangeRemark(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown()
	t.Log("/*keystoreManager*/")
	// new keystore manager
	km := &KeystoreManager{}
	var accountID string
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
		accountID1, mnemonic1, err := km.NewKeystore(tx, defaultBitSize, privPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}
		accountID = accountID1
		t.Logf("accountID: %v, mnemonic: %v", accountID1, mnemonic1)
		_, _, err = km.NewKeystore(tx, defaultBitSize, privPassphrase, "second", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}

		err = km.UseKeystoreForWallet(accountID1)
		if err != nil {
			return fmt.Errorf("failed to use keystore, %v", err)
		}

		_, err = km.NextAddresses(tx, alwaysTrueCheck, false, 2, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return fmt.Errorf("failed to new external address, %v", err)
		}

		showAddrManagerDetails(t, km.managedKeystores[accountID1])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		return km.ChangeRemark(tx, "errorAccountID", "new remark")
	})
	if err != ErrAccountNotFound {
		t.Fatalf("failed to catch error, %v", err)
	}

	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		return km.ChangeRemark(tx, accountID, "new remark")
	})
	if err != nil {
		t.Fatal(err)
	}

	showAddrManagerDetails(t, km.managedKeystores[accountID])

	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		return km.ChangeRemark(tx, accountID, "")
	})
	if err != nil {
		t.Fatal(err)
	}

	showAddrManagerDetails(t, km.managedKeystores[accountID])
}

func TestKeystoreManager_GetManangedAddressByStdAddress(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown()
	t.Log("/*keystoreManager*/")
	// new keystore manager
	km := &KeystoreManager{}
	var stdAddr string
	var accountID1 string
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		bucket, err := mwdb.GetOrCreateTopLevelBucket(tx, keystoreBucket)
		if err != nil {
			return fmt.Errorf("failed to get bucket, %v", err)
		}
		km, err = NewKeystoreManager(bucket, pubPassphrase, &config.ChainParams)
		if err != nil {
			return fmt.Errorf("failed to new keystore manager, %v", err)
		}

		// invalid privpass
		_, _, err = km.NewKeystore(tx, defaultBitSize, invalidPass, "first", &config.ChainParams, nil, addressGapLimit)
		if err != ErrIllegalPassphrase {
			return fmt.Errorf("failed to catch error, %v", err)
		}

		// privpass same as pubpass
		_, _, err = km.NewKeystore(tx, defaultBitSize, pubPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != ErrIllegalNewPrivPass {
			return fmt.Errorf("failed to catch error, %v", err)
		}

		//new keystore
		var mnemonic1 string
		accountID1, mnemonic1, err = km.NewKeystore(tx, 0, privPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}
		t.Logf("accountID: %v, mnemonic: %v", accountID1, mnemonic1)
		_, _, err = km.NewKeystore(tx, defaultBitSize, privPassphrase, "second", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}

		err = km.UseKeystoreForWallet(accountID1)
		if err != nil {
			return fmt.Errorf("failed to use keystore, %v", err)
		}

		_, err = km.NextAddresses(tx, alwaysFalseCheck, false, 21, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != ErrGapLimit {
			return fmt.Errorf("failed to catch err, %v", err)
		}

		externalAddresses, err := km.NextAddresses(tx, alwaysTrueCheck, false, 20, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return fmt.Errorf("failed to new external address, %v", err)
		}

		showAddrManagerDetails(t, km.managedKeystores[accountID1])

		for _, addr := range externalAddresses {
			stdAddr = addr.address
			break
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("address: %v", stdAddr)
	managedAddr, err := km.GetManagedAddressByStdAddress(stdAddr)
	if err != nil {
		t.Fatalf("failed to get managed address, %v", err)
	}
	managedAddressDetails(t, managedAddr)

	addrManager, err := km.GetAddrManagerByAccountID(accountID1)
	if err != nil {
		t.Fatalf("failed to get addrManager, %v", err)
	}

	mAddr, err := addrManager.Address(stdAddr)
	if err != nil {
		t.Fatalf("failed to get managed address, %v", err)
	}
	managedAddressDetails(t, mAddr)

	stdAddr = "ms1qevgq887qta2svxqu4uleczqa0a5xe4frtxvpk5"
	t.Logf("address:, %v", stdAddr)
	_, err = km.GetManagedAddressByStdAddress(stdAddr)
	if err != ErrAddressNotFound {
		t.Fatalf("failed to catch error, %v", err)
	}

	_, err = addrManager.Address(stdAddr)
	if err != ErrAddressNotFound {
		t.Fatalf("failed to catch error, %v", err)
	}
}

func TestKeystoreManager_GetManangedAddressByScriptHash(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown()
	t.Log("/*keystoreManager*/")
	// new keystore manager
	km := &KeystoreManager{}
	var script []byte
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		bucket, err := mwdb.GetOrCreateTopLevelBucket(tx, keystoreBucket)
		if err != nil {
			return fmt.Errorf("failed to get bucket, %v", err)
		}
		km, err = NewKeystoreManager(bucket, pubPassphrase, &config.ChainParams)
		if err != nil {
			return fmt.Errorf("failed to new keystore manager, %v", err)
		}

		// invalid privpass
		_, _, err = km.NewKeystore(tx, defaultBitSize, invalidPass, "first", &config.ChainParams, nil, addressGapLimit)
		if err != ErrIllegalPassphrase {
			return fmt.Errorf("failed to catch error, %v", err)
		}

		// privpass same as pubpass
		_, _, err = km.NewKeystore(tx, defaultBitSize, pubPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != ErrIllegalNewPrivPass {
			return fmt.Errorf("failed to catch error, %v", err)
		}

		//new keystore
		accountID1, mnemonic1, err := km.NewKeystore(tx, 0, privPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}
		t.Logf("accountID: %v, mnemonic: %v", accountID1, mnemonic1)
		_, _, err = km.NewKeystore(tx, defaultBitSize, privPassphrase, "second", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}

		err = km.UseKeystoreForWallet(accountID1)
		if err != nil {
			return fmt.Errorf("failed to use keystore, %v", err)
		}

		_, err = km.NextAddresses(tx, alwaysFalseCheck, false, 21, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != ErrGapLimit {
			return fmt.Errorf("failed to catch err, %v", err)
		}

		externalAddresses, err := km.NextAddresses(tx, alwaysTrueCheck, false, 20, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return fmt.Errorf("failed to new external address, %v", err)
		}

		showAddrManagerDetails(t, km.managedKeystores[accountID1])

		for _, addr := range externalAddresses {
			script = addr.scriptHash
			break
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("scriptHash: %v", script)
	managedAddr, err := km.GetManagedAddressByScriptHash(script)
	if err != nil {
		t.Fatalf("failed to get managed address, %v", err)
	}
	managedAddressDetails(t, managedAddr)

	_, err = km.GetManagedAddressByScriptHash(script[:19])
	if err == nil {
		t.Fatalf("failed to catch error")
	}

	script = []byte{70, 176, 203, 233, 32, 106, 31, 211, 97, 255, 133, 80, 104, 152, 181, 132, 64, 203, 82, 42, 79, 83, 91, 116, 101, 4, 117, 199, 165, 18, 178, 3}
	t.Logf("scriptHash:, %v", script)
	_, err = km.GetManagedAddressByScriptHash(script)
	if err != ErrScriptHashNotFound {
		t.Fatalf("failed to catch error, %v", err)
	}

}

func TestKeystoreManager_GetMnemonic(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)
	}
	defer tearDown()
	t.Log("/*keystoreManager*/")
	// new keystore manager
	km := &KeystoreManager{}

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
		accountID1, mnemonic1, err := km.NewKeystore(tx, defaultBitSize, privPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}
		t.Logf("accountID: %v, mnemonic: %v", accountID1, mnemonic1)
		_, _, err = km.NewKeystore(tx, defaultBitSize, privPassphrase, "second", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}

		err = km.UseKeystoreForWallet(accountID1)
		if err != nil {
			return fmt.Errorf("failed to use keystore, %v", err)
		}
		externalAddresses, err := km.NextAddresses(tx, alwaysTrueCheck, false, 3, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return fmt.Errorf("failed to new external address, %v", err)
		}
		for _, addr := range externalAddresses {
			managedAddressDetails(t, addr)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = mwdb.View(ldb, func(tx mwdb.ReadTransaction) error {
		accountIDs := km.ListKeystoreNames()
		accountID := accountIDs[0]
		mnemonic, err := km.GetMnemonic(tx, accountID, privPassphrase)
		if err != nil {
			return err
		}
		t.Logf("get mnemonic, accountID: %v, mnemonic: %v", accountID, mnemonic)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// non-existent accountID
	err = mwdb.View(ldb, func(tx mwdb.ReadTransaction) error {
		accountID := "wrong accountID"
		_, err := km.GetMnemonic(tx, accountID, privPassphrase)
		if err != nil {
			return err
		}
		return nil
	})
	if err != ErrAccountNotFound {
		t.Fatalf("failed to catch error, %v", err)
	}

	//
	err = mwdb.View(ldb, func(tx mwdb.ReadTransaction) error {
		accountIDs := km.ListKeystoreNames()
		accountID := accountIDs[0]
		_, err := km.GetMnemonic(tx, accountID, privPassphrase2)
		if err != nil {
			return err
		}
		return nil
	})
	if err != ErrInvalidPassphrase {
		t.Fatalf("failed to catch error, %v", err)
	}
}

func TestKeystoreManager_UseKeystoreForWallet(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)
		return
	}
	t.Log("get db")
	defer tearDown()

	km := &KeystoreManager{}
	var accountID1, accountID2 string
	err = mwdb.Update(ldb, func(dbTransaction mwdb.DBTransaction) error {
		var err error
		bucket, err := mwdb.GetOrCreateTopLevelBucket(dbTransaction, keystoreBucket)
		if err != nil {
			t.Fatalf("failed to get bucket, %v", err)
		}

		// new keystore manager
		km, err = NewKeystoreManager(bucket, pubPassphrase, &config.ChainParams)
		if err != nil {
			t.Fatalf("failed to new keystore manager, %v", err)
		}

		// new keystore
		var mnemonic1 string
		accountID1, mnemonic1, err = km.NewKeystore(dbTransaction, defaultBitSize, privPassphrase, "first account", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			t.Fatalf("failed to new keystore, %v", err)
		}
		t.Logf("accountID: %v, mnemonic: %v", accountID1, mnemonic1)

		var mnemonic2 string
		accountID2, mnemonic2, err = km.NewKeystore(dbTransaction, defaultBitSize, privPassphrase2, "second account", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			t.Fatalf("failed to new keystore, %v", err)
		}
		t.Logf("accountID: %v, mnemonic: %v", accountID2, mnemonic2)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to new keystore, %v", err)
	}

	//
	cur := km.CurrentKeystore()
	if cur != nil {
		t.Fatalf("wtf, %v", cur.keystoreName)
	}

	// wrong accountID
	err = km.UseKeystoreForWallet("wrongName")
	if err == ErrAccountNotFound {
		t.Logf("pass use keystore with wrong account name test")
	} else {
		t.Fatalf("not pass use keystore with wrong account name test, %v", err)
	}

	// use the first keystore
	err = km.UseKeystoreForWallet(accountID1)
	if err != nil {
		t.Fatalf("failed to use keystore, %v", err)
	}

	cur = km.CurrentKeystore()
	if cur.keystoreName != accountID1 {
		t.Fatalf("wtf, cur: %v, expected: %v", cur.keystoreName, accountID1)
	}

	// use it again
	err = km.UseKeystoreForWallet(accountID1)
	if err != nil {
		t.Fatalf("failed to use keystore, %v", err)
	}

	err = mwdb.Update(ldb, func(dbTransaction mwdb.DBTransaction) error {
		// new two external addresses for first account
		externalAddresses, err := km.NextAddresses(dbTransaction, alwaysTrueCheck, false, 2, addressGapLimit, massutil.AddressClassWitnessStaking)
		if err != nil {
			t.Fatalf("fialed to new 2 external addresses for first account, %v", err)
		}
		for i, addr := range externalAddresses {
			t.Logf("new external address %v for account %v:", i, accountID1)
			managedAddressDetails(t, addr)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to new addresses, %v", err)
	}

	// use the second keystore
	err = km.UseKeystoreForWallet(accountID2)
	if err != nil {
		t.Fatalf("failed to use keystore, %v", err)
	}

	err = mwdb.Update(ldb, func(dbTransaction mwdb.DBTransaction) error {
		// new two external addresses for second account
		externalAddresses, err := km.NextAddresses(dbTransaction, alwaysTrueCheck, false, 2, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			t.Fatalf("fialed to new 2 external addresses for second account, %v", err)
		}
		for i, addr := range externalAddresses {
			t.Logf("new external address %v for account %v:", i, accountID2)
			managedAddressDetails(t, addr)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to new addresses, %v", err)
	}

	// use the first keystore
	err = km.UseKeystoreForWallet(accountID1)
	if err != nil {
		t.Fatalf("failed to use keystore, %v", err)
	}

	err = mwdb.Update(ldb, func(dbTransaction mwdb.DBTransaction) error {
		// new two internal addresses for first account
		externalAddresses, err := km.NextAddresses(dbTransaction, alwaysTrueCheck, true, 2, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			t.Fatalf("fialed to new 2 internal addresses for first account, %v", err)
		}
		for i, addr := range externalAddresses {
			t.Logf("new external address %v for account %v:", i, accountID1)
			managedAddressDetails(t, addr)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("test failed, %v", err)
	}

	// check branch index
	t.Log("first account next internal index: ", km.managedKeystores[accountID1].branchInfo.nextInternalIndex)
	t.Log("first account next external index: ", km.managedKeystores[accountID1].branchInfo.nextExternalIndex)
	t.Log("second account next internal index: ", km.managedKeystores[accountID2].branchInfo.nextInternalIndex)
	t.Log("second account next external index: ", km.managedKeystores[accountID2].branchInfo.nextExternalIndex)
	t.Logf("test pass")
}

func TestKeystoreManager_GetAddrManager(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown()
	t.Log("/*keystoreManager*/")
	// new keystore manager
	km := &KeystoreManager{}
	var stdAddr string
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		bucket, err := mwdb.GetOrCreateTopLevelBucket(tx, keystoreBucket)
		if err != nil {
			return fmt.Errorf("failed to get bucket, %v", err)
		}
		km, err = NewKeystoreManager(bucket, pubPassphrase, &config.ChainParams)
		if err != nil {
			return fmt.Errorf("failed to new keystore manager, %v", err)
		}

		// invalid privpass
		_, _, err = km.NewKeystore(tx, defaultBitSize, invalidPass, "first", &config.ChainParams, nil, addressGapLimit)
		if err != ErrIllegalPassphrase {
			return fmt.Errorf("failed to catch error, %v", err)
		}

		// privpass same as pubpass
		_, _, err = km.NewKeystore(tx, defaultBitSize, pubPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != ErrIllegalNewPrivPass {
			return fmt.Errorf("failed to catch error, %v", err)
		}

		//new keystore
		accountID1, mnemonic1, err := km.NewKeystore(tx, 0, privPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}
		t.Logf("accountID: %v, mnemonic: %v", accountID1, mnemonic1)
		_, _, err = km.NewKeystore(tx, defaultBitSize, privPassphrase, "second", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}

		err = km.UseKeystoreForWallet(accountID1)
		if err != nil {
			return fmt.Errorf("failed to use keystore, %v", err)
		}

		_, err = km.NextAddresses(tx, alwaysFalseCheck, false, 21, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != ErrGapLimit {
			return fmt.Errorf("failed to catch err, %v", err)
		}

		externalAddresses, err := km.NextAddresses(tx, alwaysTrueCheck, false, 20, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return fmt.Errorf("failed to new external address, %v", err)
		}

		showAddrManagerDetails(t, km.managedKeystores[accountID1])

		for _, addr := range externalAddresses {
			stdAddr = addr.address
			break
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("address: %v", stdAddr)
	addrManager, err := km.GetAddrManager(stdAddr)
	if err != nil {
		t.Fatalf("failed to get managed address, %v", err)
	}
	showAddrManagerDetails(t, addrManager)

	stdAddr = "ms1qevgq887qta2svxqu4uleczqa0a5xe4frtxvpk5"
	t.Logf("address:, %v", stdAddr)
	_, err = km.GetAddrManager(stdAddr)
	if err != ErrAccountNotFound {
		t.Fatalf("failed to catch error, %v", err)
	}
}

func TestKeystoreManager_GetAddrs(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown()
	t.Log("/*keystoreManager*/")
	// new keystore manager
	km := &KeystoreManager{}
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		bucket, err := mwdb.GetOrCreateTopLevelBucket(tx, keystoreBucket)
		if err != nil {
			return fmt.Errorf("failed to get bucket, %v", err)
		}
		km, err = NewKeystoreManager(bucket, pubPassphrase, &config.ChainParams)
		if err != nil {
			return fmt.Errorf("failed to new keystore manager, %v", err)
		}

		// invalid privpass
		_, _, err = km.NewKeystore(tx, defaultBitSize, invalidPass, "first", &config.ChainParams, nil, addressGapLimit)
		if err != ErrIllegalPassphrase {
			return fmt.Errorf("failed to catch error, %v", err)
		}

		// privpass same as pubpass
		_, _, err = km.NewKeystore(tx, defaultBitSize, pubPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != ErrIllegalNewPrivPass {
			return fmt.Errorf("failed to catch error, %v", err)
		}

		//new keystore
		accountID1, mnemonic1, err := km.NewKeystore(tx, 0, privPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}
		t.Logf("accountID: %v, mnemonic: %v", accountID1, mnemonic1)
		_, _, err = km.NewKeystore(tx, defaultBitSize, privPassphrase, "second", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}

		err = km.UseKeystoreForWallet(accountID1)
		if err != nil {
			return fmt.Errorf("failed to use keystore, %v", err)
		}

		_, err = km.NextAddresses(tx, alwaysFalseCheck, false, 21, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != ErrGapLimit {
			return fmt.Errorf("failed to catch err, %v", err)
		}

		_, err = km.NextAddresses(tx, alwaysTrueCheck, false, 20, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return fmt.Errorf("failed to new external address, %v", err)
		}

		showAddrManagerDetails(t, km.managedKeystores[accountID1])

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = km.GetAddrs("wrong account ID")
	if err != ErrAccountNotFound {
		t.Fatalf("failed to catch error, %v", err)
	}

	for _, accountID := range km.ListKeystoreNames() {
		addrs, err := km.GetAddrs(accountID)
		if err != nil {
			t.Fatalf("failed to get addresses, %v", err)
		}
		for _, addr := range addrs {
			t.Logf("addr: %v", addr)
		}
	}
}

func TestKeystoreManager_GetAddrManagerByAccountID(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)
		return
	}
	t.Log("get db")
	defer tearDown()

	km := &KeystoreManager{}
	var accountID1, accountID2 string
	err = mwdb.Update(ldb, func(dbTransaction mwdb.DBTransaction) error {
		var err error
		bucket, err := mwdb.GetOrCreateTopLevelBucket(dbTransaction, keystoreBucket)
		if err != nil {
			t.Fatalf("failed to get bucket, %v", err)
		}

		// new keystore manager
		km, err = NewKeystoreManager(bucket, pubPassphrase, &config.ChainParams)
		if err != nil {
			t.Fatalf("failed to new keystore manager, %v", err)
		}

		// new keystore
		var mnemonic1 string
		accountID1, mnemonic1, err = km.NewKeystore(dbTransaction, defaultBitSize, privPassphrase, "first account", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			t.Fatalf("failed to new keystore, %v", err)
		}
		t.Logf("accountID: %v, mnemonic: %v", accountID1, mnemonic1)

		var mnemonic2 string
		accountID2, mnemonic2, err = km.NewKeystore(dbTransaction, defaultBitSize, privPassphrase2, "second account", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			t.Fatalf("failed to new keystore, %v", err)
		}
		t.Logf("accountID: %v, mnemonic: %v", accountID2, mnemonic2)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to new keystore, %v", err)
	}

	// use the first keystore
	err = km.UseKeystoreForWallet(accountID1)
	if err != nil {
		t.Fatalf("failed to use keystore, %v", err)
	}

	err = mwdb.Update(ldb, func(dbTransaction mwdb.DBTransaction) error {
		// new two external addresses for first account
		externalAddresses, err := km.NextAddresses(dbTransaction, alwaysTrueCheck, false, 2, addressGapLimit, massutil.AddressClassWitnessStaking)
		if err != nil {
			t.Fatalf("fialed to new 2 external addresses for first account, %v", err)
		}
		for i, addr := range externalAddresses {
			t.Logf("new external address %v for account %v:", i, accountID1)
			managedAddressDetails(t, addr)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to new addresses, %v", err)
	}

	// use the second keystore
	err = km.UseKeystoreForWallet(accountID2)
	if err != nil {
		t.Fatalf("failed to use keystore, %v", err)
	}

	err = mwdb.Update(ldb, func(dbTransaction mwdb.DBTransaction) error {
		// new two external addresses for second account
		externalAddresses, err := km.NextAddresses(dbTransaction, alwaysTrueCheck, false, 3, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			t.Fatalf("fialed to new 2 external addresses for second account, %v", err)
		}
		for i, addr := range externalAddresses {
			t.Logf("new external address %v for account %v:", i, accountID2)
			managedAddressDetails(t, addr)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to new addresses, %v", err)
	}

	_, err = km.GetAddrManagerByAccountID("wrong accountID")
	if err != ErrAccountNotFound {
		t.Fatalf("failed to catch error, %v", err)
	}

	addrManager1, err := km.GetAddrManagerByAccountID(accountID1)
	if err != nil {
		t.Fatalf("failed to get addrManager, %v", err)
	}
	showAddrManagerDetails(t, addrManager1)

	addrManager2, err := km.GetAddrManagerByAccountID(accountID1)
	if err != nil {
		t.Fatalf("failed to get addrManager, %v", err)
	}
	showAddrManagerDetails(t, addrManager2)
}

func TestKeystoreManager_CheckPrivPassphrase(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)
		return
	}
	t.Log("get db")
	defer tearDown()

	km := &KeystoreManager{}
	var accountID1, accountID2 string
	err = mwdb.Update(ldb, func(dbTransaction mwdb.DBTransaction) error {
		var err error
		bucket, err := mwdb.GetOrCreateTopLevelBucket(dbTransaction, keystoreBucket)
		if err != nil {
			t.Fatalf("failed to get bucket, %v", err)
		}

		// new keystore manager
		km, err = NewKeystoreManager(bucket, pubPassphrase, &config.ChainParams)
		if err != nil {
			t.Fatalf("failed to new keystore manager, %v", err)
		}

		// new keystore
		var mnemonic1 string
		accountID1, mnemonic1, err = km.NewKeystore(dbTransaction, defaultBitSize, privPassphrase, "first account", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			t.Fatalf("failed to new keystore, %v", err)
		}
		t.Logf("accountID: %v, mnemonic: %v", accountID1, mnemonic1)

		var mnemonic2 string
		accountID2, mnemonic2, err = km.NewKeystore(dbTransaction, defaultBitSize, privPassphrase2, "second account", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			t.Fatalf("failed to new keystore, %v", err)
		}
		t.Logf("accountID: %v, mnemonic: %v", accountID2, mnemonic2)
		return nil
	})
	if err != nil {
		t.Fatalf("failed to new keystore, %v", err)
	}

	// wrong accountID
	err = km.CheckPrivPassphrase("wrong accountID", privPassphrase2)
	if err != ErrAccountNotFound {
		t.Fatalf("failed to catch error, %v", err)
	}

	// wrong pass
	err = km.CheckPrivPassphrase(accountID1, privPassphrase2)
	if err != ErrInvalidPassphrase {
		t.Fatalf("failed to catch error, %v", err)
	}

	// right pass
	err = km.CheckPrivPassphrase(accountID1, privPassphrase)
	if err != nil {
		t.Fatalf("failed to check privpass, %v", err)
	}

}

func TestKeystoreManager_RemoveCachedKeystore(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown()
	t.Log("/*keystoreManager*/")
	// new keystore manager
	kmw := &KeystoreManager{}
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		bucket, err := mwdb.GetOrCreateTopLevelBucket(tx, keystoreBucket)
		if err != nil {
			t.Fatalf("failed to get bucket, %v", err)
		}
		kmw, err = NewKeystoreManager(bucket, pubPassphrase, &config.ChainParams)
		if err != nil {
			t.Fatalf("failed to new keystore manager, %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to new keystore manager, %v", err)
	}

	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		var err error
		_, _, err = kmw.NewKeystore(tx, defaultBitSize, privPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to new keystore, %v", err)
	}

	var accountID2 string
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		var err error
		accountID2, _, err = kmw.NewKeystore(tx, defaultBitSize, privPassphrase, "second", &config.ChainParams, nil, addressGapLimit)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to new keystore, %v", err)
	}

	kmw.RemoveCachedKeystore(accountID2)

	err = mwdb.View(ldb, func(tx mwdb.ReadTransaction) error {
		kmw.UpdateManagedKeystores(tx, accountID2)
		return nil
	})
}

func TestNewKeystoreManager(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown()
	t.Log("/*keystoreManager*/")
	// new keystore manager
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		bucket, err := mwdb.GetOrCreateTopLevelBucket(tx, keystoreBucket)
		if err != nil {
			return fmt.Errorf("failed to get bucket, %v", err)
		}
		km, err := NewKeystoreManager(bucket, pubPassphrase, &config.ChainParams)
		if err != nil {
			return fmt.Errorf("failed to new keystore manager, %v", err)
		}

		// invalid privpass
		_, _, err = km.NewKeystore(tx, defaultBitSize, invalidPass, "first", &config.ChainParams, nil, addressGapLimit)
		if err != ErrIllegalPassphrase {
			return fmt.Errorf("failed to catch error, %v", err)
		}

		// privpass same as pubpass
		_, _, err = km.NewKeystore(tx, defaultBitSize, pubPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != ErrIllegalNewPrivPass {
			return fmt.Errorf("failed to catch error, %v", err)
		}

		//new keystore
		accountID1, mnemonic1, err := km.NewKeystore(tx, 0, privPassphrase, "first", &config.ChainParams, nil, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}
		t.Logf("accountID: %v, mnemonic: %v", accountID1, mnemonic1)
		_, _, err = km.NewKeystore(tx, defaultBitSize, privPassphrase, "second", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			return fmt.Errorf("failed to new keystore, %v", err)
		}

		err = km.UseKeystoreForWallet(accountID1)
		if err != nil {
			return fmt.Errorf("failed to use keystore, %v", err)
		}

		_, err = km.NextAddresses(tx, alwaysFalseCheck, false, 21, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != ErrGapLimit {
			return fmt.Errorf("failed to catch err, %v", err)
		}

		_, err = km.NextAddresses(tx, alwaysTrueCheck, false, 20, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return fmt.Errorf("failed to new external address, %v", err)
		}

		showAddrManagerDetails(t, km.managedKeystores[accountID1])

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		bucket, err := mwdb.GetOrCreateTopLevelBucket(tx, keystoreBucket)
		if err != nil {
			return fmt.Errorf("failed to get bucket, %v", err)
		}
		km, err := NewKeystoreManager(bucket, pubPassphrase, &config.ChainParams)
		if err != nil {
			return err
		}
		params := km.ChainParams()
		t.Logf("chain params: %v", params)
		for _, addrManager := range km.GetManagedAddrManager() {
			showAddrManagerDetails(t, addrManager)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestKeystoreManager_NextAddresses_Mock(t *testing.T) {
	ldb, tearDown, err := GetDb("Tst_Manager")
	if err != nil {
		t.Fatalf("init db failed: %v", err)

	}
	defer tearDown()
	t.Log("/*keystoreManager*/")
	// new keystore manager
	km := &KeystoreManager{}
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		bucket, err := mwdb.GetOrCreateTopLevelBucket(tx, keystoreBucket)
		if err != nil {
			return fmt.Errorf("failed to get bucket, %v", err)
		}
		km, err = NewKeystoreManager(bucket, pubPassphrase, &config.ChainParams)
		if err != nil {
			return fmt.Errorf("failed to new keystore manager, %v", err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	var accountID string
	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		var err error
		accountID, _, err = km.NewKeystore(tx, 128, privPassphrase, "test", &config.ChainParams, fastScrypt, addressGapLimit)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = mwdb.Update(ldb, func(tx mwdb.DBTransaction) error {
		err := km.UseKeystoreForWallet(accountID)
		if err != nil {
			return err
		}
		addrManager := km.managedKeystores[accountID]
		err = km.managedKeystores[accountID].checkPassword(privPassphrase)
		if err != nil {
			return err
		}

		saltPassphrase := append(addrManager.privPassphraseSalt[:], privPassphrase...)
		addrManager.hashedPrivPassphrase = sha512.Sum512(saltPassphrase)
		zero.Bytes(saltPassphrase)
		addrManager.unlocked = true

		addrs, err := km.NextAddresses(tx, alwaysTrueCheck, false, 20, addressGapLimit, massutil.AddressClassWitnessV0)
		if err != nil {
			return err
		}
		for _, addr := range addrs {
			priv, err := addrManager.getPrivKeyBtcec(addr.address, privPassphrase)
			if err != nil {
				return err
			}
			fmt.Println(hex.EncodeToString(priv.Serialize()))
		}

		for _, addr := range addrs {
			fmt.Println(addr.address)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func showAddrManagerDetails(t *testing.T, addrManager *AddrManager) {
	t.Logf("keystore name: %v", addrManager.Name())
	t.Logf("reamark: %v", addrManager.Remarks())
	t.Logf("usage: %v", addrManager.AddrUse())
	t.Logf("keyscope: %v", addrManager.KeyScope())
	t.Logf("account info, type: %v", addrManager.acctInfo.acctType)
	//t.Logf("account info, privkey: %v", addrManager.acctInfo.acctKeyPriv.String())
	ex, in := addrManager.CountAddresses()
	t.Logf("branch info, external index: %v, internal index: %v", addrManager.branchInfo.nextExternalIndex, addrManager.branchInfo.nextInternalIndex)
	t.Logf("branch info, external addresses count: %v, internal addresses count: %v", ex, in)
	t.Logf("unlocked: %v", addrManager.unlocked)
	t.Logf("master pubkey: %v", addrManager.masterKeyPub.Marshal())
	t.Logf("master privkey: %v", addrManager.masterKeyPriv.Marshal())
	t.Logf("salt: %v", addrManager.privPassphraseSalt)
	t.Logf("hash: %v", addrManager.hashedPrivPassphrase)
	t.Logf("")
	for _, addr := range addrManager.ManagedAddresses() {
		managedAddressDetails(t, addr)
	}
}

func managedAddressDetails(t *testing.T, addr *ManagedAddress) {
	t.Logf("keystoreName: %v", addr.Account())
	t.Logf("address: %v", addr.String())
	t.Logf("change: %v", addr.IsChangeAddr())
	t.Logf("lock address: %v", addr.StakingAddress())
	t.Logf("derivation path: %v", addr.derivationPath)
	t.Logf("public key: %v", hex.EncodeToString(addr.PubKey().SerializeCompressed()))
	if addr.privKey != nil {
		t.Logf("private key: %v", hex.EncodeToString(addr.PrivKey().Serialize()))
	} else {
		t.Logf("private key: %v", addr.PrivKey())
	}
	t.Logf("scriptHash: %v", addr.ScriptAddress())
	redeemScript, err := addr.RedeemScript(&config.ChainParams)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("redeem script: %v", redeemScript)
	t.Log()
}
