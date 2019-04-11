package cmd

import (
	"os"
	"path/filepath"

	"massnet.org/mass-wallet/logging"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// MasscliCmd represents the base command when called without any subcommands
var MasscliCmd = &cobra.Command{
	Use:   filepath.Base(os.Args[0]),
	Short: "Command line client for MASS blockchain",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the MasscliCmd.
func Execute() {
	if err := MasscliCmd.Execute(); err != nil {
		logging.CPrint(logging.FATAL, "fail on MasscliCmd.Execute", logging.LogFormat{"err": err})
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	cobra.OnInitialize(initLogger)
	cobra.OnInitialize(logBasicInfo)

	MasscliCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./.masscli.json)")
	MasscliCmd.PersistentFlags().StringVar(&flagAPIURL, "api_url", defaultAPIURL, "API URL")
	MasscliCmd.PersistentFlags().StringVar(&flagLogDir, "log_dir", defaultLogDir, "directory for log files")
	MasscliCmd.PersistentFlags().StringVar(&flagLogLevel, "log_level", defaultLogLevel, "level of logs (debug, info, warn, error, fatal, panic)")

	viper.BindPFlag("api_url", MasscliCmd.PersistentFlags().Lookup("api_url"))
	viper.BindPFlag("log_dir", MasscliCmd.PersistentFlags().Lookup("log_dir"))
	viper.BindPFlag("log_level", MasscliCmd.PersistentFlags().Lookup("log_level"))

	MasscliCmd.AddCommand(createAddressCmd)
	MasscliCmd.AddCommand(listAddressesCmd)
	MasscliCmd.AddCommand(getTotalBalanceCmd)
	MasscliCmd.AddCommand(getAddressBalanceCmd)
	MasscliCmd.AddCommand(validateAddressCmd)
	MasscliCmd.AddCommand(listUTXOsCmd)
	MasscliCmd.AddCommand(getUTXOsByAmountCmd)
	MasscliCmd.AddCommand(createTransactionCmd)
	MasscliCmd.AddCommand(signTransactionCmd)
	MasscliCmd.AddCommand(estimateTransactionFeeCmd)

	MasscliCmd.AddCommand(sendTransactionCmd)
	MasscliCmd.AddCommand(getTransactionCmd)
	MasscliCmd.AddCommand(getTransactionStatusCmd)

	MasscliCmd.AddCommand(getClientStatusCmd)
}
