package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	pb "massnet.org/mass-wallet/api/proto"
	"massnet.org/mass-wallet/logging"

	"github.com/spf13/cobra"
)

// args for Transaction
var (
	locktime        uint64
	fee             string
	from            string
	inputs          []*pb.TransactionInput
	outputs         map[string]string
	signFlags       = "ALL"
	estimateBinding bool
	historyCount    uint32
)

var createRawTransactionCmd = &cobra.Command{
	Use:   "createrawtransaction <inputs> <outputs> [locktime=?]",
	Short: "Creates a raw transaction spending given inputs of current wallet.",
	Long: "Creates a raw transaction spending given inputs of current wallet.\n" +
		"\nArguments:\n" +
		"  <outputs>     a ToAddress-Value map\n" +
		"  <inputs>      an array of TxOut object\n" +
		"  [locktime]    optional, a integer, default 0\n",
	Example: `  createrawtransaction '[{"tx_id": "0234abef", "vout": 0},{"tx_id": "abdafe232", "vout": 1}, ...]' '{"ms1qq7xrhu6dh6r02ep42p563nmku3d9t8e6mu6yz0h7k9rnc4gr53a7sl7tw3r": "1.2",...}'` +
		"\n  // win\n" +
		`  createrawtransaction "[{\"tx_id\": \"0234abef\", \"vout\": 0},{\"tx_id\": \"abdafe232\", \"vout\": 1}, ...]" "{\"ms1qq7xrhu6dh6r02ep42p563nmku3d9t8e6mu6yz0h7k9rnc4gr53a7sl7tw3r\": \"1.2\",...}"`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.RangeArgs(2, 3)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		inputs = make([]*pb.TransactionInput, 0)
		if err := json.Unmarshal([]byte(args[0]), &inputs); err != nil {
			return err
		}

		outputs = make(map[string]string)
		if err := json.Unmarshal([]byte(args[1]), &outputs); err != nil {
			return err
		}

		for i := 2; i < len(args); i++ {
			key, value, err := parseCommandVar(args[i])
			if err != nil {
				return err
			}
			switch key {
			case "locktime":
				locktime, err = strconv.ParseUint(value, 10, 64)
				if err != nil {
					return err
				}
			default:
				return errorUnknownCommandParam(key)
			}
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "createrawtransaction called", logging.LogFormat{
			"inputs":   inputs,
			"outputs":  outputs,
			"locktime": locktime,
		})

		req := &pb.CreateRawTransactionRequest{
			Inputs:   inputs,
			Amounts:  outputs,
			LockTime: locktime,
		}

		resp := &pb.CreateRawTransactionResponse{}
		return ClientCall("/v1/transactions/create", POST, req, resp)
	},
}

var autoCreateRawTransactionCmd = &cobra.Command{
	Use:   "autocreaterawtransaction <outputs> [locktime=?] [fee=?] [from=?]",
	Short: "Creates a raw transaction spending randomly selected inputs of current wallet.",
	Long: "Creates a raw transaction spending randomly selected inputs of current wallet.\n" +
		"\nArguments:\n" +
		"  <outputs>    a ToAddress-Amount map\n" +
		"  [fee]        optional, a real with max 8 decimal places\n" +
		"  [locktime]   optional, a integer, default 0\n" +
		"  [from]       optional, a standard mass address, if not provided the inputs may be selected from any address of current wallet\n",
	Example: `  autocreaterawtransaction '{"ms1qq7xrhu6dh6r02ep42p563nmku3d9t8e6mu6yz0h7k9rnc4gr53a7sl7tw3r": "10.05", ...}' fee=3.3 from=ms1qq3lkfcujch750lt0vc2ygfgfhs3eewntfyhfy86qyclnjansksnhsxymr8g` +
		"\n  // win\n" +
		`  autocreaterawtransaction "{\"ms1qq7xrhu6dh6r02ep42p563nmku3d9t8e6mu6yz0h7k9rnc4gr53a7sl7tw3r\": \"10.05\", ...}" fee=3.3 from=ms1qq3lkfcujch750lt0vc2ygfgfhs3eewntfyhfy86qyclnjansksnhsxymr8g`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.RangeArgs(1, 4)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}

		outputs = make(map[string]string)
		err := json.Unmarshal([]byte(args[0]), &outputs)
		if err != nil {
			return err
		}

		for i := 1; i < len(args); i++ {
			key, value, err := parseCommandVar(args[i])
			if err != nil {
				return err
			}
			switch key {
			case "locktime":
				locktime, err = strconv.ParseUint(value, 10, 64)
				if err != nil {
					return err
				}
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
		logging.VPrint(logging.INFO, "autocreaterawtransaction called", logging.LogFormat{
			"outputs":  outputs,
			"locktime": locktime,
			"fee":      fee,
			"from":     from,
		})

		req := &pb.AutoCreateTransactionRequest{
			Amounts:     outputs,
			LockTime:    locktime,
			Fee:         fee,
			FromAddress: from,
		}

		resp := &pb.CreateRawTransactionResponse{}
		return ClientCall("/v1/transactions/create/auto", POST, req, resp)
	},
}

var signRawTransactionCmd = &cobra.Command{
	Use:   "signrawtransaction <hexstring> <passphrase> [mode=?]",
	Short: "Adds signatures to a raw transaction and returns the resulting raw transaction.",
	Long: "Adds signatures to a raw transaction and returns the resulting raw transaction.\n" +
		"\nArguments:\n" +
		"  <hexstring>   signed, serialized, hex-encoded transaction\n" +
		"  <passphrase>  \n" +
		"  [mode]        Optional, allowed modes(normally ALL) are:\n" +
		"            ALL:                 sign for all inputs and outputs (default)\n" +
		"            NONE:                sign for all inputs\n" +
		"            SINGLE:              sign for all inputs and one specified output\n" +
		"            ALL|ANYONECANPAY:    sign for one specified input and all outputs\n" +
		"            NONE|ANYONECANPAY:   sign for one specified input\n" +
		"            SINGLE|ANYONECANPAY: sign for one specified input and corresponding output",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.RangeArgs(2, 3)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		for i := 2; i < len(args); i++ {
			key, value, err := parseCommandVar(args[i])
			if err != nil {
				return err
			}
			switch key {
			case "mode":
				upper := strings.ToUpper(value)
				switch upper {
				case "ALL", "NONE", "SINGLE", "ALL|ANYONECANPAY", "NONE|ANYONECANPAY", "SINGLE|ANYONECANPAY":
					signFlags = upper
				default:
					return fmt.Errorf("invalid mode: %s", value)
				}
			default:
				return errorUnknownCommandParam(key)
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "signrawtransaction called", logging.LogFormat{"hex": args[0], "mode": signFlags})

		req := &pb.SignRawTransactionRequest{
			RawTx:      args[0],
			Passphrase: args[1],
			Flags:      signFlags,
		}
		resp := &pb.SignRawTransactionResponse{}
		return ClientCall("/v1/transactions/sign", POST, req, resp)
	},
}

var getTransactionFeeCmd = &cobra.Command{
	Use:   "gettransactionfee <outputs> <inputs> [binding=true] [locktime=?]",
	Short: "Estimates transaction fee.",
	Long: "Estimates transaction fee.\n" +
		"\nArguments:\n" +
		"  <outputs>    a ToAddress-Value map\n" +
		"  <inputs>     an array of TxOut\n" +
		"  [binding]   optional, indicates whether this transaction contains binding\n" +
		"  [locktime]   optional, a non negative integer, default 0\n",
	Example: `  gettransactionfee '{"ms1qq7xrhu6dh6r02ep42p563nmku3d9t8e6mu6yz0h7k9rnc4gr53a7sl7tw3r": "100.5", ...}' '[{"tx_id":"12324abef","vout":0}, ...]'` +
		"\n  // win\n" +
		`  gettransactionfee "{\"ms1qq7xrhu6dh6r02ep42p563nmku3d9t8e6mu6yz0h7k9rnc4gr53a7sl7tw3r\": \"100.5\", ...}" "[]" locktime=100`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.RangeArgs(2, 4)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}

		outputs = make(map[string]string)
		err := json.Unmarshal([]byte(args[0]), &outputs)
		if err != nil {
			return err
		}

		inputs = make([]*pb.TransactionInput, 0)
		if err := json.Unmarshal([]byte(args[1]), &inputs); err != nil {
			return err
		}

		for i := 2; i < len(args); i++ {
			key, value, err := parseCommandVar(args[i])
			if err != nil {
				return err
			}
			switch key {
			case "binding":
				estimateBinding, err = strconv.ParseBool(value)
				if err != nil {
					return err
				}
			case "locktime":
				locktime, err = strconv.ParseUint(value, 10, 64)
				if err != nil {
					return err
				}
			default:
				return errorUnknownCommandParam(key)
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "gettransactionfee called", logging.LogFormat{
			"outputs":  outputs,
			"intputs":  inputs,
			"locktime": locktime,
			"binding":  estimateBinding,
		})

		req := &pb.GetTransactionFeeRequest{
			Amounts:    outputs,
			Inputs:     inputs,
			LockTime:   locktime,
			HasBinding: estimateBinding,
		}

		resp := &pb.GetTransactionFeeResponse{}
		return ClientCall("v1/transactions/fee", POST, req, resp)
	},
}

var sendRawTransactionCmd = &cobra.Command{
	Use:   "sendrawtransaction <hexstring>",
	Short: "Submits raw transaction (signed, serialized, hex-encoded) to local node and network.",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "sendrawtransaction called", logging.LogFormat{"hex": args[0]})

		req := &pb.SendRawTransactionRequest{
			Hex: args[0],
		}
		resp := &pb.SendRawTransactionResponse{}
		return ClientCall("/v1/transactions/send", POST, req, resp)
	},
}

var getRawTransactionCmd = &cobra.Command{
	Use:   "getrawtransaction <txid>",
	Short: "Returns raw transaction representation for given transaction id.",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "getrawtransaction called", logging.LogFormat{"txid": args[0]})

		resp := &pb.GetRawTransactionResponse{}
		return ClientCall(fmt.Sprintf("/v1/transactions/%s/details", args[0]), GET, nil, resp)
	},
}

var getTxStatusCmd = &cobra.Command{
	Use:   "gettransactionstatus <txid>",
	Short: "Returns transaction status for given transaction id.",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "gettransactionstatus called", logging.LogFormat{"txid": args[0]})

		resp := &pb.GetTxStatusResponse{}
		return ClientCall(fmt.Sprintf("/v1/transactions/%s/status", args[0]), GET, nil, resp)
	},
}

var listTrasactionsCmd = &cobra.Command{
	Use:   "listtransactions [count=?] [address=?]",
	Short: "Returns up to N most recent transactions for current wallet.",
	Long: "Returns up to N most recent transactions for current wallet.\n" +
		"\nArguments:\n" +
		"  [count]     optional, up to count most recent transactions, if not provided(or 0) a default value will be used.\n" +
		"  [address]   optional, target address, if not provided it'll return transactions from all address of current wallet.\n",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.RangeArgs(0, 2)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		for i := 0; i < len(args); i++ {
			key, value, err := parseCommandVar(args[i])
			if err != nil {
				return err
			}
			switch key {
			case "count":
				c, err := strconv.Atoi(value)
				if err != nil {
					return err
				}
				historyCount = uint32(c)
			case "address":
				from = value
			default:
				return errorUnknownCommandParam(key)
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "listtransactions called", logging.LogFormat{
			"count":   historyCount,
			"address": from,
		})

		req := &pb.TxHistoryRequest{
			Count:   historyCount,
			Address: from,
		}
		resp := &pb.TxHistoryResponse{}
		return ClientCall("/v1/transactions/history", POST, req, resp)
	},
}
