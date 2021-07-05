package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/massnetorg/mass-core/cmdutils"
	"github.com/massnetorg/mass-core/logging"
	"github.com/spf13/cobra"
	walletcfg "massnet.org/mass-wallet/config"
)

var exportChainCmd = &cobra.Command{
	Use:   "exportchain <datastorePath> <filename> [lastHeight]",
	Short: "Export blockchain into file, *.gz for compression",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		bc, close, err := cmdutils.MakeChain(args[0], true, walletcfg.ChainParams)
		if err != nil {
			return fmt.Errorf("MakeChain failed: %v", err)
		}
		defer close()

		start := time.Now()

		height := uint64(0)
		if len(args) == 3 {
			if height, err = strconv.ParseUint(args[2], 10, 64); err != nil {
				return err
			}
		}

		err = cmdutils.ExportChain(bc, args[1], height)
		if err != nil {
			return fmt.Errorf("ExportChain failed: %v", err)
		}
		fmt.Printf("Export done in %v\n", time.Since(start))
		return nil
	},
}

var NoExpensiveValidation bool

var importChainCmd = &cobra.Command{
	Use:   "importchain <filename> <datastorePath>",
	Short: "Import a blockchain file",

	Args: func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(2)(cmd, args); err != nil {
			logging.CPrint(logging.ERROR, "wrong argument count", logging.LogFormat{"count": len(args)})
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		bc, close, err := cmdutils.MakeChain(args[1], false, walletcfg.ChainParams)
		if err != nil {
			return err
		}
		defer close()

		start := time.Now()

		logging.CPrint(logging.INFO, "Import start", logging.LogFormat{
			"head":   bc.BestBlockHash(),
			"height": bc.BestBlockHeight(),
		})

		err = cmdutils.ImportChain(bc, args[0], NoExpensiveValidation)
		if err != nil {
			logging.CPrint(logging.ERROR, "Import abort", logging.LogFormat{
				"head":   bc.BestBlockHash(),
				"height": bc.BestBlockHeight(),
				"err":    err,
			})
			return err
		}
		fmt.Printf("Import done in %v\n", time.Since(start))
		return nil
	},
}
