package cmd_test

import (
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	pb "massnet.org/mass-wallet/api/proto"
)

func TestF(t *testing.T) {
	s := `{"amounts":{"ms1qq5wjn82ffjmaaf5gfw02whhjsyund6l4wqmuvgnet829yquy7dgms6yc8pf":"1"},"change_address":"ms1qq0h5g69xyfxqlyrszuqgpkywlzs640waazr3mz66zvxlr8qmwpm5sf4l9z8","fee":"0.0001","from_address":"ms1qq0h5g69xyfxqlyrszuqgpkywlzs640waazr3mz66zvxlr8qmwpm5sf4l9z8"}`
	// s := `{"amounts":{"ms1qq5wjn82ffjmaaf5gfw02whhjsyund6l4wqmuvgnet829yquy7dgms6yc8pf":"1"},"change_address":"ms1qq0h5g69xyfxqlyrszuqgpkywlzs640waazr3mz66zvxlr8qmwpm5sf4l9z8","fee":"0.0001"}`
	req := &pb.AutoCreateTransactionRequest{}
	err := json.Unmarshal([]byte(s), req)
	require.NoError(t, err)

}

func TestNo(t *testing.T) {
	s1 := "ms1qq5wjn82ffjmaaf5gfw02whhjsyund6l4wqmuvgnet829yquy7dgms6yc8pf"
	// mc, err := regexp.MatchString("^[0-9A-Za-z]{63}$", s1)
	// mc, err := regexp.MatchString("^[1-9A-HJ-NP-Za-km-z]{63}$", s1)
	mc, err := regexp.MatchString("^ms1qq[qpzry9x8gf2tvdw0s3jn54khce6mua7l]{58}$", s1)
	fmt.Println(mc, err)
}
