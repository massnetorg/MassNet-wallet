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

// TestBlockHeader tests the BlockHeader API.
func TestBlockHeader(t *testing.T) {
	var testRound = 100

	for i := 0; i < testRound; i++ {
		header := mockHeader()
		var wBuf bytes.Buffer
		err := header.Serialize(&wBuf, DB)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		newHeader := new(BlockHeader)
		err = newHeader.Deserialize(&wBuf, DB)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		// compare header and newHeader
		if !reflect.DeepEqual(header, newHeader) {
			t.Error("header and newHeader is not equal")
		}
	}
}
