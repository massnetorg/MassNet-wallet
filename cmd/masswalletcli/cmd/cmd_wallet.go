package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/massnetorg/mass-core/logging"
	pb "massnet.org/mass-wallet/api/proto"

	"github.com/spf13/cobra"
)

var createAddressCmd = &cobra.Command{
	Use:   "createaddress <version>",
	Short: "Creates a new address within current wallet.",
	Long: "Creates a new address within current wallet.\n" +
		"\nArguments:\n" +
		"  <version>    0 - create a standard transaction address\n" +
		"               1 - create a staking transaction address\n",
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		version, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
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
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		version, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
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
	Args: cobra.MinimumNArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			minconf = 1
			detail  = false
		)
		for i := 0; i < len(args); i++ {
			key, value, err := parseCommandVar(args[i])
			if err != nil {
				continue
			}
			switch key {
			case "minconf":
				c, err := strconv.Atoi(value)
				if err == nil {
					minconf = c
				}
			case "detail":
				detail, _ = strconv.ParseBool(value)
			default:
			}
		}
		logging.VPrint(logging.INFO, "getwalletbalance called", logging.LogFormat{
			"min_confirmations": minconf,
			"detail":            detail,
		})

		req := &pb.GetWalletBalanceRequest{
			RequiredConfirmations: int32(minconf),
			Detail:                detail,
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
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		confs, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
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
	Args:  cobra.ExactArgs(1),
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
	Args: cobra.MinimumNArgs(0),
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
	Use:   "createwallet [entropy=?] [remarks=?]",
	Short: "Creates a new wallet of latest version(1).",
	Long: "Creates a new wallet of latest version(1), both walletId and mnemonic are included in response.\n" +
		"\nArguments:\n" +
		"  [entropy]     optional, initial entropy length, it must be a multiple of 32 bits, the allowed size is 128-256.\n" +
		"  [remarks]     optional.\n",
	Example: `  createwallet entropy=160 remarks='create a wallet for test'`,
	Args:    cobra.MinimumNArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			entropy = 128
			remarks = ""
		)
		for i := 0; i < len(args); i++ {
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
		logging.VPrint(logging.INFO, "createwallet called", logging.LogFormat{
			"bitsize": entropy,
			"remards": remarks,
		})

		req := &pb.CreateWalletRequest{
			Passphrase: readPassword(),
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
	Args:  cobra.ExactArgs(1),
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
	Use:   "removewallet <wallet_id>",
	Short: "Removes specified wallet from server.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "removewallet called", logging.LogFormat{"walletid": args[0]})

		req := &pb.RemoveWalletRequest{
			WalletId:   args[0],
			Passphrase: readPassword(),
		}
		resp := &pb.RemoveWalletResponse{}
		return ClientCall("/v1/wallets/remove", POST, req, resp)
	},
}

var exportWalletCmd = &cobra.Command{
	Use:   "exportwallet <wallet_id>",
	Short: "Exports wallet keystore.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "exportwallet called", logging.LogFormat{"walletid": args[0]})

		req := &pb.ExportWalletRequest{
			WalletId:   args[0],
			Passphrase: readPassword(),
		}
		resp := &pb.ExportWalletResponse{}
		return ClientCall("/v1/wallets/export", POST, req, resp)
	},
}

var importWalletCmd = &cobra.Command{
	Use:   "importwallet <keystore>",
	Short: "Imports a wallet keystore.",
	Long: "Imports a wallet keystore, both version 0 and 1 are compatible\n" +
		"\nArguments:\n" +
		"  <keystore>     raw json of keystore.\n",
	Example: `  importwallet '{"crypto":"", "hdPath":"","remarks":"",...}' 123456`,
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "importwallet called", EmptyLogFormat)

		req := &pb.ImportWalletRequest{
			Keystore:   args[0],
			Passphrase: readPassword(),
		}
		resp := &pb.ImportWalletResponse{}
		return ClientCall("/v1/wallets/import", POST, req, resp)
	},
}

var importMnemonicCmd = &cobra.Command{
	Use:   "importmnemonic <mnemonic> [initial=?] [remarks=?]",
	Short: "Imports a wallet backup mnemonic.",
	Long: "Imports a wallet backup mnemonic.\n" +
		"\nArguments:\n" +
		"  <mnemonic>	mnemonic phrase\n" +
		"  [initial]	number of initial addresses, default 0\n",
	Example: `  importmnemonic 'tomorrow entry oval ...' 123456 initial=10 remarks='backup mnemonic'`,
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		initial := 0
		remarks := ""
		for i := 1; i < len(args); i++ {
			key, value, err := parseCommandVar(args[i])
			if err != nil {
				return err
			}
			switch key {
			case "initial":
				initial, err = strconv.Atoi(value)
				if err != nil {
					return err
				}
			case "remarks":
				remarks = value
			default:
				return errorUnknownCommandParam(key)
			}
		}

		logging.VPrint(logging.INFO, "importmnemonic called", logging.LogFormat{
			"initial": initial,
			"remarks": remarks,
		})

		req := &pb.ImportMnemonicRequest{
			Mnemonic:      args[0],
			Passphrase:    readPassword(),
			ExternalIndex: uint32(initial),
			Remarks:       remarks,
		}
		resp := &pb.ImportWalletResponse{}
		return ClientCall("/v1/wallets/import/mnemonic", POST, req, resp)
	},
}

var getWalletMnemonicCmd = &cobra.Command{
	Use:   "getwalletmnemonic <wallet_id>",
	Short: "Returns mnemonic of the specified wallet.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "getwalletmnemonic called", logging.LogFormat{
			"walletid": args[0],
		})

		req := &pb.GetWalletMnemonicRequest{
			WalletId:   args[0],
			Passphrase: readPassword(),
		}
		resp := &pb.GetWalletMnemonicResponse{}
		return ClientCall("/v1/wallets/mnemonic", POST, req, resp)
	},
}
