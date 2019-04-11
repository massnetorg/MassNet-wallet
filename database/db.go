// Modified for MassNet
// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package database

import (
	"errors"

	"massnet.org/mass-wallet/config"

	"massnet.org/mass-wallet/wire"

	"massnet.org/mass-wallet/massutil"

	"github.com/btcsuite/golangcrypto/ripemd160"
)

// Errors that the various database functions may return.
var (
	ErrAddrIndexDoesNotExist  = errors.New("address index hasn't been built or is an older version")
	ErrUnsupportedAddressType = errors.New("address type is not supported " +
		"by the address-index")
	ErrPrevShaMissing  = errors.New("previous sha missing from database")
	ErrTxShaMissing    = errors.New("requested transaction does not exist")
	ErrBlockShaMissing = errors.New("requested block does not exist")
	ErrDuplicateSha    = errors.New("duplicate insert attempted")
	ErrDbDoesNotExist  = errors.New("non-existent database")
	ErrDbUnknownType   = errors.New("non-existent database type")
	ErrNotImplemented  = errors.New("method has not yet been implemented")
)

type Db interface {
	// Close cleanly shuts down the database and syncs all data.
	Close() (err error)

	// DropAfterBlockBySha will remove any blocks from the database after
	// the given block.  It terminates any existing transaction and performs
	// its operations in an atomic transaction which is commited before
	// the function returns.
	DropAfterBlockBySha(*wire.Hash) (err error)

	// ExistsSha returns whether or not the given block hash is present in
	// the database.
	ExistsSha(sha *wire.Hash) (exists bool, err error)

	// FetchBlockBySha returns a massutil Block.  The implementation may
	// cache the underlying data if desired.
	FetchBlockBySha(sha *wire.Hash) (blk *massutil.Block, err error)

	// FetchBlockHeightBySha returns the block height for the given hash.
	FetchBlockHeightBySha(sha *wire.Hash) (height int32, err error)

	// FetchBlockHeaderBySha returns a wire.BlockHeader for the given
	// sha.  The implementation may cache the underlying data if desired.
	FetchBlockHeaderBySha(sha *wire.Hash) (bh *wire.BlockHeader, err error)

	// FetchBlockShaByHeight returns a block hash based on its height in the
	// block chain.
	FetchBlockShaByHeight(height int32) (sha *wire.Hash, err error)

	// FetchHeightRange looks up a range of blocks by the start and ending
	// heights.  Fetch is inclusive of the start height and exclusive of the
	// ending height. To fetch all hashes from the start height until no
	// more are present, use the special id `AllShas'.
	FetchHeightRange(startHeight, endHeight int32) (rshalist []wire.Hash, err error)

	// ExistsTxSha returns whether or not the given tx hash is present in
	// the database
	ExistsTxSha(sha *wire.Hash) (exists bool, err error)

	// FetchTxBySha returns some data for the given transaction hash. The
	// implementation may cache the underlying data if desired.
	FetchTxBySha(txsha *wire.Hash) ([]*TxListReply, error)

	// FetchTxByShaList returns a TxListReply given an array of transaction
	// hashes.  The implementation may cache the underlying data if desired.
	// This differs from FetchUnSpentTxByShaList in that it will return
	// the most recent known Tx, if it is fully spent or not.
	//
	// NOTE: This function does not return an error directly since it MUST
	// return at least one TxListReply instance for each requested
	// transaction.  Each TxListReply instance then contains an Err field
	// which can be used to detect errors.
	FetchTxByShaList(txShaList []*wire.Hash) []*TxListReply

	// FetchUnSpentTxByShaList returns a TxListReply given an array of
	// transaction hashes.  The implementation may cache the underlying
	// data if desired. Fully spent transactions will not normally not
	// be returned in this operation.
	//
	// NOTE: This function does not return an error directly since it MUST
	// return at least one TxListReply instance for each requested
	// transaction.  Each TxListReply instance then contains an Err field
	// which can be used to detect errors.
	FetchUnSpentTxByShaList(txShaList []*wire.Hash) []*TxListReply

	// InsertBlock inserts raw block and transaction data from a block
	// into the database.  The first block inserted into the database
	// will be treated as the genesis block.  Every subsequent block insert
	// requires the referenced parent block to already exist.
	InsertBlock(block *massutil.Block) (height int32, err error)

	// NewestSha returns the hash and block height of the most recent (end)
	// block of the block chain.  It will return the zero hash, -1 for
	// the block height, and no error (nil) if there are not any blocks in
	// the database yet.
	NewestSha() (sha *wire.Hash, height int32, err error)

	// FetchAddrIndexTip returns the hash and block height of the most recent
	// block which has had its address index populated. It will return
	// ErrAddrIndexDoesNotExist along with a zero hash, and -1 if the
	// addrindex hasn't yet been built up.
	FetchAddrIndexTip() (sha *wire.Hash, height int32, err error)

	// UpdateAddrIndexForBlock updates the stored addrindex with passed
	// index information for a particular block height. Additionally, it
	// will update the stored meta-data related to the curent tip of the
	// addr index. These two operations are performed in an atomic
	// transaction which is commited before the function returns.
	// Addresses are indexed by the raw bytes of their base58 decoded
	// hash160.
	UpdateAddrIndexForBlock(blkSha *wire.Hash, height int32,
		addrIndex BlockAddrIndex) error

	// FetchTxsForAddr looks up and returns all transactions which either
	// spend a previously created output of the passed address, or create
	// a new output locked to the passed address. The, `limit` parameter
	// should be the max number of transactions to be returned.
	// Additionally, if the caller wishes to skip forward in the results
	// some amount, the 'seek' represents how many results to skip.
	// The transactions are returned in chronological order by block height
	// from old to new, or from new to old if `reverse` is set.
	// NOTE: Values for both `seek` and `limit` MUST be positive.
	// It will return the array of fetched transactions, along with the amount
	// of transactions that were actually skipped.
	FetchTxsForAddr(addr massutil.Address, skip int, limit int, reverse bool) ([]*TxListReply, int, error)

	// FetchUtxosForAddrs looks up and returns all unspent transaction outpoints
	// locked to the passed addresses.
	FetchUtxosForAddrs(addrs []massutil.Address, chainParams *config.Params) ([][]*UtxoListReply, error)

	// DeleteAddrIndex deletes the entire addrindex stored within the DB.
	DeleteAddrIndex() error

	// RollbackClose discards the recent database changes to the previously
	// saved data at last Sync and closes the database.
	RollbackClose() (err error)

	// Sync verifies that the database is coherent on disk and no
	// outstanding transactions are in flight.
	Sync() (err error)

	// address_package related

	//
	InsertRootKey(rootKeyEnc []byte, rootPkExStr string, defaultPasswordUsed bool) error
	////
	//InsertRootKey(rootKeyEnc []byte, rootPkExStr string) error

	// return all rootPubKeyString
	FetchAllRootPkStr() ([]string, error)

	//return rootPrivKeyEnc and defaultPasswordUsed
	FetchRootKey(rootPkExStr string) ([]byte, bool, error)

	//// return rootkeyEnc
	//FetchRootKey(rootPkExStr string) ([]byte, error)

	//create childKeyNumPrefix+[]byte(rootPkExStr)->0
	InitChildKeyNum(rootPkExStr string) error

	//
	FetchChildKeyNum(rootPkExStr string) (int, error)

	//ChildKeyNum+1
	UpdateChildKeyNum(rootPkExStr string) (int, error)

	//
	InsertWitnessAddr(witAddr string, redeemScript []byte) error

	//
	FetchWitnessAddrToRedeem() (map[string][]byte, error)

	// import
	ImportChildKeyNum(rootPkExStr string, num string) error
}

// DriverDB defines a structure for backend drivers to use when they registered
// themselves as a backend which implements the Db interface.
type DriverDB struct {
	DbType   string
	CreateDB func(args ...interface{}) (pbdb Db, err error)
	OpenDB   func(args ...interface{}) (pbdb Db, err error)
}

// TxListReply is used to return individual transaction information when
// data about multiple transactions is requested in a single call.
type TxListReply struct {
	Sha     *wire.Hash
	Tx      *wire.MsgTx
	BlkSha  *wire.Hash
	Height  int32
	TxSpent []bool
	Err     error
}

//
type UtxoListReply struct {
	TxSha    *wire.Hash
	Height   int32
	Coinbase bool
	Index    uint32
	Value    massutil.Amount
}

// AddrIndexKeySize is the number of bytes used by keys into the BlockAddrIndex.
const AddrIndexKeySize = ripemd160.Size

type AddrIndexOutPoint struct {
	TxLoc *wire.TxLoc
	Index uint32
}

// BlockAddrIndex represents the indexing structure for addresses.
// It maps a hash160 to a list of transaction locations within a block that
// either pays to or spends from the passed UTXO for the hash160.
type BlockAddrIndex map[[AddrIndexKeySize]byte][]*AddrIndexOutPoint

// driverList holds all of the registered database backends.
var driverList []DriverDB

// AddDBDriver adds a back end database driver to available interfaces.
func AddDBDriver(instance DriverDB) {
	for _, drv := range driverList {
		if drv.DbType == instance.DbType {
			return
		}
	}
	driverList = append(driverList, instance)
}

// CreateDB intializes and opens a database.
func CreateDB(dbtype string, args ...interface{}) (pbdb Db, err error) {
	for _, drv := range driverList {
		if drv.DbType == dbtype {
			return drv.CreateDB(args...)
		}
	}
	return nil, ErrDbUnknownType
}

// OpenDB opens an existing database.
func OpenDB(dbtype string, args ...interface{}) (pbdb Db, err error) {
	for _, drv := range driverList {
		if drv.DbType == dbtype {
			return drv.OpenDB(args...)
		}
	}
	return nil, ErrDbUnknownType
}

// SupportedDBs returns a slice of strings that represent the database drivers
// that have been registered and are therefore supported.
func SupportedDBs() []string {
	var supportedDBs []string
	for _, drv := range driverList {
		supportedDBs = append(supportedDBs, drv.DbType)
	}
	return supportedDBs
}
