package cmd

import (
	rpcprotobuf "massnet.org/mass-wallet/api/proto"
	"massnet.org/mass-wallet/logging"

	"github.com/spf13/cobra"
)

// getClientStatusCmd represents the getClientStatus command
var getClientStatusCmd = &cobra.Command{
	Use:   "get-client-status",
	Short: "Get client status",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		logging.CPrint(logging.INFO, "get-client-status called", logging.LogFormat{})

		resp := &rpcprotobuf.GetClientStatusResponse{}
		ClientCall("/v1/client/status", GET, nil, resp)
		printJSON(resp)
	},
}
