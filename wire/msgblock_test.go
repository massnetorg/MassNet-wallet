// Modified for MassNet
// Copyright (c) 2013-2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"bytes"
	"reflect"
	"testing"
)

// TestBlock tests the MsgBlock API.
func TestBlock(t *testing.T) {
	var testRound = 100

	for i := 1; i < testRound; i += 20 {
		blk := mockBlock(2000 / i)
		var wBuf bytes.Buffer
		err := blk.Serialize(&wBuf, DB)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		newBlk := new(MsgBlock)
		err = newBlk.Deserialize(&wBuf, DB)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		// compare blk and newBlk
		if !reflect.DeepEqual(blk, newBlk) {
			t.Error("blk and newBlk is not equal")
		}
	}
}
