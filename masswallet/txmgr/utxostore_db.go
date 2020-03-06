package txmgr

import (
	"encoding/binary"
	"errors"
	"fmt"

	"massnet.org/mass-wallet/massutil"
	mwdb "massnet.org/mass-wallet/masswallet/db"
	"massnet.org/mass-wallet/masswallet/utils"
	"massnet.org/mass-wallet/wire"
)

func canonicalOutPoint(txHash *wire.Hash, index uint32) []byte {
	k := make([]byte, 36)
	copy(k, txHash[:])
	binary.BigEndian.PutUint32(k[32:36], index)
	return k
}

func canonicalUnspentKey(walletId string, txHash *wire.Hash, index uint32) []byte {
	k := make([]byte, 78)
	copy(k[0:42], []byte(walletId))
	copy(k[42:74], txHash[:])
	binary.BigEndian.PutUint32(k[74:78], index)
	return k
}

func readCanonicalUnspentKey(k []byte, op *wire.OutPoint) error {
	if len(k) < 78 {
		return fmt.Errorf("short canonical unspent key (actual %d bytes)", len(k))
	}
	copy(op.Hash[:], k[42:74])
	op.Index = binary.BigEndian.Uint32(k[74:78])
	return nil
}

func existsUnspent(ns mwdb.Bucket, walletId string, outPoint *wire.OutPoint) (k, credKey []byte, err error) {

	widLen := len(walletId)
	if widLen != 42 {
		return nil, nil, fmt.Errorf("short walletId value (expect 42 bytes, actual %d bytes)", widLen)
	}

	k = canonicalUnspentKey(walletId, &outPoint.Hash, outPoint.Index)
	credKey, err = existsRawUnspent(ns, k)
	if err != nil {
		return nil, nil, err
	}
	return k, credKey, nil
}

// existsRawUnspent return bucketCredits key
func existsRawUnspent(ns mwdb.Bucket, k []byte) (credKey []byte, err error) {
	if len(k) < 78 {
		return nil, fmt.Errorf("short unspent key (expect 78 bytes, actual %d bytes)", len(k))
	}
	v, err := ns.Get(k)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	credKey = make([]byte, 76)
	copy(credKey, k[42:74])
	copy(credKey[32:72], v)
	copy(credKey[72:76], k[74:78])
	return credKey, nil
}

func deleteRawUnspent(ns mwdb.Bucket, k []byte) error {
	err := ns.Delete(k)
	if err != nil {
		return fmt.Errorf("failed to delete unspent: %v", err)
	}
	return nil
}

func putRawUnminedInput(ns mwdb.Bucket, k, v []byte) error {
	spendTxHashes, err := ns.Get(k)
	if err != nil {
		return err
	}
	spendTxHashes = append(spendTxHashes, v...)
	return ns.Put(k, spendTxHashes)
}
func existsRawUnminedInput(ns mwdb.Bucket, k []byte) (v []byte) {
	v, _ = ns.Get(k)
	return v
}

func deleteRawUnminedInput(ns mwdb.Bucket, k []byte) error {
	err := ns.Delete(k)
	if err != nil {
		return fmt.Errorf("failed to delete unmined input: %v", err)
	}
	return nil
}

// fetchUnminedInputSpendTxHashes fetches the list of unmined transactions that
// spend the serialized outpoint.
func fetchUnminedInputSpendTxHashes(ns mwdb.Bucket, k []byte) []wire.Hash {
	rawSpendTxHashes, _ := ns.Get(k)
	if rawSpendTxHashes == nil {
		return nil
	}

	// Each transaction hash is 32 bytes.
	spendTxHashes := make([]wire.Hash, 0, len(rawSpendTxHashes)/32)
	for len(rawSpendTxHashes) > 0 {
		var spendTxHash wire.Hash
		copy(spendTxHash[:], rawSpendTxHashes[:32])
		spendTxHashes = append(spendTxHashes, spendTxHash)
		rawSpendTxHashes = rawSpendTxHashes[32:]
	}

	return spendTxHashes
}

//    credit
func keyCredit(txHash *wire.Hash, index uint32, block *BlockMeta) []byte {
	k := make([]byte, 76)
	copy(k, txHash[:])
	binary.BigEndian.PutUint64(k[32:40], block.Height)
	copy(k[40:72], block.Hash[:])
	binary.BigEndian.PutUint32(k[72:76], index)
	return k
}

// valueUnspentCredit creates a new credit value for an unspent credit.  All
// credits are created unspent, and are only marked spent later, so there is no
// value function to create either spent or unspent credits.
func valueUnspentCredit(cred *credit) ([]byte, error) {
	if len(cred.scriptHash) != 32 {
		return nil, fmt.Errorf("short script hash (expect 32 bytes)")
	}
	v := make([]byte, 45)
	binary.BigEndian.PutUint64(v, cred.amount.UintValue())
	if cred.flags.Change {
		v[8] |= 1 << 1
	}
	if cred.flags.Class == ClassStakingUtxo {
		v[8] |= 1 << 2
	}
	if cred.flags.Class == ClassBindingUtxo {
		v[8] |= 1 << 3
	}
	binary.BigEndian.PutUint32(v[9:13], cred.maturity)
	copy(v[13:45], cred.scriptHash)
	return v, nil
}

// including both mined credit and unmined credit
func readCreditValue(v []byte, cred *credit) error {
	if len(v) < 45 {
		return fmt.Errorf("short credit(mined or unmined) value")
	}
	amount, err := massutil.NewAmountFromUint(binary.BigEndian.Uint64(v))
	if err != nil {
		return err
	}
	cred.amount = amount
	cred.flags.Spent = v[8]&(1<<0) != 0
	cred.flags.Change = v[8]&(1<<1) != 0
	cred.maturity = binary.BigEndian.Uint32(v[9:13])
	cred.scriptHash = v[13:45]

	f := (v[8] & (3 << 2)) >> 2
	switch f {
	case 0:
		cred.flags.Class = ClassStandardUtxo
	case 1:
		cred.flags.Class = ClassStakingUtxo
	case 2:
		cred.flags.Class = ClassBindingUtxo
	default:
		return fmt.Errorf("unknown utxo class")
	}
	return nil
}

func readRawCreditKey(k []byte, cred *credit) error {
	if len(k) < 76 {
		return fmt.Errorf("short credit key")
	}
	copy(cred.outPoint.Hash[:], k[0:32])
	cred.block.Height = binary.BigEndian.Uint64(k[32:40])
	copy(cred.block.Hash[:], k[40:72])
	cred.outPoint.Index = binary.BigEndian.Uint32(k[72:76])
	return nil
}

func readUnminedCreditKey(k []byte, cred *credit) error {
	if len(k) != 36 {
		return fmt.Errorf("short k value (expected 36 bytes, read %v)", len(k))
	}

	copy(cred.outPoint.Hash[:], k[0:32])
	cred.outPoint.Index = binary.BigEndian.Uint32(k[32:36])
	return nil
}

func existsRawUnminedCredit(ns mwdb.Bucket, k []byte) ([]byte, error) {
	if len(k) < 36 {
		return nil, fmt.Errorf("short k read (expected 36 bytes, read %v)", len(k))
	}
	return ns.Get(k)
}

func deleteRawUnminedCredit(ns mwdb.Bucket, k []byte) error {
	err := ns.Delete(k)
	if err != nil {
		return fmt.Errorf("failed to delete unmined credit: %v", err)
	}
	return nil
}

func valueUnminedCredit(amount massutil.Amount, change bool, maturity uint32,
	scriptHash []byte, ps utils.PkScript) ([]byte, error) {
	if len(scriptHash) != 32 {
		return nil, fmt.Errorf("invalid script hash length (expected 32 bytes, read %v)", len(scriptHash))
	}
	v := make([]byte, 45)
	binary.BigEndian.PutUint64(v, amount.UintValue())
	if change {
		v[8] = 1 << 1
	}
	if ps.IsStaking() {
		v[8] |= 1 << 2
	}
	if ps.IsBinding() {
		v[8] |= 1 << 3
	}
	binary.BigEndian.PutUint32(v[9:13], maturity)
	copy(v[13:45], scriptHash)
	return v, nil
}

func valueUnminedCreditFromMined(credValue []byte) ([]byte, error) {
	if len(credValue) < 45 {
		return nil, fmt.Errorf("short v read (expected 45 bytes, read %v)", len(credValue))
	}
	return credValue[0:45], nil
}

func putRawUnminedCredit(ns mwdb.Bucket, k, v []byte) error {
	err := ns.Put(k, v)
	if err != nil {
		return fmt.Errorf("cannot put unmined credit: %v", err)
	}
	return nil
}

func existsCredit(ns mwdb.Bucket, txHash *wire.Hash, index uint32, block *BlockMeta) (k, v []byte, err error) {
	k = keyCredit(txHash, index, block)
	v, err = ns.Get(k)
	if err != nil {
		k = nil
		v = nil
	}
	return
}

func existsRawCredit(ns mwdb.Bucket, k []byte) (v []byte, err error) {
	v, err = ns.Get(k)
	if err != nil {
		v = nil
	}
	return
}

func readCreditSpender(credValue []byte) (debitKey []byte) {
	if len(credValue) < 121 {
		return nil
	}
	debitKey = make([]byte, 76)
	copy(debitKey, credValue[45:121])
	return
}

func getCreditsByTxHash(ns mwdb.Bucket, txsha *wire.Hash) ([]*mwdb.Entry, error) {
	return ns.GetByPrefix(txsha[:])
}

func getCreditsByTxHashHeight(ns mwdb.Bucket, txsha *wire.Hash, height uint64) (map[uint32]*mwdb.Entry, error) {
	if len(txsha) != 32 {
		return nil, fmt.Errorf("short hash value (expected 32 bytes, read %d)", len(txsha))
	}
	prefix := make([]byte, 40)
	copy(prefix[0:32], txsha[:])
	binary.BigEndian.PutUint64(prefix[32:40], height)
	entries, err := ns.GetByPrefix(prefix)
	if err != nil {
		return nil, err
	}
	result := make(map[uint32]*mwdb.Entry)
	for _, entry := range entries {
		index := binary.BigEndian.Uint32(entry.Key[72:76])
		result[index] = entry
	}
	return result, nil
}

func getLastCreditByTxHashIndexTillHeight(ns mwdb.Bucket, txsha *wire.Hash,
	index uint32, height uint64) (*mwdb.Entry, error) {
	if len(txsha) != 32 {
		return nil, fmt.Errorf("short hash value (expected 32 bytes, read %d)", len(txsha))
	}
	entries, err := ns.GetByPrefix(txsha[:])
	if err != nil {
		return nil, err
	}
	target := uint64(0)
	var ret *mwdb.Entry
	for _, entry := range entries {
		eIdx := binary.BigEndian.Uint32(entry.Key[72:76])
		if eIdx == index {
			eHeight := binary.BigEndian.Uint64(entry.Key[32:40])
			if eHeight <= height && eHeight > target {
				target = eHeight
				ret = entry
			}
		}
	}
	return ret, nil
}

func putRawCredit(ns mwdb.Bucket, k, v []byte) error {
	err := ns.Put(k, v)
	if err != nil {
		return fmt.Errorf("failed to put credit: %v", err)
	}
	return nil
}

func deleteRawCredit(ns mwdb.Bucket, k []byte) error {
	err := ns.Delete(k)
	if err != nil {
		return fmt.Errorf("failed to delete credit: %v", err)
	}
	return nil
}

func spendCredit(ns mwdb.Bucket, credKey []byte, spender *indexedIncidence) (massutil.Amount, error) {
	v, err := ns.Get(credKey)
	if err != nil {
		return massutil.ZeroAmount(), err
	}
	vLen := len(v)
	if vLen != 45 {
		return massutil.ZeroAmount(), fmt.Errorf("short v read (expected 45 bytes, read %v)", vLen)
	}
	newv := make([]byte, vLen+76) // 76 is the length of spender info
	copy(newv, v)
	v = newv
	v[8] |= 1 << 0
	copy(v[vLen:vLen+32], spender.txHash[:])
	binary.BigEndian.PutUint64(v[vLen+32:vLen+40], spender.block.Height)
	copy(v[vLen+40:vLen+72], spender.block.Hash[:])
	binary.BigEndian.PutUint32(v[vLen+72:vLen+76], spender.index)

	amount, err := massutil.NewAmountFromUint(binary.BigEndian.Uint64(v[0:8]))
	if err != nil {
		return massutil.ZeroAmount(), err
	}
	return amount, putRawCredit(ns, credKey, v)
}

// unspendRawCredit rewrites the credit for the given key as unspent.  The
// output amount of the credit is returned.  It returns without error if no
// credit exists for the key.
func unspendRawCredit(ns mwdb.Bucket, credKey []byte) (*credit, error) {

	v, err := ns.Get(credKey)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}

	newv := make([]byte, 45)
	copy(newv, v)
	newv[8] &^= 1 << 0

	err = ns.Put(credKey, newv)
	if err != nil {
		return nil, fmt.Errorf("failed to put unspend credit: %v", err)
	}

	c := &credit{
		block: &BlockMeta{},
	}
	err = readCreditValue(newv, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// fetchMinedBalance return nil if account not found
// account is bech32 encoded address
func fetchMinedBalance(ns mwdb.Bucket, account string) ([]*mwdb.Entry, error) {
	// Entry.Key is the keystore name
	// Entry.Value is amount
	if len(account) > 0 {
		v, err := ns.Get([]byte(account))
		if err != nil {
			return nil, err
		}
		if v == nil {
			return nil, nil
		}
		return []*mwdb.Entry{&mwdb.Entry{
			Key:   []byte(account),
			Value: v,
		}}, nil
	}
	return ns.GetByPrefix(nil)
}

func putMinedBalance(ns mwdb.Bucket, walletId string, amt massutil.Amount) error {
	if len(walletId) != 42 {
		return fmt.Errorf("putMinedBalance: short read (expected 42 bytes, read %v)", len(walletId))
	}
	v := make([]byte, 8)
	binary.BigEndian.PutUint64(v, amt.UintValue())
	err := ns.Put([]byte(walletId), v)
	if err != nil {
		return fmt.Errorf("failed to put balance, account: %s, amount: %d, err: %v", walletId, amt.UintValue(), err)
	}
	return nil
}

func deleteMinedBalance(ns mwdb.Bucket, account string) error {
	return ns.Delete([]byte(account))
}

func keyDebit(txHash *wire.Hash, index uint32, block *BlockMeta) []byte {
	k := make([]byte, 76)
	copy(k, txHash[:])
	binary.BigEndian.PutUint64(k[32:40], block.Height)
	copy(k[40:72], block.Hash[:])
	binary.BigEndian.PutUint32(k[72:76], index)
	return k
}

func putDebit(ns mwdb.Bucket, txHash *wire.Hash, index uint32, amount massutil.Amount, block *BlockMeta, credKey []byte) error {
	k := keyDebit(txHash, index, block)

	v := make([]byte, 84)
	binary.BigEndian.PutUint64(v, amount.UintValue())
	copy(v[8:84], credKey)

	err := ns.Put(k, v)
	if err != nil {
		return fmt.Errorf("failed to put debit %s input %d, err: %v",
			txHash, index, err)
	}
	return nil
}

func existsDebit(ns mwdb.Bucket, txHash *wire.Hash, index uint32, block *BlockMeta) (k, credKey []byte, err error) {
	k = keyDebit(txHash, index, block)
	v, err := ns.Get(k)
	if err != nil {
		return nil, nil, err
	}
	if v == nil {
		return nil, nil, nil
	}
	if len(v) < 84 {
		return nil, nil, fmt.Errorf("%s: short read (expected 84 bytes, read %v)", bucketDebits, len(v))
	}
	return k, v[8:84], nil
}

func deleteRawDebit(ns mwdb.Bucket, k []byte) error {
	err := ns.Delete(k)
	if err != nil {
		return fmt.Errorf("failed to delete debit: %v", err)
	}
	return nil
}

func fetchTxRecordKeyFromRawCreditKey(k []byte) ([]byte, error) {
	if len(k) < 72 {
		return nil, fmt.Errorf("short k read (expected at least 72 bytes, read %d)", len(k))
	}
	return k[0:72], nil
}

func fetchRawCreditAmountSpent(v []byte) (massutil.Amount, bool, error) {
	if len(v) < 45 {
		return massutil.ZeroAmount(), false,
			fmt.Errorf("short v read (expected 45 bytes, read %d)", len(v))
	}
	amt, err := massutil.NewAmountFromUint(binary.BigEndian.Uint64(v[0:8]))
	if err != nil {
		return massutil.ZeroAmount(), false, err
	}
	spent := v[8]&(1<<0) != 0
	return amt, spent, nil
}

func fetchRawCreditMaturityScriptHash(v []byte) (uint32, []byte, error) {
	if len(v) < 45 {
		return 0, nil, fmt.Errorf("short v read (expected 45 bytes, read %d)", len(v))
	}
	return binary.BigEndian.Uint32(v[9:13]), v[13:45], nil
}

// fetchNsUnspentValueFromRawCredit returns the unspent value for a raw credit key.
// This may be used to mark a credit as unspent.
func fetchNsUnspentValueFromRawCredit(k []byte) ([]byte, error) {
	if len(k) < 76 {
		return nil, fmt.Errorf("short key (expected 76 bytes, read %d)", len(k))
	}
	return k[32:72], nil
}

func valueUnspent(block *BlockMeta) []byte {
	v := make([]byte, 40)
	binary.BigEndian.PutUint64(v, block.Height)
	copy(v[8:40], block.Hash[:])
	return v
}

func putUnspent(ns mwdb.Bucket, walletId string, outPoint *wire.OutPoint, block *BlockMeta) error {
	widLen := len(walletId)
	if widLen != 42 {
		return fmt.Errorf("short walletId value (expect 42 bytes, actual %d bytes)", widLen)
	}

	k := canonicalUnspentKey(walletId, &outPoint.Hash, outPoint.Index)
	v := valueUnspent(block)
	err := ns.Put(k, v)
	if err != nil {
		return fmt.Errorf("cannot put unspent: %v", err)
	}
	return nil
}

func putRawUnspent(ns mwdb.Bucket, k, v []byte) error {
	if len(k) != 78 || len(v) != 40 {
		return fmt.Errorf("invalid k/v length (key %d bytes, value %d bytes)", len(k), len(v))
	}
	err := ns.Put(k, v)
	if err != nil {
		return fmt.Errorf("cannot put unspent: %v", err)
	}
	return nil
}

func readBlockOfUnspent(v []byte, block *BlockMeta) error {
	if len(v) < 40 {
		return fmt.Errorf("short unspent value (expect %d bytes, read %d)", 40, len(v))
	}
	block.Height = binary.BigEndian.Uint64(v)
	copy(block.Hash[:], v[8:40])
	return nil
}

func keyAddressRecord(rec *addressRecord) ([]byte, error) {
	widLen := len(rec.walletId)
	if widLen != 42 {
		return nil, fmt.Errorf("short walletId value (expect 42 bytes, actual %d bytes)", widLen)
	}
	if len(rec.encodeAddress) == 0 {
		return nil, errors.New("empty encodeAddr value")
	}
	if !massutil.IsValidAddressClass(rec.addressClass) {
		return nil, fmt.Errorf("unexpected address class %#x", rec.addressClass)
	}

	k := make([]byte, widLen+2+len(rec.encodeAddress))
	copy(k, []byte(rec.walletId))
	binary.BigEndian.PutUint16(k[widLen:widLen+2], rec.addressClass)
	copy(k[widLen+2:], []byte(rec.encodeAddress))
	return k, nil
}

func valueAddressRecord(rec *addressRecord) []byte {
	v := make([]byte, 8)
	binary.BigEndian.PutUint64(v, rec.blockHeight)
	return v
}

func putRawAddressRecord(ns mwdb.Bucket, k, v []byte) error {
	return ns.Put(k, v)
}

func existsRawAddressRecord(ns mwdb.Bucket, k []byte) ([]byte, error) {
	return ns.Get(k)
}

func deleteRawAddressRecord(ns mwdb.Bucket, k []byte) error {
	return ns.Delete(k)
}

func readAddressHeight(v []byte) uint64 {
	return binary.BigEndian.Uint64(v)
}

func fetchAddressesByWalletId(ns mwdb.Bucket, walletId string) ([]*AddressDetail, error) {
	entries, err := ns.GetByPrefix([]byte(walletId))
	if err != nil {
		return nil, err
	}
	ret := make([]*AddressDetail, 0)
	for _, entry := range entries {
		ad := &AddressDetail{
			Address:      string(entry.Key[44:]),
			AddressClass: binary.BigEndian.Uint16(entry.Key[42:44]),
			Used:         binary.BigEndian.Uint64(entry.Value) > 0,
		}
		ret = append(ret, ad)
	}
	return ret, nil
}

func keyMinedLGHistory(out *lgTxHistory) []byte {
	k := make([]byte, 84)
	copy(k, []byte(out.walletId))
	if out.isBinding {
		k[42] = 1
	}
	if out.isWithdraw {
		k[43] = 1
	}
	copy(k[44:76], out.txhash[:])
	// binary.BigEndian.PutUint32(k[76:80], out.index)
	binary.BigEndian.PutUint64(k[76:84], out.blockHeight)
	return k
}

func keyUnminedLGHistory(out *lgTxHistory) []byte {
	k := make([]byte, 76)
	copy(k, []byte(out.walletId))
	if out.isBinding {
		k[42] = 1
	}
	if out.isWithdraw {
		k[43] = 1
	}
	copy(k[44:76], out.txhash[:])
	return k
}

func valueLGHistory(out *lgTxHistory) []byte {
	num := len(out.indexes)
	v := make([]byte, (num+1)*4)
	binary.BigEndian.PutUint32(v, uint32(num))
	for i := 0; i < num; i++ {
		binary.BigEndian.PutUint32(v[4*(i+1):], out.indexes[i])
	}
	return v
}

func existsRawLGOutput(ns mwdb.Bucket, k []byte) ([]byte, error) {
	return ns.Get(k)
}

func deleteRawLGOutput(ns mwdb.Bucket, k []byte) error {
	return ns.Delete(k)
}

func putUnminedLGHistory(ns mwdb.Bucket, out *lgTxHistory) error {
	k := keyUnminedLGHistory(out)
	v := valueLGHistory(out)
	return ns.Put(k, v)
}

func putRawUnminedLGHistory(ns mwdb.Bucket, k, v []byte) error {
	return ns.Put(k, v)
}

func putLGHistory(ns mwdb.Bucket, out *lgTxHistory) error {
	k := keyMinedLGHistory(out)
	v := valueLGHistory(out)
	return ns.Put(k, v)
}

func deleteUnminedLGHistory(ns mwdb.Bucket, out *lgTxHistory) error {
	k := keyUnminedLGHistory(out)
	return ns.Delete(k)
}

// used for unmined/mined history
func fetchRawLGHistoryByWalletId(ns mwdb.Bucket, walletId string, tp outputType) ([]*mwdb.Entry, error) {
	if len(walletId) != 42 {
		return nil, fmt.Errorf("invalid walletId value (expect 42 bytes, actual %d bytes)", len(walletId))
	}
	prefix := make([]byte, 43)
	copy(prefix, []byte(walletId))
	prefix[42] = byte(tp)
	return ns.GetByPrefix(prefix)
}

// used for unmined/mined history
func readLGHistory(isUnmined bool, k, v []byte, lout *lgTxHistory) error {
	if (isUnmined && len(k) < 76) || (!isUnmined && len(k) < 84) {
		return fmt.Errorf("invalid lg history k value (unmined %v, actual %d bytes)", isUnmined, len(k))
	}
	if len(v) < 4 {
		return fmt.Errorf("invalid lg history v value (expect 4 bytes, actual %d bytes)", len(v))
	}
	lout.walletId = string(k[0:42])
	lout.isBinding = k[42]&(1<<0) != 0
	lout.isWithdraw = k[43]&(1<<0) != 0
	copy(lout.txhash[:], k[44:76])
	if !isUnmined {
		lout.blockHeight = binary.BigEndian.Uint64(k[76:84])
	}

	num := binary.BigEndian.Uint32(v[0:4])
	for i := uint32(1); i <= num; i++ {
		lout.indexes = append(lout.indexes, binary.BigEndian.Uint32(v[4*i:4*(i+1)]))
	}
	return nil
}

func deleteByPrefix(ns mwdb.Bucket, prefix []byte) error {
	entries, err := ns.GetByPrefix(prefix)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		err = ns.Delete(entry.Key)
		if err != nil {
			return err
		}
	}
	return nil
}
