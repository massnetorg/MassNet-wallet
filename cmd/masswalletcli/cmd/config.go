package cmd

import (
	"os/user"
	"path/filepath"
	"strings"

	"massnet.org/mass-wallet/logging"

	"github.com/spf13/viper"
)

const (
	defaultServer      = "https://localhost:9686"
	defaultLogFilename = "clilog"
	defaultLogLevel    = "info"
	defaultMassHomeDir = ".massnet"
	defaultLogDir      = "./logs"
	rpcCert            = "./conf/cert.crt"
	rpcKey             = "./conf/cert.key"
	configFile         = "./conf/masswalletcli.json"
)

var (
	config = new(Config)
)

type Config struct {
	Server   string `json:"server"`
	LogDir   string `json:"log_dir"`
	LogLevel string `json:"log_level"`
	RPCCert  string `json:"rpc_cert"`
	RPCKey   string `json:"rpc_key"`
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	viper.SetConfigFile(ensureMassAbsPath(configFile))
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

	config.RPCCert = viper.GetString("rpc_cert")
	if config.RPCCert == "" {
		config.RPCCert = rpcCert
	}

	config.RPCKey = viper.GetString("rpc_key")
	if config.RPCKey == "" {
		config.RPCKey = rpcKey
	}

	config.LogDir = ensureMassAbsPath(config.LogDir)
	config.RPCCert = ensureMassAbsPath(config.RPCCert)
	config.RPCKey = ensureMassAbsPath(config.RPCKey)

	logging.Init(config.LogDir, defaultLogFilename, config.LogLevel, 1, false)
}

// ensureMassAbsPath to ensure the configured path is absolute
func ensureMassAbsPath(configuredPath string) string {
	if configuredPath == "" {
		return ""
	}
	if strings.Index(configuredPath, "/") == 0 {
		return configuredPath
	}
	usr, err := user.Current()
	if err != nil {
		return configuredPath
	}
	if strings.Index(configuredPath, ".") == 0 {
		configuredPath = strings.TrimPrefix(configuredPath, ".")
	}
	subPaths := []string{usr.HomeDir, defaultMassHomeDir}
	subPaths = append(subPaths, strings.Split(configuredPath, "/")...)
	return filepath.Join(subPaths...)
}
