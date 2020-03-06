package txmgr

import (
	"errors"

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
	return putWalletStatus(nsWalletStatus, ws)
}

func (s *SyncStore) DeleteWalletStatus(tx mwdb.DBTransaction, walletId string) error {
	nsWalletStatus := tx.FetchBucket(s.bucketMeta.nsWalletStatus)
	return deleteByKey(nsWalletStatus, []byte(walletId))
}
