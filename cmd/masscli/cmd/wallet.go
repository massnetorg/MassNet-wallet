package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"massnet.org/mass-wallet/api/proto"
	"massnet.org/mass-wallet/logging"

	"github.com/spf13/cobra"
)

// args for createAddressCmd
var (
	createAddressArgPubNumber   int32
	createAddressArgSigRequired int32
)

// createAddressCmd represents the createAddress command
var createAddressCmd = &cobra.Command{
	Use:   "create-address <sig_required> <pub_number>",
	Short: "Create an address by specifying multi-signature address arguments",
	Long: "Create an address by specifying sigRequired/pubKey number in multi-signature:\n" +
		"Normally we use 1-1 address, while you can create multi-sig addresses with no more than 20 pubKeys.",
	Example: "Normal address:\n" +
		"create-address 1 1\n" +
		"2-3 address:\n" +
		"create-address 2 3",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(2)(cmd, args); err != nil {
			logging.CPrint(logging.ERROR, "wrong argument count", logging.LogFormat{"count": len(args)})
			return err
		}
		sigRequired, err := strconv.ParseUint(args[0], 10, 64)
		if err != nil {
			logging.CPrint(logging.ERROR, "invalid sig_required", logging.LogFormat{"value": args[0]})
			return err
		}
		pubNumber, err := strconv.ParseUint(args[1], 10, 64)
		if err != nil {
			logging.CPrint(logging.ERROR, "invalid pub_number", logging.LogFormat{"value": args[1]})
			return err
		}

		createAddressArgSigRequired, createAddressArgPubNumber = int32(sigRequired), int32(pubNumber)
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		pubNumber, sigRequired := createAddressArgPubNumber, createAddressArgSigRequired
		logging.CPrint(logging.INFO, "create-address called", logging.LogFormat{"pk_number": pubNumber, "sig_required": sigRequired})

		resp := &rpcprotobuf.CreateAddressResponse{}
		ClientCall("/v1/addresses", POST, &rpcprotobuf.CreateAddressRequest{SignRequire: sigRequired, PubKeyNumber: pubNumber}, resp)
		printJSON(resp)
	},
}

// listAddressesCmd represents the listAddresses command
var listAddressesCmd = &cobra.Command{
	Use:   "list-addresses",
	Short: "List the addresses",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		logging.CPrint(logging.INFO, "list-addresses called", logging.LogFormat{})

		resp := &rpcprotobuf.GetAddressesResponse{}
		ClientCall("/v1/addresses", GET, nil, resp)
		printJSON(resp)
	},
}

// getTotalBalanceCmd represents the getTotalBalance command
var getTotalBalanceCmd = &cobra.Command{
	Use:   "get-total-balance",
	Short: "Get total balance of all addresses",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		logging.CPrint(logging.INFO, "get-total-balance called", logging.LogFormat{})

		resp := &rpcprotobuf.GetAllBalanceResponse{}
		ClientCall("/v1/addresses/balance", GET, nil, resp)
		printJSON(resp)
	},
}

// getAddressBalanceCmd represents the getAddressBalance command
var getAddressBalanceCmd = &cobra.Command{
	Use:   "get-address-balance <addr1> <addr2> ... <addrN>",
	Short: "Get balance of specified addresses",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.MinimumNArgs(1)(cmd, args); err != nil {
			logging.CPrint(logging.ERROR, "wrong argument count", logging.LogFormat{"count": len(args)})
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		addrList := strings.Join(args, ",")
		logging.CPrint(logging.INFO, "get-address-balance called", logging.LogFormat{"address_list": addrList})

		resp := &rpcprotobuf.GetBalanceResponse{}
		ClientCall(fmt.Sprintf("/v1/addresses/%s/balance", addrList), GET, nil, resp)
		printJSON(resp)
	},
}

// validateAddressCmd represents the validateAddress command
var validateAddressCmd = &cobra.Command{
	Use:   "validate-address <addr>",
	Short: "Validate specified address",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			logging.CPrint(logging.ERROR, "wrong argument count", logging.LogFormat{"count": len(args)})
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		addr := args[0]
		logging.CPrint(logging.INFO, "validate-address called", logging.LogFormat{"address": addr})

		resp := &rpcprotobuf.ValidateAddressResponse{}
		ClientCall(fmt.Sprintf("/v1/addresses/%s/validate", addr), GET, nil, resp)
		printJSON(resp)
	},
}

// listUTXOsCmd represents the listUTXOs command
var listUTXOsCmd = &cobra.Command{
	Use:   "list-utxos",
	Short: "List the UTXOs in wallet",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		logging.CPrint(logging.INFO, "list-utxos called", logging.LogFormat{})

		resp := &rpcprotobuf.GetUtxoResponse{}
		ClientCall("/v1/addresses/utxos", GET, nil, resp)
		printJSON(resp)
	},
}

// getUTXOsByAmountCmd represents the getUTXOsByAmount command
var getUTXOsByAmountCmd = &cobra.Command{
	Use:   "get-utxos-by-amount <amount>",
	Short: "Get available utxos by amount",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			logging.CPrint(logging.ERROR, "wrong argument count", logging.LogFormat{"count": len(args)})
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		amount := args[0]
		logging.CPrint(logging.INFO, "get-utxos-by-amount called", logging.LogFormat{"amount": amount})

		resp := &rpcprotobuf.GetUtxoByAmountResponse{}
		ClientCall(fmt.Sprintf("/v1/addresses/utxos/%s", amount), GET, nil, resp)
		printJSON(resp)
	},
}

// args for createTransactionCmd
var (
	createTransactionFlagAuto bool
	createTransactionArgReq   = &rpcprotobuf.CreateRawTransactionRequest{}
)

// createTransactionCmd represents the createTransaction command
var createTransactionCmd = &cobra.Command{
	Use:   "create-tx <args_json> [--auto]",
	Short: "Create transaction using arguments in json format",
	Long: "Create transaction by arguments in json format, here are two creating modes:\n" +
		"1. Normal Mode:\n" +
		"Create transaction by manually specify input UTXO and output address/value in MASS unit.\n" +
		"2. Auto Mode:\n" +
		"Create transaction by simply specify output address/value in MASS unit, txFee would be auto calculated when set 0.\n" +
		"Use auto mode by setting --auto flag.",
	Example: "1. Normal Mode:\n" +
		`create-tx "{\"inputs\": [{\"txId\": \"txid_1\", \"vout\": out_index_1}], \"amounts\": {\"address_1\": amount_1}, \"lockTime\": 0}"` + "\n" +
		"2. Auto Mode:\n" +
		`create-tx "{\"amounts\": {\"address_1\": amount_1, \"address_2\": amount_2}, \"lockTime\": 0, \"userTxFee\":0}" --auto`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			logging.CPrint(logging.ERROR, "wrong argument count", logging.LogFormat{"count": len(args)})
			return err
		}
		err := json.Unmarshal([]byte(args[0]), createTransactionArgReq)
		if err != nil {
			logging.CPrint(logging.ERROR, "fail to unmarshal json", logging.LogFormat{"err": err, "args_json": args[0]})
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		auto, req := createTransactionFlagAuto, createTransactionArgReq
		jBytes, _ := json.Marshal(req)
		logging.CPrint(logging.INFO, "create-tx called", logging.LogFormat{"auto": auto, "args_json": string(jBytes)})

		var path = "/v1/transactions/creating"
		if auto {
			path = strings.Join([]string{path, "auto"}, "/")
		}

		resp := &rpcprotobuf.CreateRawTransactionResponse{}
		ClientCall(fmt.Sprintf(path), POST, req, resp)
		printJSON(resp)
	},
}

// signTransactionCmd represents the signTransaction command
var signTransactionCmd = &cobra.Command{
	//Use:   "sign-transaction <hex> <mode> <password>",
	Use:   "sign-tx <hex> <mode>",
	Short: "Sign transaction in hex format, normally mode should be ALL",
	Long: "Sign transaction in hex format, here are different modes(normally should be ALL):\n" +
		"ALL:                 sign for all inputs and outputs\n" +
		"NONE:                sign for all inputs\n" +
		"SINGLE:              sign for all inputs and one specified output\n" +
		"ALL|ANYONECANPAY:    sign for one specified input and all outputs\n" +
		"NONE|ANYONECANPAY:   sign for one specified input\n" +
		"SINGLE|ANYONECANPAY: sign for one specified input and corresponding output",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(2)(cmd, args); err != nil {
			logging.CPrint(logging.ERROR, "wrong argument count", logging.LogFormat{"count": len(args)})
			return err
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		//hex, mode, passwd := args[0], args[1], args[2]
		hex, mode := args[0], args[1]
		logging.CPrint(logging.INFO, "sign-tx called", logging.LogFormat{"hex": hex, "mode": mode})

		req := &rpcprotobuf.SignRawTransactionRequest{
			RawTx: hex,
			Flags: mode,
			//Password: passwd,
		}
		resp := &rpcprotobuf.SignRawTransactionResponse{}
		ClientCall("/v1/transactions/signing", POST, req, resp)
		printJSON(resp)
	},
}

// args for estimateTransactionFeeCmd
var (
	estimateTransactionFeeArgReq = &rpcprotobuf.GetTransactionFeeRequest{}
)

// estimateTransactionFeeCmd represents the estimateTransactionFee command
var estimateTransactionFeeCmd = &cobra.Command{
	Use:     "estimate-tx-fee <args_json>",
	Short:   "Estimate transaction fee using arguments in json format",
	Example: `estimate-tx-fee "{\"amounts\": {\"address_1\": amount_1, \"address_2\": amount_2}, \"lockTime\": 0}"`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			logging.CPrint(logging.ERROR, "wrong argument count", logging.LogFormat{"count": len(args)})
			return err
		}
		err := json.Unmarshal([]byte(args[0]), estimateTransactionFeeArgReq)
		if err != nil {
			logging.CPrint(logging.ERROR, "fail to unmarshal json", logging.LogFormat{"err": err, "args_json": args[0]})
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		req := estimateTransactionFeeArgReq
		jBytes, _ := json.Marshal(req)
		logging.CPrint(logging.INFO, "estimate-tx-fee called", logging.LogFormat{"args_json": string(jBytes)})

		resp := &rpcprotobuf.GetTransactionFeeResponse{}
		ClientCall("v1/transactions/fee", POST, req, resp)
		printJSON(resp)
	},
}

func init() {
	createTransactionCmd.PersistentFlags().BoolVar(&createTransactionFlagAuto, "auto", false, "auto create transaction")
}
