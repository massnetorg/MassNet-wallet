package cmd

import (
	"os"
	"path/filepath"

	"massnet.org/mass-wallet/logging"

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
	rootCmd.AddCommand(stopCmd)

	// cmd_wallet
	rootCmd.AddCommand(listWalletsCmd)
	rootCmd.AddCommand(createWalletCmd)
	rootCmd.AddCommand(useWalletCmd)
	rootCmd.AddCommand(importWalletCmd)
	rootCmd.AddCommand(importWalletByMnemonicCmd)
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
	rootCmd.AddCommand(getTxStatusCmd)
	rootCmd.AddCommand(listTrasactionsCmd)

	rootCmd.AddCommand(createStakingTransactionCmd)
	rootCmd.AddCommand(getStakingHistoryCmd)
	rootCmd.AddCommand(getLatestRewardListCmd)

	rootCmd.AddCommand(createBindingTransactionCmd)
	rootCmd.AddCommand(getBindingHistoryCmd)
	rootCmd.AddCommand(getAddressBindingCmd)
}
