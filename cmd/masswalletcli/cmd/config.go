package cmd

import (
	"os"
	"path/filepath"

	"github.com/massnetorg/mass-core/logging"

	"github.com/spf13/viper"
)

const (
	defaultServer      = "http://localhost:9688"
	defaultLogFilename = "walletcli"
	defaultLogLevel    = "info"
	defaultLogDir      = "walletcli-logs"
	defaultRpcCert     = "cert.crt"
	defaultRpcKey      = "cert.key"
)

var (
	config = new(CliConfig)
)

type CliConfig struct {
	Server   string `json:"server"`
	LogDir   string `json:"log_dir"`
	LogLevel string `json:"log_level"`
	RpcCert  string `json:"rpc_cert"`
	RpcKey   string `json:"rpc_key"`
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.AutomaticEnv() // read in environment variables that match

	if fs, _ := os.Stat("walletcli-config.json"); fs != nil {
		// If a config file is found, read it in.
		viper.SetConfigFile("walletcli-config.json")
		if err := viper.ReadInConfig(); err != nil {
			panic(err)
		}
	}

	// Load config to memory.
	config.Server = viper.GetString("server")
	if config.Server == "" {
		config.Server = defaultServer
	}

	config.LogDir = viper.GetString("log_dir")
	if config.LogDir == "" {
		config.LogDir = defaultLogDir
	}

	config.LogLevel = viper.GetString("log_level")
	if config.LogLevel == "" {
		config.LogLevel = defaultLogLevel
	}

	config.RpcCert = viper.GetString("rpc_cert")
	if config.RpcCert == "" {
		config.RpcCert = defaultRpcCert
	}

	config.RpcKey = viper.GetString("rpc_key")
	if config.RpcKey == "" {
		config.RpcKey = defaultRpcKey
	}

	logging.Init(filepath.Join(".", config.LogDir), defaultLogFilename, config.LogLevel, 1, false)
}
