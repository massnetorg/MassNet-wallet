package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"massnet.org/mass-wallet/api/proto"
	"massnet.org/mass-wallet/logging"
)

// sendTransactionCmd represents the sendTransaction command
var sendTransactionCmd = &cobra.Command{
	Use:   "send-tx <hex>",
	Short: "Send transaction in hex format",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			logging.CPrint(logging.ERROR, "wrong argument count", logging.LogFormat{"count": len(args)})
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		hex := args[0]
		logging.CPrint(logging.INFO, "send-tx called", logging.LogFormat{"hex": hex})

		req := &rpcprotobuf.SendRawTransactionRequest{
			HexTx: hex,
		}
		resp := &rpcprotobuf.SendRawTransactionResponse{}
		ClientCall("/v1/transactions/sending", POST, req, resp)
		printJSON(resp)
	},
}

// args for getTransactionCmd
var (
	getTransactionArgVerbose bool
)

// getTransactionCmd represents the getTransaction command
var getTransactionCmd = &cobra.Command{
	Use:   "get-tx <txid>",
	Short: "Get transaction in details",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			logging.CPrint(logging.ERROR, "wrong argument count", logging.LogFormat{"count": len(args)})
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		txid := args[0]
		logging.CPrint(logging.INFO, "get-tx called", logging.LogFormat{"txid": txid})

		resp := &rpcprotobuf.GetRawTransactionResponse{}
		ClientCall(fmt.Sprintf("/v1/transactions/%s/details", txid), GET, nil, resp)
		printJSON(resp)
	},
}

// getTransactionStatusCmd represents the getTransactionStatus command
var getTransactionStatusCmd = &cobra.Command{
	Use:   "get-tx-status <txid>",
	Short: "Get transaction status",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			logging.CPrint(logging.ERROR, "wrong argument count", logging.LogFormat{"count": len(args)})
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		txid := args[0]
		logging.CPrint(logging.INFO, "get-tx-status called", logging.LogFormat{"txid": txid})

		resp := &rpcprotobuf.GetTxStatusResponse{}
		ClientCall(fmt.Sprintf("/v1/transactions/%s/status", txid), GET, nil, resp)
		printJSON(resp)
	},
}

func init() {
	getTransactionCmd.PersistentFlags().BoolVar(&getTransactionArgVerbose, "verbose", false, "show verbose information")
}
