package txmgr

import (
	"encoding/binary"
	"errors"
	"math"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"massnet.org/mass-wallet/massutil"
	mwdb "massnet.org/mass-wallet/masswallet/db"
	"massnet.org/mass-wallet/masswallet/utils"
	"massnet.org/mass-wallet/wire"
)

const (
	//-----------------tx buckets-----------------

	// Key:
	//    [0:32]  - txrecord.hash
	// Value:
	//    [0:8]  - tx received time
	//    [8:8+serialSize(tx)] - serialize(MsgTx)
	bucketUnmined = "m"

	// Key:
	//    [0:32]  - txhash
	//    [32:40] - block.height
	//    [40:72] - block.hash
	// Value:
	//    [0:8]  - received unix time
	//    [8:8+serialSize(tx)] - serialize(MsgTx)
	bucketTxRecords = "t"

	// Key:
	//    [0:8]  - block.height
	// Value:
	//    [0:32]  - block.hash
	//    [32:40]  - block.timestamp
	//    [40:44]  - num of tx
	//    [44:44+32*num]  - txhash*num
	bucketBlocks = "b"

	// Key:
	//    [:42]     - wallet id (bech32 string)
	//    [42:44]   - address class(2 byte or big-endian uint16)
	//                massutil.AddressClassWitnessV0: 	0x0000
	//                massutil.AddressClassWitnessStaking: 0x0001
	//    [44:]     - address (bech32 string)
	// Value:
	//    [:8] -  first use height, default 0(not used)
	bucketAddresses = "a"

	// Key:
	//    [0:42]		- wallet id(bech32 string)
	//    [42:43]		- type
	//           			0: staking
	//                		1: binding
	//    [43:44]    	- op
	//              		0: deposit
	//              		1: withdraw
	//    [44:76]		- tx hash
	//    [76:84]       - block height
	//
	// Value:
	//	  [0:4]         - num of index of output(op = 0) or input(op = 1)
	//    [4:4+4*num]   - index * num
	bucketLGOutput = "lg"

	// Key:
	//    [0:42]		- wallet id(bech32 string)
	//    [42:43]		- type
	//           			0: staking
	//                		1: binding
	//    [43:44]    	- op
	//              		0: deposit
	//              		1: withdraw
	//    [44:76]		- tx hash
	//
	// Value:
	//	  [0:4]         - num of index of output(op = 0) or input(op = 1)
	//    [4:4+4*num]  - index * num
	bucketUnminedLGOutput = "LG"

	//-----------------utxo buckets-----------------

	// Key:
	//    [0:32]  - hash
	//    [32:36] - index
	// Value:
	//    [0:32*N]  - spendTxHash * N
	bucketUnminedInputs = "mi"

	// Key:
	//    [0:32]  - hash
	//    [32:36] - index
	// Value:
	//    [0:8]  - amount: uint64
	//    [8:9]     - Flags (1 byte)
	//               0x01: Spent
	//               0x02: Change
	//				 0x04: Staking
	//				 0x08: Binding
	//    [9:13]  - utxo maturity
	//    [13:45] - witness script hash
	bucketUnminedCredits = "mc"

	// Key:
	//    [0:32]  - hash of txrecord
	//    [32:40] - block.height
	//    [40:72] - block.hash
	//    [72:76] - index of txout
	// Value:
	//    [0:8]   - Amount (8 bytes)
	//    [8:9]     - Flags (1 byte)
	//               0x01: Spent
	//               0x02: Change
	//				 0x04: Staking
	//				 0x08: Binding
	//    [9:13]  - utxo maturity
	//    [13:45] - witness script hash
	//    optional spender info:
	//        [45:77]     - hash of spender tx
	//        [77:85]     - block height
	//        [85:117]    - block hash
	//        [117:121]   - input index of spender tx
	bucketCredits = "c"

	// Key:
	//   [0:42]	   - wallet id(bech32 string)
	//   [42:74]   - Transaction hash (32 bytes)
	//   [74:78]   - Output index (4 bytes)
	// Value:
	//    [0:8]   - block.height
	//    [8:40]  - block.hash
	bucketUnspent = "u"

	// Key:
	//    [0:32]  - hash of txrecord
	//    [32:40] - block.height
	//    [40:72] - block.hash
	//    [72:76] - index of txin
	// Value:
	//    [0:8]  - amount
	//    [8:84]  - credit.key
	bucketDebits = "d"
)

const (
	// Key:
	//    encoded account string
	// Value:
	//    [0:8]  - amount
	bucketMinedBalance = "bal"
)

const (
	// Key:
	//    [0:8]   - height
	// Value:
	//    [0:32]  - hash
	//	  [32:36]   - timestamp
	syncBucketName = "sync"
	// 'syncedToName' is one of the keys in bucket 'syncBucketName', the value is:
	//    [0:8]     - height
	syncedToName = "syncedto"

	// Key:
	//    [0:42] - wallet_id
	// Value:
	//    [0:8]  - synced height, it will be max uint64 when syncing done
	bucketWalletStatus = "ws"
)

type outputType byte

const (
	outputStaking outputType = iota
	outputBinding
)

// var
var (
	// ErrBlockRecordNotFound = errors.New("block record not found")
	ErrNotFound = errors.New("not found")
	// ErrMaybeChainForks     = errors.New("maybe chain forks")
)

type RelevantMeta struct {
	Index        int
	PkScript     utils.PkScript
	WalletId     string
	IsChangeAddr bool
}

// TxRecord ...
type TxRecord struct {
	MsgTx         wire.MsgTx
	Hash          wire.Hash
	Received      time.Time
	RelevantTxIn  []*RelevantMeta
	RelevantTxOut []*RelevantMeta
	HasBindingIn  bool
	HasBindingOut bool
}

// NewTxRecordFromMsgTx ...
func NewTxRecordFromMsgTx(msgTx *wire.MsgTx, received time.Time) (*TxRecord, error) {
	rec := &TxRecord{
		MsgTx:         *msgTx,
		Hash:          msgTx.TxHash(),
		Received:      received,
		RelevantTxIn:  make([]*RelevantMeta, 0),
		RelevantTxOut: make([]*RelevantMeta, 0),
	}

	return rec, nil
}

// BlockMeta ...
type BlockMeta struct {
	Height    uint64
	Hash      wire.Hash
	Timestamp time.Time
}

type blockRecord struct {
	BlockMeta
	transactions []wire.Hash
}

type incidence struct {
	txHash wire.Hash
	block  BlockMeta
}

// indexedIncidence records the transaction incidence and an input or output
// index.
type indexedIncidence struct {
	incidence
	index uint32
}

type addressRecord struct {
	walletId      string
	encodeAddress string
	addressClass  uint16 // massutil.AddressClassWitnessV0 or massutil.AddressClassWitnessStaking
	blockHeight   uint64
}

type UtxoClass int

const (
	ClassStandardUtxo UtxoClass = 0
	ClassStakingUtxo  UtxoClass = 1
	ClassBindingUtxo  UtxoClass = 2
	ClassUnknownUtxo  UtxoClass = math.MaxInt32
)

type credit struct {
	outPoint   wire.OutPoint
	block      *BlockMeta
	amount     massutil.Amount
	maturity   uint32
	scriptHash []byte // length = txscript.WitnessV0ScriptHashDataSize(32)
	flags      UtxoFlags
	spentBy    indexedIncidence // Index == ^uint32(0) if unspent
}

// Credit ...
type Credit struct {
	wire.OutPoint
	BlockMeta
	Amount   massutil.Amount
	Maturity uint32

	Confirmations uint32
	Flags         UtxoFlags
	ScriptHash    []byte // for ScriptAddressUnspents
}

type UtxoFlags struct {
	Spent          bool
	SpentByUnmined bool
	Change         bool
	IsUnmined      bool
	Class          UtxoClass
}

// Debit ...
type Debit struct {
	PreviousOutPoint wire.OutPoint
}

type BalanceDetail struct {
	Total               massutil.Amount
	Spendable           massutil.Amount
	WithdrawableStaking massutil.Amount
	WithdrawableBinding massutil.Amount
}

type AddressDetail struct {
	Address      string
	AddressClass uint16
	Used         bool
	StdAddress   string // not empty when AddressClass=massutil.AddressClassWitnessStaking
	PubKey       *btcec.PublicKey
}

// staking & binding tx history
type lgTxHistory struct {
	walletId    string
	txhash      wire.Hash
	indexes     []uint32 // index of outputs(isWithdraw = false) or inputs(isWithdraw = true)
	isWithdraw  bool     // false-deposit, true-withdraw
	isBinding   bool     // false-staking, true-binding
	blockHeight uint64   // 0 means unmined
}

type StakingUtxo struct {
	Hash           wire.Hash
	Index          uint32
	Address        string
	Amount         massutil.Amount
	Spent          bool
	SpentByUnmined bool
	FrozenPeriod   uint32
}
type StakingHistoryDetail struct {
	TxHash      wire.Hash
	Index       uint32
	Op          byte // 0-deposit, 1-withdraw
	BlockHeight uint64
	Utxo        StakingUtxo
}

func (l *StakingHistoryDetail) IsDeposit() bool {
	return l.Op == 0
}

type BindingUtxo struct {
	Hash           wire.Hash
	Index          uint32
	HolderAddress  string
	BindingAddress string
	Amount         massutil.Amount
	Spent          bool
	SpentByUnmined bool
}
type BindingHistoryDetail struct {
	TxHash      wire.Hash
	Index       uint32
	Op          byte // 0-deposit, 1-withdraw
	BlockHeight uint64
	Utxo        BindingUtxo
	MsgTx       *wire.MsgTx
}

func (g *BindingHistoryDetail) IsDeposit() bool {
	return g.Op == 0
}

// Hash credit hash
func (c *Credit) Hash() []byte {
	k := make([]byte, 36)
	copy(k, c.OutPoint.Hash[:])
	binary.BigEndian.PutUint32(k[32:36], c.OutPoint.Index)
	return k
}

func (c *credit) isStaking() bool {
	return c.flags.Class == ClassStakingUtxo
}

func (c *credit) isBinding() bool {
	return c.flags.Class == ClassBindingUtxo
}

// Hash debit hash
func (d *Debit) Hash() []byte {
	k := make([]byte, 36)
	copy(k, d.PreviousOutPoint.Hash[:])
	binary.BigEndian.PutUint32(k[32:36], d.PreviousOutPoint.Index)
	return k
}

const WalletSyncedDone = math.MaxUint64

type WalletStatus struct {
	WalletID     string
	SyncedHeight uint64
}

func (s *WalletStatus) Ready() bool {
	return s.SyncedHeight == WalletSyncedDone
}

type StoreBucketMeta struct {
	// TxStore
	nsUnmined          mwdb.BucketMeta
	nsTxRecords        mwdb.BucketMeta
	nsBlocks           mwdb.BucketMeta
	nsAddresses        mwdb.BucketMeta
	nsLGHistory        mwdb.BucketMeta
	nsUnminedLGHistory mwdb.BucketMeta

	// UtxoStore
	nsUnspent        mwdb.BucketMeta
	nsUnminedInputs  mwdb.BucketMeta
	nsUnminedCredits mwdb.BucketMeta
	nsMinedBalance   mwdb.BucketMeta
	nsCredits        mwdb.BucketMeta
	nsDebits         mwdb.BucketMeta

	// SyncStore
	nsSyncBucketName mwdb.BucketMeta
	nsWalletStatus   mwdb.BucketMeta
}

func (s *StoreBucketMeta) CheckInit() error {
	if s.nsUnmined == nil {
		return errors.New("StoreBucketMeta.nsUnmined not initialized")
	}
	if s.nsTxRecords == nil {
		return errors.New("StoreBucketMeta.nsTxRecords not initialized")
	}
	if s.nsBlocks == nil {
		return errors.New("StoreBucketMeta.nsBlocks not initialized")
	}
	if s.nsAddresses == nil {
		return errors.New("StoreBucketMeta.nsAddresses not initialized")
	}
	if s.nsLGHistory == nil {
		return errors.New("StoreBucketMeta.nsLGHistory not initialized")
	}
	if s.nsUnminedLGHistory == nil {
		return errors.New("StoreBucketMeta.nsUnminedLGHistory not initialized")
	}
	if s.nsUnspent == nil {
		return errors.New("StoreBucketMeta.nsUnspent not initialized")
	}
	if s.nsUnminedInputs == nil {
		return errors.New("StoreBucketMeta.nsUnminedInputs not initialized")
	}
	if s.nsUnminedCredits == nil {
		return errors.New("StoreBucketMeta.nsUnminedCredits not initialized")
	}
	if s.nsMinedBalance == nil {
		return errors.New("StoreBucketMeta.nsMinedBalance not initialized")
	}
	if s.nsCredits == nil {
		return errors.New("StoreBucketMeta.nsCredits not initialized")
	}
	if s.nsDebits == nil {
		return errors.New("StoreBucketMeta.nsDebits not initialized")
	}
	if s.nsSyncBucketName == nil {
		return errors.New("StoreBucketMeta.nsSyncBucketName not initialized")
	}
	if s.nsWalletStatus == nil {
		return errors.New("StoreBucketMeta.nsWalletStatus not initialized")
	}
	return nil
}
