// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	coreconfig "github.com/massnetorg/mass-core/config"
	"github.com/massnetorg/mass-core/consensus"
	"github.com/massnetorg/mass-core/version"
	"github.com/massnetorg/mass-core/wire"

	"errors"

	configpb "massnet.org/mass-wallet/config/pb"

	flags "github.com/btcsuite/go-flags"
)

const (
	DefaultConfigFilename  = "config.json"
	DefaultChainDataDir    = "chain"
	DefaultElkFilename     = "json-masswallet"
	DefaultLoggingFilename = "masswalletlog"

	defaultChainTag    = "mainnet"
	defaultShowVersion = false
	defaultCreate      = false

	defaultBlockMinSize      = 0
	defaultBlockMaxSize      = wire.MaxBlockPayload
	defaultBlockPrioritySize = consensus.DefaultBlockPrioritySize

	DefaultAddressGapLimit         = 20
	DefaultMaxUnusedStakingAddress = 8
	DefaultMaxTxFee                = "1.0" // MASS
)

var (
	MassWalletHomeDir            = AppDataDir("masswallet", false)
	knownDbTypes                 = []string{"leveldb", "rocksdb", "memdb"}
	FreeTxRelayLimit             = 15.0
	AddrIndex                    = true
	NoRelayPriority              = true
	BlockPrioritySize     uint32 = defaultBlockPrioritySize
	BlockMinSize          uint32 = defaultBlockMinSize
	BlockMaxSize          uint32 = defaultBlockMaxSize
	HDCoinTypeTestNet     uint32 = 1
	HDCoinTypeMassMainNet uint32 = 297
	MaxPeers                     = 50
	Moniker                      = "anonymous"
	ChainTag                     = defaultChainTag
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
	Core           *coreconfig.Config      `json:"core"`
	Wallet         *configpb.WalletConfig  `json:"wallet"`
	AddCheckpoints []coreconfig.Checkpoint `json:"-"`
	ShowVersion    bool                    `short:"V" long:"version" description:"Display Version information and exit" json:"-"`
	Create         bool                    `long:"create" description:"Create the wallet if it does not exist" json:"-"`
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
		ShowVersion: defaultShowVersion,
		Create:      defaultCreate,
		Core:        NewDefCoreConfig(),
		Wallet:      NewDefWalletConfig(),
	}

	// Service options which are only added on Windows.
	serviceOpts := serviceOptions{}

	// Pre-parse the command line options to see if an alternative config
	// file or the version flag was specified.  Any errors aside from the
	// help message error can be ignored here since they will be caught by
	// the final parse below.
	preCfg := cfg
	preParser := newConfigParser(&preCfg, &serviceOpts, flags.HelpFlag)
	_, err := preParser.Parse()
	if err != nil {
		if e, ok := err.(*flags.Error); ok && e.Type == flags.ErrHelp {
			fmt.Fprintln(os.Stderr, err)
			return nil, nil, err
		}
	}

	// Show the version and exit if the Version flag was specified.
	appName := filepath.Base(os.Args[0])
	usageMessage := fmt.Sprintf("Use %s -h to show usage", appName)
	if preCfg.ShowVersion {
		fmt.Println(appName, "version", version.GetVersion())
		os.Exit(0)
	}

	// Perform service command and exit if specified.  Invalid service
	// commands show an appropriate error.  Only runs on Windows since
	// the RunServiceCommand function will be nil when not on Windows.
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

func LoadConfig(cfg *Config) {
	b, err := ioutil.ReadFile(DefaultConfigFilename)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(0)
	}

	if err := json.Unmarshal(b, cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(0)
	}
}

func CheckConfig(cfg *Config) *Config {
	// Checks for P2PConfig
	cfg.Core.P2P.Seeds = NormalizeSeeds(cfg.Core.P2P.Seeds, ChainParams.DefaultPort)
	if cfg.Wallet.API.DisableTls {
		cfg.Wallet.API.RpcCert = ""
		cfg.Wallet.API.RpcKey = ""
	}

	// Checks for DataConfig
	if !validDbType(cfg.Core.Datastore.DBType) {
		err := errors.New(fmt.Sprintf("invalid db_type %s", cfg.Core.Datastore.DBType))
		fmt.Fprintln(os.Stderr, err)
		os.Exit(0)
	}
	cfg.Core.Datastore.Dir = cleanAndExpandPath(cfg.Core.Datastore.Dir)

	// Checks for LogConfig
	cfg.Core.Log.LogDir = cleanAndExpandPath(cfg.Core.Log.LogDir)

	// add checkpoints
	if !cfg.Core.Chain.DisableCheckpoints {
		add, err := coreconfig.ParseCheckpoints(cfg.Core.Chain.AddCheckpoints)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(0)
		}
		cfg.AddCheckpoints = add
	}

	// Checks for AdvancedConfig
	if cfg.Wallet.Settings.AddressGapLimit <= 1 {
		err := errors.New(fmt.Sprintf("AddressGapLimit should be larger than 1"))
		fmt.Fprintln(os.Stderr, err)
		os.Exit(0)
	}
	if cfg.Wallet.Settings.AddressGapLimit <= cfg.Wallet.Settings.MaxUnusedStakingAddress {
		cfg.Wallet.Settings.MaxUnusedStakingAddress = (uint32)(float32(cfg.Wallet.Settings.AddressGapLimit) * 0.2)
	}
	if cfg.Wallet.Settings.MaxUnusedStakingAddress == 0 {
		cfg.Wallet.Settings.MaxUnusedStakingAddress = 1
	}
	if len(cfg.Wallet.Settings.MaxTxFee) == 0 {
		cfg.Wallet.Settings.MaxTxFee = DefaultMaxTxFee
	}

	return cfg
}

// cleanAndExpandPath expands environment variables and leading ~ in the
// passed path, cleans the result, and returns it.
func cleanAndExpandPath(path string) string {
	// Expand initial ~ to OS specific home directory.
	if strings.HasPrefix(path, "~") {
		homeDir := filepath.Dir(MassWalletHomeDir)
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

func fileExists(filePath string) (bool, error) {
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
