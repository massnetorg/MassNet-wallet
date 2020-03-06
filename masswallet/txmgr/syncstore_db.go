package txmgr

import (
	"encoding/binary"
	"fmt"
	"time"

	mwdb "massnet.org/mass-wallet/masswallet/db"
)

func fetchSyncedBlock(bucket mwdb.Bucket, height uint64) (*BlockMeta, error) {

	k := make([]byte, 8)
	binary.BigEndian.PutUint64(k, height)
	v, err := bucket.Get(k)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	if len(v) != 36 {
		return nil, fmt.Errorf("couldn't get BlockMeta from database for height %d", height)
	}

	block := &BlockMeta{}
	block.Height = height
	copy(block.Hash[:], v[0:32])
	block.Timestamp = time.Unix(
		int64(binary.BigEndian.Uint32(v[32:])), 0,
	)
	return block, nil
}

func fetchSyncedTo(bucket mwdb.Bucket) (*BlockMeta, error) {
	v, err := bucket.Get([]byte(syncedToName))
	if err != nil {
		return nil, err
	}
	if v == nil { // db init
		return nil, nil
	}
	height := binary.BigEndian.Uint64(v[0:8])
	return fetchSyncedBlock(bucket, height)
}

func putSyncedBucket(bucket mwdb.Bucket, bs *BlockMeta) error {
	k := make([]byte, 8)
	binary.BigEndian.PutUint64(k, bs.Height)

	v := make([]byte, 36)
	copy(v, bs.Hash[0:32])
	binary.BigEndian.PutUint32(v[32:], uint32(bs.Timestamp.Unix()))

	err := bucket.Put(k, v)
	if err != nil {
		return fmt.Errorf("failed to store synced block info %v: %v", bs.Hash.String(), err)
	}
	return nil
}

func resetSyncedTo(bucket mwdb.Bucket, height uint64) error {

	v, err := bucket.Get([]byte(syncedToName))
	if err != nil {
		return err
	}
	if len(v) != 8 {
		return fmt.Errorf("incorrect syncedTo information")
	}
	cur := binary.BigEndian.Uint64(v[0:8])

	k := make([]byte, 8)
	for ; cur > height; cur-- {
		binary.BigEndian.PutUint64(k, cur)
		err = bucket.Delete(k)
		if err != nil {
			return err
		}
	}

	binary.BigEndian.PutUint64(k, cur)
	return bucket.Put([]byte(syncedToName), k)
}

func putSyncedTo(bucket mwdb.Bucket, bs *BlockMeta) error {
	if bs.Height > 0 {
		b, err := fetchSyncedBlock(bucket, bs.Height-1)
		if err != nil {
			return fmt.Errorf("failed to fetch synced block info %v: %v", bs.Hash.String(), err)
		}
		if b == nil {
			return fmt.Errorf("syncedTo is too greater than last synced block")
		}
	}

	b, err := fetchSyncedBlock(bucket, bs.Height+1)
	if err != nil {
		return fmt.Errorf("failed to fetch synced block info %v: %v", bs.Hash.String(), err)
	}
	if b != nil {
		return fmt.Errorf("syncedTo is smaller than last synced block:%d %d", b.Height, bs.Height)
	}

	err = putSyncedBucket(bucket, bs)
	if err != nil {
		return err
	}

	v := make([]byte, 8)
	binary.BigEndian.PutUint64(v, bs.Height)

	err = bucket.Put([]byte(syncedToName), v)
	if err != nil {
		return fmt.Errorf("failed to store sync information %v: %v", bs.Hash, err)
	}
	return nil
}

func putWalletStatus(bucket mwdb.Bucket, ws *WalletStatus) error {
	if len(ws.WalletID) != 42 {
		return fmt.Errorf("putWalletStatus expect 42 bytes key(acutal %d)", len(ws.WalletID))
	}
	v := make([]byte, 8)
	binary.BigEndian.PutUint64(v, ws.SyncedHeight)
	return bucket.Put([]byte(ws.WalletID), v)
}

func readWalletStatus(k, v []byte, ws *WalletStatus) error {
	if len(k) != 42 {
		return fmt.Errorf("readWalletStatus expect 42 bytes key(acutal %d)", len(k))
	}
	if len(v) != 8 {
		return fmt.Errorf("readWalletStatus expect 8 bytes value(acutal %d)", len(v))
	}
	ws.WalletID = string(k)
	ws.SyncedHeight = binary.BigEndian.Uint64(v[0:8])
	return nil
}
