package cmd

import (
	"encoding/json"

	"github.com/spf13/cobra"
	pb "massnet.org/mass-wallet/api/proto"
	"massnet.org/mass-wallet/logging"
)

var createBindingTransactionCmd = &cobra.Command{
	Use:   "createbindingtransaction <outputs> [fee=?] [from=?]",
	Short: "Creates a binding transaction.",
	Long: "Creates a binding transaction.\n" +
		"\nArguments:\n" +
		`  <outputs>    an array of {"holder_address":"","binding_address":"","amount":""}` +
		"\n                 holder_address     -   actual address to which pay MASS\n" +
		"                 binding_address   -   poc address\n" +
		"  [fee]        optional, MASS paid to miner, a real with max 8 decimal places\n" +
		"  [from]       optional, the address of current wallet from which all inputs selected. \n" +
		"               If not provided inputs may be selected from any address of current wallet\n",
	Example: `  createbindingtransaction '[{"holder_address":"ms1qq7xrhu6dh6r02ep42p563nmku3d9t8e6mu6yz0h7k9rnc4gr53a7sl7tw3r","binding_address":"18gsEwbYu65Qjwz4dUtKpYqfyYawQF8yga","amount":"1000.001"},...]'` +
		` from=ms1qq0d99znj2pc032frunvme29ypquxprxrrexthv2d9t5v6zgul4a7qapk0jj fee=0.01` +
		"\n  // win\n" +
		`  createbindingtransaction "[{\"holder_address\":\"ms1qq7xrhu6dh6r02ep42p563nmku3d9t8e6mu6yz0h7k9rnc4gr53a7sl7tw3r\",\"binding_address\":\"18gsEwbYu65Qjwz4dUtKpYqfyYawQF8yga\",\"amount\":\"1000.001\"},...]"` +
		` from=ms1qq0d99znj2pc032frunvme29ypquxprxrrexthv2d9t5v6zgul4a7qapk0jj fee=0.01`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.RangeArgs(1, 3)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		for i := 1; i < len(args); i++ {
			key, value, err := parseCommandVar(args[i])
			if err != nil {
				return err
			}
			switch key {
			case "fee":
				fee = value
			case "from":
				from = value
			default:
				return errorUnknownCommandParam(key)
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {

		outs := make([]*pb.CreateBindingTransactionRequest_Output, 0)
		err := json.Unmarshal([]byte(args[0]), &outs)
		if err != nil {
			return err
		}

		logging.VPrint(logging.INFO, "createbindingtransaction called", logging.LogFormat{
			"outputs": outs,
			"from":    from,
			"fee":     fee,
		})

		req := &pb.CreateBindingTransactionRequest{
			Outputs:     outs,
			FromAddress: from,
			Fee:         fee,
		}

		resp := &pb.CreateRawTransactionResponse{}
		return ClientCall("/v1/transactions/binding", POST, req, resp)
	},
}

var getBindingHistoryCmd = &cobra.Command{
	Use:   "listbindingtransactions",
	Short: "Returns all binding transaction of current wallet.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "listbindingtransactions called", EmptyLogFormat)

		resp := &pb.GetBindingHistoryResponse{}
		return ClientCall("/v1/transactions/binding/history", GET, nil, resp)
	},
}

var getAddressBindingCmd = &cobra.Command{
	Use:   "getaddresstotalbinding <poc_address>...",
	Short: "Returns total binding MASS of each PoC address.",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "getaddresstotalbinding called", logging.LogFormat{"addresses": args})
		req := &pb.GetAddressBindingRequest{
			Addresses: args,
		}
		resp := &pb.GetAddressBindingResponse{}
		return ClientCall("/v1/addresses/binding", POST, req, resp)
	},
}
