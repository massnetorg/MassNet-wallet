package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
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
		"data": {
			"db_dir": "chain",
			"db_type": "leveldb"
		},
		"log": {
			"log_dir": "logs",
			"log_level": "info"
		},
		"advanced": {
			"address_gap_limit": 3000
		}
	}   
	`
)

func TestMarshal(t *testing.T) {
	cfg := &Config{
		Config: NewDefaultConfig(),
	}
	err := json.Unmarshal([]byte(cfgjson), cfg.Config)
	assert.Nil(t, err)
	CheckConfig(cfg)
	assert.Empty(t, cfg.Config.Network.API.RpcCert)
	assert.Empty(t, cfg.Config.Network.API.RpcKey)
}

func TestMarshalMaxTxFee(t *testing.T) {
	tests := []struct {
		name    string
		cfgjson string
		expect  string
	}{
		{
			"notset",
			`{
				"advanced": {}
			}`,
			DefaultMaxTxFee,
		},
		{
			"valid",
			`{
				"advanced": {
					"max_tx_fee": "5.555"
				}
			}`,
			"5.555",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := &Config{
				Config: NewDefaultConfig(),
			}
			err := json.Unmarshal([]byte(test.cfgjson), cfg.Config)
			assert.Nil(t, err)
			CheckConfig(cfg)
			assert.True(t, cfg.Config.Advanced.MaxTxFee == test.expect)
		})
	}
}
