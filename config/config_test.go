package config_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"massnet.org/mass-wallet/config"
)

var (
	cfgjson = `
	{
		"app" : {
			"profile": "",
			"cpu_profile": ""
		},
		"network": {
			"p2p": {
				"seeds": "",
				"listen_address": ""
			},
			"api": {
				"grpc_port": "9687",
				"http_port": "9688",
				"disable_tls": true,
				"rpc_cert": "./cert.crt",
				"rpc_key": "./cert.key"
			}
		},
		"chain": {
			"data_dir": "ldb/chain"
		},
		"log": {
			"log_dir": "ldb/logs",
			"log_level": "info"
		},
		"wallet": {
			"data_dir": "ldb/wallet",
			"pub_pass": "passphrase"
		}
	}   
	`
)

func TestMarshal(t *testing.T) {
	cfg := &config.Config{
		Config: config.NewDefaultConfig(),
	}
	err := json.Unmarshal([]byte(cfgjson), cfg.Config)
	assert.Nil(t, err)
	config.CheckConfig(cfg)
	if cfg.Config.Network.API.DisableTls {
		assert.Empty(t, cfg.Config.Network.API.RpcCert)
		assert.Empty(t, cfg.Config.Network.API.RpcKey)
	}
}
