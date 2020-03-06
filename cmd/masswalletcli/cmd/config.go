package cmd

import (
	"massnet.org/mass-wallet/logging"

	"github.com/spf13/viper"
)

const (
	defaultServer      = "https://localhost:9686"
	defaultLogFilename = "clilog"
	defaultLogLevel    = "info"
	defaultLogDir      = "./logs"
	rpcCert            = "./conf/cert.crt"
	rpcKey             = "./conf/cert.key"
	configFile         = "./cli-config.json"
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
	// viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	viper.SetConfigFile(configFile)
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
