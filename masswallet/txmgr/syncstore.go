package txmgr

import (
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/massnetorg/mass-core/logging"
	"massnet.org/mass-wallet/config"
	mwdb "massnet.org/mass-wallet/masswallet/db"
)

// SyncStore ...
type SyncStore struct {
	chainParams *config.Params
	bucketMeta  *StoreBucketMeta
}

// NewSyncStore ...
func NewSyncStore(store mwdb.Bucket, bucketMeta *StoreBucketMeta, chainParams *config.Params) (*SyncStore, error) {
	s := &SyncStore{
		bucketMeta:  bucketMeta,
		chainParams: chainParams,
	}

	//syncBucketName
	bucket, err := mwdb.GetOrCreateBucket(store, syncBucketName)
	if err != nil {
		return nil, err
	}
	s.bucketMeta.nsSyncBucketName = bucket.GetBucketMeta()

	syncedTo, err := fetchSyncedTo(bucket)
	if err != nil {
		return nil, err
	}
	if syncedTo == nil {
		syncedTo = &BlockMeta{
			Height:    s.chainParams.GenesisBlock.Header.Height,
			Hash:      *s.chainParams.GenesisHash,
			Timestamp: s.chainParams.GenesisBlock.Header.Timestamp,
		}
		err = putSyncedTo(bucket, syncedTo)
		if err != nil {
			return nil, err
		}
	}

	//bucketWalletStatus
	bucket, err = mwdb.GetOrCreateBucket(store, bucketWalletStatus)
	if err != nil {
		return nil, err
	}
	s.bucketMeta.nsWalletStatus = bucket.GetBucketMeta()

	return s, nil
}

// BlockHash ...
func (s *SyncStore) SyncedBlock(tx mwdb.DBTransaction, height uint64) (*BlockMeta, error) {
	nsSyncBucketName := tx.FetchBucket(s.bucketMeta.nsSyncBucketName)
	return fetchSyncedBlock(nsSyncBucketName, height)
}

// SetSyncedTo ...
func (s *SyncStore) SetSyncedTo(tx mwdb.DBTransaction, bs *BlockMeta) error {
	return putSyncedTo(tx.FetchBucket(s.bucketMeta.nsSyncBucketName), bs)
}

func (s *SyncStore) ResetSyncedTo(tx mwdb.DBTransaction, height uint64) error {
	return resetSyncedTo(tx.FetchBucket(s.bucketMeta.nsSyncBucketName), height)
}

// SyncedTo ...
func (s *SyncStore) SyncedTo(tx mwdb.ReadTransaction) (bm *BlockMeta, err error) {
	nsSyncBucketName := tx.FetchBucket(s.bucketMeta.nsSyncBucketName)
	bm, err = fetchSyncedTo(nsSyncBucketName)
	if err != nil {
		return
	}
	if bm == nil {
		return nil, errors.New("fetchSyncedTo return nil")
	}
	return
}

func (s *SyncStore) GetAllWalletStatus(tx mwdb.ReadTransaction) ([]*WalletStatus, error) {
	nsWalletStatus := tx.FetchBucket(s.bucketMeta.nsWalletStatus)
	entries, err := nsWalletStatus.GetByPrefix(nil)
	if err != nil {
		return nil, err
	}
	ret := make([]*WalletStatus, 0)
	for _, entry := range entries {
		var status WalletStatus
		if err = readWalletStatus(entry.Key, entry.Value, &status); err != nil {
			return nil, err
		}
		ret = append(ret, &status)
	}
	return ret, nil
}

func (s *SyncStore) GetWalletStatus(tx mwdb.ReadTransaction, walletId string) (*WalletStatus, error) {
	nsWalletStatus := tx.FetchBucket(s.bucketMeta.nsWalletStatus)
	v, err := existsValue(nsWalletStatus, []byte(walletId))
	if err != nil {
		return nil, err
	}
	var status WalletStatus
	err = readWalletStatus([]byte(walletId), v, &status)
	if err != nil {
		return nil, err
	}
	return &status, nil
}

func (s *SyncStore) PutWalletStatus(tx mwdb.DBTransaction, ws *WalletStatus) error {
	nsWalletStatus := tx.FetchBucket(s.bucketMeta.nsWalletStatus)
	if len(ws.WalletID) != 42 {
		return fmt.Errorf("putWalletStatus expect 42 bytes key(acutal %d)", len(ws.WalletID))
	}
	v := make([]byte, 9)
	binary.BigEndian.PutUint64(v[0:8], ws.SyncedHeight)
	v[8] = ws.Flags
	return putKeyValue(nsWalletStatus, []byte(ws.WalletID), v)
}

func (s *SyncStore) DeleteWalletStatus(tx mwdb.DBTransaction, walletId string) error {
	nsWalletStatus := tx.FetchBucket(s.bucketMeta.nsWalletStatus)
	return deleteKey(nsWalletStatus, []byte(walletId))
}

func (s *SyncStore) MarkDeleteWallet(tx mwdb.DBTransaction, walletId string) error {
	status, err := s.GetWalletStatus(tx, walletId)
	if err != nil {
		return err
	}
	status.Flags |= WalletFlagsRemove
	return s.PutWalletStatus(tx, status)
}

func (s *SyncStore) CountAll(tx mwdb.ReadTransaction) {
	buckets, err := tx.BucketNames()
	logging.CPrint(logging.INFO, "---->buckets",
		logging.LogFormat{
			"buckets": buckets,
			"err":     err,
		})
	nsWalletStatus := tx.FetchBucket(s.bucketMeta.nsWalletStatus)
	printEntryCount("nsWalletStatus", nsWalletStatus)
	nsSyncBucketName := tx.FetchBucket(s.bucketMeta.nsSyncBucketName)
	printEntryCount("nsSyncBucketName", nsSyncBucketName)
	// utxo
	nsDebits := tx.FetchBucket(s.bucketMeta.nsDebits)
	printEntryCount("nsDebits", nsDebits)
	nsCredits := tx.FetchBucket(s.bucketMeta.nsCredits)
	printEntryCount("nsCredits", nsCredits)
	nsMinedBalance := tx.FetchBucket(s.bucketMeta.nsMinedBalance)
	printEntryCount("nsMinedBalance", nsMinedBalance)
	nsUnminedCredits := tx.FetchBucket(s.bucketMeta.nsUnminedCredits)
	printEntryCount("nsUnminedCredits", nsUnminedCredits)
	nsUnminedInputs := tx.FetchBucket(s.bucketMeta.nsUnminedInputs)
	printEntryCount("nsUnminedInputs", nsUnminedInputs)
	nsUnspent := tx.FetchBucket(s.bucketMeta.nsUnspent)
	printEntryCount("nsUnspent", nsUnspent)

	// nsUnminedGameHistory := tx.FetchBucket(s.bucketMeta.nsUnminedGameHistory)
	// printEntryCount("nsUnminedGameHistory", nsUnminedGameHistory)
	// nsGameHistory := tx.FetchBucket(s.bucketMeta.nsGameHistory)
	// printEntryCount("nsGameHistory", nsGameHistory)
	nsAddresses := tx.FetchBucket(s.bucketMeta.nsAddresses)
	printEntryCount("nsAddresses", nsAddresses)
	nsBlocks := tx.FetchBucket(s.bucketMeta.nsBlocks)
	printEntryCount("nsBlocks", nsBlocks)
	nsTxRecords := tx.FetchBucket(s.bucketMeta.nsTxRecords)
	printEntryCount("nsTxRecords", nsTxRecords)
	nsUnmined := tx.FetchBucket(s.bucketMeta.nsUnmined)
	printEntryCount("nsUnmined", nsUnmined)
}

func printEntryCount(name string, bkt mwdb.Bucket) {
	num := 0
	size := 0
	it := bkt.NewIterator(mwdb.BytesPrefix(nil))
	defer it.Release()
	for it.Next() {
		num++
		size += len(it.Key()) + len(it.Value())
	}
	logging.CPrint(logging.INFO, fmt.Sprintf("----%s----", name),
		logging.LogFormat{
			"num":  num,
			"size": size,
		})
}
