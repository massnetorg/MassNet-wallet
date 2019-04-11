package ldb

import "testing"

const (
	dbtype = "leveldb"
)

func TestLevelDb_DropAfterBlockBySha(t *testing.T) {
	db, tearDown, err := GetDb("DbTestDropAfterBlockBySha")
	if err != nil {
		t.Errorf("get Db error%v", err)
	}
	defer tearDown()
	sha, err := db.FetchBlockShaByHeight(2)
	if err != nil {
		t.Errorf("get newestSha error: %v", err)
	}
	err = db.DropAfterBlockBySha(sha)
	if err != nil {
		t.Errorf("DropAfterBlockBySha error: %v", err)
	}
	_, height, err := db.NewestSha()
	if err != nil {
		t.Errorf("get NewestSha error: %v", err)
	}
	if height != 2 {
		if err != nil {
			t.Errorf("DropAfterBlockBySha failed: %v", err)
		}
	}
}
