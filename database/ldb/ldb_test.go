package ldb_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/massnetorg/MassNet-wallet/database"
	"github.com/massnetorg/MassNet-wallet/massutil"
)

const (
	blockDbNamePrefix = "blocks"
	dbtype            = "leveldb"
)

func blockDbPath(dbType, absolutePath string) string {
	// The database name is based on the database type.
	dbName := blockDbNamePrefix + "_" + dbType
	if dbType == "sqlite" {
		dbName = dbName + ".db"
	}

	dbPath := filepath.Join(absolutePath, dbName)
	return dbPath
}

func GetDb(absolutePath string) (database.Db, error) {
	dbPath := blockDbPath(dbtype, absolutePath)
	fmt.Println("dbPath:", dbPath)
	db, err := database.OpenDB(dbtype, dbPath)
	if err != nil {
		fmt.Println("opendb error: ", err)
		return nil, err
	}
	return db, nil
}

func GetBLock(blkHeight int32, absolutePath string) (*massutil.Block, error) {
	db, err := GetDb(absolutePath)
	if err != nil {
		fmt.Println("GetDb()-err:", err)
		return nil, err
	}
	defer db.Close()
	blkSha, err := db.FetchBlockShaByHeight(blkHeight)
	if err != nil {
		fmt.Println("GetBlockSha()-err:", err)
		return nil, err
	}
	block, err := db.FetchBlockBySha(blkSha)
	if err != nil {
		fmt.Println("GetBlock()-err:", err)
		return nil, err
	}
	fmt.Printf("blockHeight: %d\nblockHash: %s\nprevHash: %s\n",
		block.Height(), block.Hash().String(), block.MsgBlock().Header.Previous.String())
	return block, nil
}

func TestLDb_GetBlock(t *testing.T) {
	path := "../../bin/chain"
	for i := 5000; i < 5500; i++ {
		GetBLock(int32(i), path)
		fmt.Println()
	}
}
