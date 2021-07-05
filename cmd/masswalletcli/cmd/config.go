package cmd

import (
	"github.com/massnetorg/mass-core/logging"

	"github.com/spf13/viper"
)

const (
	defaultServer      = "http://localhost:9688"
	defaultLogFilename = "walletcli"
	defaultLogLevel    = "info"
	defaultLogDir      = "walletcli-logs"
	rpcCert            = "cert.crt"
	rpcKey             = "cert.key"
)

var (
	config = new(Config)
)

type Config struct {
	Server   string `json:"server"`
	LogDir   string `json:"log_dir"`
	LogLevel string `json:"log_level"`
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	viper.SetConfigFile("walletcli-config.json")
	if err := viper.ReadInConfig(); err != nil {
		panic(err)
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

	logging.Init(config.LogDir, defaultLogFilename, config.LogLevel, 1, false)
}
