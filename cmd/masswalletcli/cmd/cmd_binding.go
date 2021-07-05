package cmd

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/massnetorg/mass-core/blockchain"
	"github.com/massnetorg/mass-core/logging"
	"github.com/massnetorg/mass-core/massutil"
	"github.com/massnetorg/mass-core/poc"
	"github.com/massnetorg/mass-core/poc/chiapos"
	"github.com/massnetorg/mass-core/poc/chiawallet"
	"github.com/spf13/cobra"
	pb "massnet.org/mass-wallet/api/proto"
	wcfg "massnet.org/mass-wallet/config"
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
	Use:   "listbindingtransactions [all]",
	Short: "Returns binding transaction of current wallet.",
	Long: "By default, returns bindings not withdrawn.\n" +
		"\nArguments:\n" +
		"  [all]                returns all bindings, including withdrawn.\n",
	Args: cobra.MinimumNArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "listbindingtransactions called", logging.LogFormat{"args": args})

		resp := &pb.GetBindingHistoryResponse{}
		if len(args) == 0 || strings.ToLower(args[0]) != "all" {
			return ClientCall("/v1/transactions/binding/history", GET, nil, resp)
		}
		return ClientCall("/v1/transactions/binding/history/all", GET, nil, resp)
	},
}

var batchBindPoolPkCmd = &cobra.Command{
	Use:   "batchbindpoolpk <chiaKeystore> <from> [coinbase]",
	Short: "Check or bind coinbase for chia pool pubkey.",
	Long: "Check or bind coinbase for chia pool pubkey.\n" +
		"\nArguments:\n" +
		"  <chiaKeystore>       Required, keystore storing chia poolSks/poolPks. Exported by 'massminercli'.\n" +
		"  <from>               Specify the address to pay for the transaction. Ensure it has at least 1.01 MASS.\n" +
		"                       Ignored if flag '-c' is set.\n" +
		"  [coinbase]           Specify coinbase to be bound to poolpk, clear already bound coinbase if not provided.\n" +
		"                       Ignored if flag '-c' is set.",
	Example: "  batchbindpoolpk chia-keystore.json ms1qq0d99znj2pc032frunvme29ypquxprxrrexthv2d9t5v6zgul4a7qapk0jj" +
		" ms1qqyq0y0wt4el4834acfq9g3t4p2jjsnqg3msw4jdm4u45ext3kr6yqwc06xr",
	Args: cobra.RangeArgs(1, 3),
	RunE: func(cmd *cobra.Command, args []string) error {

		isCheck, err := cmd.Flags().GetBool("check")
		if err != nil {
			return fmt.Errorf("failed to get flag 'check'")
		}

		store, err := chiawallet.NewKeystoreFromFile(args[0])
		if err != nil {
			return err
		}

		type pkInfo struct {
			Pk    *chiapos.G1Element
			Sk    *chiapos.PrivateKey
			Nonce uint32
		}

		pkToInfo := make(map[string]*pkInfo)
		for _, minerKey := range store.GetAllMinerKeys() {
			pkToInfo[hex.EncodeToString(minerKey.PoolPublicKey.Bytes())] = &pkInfo{
				Pk:    minerKey.PoolPublicKey,
				Sk:    minerKey.PoolPrivateKey,
				Nonce: 1,
			}
		}

		if len(pkToInfo) == 0 {
			fmt.Println("no pool pk found, do nothing")
			return nil
		}

		req := &pb.CheckPoolPkCoinbaseRequest{}
		for pkHex := range pkToInfo {
			req.PoolPubkeys = append(req.PoolPubkeys, pkHex)
		}
		resp := &pb.CheckPoolPkCoinbaseResponse{}

		if isCheck {
			return ClientCall("/v1/bindings/poolpubkeys", POST, req, resp)
		}

		// bind
		if len(args) < 2 {
			return fmt.Errorf("require at least 2 arguments")
		}

		var coinbaseAddr []byte
		if len(args) > 2 {
			addr, err := massutil.DecodeAddress(args[2], wcfg.ChainParams)
			if err != nil {
				return err
			}
			if !massutil.IsWitnessV0Address(addr) {
				return fmt.Errorf("invalid coinbase: %s", args[2])
			}
			coinbaseAddr = addr.ScriptAddress()
		}

		// prepare nonce
		if err = ClientCallWithoutPrintResponse("/v1/bindings/poolpubkeys", POST, req, resp); err != nil {
			return err
		}
		for pkHex, info := range pkToInfo {
			if result := resp.Result[pkHex]; result != nil {
				info.Nonce = result.Nonce + 1
			}
		}

		pass := readPassword()
		var txIds []string
		// send transactions
		for pk, info := range pkToInfo {
			// sign payload
			sig, err := blockchain.SignPoolPkPayload(info.Sk, coinbaseAddr, info.Nonce)
			if err != nil {
				return err
			}
			payload := blockchain.NewBindPoolCoinbasePayload(info.Pk, sig, coinbaseAddr, info.Nonce)

			reqCreate := &pb.CreatePoolPkCoinbaseTransactionRequest{
				FromAddress: args[1],
				Payload:     hex.EncodeToString(blockchain.EncodePayload(payload)),
			}
			respCreate := &pb.CreateRawTransactionResponse{}
			if err := ClientCallWithoutPrintResponse("/v1/transactions/poolpkcoinbase", POST, reqCreate, respCreate); err != nil {
				return err
			}

			txId, err := signSendTx(respCreate.Hex, pass)
			if err != nil {
				return err
			}
			logging.CPrint(logging.INFO, fmt.Sprintf("bind poolpk %s, txid %s", pk, txId))
			txIds = append(txIds, txId)
		}

		return waitConfirmed(txIds)
	},
}

var checkPoolPkCoinbaseCmd = &cobra.Command{
	Use:   "checkpoolpkcoinbase <pubkey> <pubkey> ...",
	Short: "Query coinbase bound to chia pool pubkey.",
	Long: "Query coinbase bound to chia pool pubkey.\n" +
		"\nArguments:\n" +
		"  <pubkey>       Required, hex-encoded chia pool pubkey.\n",
	Example: "  checkpoolpkcoinbase 8919b3715c0e8998c5d2f36f1236c7ab0d44b8285644effe2ee0d9f54a6dadf0efc6bbd0917371b2e9462186ac99c948",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		req := &pb.CheckPoolPkCoinbaseRequest{
			PoolPubkeys: args,
		}

		resp := &pb.CheckPoolPkCoinbaseResponse{}
		return ClientCall("/v1/bindings/poolpubkeys", POST, req, resp)
	},
}

var getNetworkBindingCmd = &cobra.Command{
	Use:   "getnetworkbinding [height]",
	Short: "Gets total network binding and new binding price.",
	Long: "Gets total network binding and new binding price.\n" +
		"\nArguments:\n" +
		"  [height]                Specify height to query at.\n",
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "getnetworkbinding called")

		resp := &pb.GetNetworkBindingResponse{}
		if len(args) == 0 {
			return ClientCall("/v1/bindings/networkbinding", GET, nil, resp)
		}

		height, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}

		return ClientCall(fmt.Sprintf("/v1/bindings/networkbinding/%d", height), GET, nil, resp)
	},
}

var checkTargetBindingCmd = &cobra.Command{
	Use:   "checktargetbinding <target> <target> ...",
	Short: "Query binding info.",
	Long: "Query binding info.\n" +
		"\nArguments:\n" +
		"  <target>       Required, base58 encoded binding target address, not wallet address.\n",
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return ClientCall("/v1/bindings/targets", POST, &pb.CheckTargetBindingRequest{
			Targets: args,
		}, &pb.CheckTargetBindingResponse{})
	},
}

var batchBindingCmd = &cobra.Command{
	Use:   `batchbinding <file> <from>`,
	Short: "Batch check or send binding transactions from file.",
	Long: "Batch check or send binding transactions from file.\n" +
		"\nArguments:\n" +
		"  <file>       Required, file storing targets to be bound. Exported by 'massminercli'.\n" +
		"  <from>       Specify the address to pay for bindings.\n" +
		"               Ignored if flag '-c' is set.",
	Example: "  batchbinding targets.json ms1qq0d99znj2pc032frunvme29ypquxprxrrexthv2d9t5v6zgul4a7qapk0jj",
	Args:    cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {

		isCheck, err := cmd.Flags().GetBool("check")
		if err != nil {
			return fmt.Errorf("failed to get flag 'check'")
		}

		list, err := massutil.NewBindingListFromFile(args[0])
		if err != nil {
			return err
		}
		list = list.RemoveDuplicate()

		// filter unbound
		unbound := make([]string, 0, 1000)
		targetPlot := make(map[string]massutil.BindingPlot)

		batchSize := 500
		batch := make([]string, 0, batchSize)
		for _, plot := range list.Plots {
			target, err := massutil.DecodeAddress(plot.Target, wcfg.ChainParams)
			if err != nil {
				return err
			}
			if _, ok := target.(*massutil.AddressBindingTarget); !ok {
				return fmt.Errorf("invalid binding target: %s", plot.Target)
			}
			targetPlot[plot.Target] = plot

			batch = append(batch, plot.Target)
			if len(batch) >= batchSize {
				ub, err := filterUnboundTargets(batch)
				if err != nil {
					return err
				}
				unbound = append(unbound, ub...)
				batch = batch[:0]
			}
		}
		if len(batch) > 0 {
			ub, err := filterUnboundTargets(batch)
			if err != nil {
				return err
			}
			unbound = append(unbound, ub...)
		}
		fmt.Printf("total %d, unbound %d\n", list.TotalCount, len(unbound))

		if isCheck {
			unboundList := &massutil.BindingList{}
			for _, ub := range unbound {
				unboundList.Plots = append(unboundList.Plots, targetPlot[ub])
			}
			if err := unboundList.WriteToFile(args[0] + ".unbound"); err != nil {
				return fmt.Errorf("failed to write unbound targets to file %s: %v", args[0]+".unbound", err)
			}
			fmt.Printf("unbound targets have been wrote to file '%s'\n", args[0]+".unbound")
			return nil
		}

		// create bindings
		if len(args) < 2 {
			return fmt.Errorf("require 2 arguments, <file> and <from>")
		}
		if len(unbound) == 0 {
			fmt.Println("all targets bound, do nothing!")
			return nil
		}

		password := readPassword()

		da, err := massutil.DecodeAddress(args[1], wcfg.ChainParams)
		if err != nil {
			return err
		}
		from := da.EncodeAddress()

		// get prices
		massPrices, chiaPrices, err := queryBindingPrices()
		if err != nil {
			return err
		}

		// prepare outputs
		i := 0
		total := 0
		var txids []string
		outputs := make([]*pb.CreateBindingTransactionRequest_Output, 0, batchSize)
		for _, target := range unbound {
			plot := targetPlot[target]
			var (
				price string
				ok    bool
			)
			switch plot.Type {
			case uint8(poc.ProofTypeDefault):
				price, ok = massPrices[uint32(plot.Size)]
			case uint8(poc.ProofTypeChia):
				price, ok = chiaPrices[uint32(plot.Size)]
			default:
				return fmt.Errorf("unknown plot type %d", plot.Type)
			}
			if !ok {
				return fmt.Errorf("price not found for target %s, type %d, size %d", target, plot.Type, plot.Size)
			}

			outputs = append(outputs, &pb.CreateBindingTransactionRequest_Output{
				HolderAddress:  from,
				BindingAddress: target,
				Amount:         price,
			})

			if len(outputs) >= batchSize {
				txid, err := createSignSend(outputs, password, from)
				if err != nil {
					return err
				}
				if len(txid) != 0 {
					txids = append(txids, txid)
					total += len(outputs)
					logging.CPrint(logging.INFO, fmt.Sprintf("%d: send binding transaction %s, total bound %d", i, txid, total))
					i++
				}
				outputs = outputs[:0]
			}
		}
		if len(outputs) >= 0 {
			txid, err := createSignSend(outputs, password, from)
			if err != nil {
				return err
			}
			if len(txid) != 0 {
				txids = append(txids, txid)
				total += len(outputs)
				logging.CPrint(logging.INFO, fmt.Sprintf("%d: send binding transaction %s, total bound %d", i, txid, total))
			}
		}

		// wait confirmed
		if err := waitConfirmed(txids); err != nil {
			return fmt.Errorf("waitConfirmed failed: %v", err)
		}

		if total < len(unbound) {
			msg := fmt.Sprintf("total bound %d/%d, please try!", total, len(unbound))
			fmt.Fprintln(os.Stdout, fmt.Sprintf("%c[1;;31m%s%s%c[0m", 0x1B, "Failed: ", msg, 0x1B))
			return nil
		}

		fmt.Printf("total bound %d/%d, done!\n", total, len(unbound))
		return nil
	},
}

func filterUnboundTargets(targets []string) ([]string, error) {
	var resp pb.CheckTargetBindingResponse
	if err := ClientCallWithoutPrintResponse("/v1/bindings/targets", POST, &pb.CheckTargetBindingRequest{
		Targets: targets,
	}, &resp); err != nil {
		return nil, err
	}

	var unbound []string
	for target, info := range resp.Result {
		if strings.TrimSuffix(info.Amount, " MASS") == "0" {
			unbound = append(unbound, target)
		}
	}
	return unbound, nil
}

func queryBindingPrices() (massPrices, chiaPrices map[uint32]string, err error) {
	resp := &pb.GetNetworkBindingResponse{}
	if err := ClientCallWithoutPrintResponse("/v1/bindings/networkbinding", GET, nil, resp); err != nil {
		return nil, nil, err
	}

	massPrices = make(map[uint32]string)
	for size, price := range resp.BindingPriceMassBitlength {
		massPrices[size] = strings.TrimSuffix(price, " MASS")
	}

	chiaPrices = make(map[uint32]string)
	for size, price := range resp.BindingPriceChiaK {
		chiaPrices[size] = strings.TrimSuffix(price, " MASS")
	}
	return
}

func createSignSend(outputs []*pb.CreateBindingTransactionRequest_Output, pass, from string) (string, error) {
	// create
	req1 := &pb.CreateBindingTransactionRequest{
		Outputs:     outputs,
		FromAddress: from,
	}

	unsigned := ""
	retry := 0
	for {
		resp1 := &pb.CreateRawTransactionResponse{}
		if err := ClientCallWithoutPrintResponse("/v1/transactions/binding", POST, req1, resp1); err != nil {
			if strings.Contains(err.Error(), "Insufficient wallet balance") {
				retry++
				if retry > 30 {
					return "", fmt.Errorf("wait too long for available funds")
				}
				fmt.Println("no available funds, sleep 30s...")
				time.Sleep(30 * time.Second)
				continue
			}
			return "", err
		}
		unsigned = resp1.Hex
		break
	}

	return signSendTx(unsigned, pass)
}

func signSendTx(txHex, pass string) (string, error) {
	reqSign := &pb.SignRawTransactionRequest{
		RawTx:      txHex,
		Passphrase: pass,
		Flags:      "ALL",
	}
	respSign := &pb.SignRawTransactionResponse{}
	if err := ClientCallWithoutPrintResponse("/v1/transactions/sign", POST, reqSign, respSign); err != nil {
		return "", fmt.Errorf("sign binding failed: %v", err)
	}

	// send
	reqSend := &pb.SendRawTransactionRequest{
		Hex: respSign.Hex,
	}
	respSend := &pb.SendRawTransactionResponse{}
	if err := ClientCallWithoutPrintResponse("/v1/transactions/send", POST, reqSend, respSend); err != nil {
		if strings.Contains(err.Error(), "plot pk already bound") { // only for batchbinding
			return "", nil
		}
		return "", fmt.Errorf("send binding failed: %v", err)
	}
	return respSend.TxId, nil
}

func waitConfirmed(txids []string) error {

	fmt.Printf("waiting %d transactions to be packed\n", len(txids))

	unconfirmed := make([]string, 0, len(txids))
	for {
		for _, txid := range txids {
			resp := &pb.GetTxStatusResponse{}
			if err := ClientCallWithoutPrintResponse(fmt.Sprintf("/v1/transactions/%s/status", txid), GET, nil, resp); err != nil {
				return err
			}
			switch resp.Code {
			case 1: // "confirmed"
			case 2: // "missing"
				logging.CPrint(logging.WARN, "transaction not found "+txid)
			case 3, 4: // "packing", "confirming"
				unconfirmed = append(unconfirmed, txid)
			default:
				logging.CPrint(logging.ERROR, fmt.Sprintf("unknown tx status %d for transaction %s", resp.Code, txid))
			}
		}
		if len(unconfirmed) == 0 {
			fmt.Println("done!")
			return nil
		}

		fmt.Println("sleep 10s before re-check tx status")
		time.Sleep(10 * time.Second)
		txids = append(txids[:0], unconfirmed...)
		unconfirmed = unconfirmed[:0]
	}
}
