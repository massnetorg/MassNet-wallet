package wire

import (
	"bytes"
	"reflect"
	"testing"
)

// TestProposalArea tests the ProposalArea
func TestProposalArea(t *testing.T) {
	var testRound = 100

	for i := 0; i < testRound; i++ {
		pa := mockProposalArea()
		var wBuf bytes.Buffer
		err := pa.Serialize(&wBuf, DB)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}

		newPa := new(ProposalArea)
		err = newPa.Deserialize(&wBuf, DB)
		if err != nil {
			t.Error(err, pa.PunishmentArea[0], pa.PunishmentArea[0].version, pa.PunishmentArea[0].proposalType)
			t.FailNow()
		}

		// compare pa and newPa
		if !reflect.DeepEqual(pa, newPa) {
			t.Error("pa and newPa is not equal")
		}
	}
}
