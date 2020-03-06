package cmd

import (
	"fmt"
	"strconv"
	"strings"

	pb "massnet.org/mass-wallet/api/proto"
	"massnet.org/mass-wallet/logging"

	"github.com/spf13/cobra"
)

var (
	externalIndex = 0
	entropy       = 128
	remarks       = ""
	keystore      []byte
	confs         = 0
	version       = 0
	bool1         = false
)

var createAddressCmd = &cobra.Command{
	Use:   "createaddress <version>",
	Short: "Creates a new address within current wallet.",
	Long: "Creates a new address within current wallet.\n" +
		"\nArguments:\n" +
		"  <version>    0 - create a standard transaction address\n" +
		"               1 - create a staking transaction address\n",
	Args: func(cmd *cobra.Command, args []string) (err error) {
		if err = cobra.ExactArgs(1)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		version, err = strconv.Atoi(args[0])
		return err
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "createaddress called", logging.LogFormat{"version": version})

		resp := &pb.CreateAddressResponse{}
		return ClientCall("/v1/addresses/create", POST, &pb.CreateAddressRequest{Version: int32(version)}, resp)
	},
}

var listAddressesCmd = &cobra.Command{
	Use:   "listaddresses <version>",
	Short: "Lists addresses of current wallet.",
	Long: "Lists addresses of current wallet.\n" +
		"\nArguments:\n" +
		"  <version>    0 - list all standard transaction addresses\n" +
		"               1 - list all staking transaction addresses\n",
	Args: func(cmd *cobra.Command, args []string) (err error) {
		if err = cobra.ExactArgs(1)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		version, err = strconv.Atoi(args[0])
		return err
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "listaddresses called", logging.LogFormat{"version": version})

		resp := &pb.GetAddressesResponse{}
		return ClientCall(fmt.Sprintf("/v1/addresses/%s", args[0]), GET, nil, resp)
	},
}

var getWalletBalanceCmd = &cobra.Command{
	Use:   "getwalletbalance [minconf=?] [detail=?]",
	Short: "Returns total balance of current wallet.",
	Long: "Returns total balance of current wallet.\n" +
		"\nArguments:\n" +
		"  [minconf]   optional. minimum number of blockchain confirmations of UTXOs, default 1\n" +
		"  [detail]   optional boolean. if query balance detail, default false\n",
	Args: func(cmd *cobra.Command, args []string) (err error) {
		if err = cobra.RangeArgs(0, 2)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		confs = 1
		bool1 = false
		for i := 0; i < len(args); i++ {
			var key, value string
			key, value, err = parseCommandVar(args[i])
			if err == nil {
				switch key {
				case "minconf":
					confs, err = strconv.Atoi(value)
				case "detail":
					bool1, err = strconv.ParseBool(value)
				default:
					err = errorUnknownCommandParam(key)
				}
			}
		}
		return err
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "getwalletbalance called", logging.LogFormat{
			"min_confirmations": confs,
			"detail":            bool1,
		})

		req := &pb.GetWalletBalanceRequest{
			RequiredConfirmations: int32(confs),
			Detail:                bool1,
		}
		resp := &pb.GetWalletBalanceResponse{}
		return ClientCall("/v1/wallets/current/balance", POST, req, resp)
	},
}

var getAddressBalanceCmd = &cobra.Command{
	Use:   "getaddressbalance <min_conf> [<address> <address> ...]",
	Short: "Returns balance of specified addresses of current wallet.",
	Long: "Returns balance of specified addresses of current wallet.\n" +
		"\nArguments:\n" +
		"  <min_conf>         minimum number of blockchain confirmations of UTXOs\n" +
		"  [<address>...]     standard mass address, if not provided, it'll return balance of all addresses of current wallet\n",
	Args: func(cmd *cobra.Command, args []string) (err error) {
		if err = cobra.MinimumNArgs(1)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		confs, err = strconv.Atoi(args[0])
		return err
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "getaddressbalance called", logging.LogFormat{
			"min_confirmations": confs,
			"address_list":      strings.Join(args[1:], ","),
		})
		req := &pb.GetAddressBalanceRequest{
			RequiredConfirmations: int32(confs),
		}
		if len(args) > 1 {
			req.Addresses = args[1:]
		}
		resp := &pb.GetAddressBalanceResponse{}
		return ClientCall("/v1/addresses/balance", POST, req, resp)
	},
}

var validateAddressCmd = &cobra.Command{
	Use:   "validateaddress <address>",
	Short: "Checks if the address is in correct format and belong to current wallet.",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "validateaddress called", logging.LogFormat{"address": args[0]})

		resp := &pb.ValidateAddressResponse{}
		return ClientCall(fmt.Sprintf("/v1/addresses/%s/validate", args[0]), GET, nil, resp)
	},
}

var listUtxoCmd = &cobra.Command{
	Use:   "listutxo <address> <address> ...",
	Short: "Lists UTXO of specified addresses of current wallet.",
	Long: "Lists UTXO of specified addresses of current wallet.\n" +
		"\nArguments:\n" +
		"  <address>  if not provided, it'll return all UTXOs of current wallet",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.MinimumNArgs(0)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "listutxo called", logging.LogFormat{"addresses": args})

		resp := &pb.GetUtxoResponse{}
		return ClientCall("/v1/addresses/utxos", POST, &pb.GetUtxoRequest{Addresses: args}, resp)
	},
}

var listWalletsCmd = &cobra.Command{
	Use:   "listwallets",
	Short: "Returns all wallets imported into this server.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "listwallets called", EmptyLogFormat)

		resp := &pb.WalletsResponse{}
		return ClientCall("/v1/wallets", GET, nil, resp)
	},
}

var createWalletCmd = &cobra.Command{
	Use:   "createwallet <passphrase> [entropy=?] [remarks=?]",
	Short: "Creates a new wallet and returns walletId and mnemonic.",
	Long: "Creates a new wallet and returns walletId and mnemonic.\n" +
		"\nArguments:\n" +
		"  <passphrase>  used to protect mnemonic\n" +
		"  [entropy]     optional, initial entropy length, it must be a multiple of 32 bits, the allowed size is 128-256.\n" +
		"  [remarks]     optional.\n",
	Example: `  createwallet 123456 entropy=160 remarks='create a wallet for test'`,
	Args: func(cmd *cobra.Command, args []string) error {
		var err error
		if err = cobra.RangeArgs(1, 3)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		for i := 1; i < len(args); i++ {
			key, value, err := parseCommandVar(args[i])
			if err != nil {
				return err
			}
			switch key {
			case "entropy":
				entropy, err = strconv.Atoi(value)
				if err != nil {
					return err
				}
				if entropy < 128 || entropy > 256 || entropy%32 != 0 {
					return fmt.Errorf("invalid entropy: %d", entropy)
				}
			case "remarks":
				remarks = value
			default:
				return errorUnknownCommandParam(key)
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "createwallet called", logging.LogFormat{
			"bitsize": entropy,
			"remards": remarks,
		})

		req := &pb.CreateWalletRequest{
			Passphrase: args[0],
			BitSize:    int32(entropy),
			Remarks:    remarks,
		}
		resp := &pb.CreateWalletResponse{}
		return ClientCall("/v1/wallets/create", POST, req, resp)
	},
}

var useWalletCmd = &cobra.Command{
	Use:   "usewallet <wallet_id>",
	Short: "Switches transaction context to <wallet_id>.",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(1)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "usewallet called", logging.LogFormat{"walletid": args[0]})

		req := &pb.UseWalletRequest{
			WalletId: args[0],
		}
		resp := &pb.UseWalletResponse{}
		return ClientCall("/v1/wallets/use", POST, req, resp)
	},
}

var removeWalletCmd = &cobra.Command{
	Use:   "removewallet <wallet_id> <passphrase>",
	Short: "Removes specified wallet from server.",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(2)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "removewallet called", logging.LogFormat{"walletid": args[0]})

		req := &pb.RemoveWalletRequest{
			WalletId:   args[0],
			Passphrase: args[1],
		}
		resp := &pb.RemoveWalletResponse{}
		return ClientCall("/v1/wallets/remove", POST, req, resp)
	},
}

var exportWalletCmd = &cobra.Command{
	Use:   "exportwallet <wallet_id> <passphrase>",
	Short: "Exports wallet keystore.",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(2)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "exportwallet called", logging.LogFormat{"walletid": args[0]})

		req := &pb.ExportWalletRequest{
			WalletId:   args[0],
			Passphrase: args[1],
		}
		resp := &pb.ExportWalletResponse{}
		return ClientCall("/v1/wallets/export", POST, req, resp)
	},
}

var importWalletCmd = &cobra.Command{
	Use:   "importwallet <keystore> <passphrase>",
	Short: "Imports wallet into server by keystore.",
	Long: "Imports wallet into server by keystore.\n" +
		"\nArguments:\n" +
		"  <keystore>     raw json of keystore.\n" +
		"  <passphrase>   used to decrypt keystore data.\n",
	Example: `  importwallet '{"crypto":"", "hdPath":"","remarks":"",...}' 123456` +
		"\n  // win\n" +
		`  importwallet "{\"crypto\":\"...\", \"hdPath\":\"...\",\"remarks\":\"...\",...}" 123456`,
	Args: func(cmd *cobra.Command, args []string) (err error) {
		if err = cobra.ExactArgs(2)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "importwallet called", logging.LogFormat{"keystore": string(keystore)})

		req := &pb.ImportWalletRequest{
			Keystore:   args[0],
			Passphrase: args[1],
		}
		resp := &pb.ImportWalletResponse{}
		return ClientCall("/v1/wallets/import", POST, req, resp)
	},
}

var importWalletByMnemonicCmd = &cobra.Command{
	Use:   "importwalletbymnemonic <mnemonic> <passphrase> [externalindex=?] [remarks=?]",
	Short: "Imports wallet into server by mnemonic.",
	Long: "Imports wallet into server by mnemonic.\n" +
		"\nArguments:\n" +
		"  <mnemonic>      standard mnemonic phrase\n" +
		"  <passphrase>    used to protect mnemonic\n" +
		"  [externalindex] the initial pub address count, default 0\n",
	Example: `  importwalletbymnemonic 'one two three four ...' 123456 externalindex=10 remarks='import wallet by mnemonic'`,
	Args: func(cmd *cobra.Command, args []string) (err error) {
		if err = cobra.RangeArgs(2, 4)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		for i := 2; i < len(args); i++ {
			key, value, err := parseCommandVar(args[i])
			if err != nil {
				return err
			}
			switch key {
			case "externalindex":
				externalIndex, err = strconv.Atoi(value)
				if err != nil {
					return err
				}
				if externalIndex < 0 {
					return fmt.Errorf("invalid externalindex: %d", externalIndex)
				}
			case "remarks":
				remarks = value
			default:
				return errorUnknownCommandParam(key)
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "importwalletbymnemonic called", logging.LogFormat{
			"externalindex": externalIndex,
			"remarks":       remarks,
		})

		req := &pb.ImportWalletWithMnemonicRequest{
			Mnemonic:      args[0],
			Passphrase:    args[1],
			ExternalIndex: uint32(externalIndex),
			Remarks:       remarks,
		}
		resp := &pb.ImportWalletResponse{}
		return ClientCall("/v1/wallets/import/mnemonic", POST, req, resp)
	},
}

var getWalletMnemonicCmd = &cobra.Command{
	Use:   "getwalletmnemonic <wallet_id> <passphrase>",
	Short: "Returns mnemonic of the specified wallet.",
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(2)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "getwalletmnemonic called", logging.LogFormat{
			"walletid": args[0],
		})

		req := &pb.GetWalletMnemonicRequest{
			WalletId:   args[0],
			Passphrase: args[1],
		}
		resp := &pb.GetWalletMnemonicResponse{}
		return ClientCall("/v1/wallets/mnemonic", POST, req, resp)
	},
}
