package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/massnetorg/mass-core/logging"
	"github.com/spf13/cobra"
	pb "massnet.org/mass-wallet/api/proto"
)

var (
	frozenPeriod uint32
)

var createStakingTransactionCmd = &cobra.Command{
	Use:   "createstakingtransaction <staking_address> <frozen_period> <value> [fee=?] [from=?]",
	Short: "Creates a staking transaction.",
	Long: "Creates a staking transaction.\n" +
		"\nArguments:\n" +
		"  <staking_address>    a staking address of current wallet\n" +
		"  <frozen_period>      number of blocks this staking transaction would be locked\n" +
		"  <value>              amount of staked MASS, a real with max 8 decimal places\n" +
		"  [fee]                optional, MASS paid to miner, a real with max 8 decimal places\n" +
		"  [from]               optional, the address of current wallet from which all inputs selected. \n" +
		"                       if not provided inputs may be selected from any address of current wallet\n",
	Example: `  createstakingtransaction ms1qp0czrc8errz8gdmpjgxd59kwvydf3g3ch72d6qm2kqwzlgm232pksqw0eky 1000 95.5 fee=0.05`,
	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.RangeArgs(3, 5)(cmd, args); err != nil {
			logging.VPrint(logging.ERROR, LogMsgIncorrectArgsNumber, logging.LogFormat{"actual": len(args)})
			return err
		}
		lh, err := strconv.ParseUint(args[1], 10, 32)
		if err != nil {
			return err
		}
		frozenPeriod = uint32(lh)
		for i := 3; i < len(args); i++ {
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
		logging.VPrint(logging.INFO, "createstakingtransaction called", logging.LogFormat{
			"from_address":    from,
			"staking_address": args[0],
			"frozen_period":   frozenPeriod,
			"amount":          args[2],
			"fee":             fee,
		})

		req := &pb.CreateStakingTransactionRequest{
			FromAddress:    from,
			StakingAddress: args[0],
			FrozenPeriod:   frozenPeriod,
			Amount:         args[2],
			Fee:            fee,
		}
		resp := &pb.CreateRawTransactionResponse{}
		return ClientCall("/v1/transactions/staking", POST, req, resp)
	},
}

var getStakingHistoryCmd = &cobra.Command{
	Use:   "liststakingtransactions [all]",
	Short: "Returns staking transactions of current wallet.",
	Long: "By default, returns stakings not withdrawn.\n" +
		"\nArguments:\n" +
		"  [all]                returns all stakings, including withdrawn.\n",
	Args: cobra.MinimumNArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		logging.VPrint(logging.INFO, "liststakingtransactions called", logging.LogFormat{"args": args})
		resp := &pb.GetStakingHistoryResponse{}
		if len(args) == 0 || strings.ToLower(args[0]) != "all" {
			return ClientCall("/v1/transactions/staking/history", GET, nil, resp)
		}
		return ClientCall("/v1/transactions/staking/history/all", GET, nil, resp)
	},
}

var getBlockStakingReward = &cobra.Command{
	Use:   "getblockstakingreward [height]",
	Short: "Returns staking reward list at target height.",
	Long: "\nBy default, returns the reward list at best height.\n" +
		"\nArguments:\n" +
		"  [height]                optional, default value 0.\n",
	Args: cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		logging.VPrint(logging.INFO, "getblockstakingreward called", EmptyLogFormat)

		height := 0
		if len(args) > 0 {
			height, err = strconv.Atoi(args[0])
			if err != nil {
				return err
			}
		}
		resp := &pb.GetBlockStakingRewardResponse{}
		return ClientCall(fmt.Sprintf("/v1/blocks/%d/stakingreward", height), GET, nil, resp)
	},
}
