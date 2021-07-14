package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/massnetorg/mass-core/limits"
	"github.com/massnetorg/mass-core/logging"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   filepath.Base(os.Args[0]),
	Short: "Command line client for MASS wallet",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	// Use all processor cores.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Up some limits.
	if err := limits.SetLimits(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set limits: %v\n", err)
		os.Exit(1)
	}

	if err := rootCmd.Execute(); err != nil {
		logging.VPrint(logging.FATAL, "Command failed", logging.LogFormat{"err": err})
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// cmd_others
	rootCmd.AddCommand(createCertCmd)
	rootCmd.AddCommand(getClientStatusCmd)
	rootCmd.AddCommand(getBestBlockCmd)
	rootCmd.AddCommand(getBlockByHeightCmd)
	rootCmd.AddCommand(stopCmd)

	// cmd_wallet
	rootCmd.AddCommand(listWalletsCmd)
	rootCmd.AddCommand(createWalletCmd)
	rootCmd.AddCommand(useWalletCmd)
	rootCmd.AddCommand(importWalletCmd)
	rootCmd.AddCommand(importMnemonicCmd)
	rootCmd.AddCommand(exportWalletCmd)
	rootCmd.AddCommand(removeWalletCmd)
	rootCmd.AddCommand(getWalletMnemonicCmd)
	rootCmd.AddCommand(getWalletBalanceCmd)
	rootCmd.AddCommand(getAddressBalanceCmd)
	rootCmd.AddCommand(listUtxoCmd)
	rootCmd.AddCommand(createAddressCmd)
	rootCmd.AddCommand(listAddressesCmd)
	rootCmd.AddCommand(validateAddressCmd)

	//
	rootCmd.AddCommand(createRawTransactionCmd)
	rootCmd.AddCommand(autoCreateRawTransactionCmd)
	rootCmd.AddCommand(signRawTransactionCmd)
	rootCmd.AddCommand(getTransactionFeeCmd)
	rootCmd.AddCommand(sendRawTransactionCmd)
	rootCmd.AddCommand(getRawTransactionCmd)
	rootCmd.AddCommand(decodeRawTransactionCmd)
	rootCmd.AddCommand(getTxStatusCmd)
	rootCmd.AddCommand(listTrasactionsCmd)

	rootCmd.AddCommand(createStakingTransactionCmd)
	rootCmd.AddCommand(getStakingHistoryCmd)
	rootCmd.AddCommand(getBlockStakingReward)

	rootCmd.AddCommand(createBindingTransactionCmd)

	batchBindPoolPkCmd.Flags().BoolP("check", "c", false, "only check current bound coinbase")
	rootCmd.AddCommand(batchBindPoolPkCmd)

	rootCmd.AddCommand(checkPoolPkCoinbaseCmd)
	rootCmd.AddCommand(checkTargetBindingCmd)
	rootCmd.AddCommand(getNetworkBindingCmd)
	rootCmd.AddCommand(getBindingHistoryCmd)

	rootCmd.AddCommand(exportChainCmd)

	importChainCmd.Flags().BoolVarP(&NoExpensiveValidation, "fast-validation", "f", false, "disable expensive validations")
	rootCmd.AddCommand(importChainCmd)

	batchBindingCmd.Flags().BoolP("check", "c", false, "only check unbound targets")
	rootCmd.AddCommand(batchBindingCmd)

	getBindingListCmd.Flags().BoolVarP(&getBindingListFlagOverwrite, "overwrite", "o", false, "overwrite existed file")
	getBindingListCmd.Flags().BoolVarP(&getBindingListFlagListAll, "all", "a", false, "list all files instead of only plotted files")
	getBindingListCmd.Flags().StringVarP(&getBindingListFlagKeystore, "keystore", "", "", "specify the keystore to eliminate files without private key")
	getBindingListCmd.Flags().StringVarP(&getBindingListFlagPlotType, "type", "t", "", "specify the searching plot type: m1 (for native MassDB) or m2 (for Chia Plot)")
	getBindingListCmd.Flags().StringSliceVarP(&getBindingListFlagDirectories, "dirs", "d", nil, "specify the searching directories")
	rootCmd.AddCommand(getBindingListCmd)
}
