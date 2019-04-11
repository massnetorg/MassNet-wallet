// Modified for MassNet
// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"github.com/massnetorg/MassNet-wallet/consensus"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/massnetorg/MassNet-wallet/version"
	"github.com/massnetorg/MassNet-wallet/wire"

	"errors"

	"github.com/btcsuite/go-flags"
	"github.com/massnetorg/MassNet-wallet/config/pb"
)

const (
	defaultChainTag          = "testnet"
	DefaultConfigFilename    = "config.json"
	defaultShowVersion       = false
	DefaultDataDirname       = "chain"
	DefaultLogLevel          = "info"
	defaultLogDirname        = "logs"
	defaultWalletFileDirname = "wallet"
	defaultDbType            = "leveldb"

	defaultBlockMinSize      = 0
	defaultBlockMaxSize      = wire.MaxBlockPayload
	defaultBlockPrioritySize = consensus.DefaultBlockPrioritySize
	defaultSigCacheMaxSize   = 50000
)

var (
	FreeTxRelayLimit         = 15.0
	AddrIndex                = true
	DropAddrIndex            = false
	MaxOrphanTxs             = consensus.MaxOrphanTransactions
	NoRelayPriority          = true
	MinRelayTxFee            = consensus.DefaultMinRelayTxFee
	BlockPrioritySize uint32 = defaultBlockPrioritySize
	BlockMinSize      uint32 = defaultBlockMinSize
	BlockMaxSize      uint32 = defaultBlockMaxSize
	SigCacheMaxSize   uint   = defaultSigCacheMaxSize
)

const (
	defaultListenAddress    = "tcp://0.0.0.0:43453"
	defaultDialTimeout      = 3
	defaultHandshakeTimeout = 30
)

var (
	MaxPeers            = 50
	Moniker             = "anonymous"
	ChainTag            = defaultChainTag
	BanDuration         = time.Hour * 24
	BanThreshold uint32 = 100
)

const (
	defaultAPIUrl      = "localhost"
	defaultAPIPortGRPC = "9685"
	defaultAPIPortHttp = "9686"
	defaultAPICORSAddr = ""
)

var (
	MassHomeDir          = AppDataDir("mass", false)
	defaultConfigFile    = DefaultConfigFilename
	defaultDataDir       = DefaultDataDirname
	knownDbTypes         = []string{"leveldb", "memdb"}
	defaultWalletFileDir = defaultWalletFileDirname
	defaultLogDir        = defaultLogDirname
)

// RunServiceCommand is only set to a real function on Windows.  It is used
// to parse and execute service commands specified via the -s flag.
var RunServiceCommand func(string) error

// serviceOptions defines the configuration options for mass as a service on
// Windows.
type serviceOptions struct {
	ServiceCommand string `short:"s" long:"service" description:"Service command {install, remove, start, stop}"`
}

type Config struct {
	*configpb.Config
	ConfigFile  string `short:"C" long:"configfile" description:"Path to configuration file"`
	ShowVersion bool   `short:"V" long:"version" description:"Display Version information and exit"`
	Generate    bool   `long:"generate" description:"Generate (mine) coins when start"`
}

// newConfigParser returns a new command line flags parser.
func newConfigParser(cfg *Config, so *serviceOptions, options flags.Options) *flags.Parser {
	parser := flags.NewParser(cfg, options)
	if runtime.GOOS == "windows" {
		parser.AddGroup("Service Options", "Service Options", so)
	}
	return parser
}

// ParseConfig reads and parses the config using a Config file and command
// line options.
// This func proceeds as follows:
//  1) Start with a default config with sane settings
//  2) Pre-parse the command line to check for an alternative config file
func ParseConfig() (*Config, []string, error) {
	// Default config.
	cfg := Config{
		ConfigFile:  defaultConfigFile,
		ShowVersion: defaultShowVersion,
		Config:      configpb.NewConfig(),
	}

	serviceOpts := serviceOptions{}

	preCfg := cfg
	preParser := newConfigParser(&preCfg, &serviceOpts, flags.HelpFlag)
	_, err := preParser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			fmt.Fprintln(os.Stderr, err)
			return nil, nil, err
		}
	}

	appName := "mass-wallet"
	usageMessage := fmt.Sprintf("Use %s -h to show usage", appName)
	if preCfg.ShowVersion {
		fmt.Println(appName, "version", version.GetVersion())
		os.Exit(0)
	}

	if serviceOpts.ServiceCommand != "" && RunServiceCommand != nil {
		err := RunServiceCommand(serviceOpts.ServiceCommand)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(0)
	}

	// Load additional config from file.
	parser := newConfigParser(&cfg, &serviceOpts, flags.Default)

	// Parse command line options again to ensure they take precedence.
	remainingArgs, err := parser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); !ok || e.Type != flags.ErrHelp {
			fmt.Fprintln(os.Stderr, usageMessage)
		}
		return nil, nil, err
	}

	return &cfg, remainingArgs, nil
}

func LoadConfig(cfg *Config) (*Config, error) {
	b, err := ioutil.ReadFile(cfg.ConfigFile)
	if err != nil {
		return cfg, err
	}

	if err := json.Unmarshal(b, cfg.Config); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func CheckConfig(cfg *Config) (*Config, error) {
	if cfg.Common == nil {
		cfg.Common = new(configpb.CommonConfig)
	}
	if cfg.App == nil {
		cfg.App = new(configpb.AppConfig)
	}
	if cfg.Network == nil {
		cfg.Network = new(configpb.NetworkConfig)
	}
	if cfg.Network.P2P == nil {
		cfg.Network.P2P = new(configpb.P2PConfig)
		cfg.Network.P2P.AddPeer = make([]string, 0)
	}
	if cfg.Network.API == nil {
		cfg.Network.API = new(configpb.APIConfig)
	}
	if cfg.Db == nil {
		cfg.Db = new(configpb.DataConfig)
	}
	if cfg.Log == nil {
		cfg.Log = new(configpb.LogConfig)
	}
	if cfg.Chain == nil {
		cfg.Chain = new(configpb.ChainConfig)
	}
	if cfg.Pool == nil {
		cfg.Pool = new(configpb.MemPoolConfig)
	}
	if cfg.Wallet == nil {
		cfg.Wallet = new(configpb.WalletConfig)
	}

	// Checks for APIConfig
	if cfg.Network.API.APIUrl == "" {
		cfg.Network.API.APIUrl = defaultAPIUrl
	}
	if cfg.Network.API.APIPortHttp == "" {
		cfg.Network.API.APIPortHttp = defaultAPIPortHttp
	}
	if cfg.Network.API.APIPortGRPC == "" {
		cfg.Network.API.APIPortGRPC = defaultAPIPortGRPC
	}
	if cfg.Network.API.APICORSAddr == "" {
		cfg.Network.API.APICORSAddr = defaultAPICORSAddr
	}

	// Checks for P2PConfig
	cfg.Network.P2P.Seeds = NormalizeSeeds(cfg.Network.P2P.Seeds, ChainParams.DefaultPort)
	if cfg.Network.P2P.ListenAddress == "" {
		cfg.Network.P2P.ListenAddress = defaultListenAddress
	}
	if cfg.Network.P2P.DialTimeout == 0 {
		cfg.Network.P2P.DialTimeout = defaultDialTimeout
	}
	if cfg.Network.P2P.HandshakeTimeout == 0 {
		cfg.Network.P2P.HandshakeTimeout = defaultHandshakeTimeout
	}

	var dealWithDir = func(path string) string {
		return cleanAndExpandPath(path)
	}

	// Checks for DataConfig
	if cfg.Db.DbType == "" {
		cfg.Db.DbType = defaultDbType
	}
	if !validDbType(cfg.Db.DbType) {
		return cfg, errors.New(fmt.Sprintf("invalid db_type %s", cfg.Db.DbType))
	}
	if cfg.Db.DataDir == "" {
		cfg.Db.DataDir = defaultDataDir
	}
	cfg.Db.DataDir = dealWithDir(cfg.Db.DataDir)

	// Checks for LogConfig
	if cfg.Log.LogDir == "" {
		cfg.Log.LogDir = defaultLogDir
	}
	cfg.Log.LogDir = dealWithDir(cfg.Log.LogDir)
	if cfg.Log.DebugLevel == "" {
		cfg.Log.DebugLevel = DefaultLogLevel
	}

	// Checks for WalletConfig
	if cfg.Wallet.WalletDir == "" {
		cfg.Wallet.WalletDir = defaultWalletFileDir
	}
	cfg.Wallet.WalletDir = dealWithDir(cfg.Wallet.WalletDir)

	return cfg, nil
}

// cleanAndExpandPath expands environment variables and leading ~ in the
// passed path, cleans the result, and returns it.
func cleanAndExpandPath(path string) string {
	// Expand initial ~ to OS specific home directory.
	if strings.HasPrefix(path, "~") {
		homeDir := filepath.Dir(MassHomeDir)
		path = strings.Replace(path, "~", homeDir, 1)
	}

	// NOTE: The os.ExpandEnv doesn't work with Windows-style %VARIABLE%,
	// but they variables can still be expanded via POSIX-style $VARIABLE.
	return filepath.Clean(os.ExpandEnv(path))
}

// validDbType returns whether or not dbType is a supported database type.
func validDbType(dbType string) bool {
	for _, knownType := range knownDbTypes {
		if dbType == knownType {
			return true
		}
	}

	return false
}
