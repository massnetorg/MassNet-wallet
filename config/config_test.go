package config

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	cfgjson = `{
		"core": {
			"p2p": {
				"seeds": "",
				"listen_address": "tcp://0.0.0.0:43453"
			},
			"log": {
				"log_dir": "./logs"
			},
			"datastore": {
				"dir": "./chain"
			}
		},
		"wallet": {
			"api": {
				"host": "localhost",
				"http_port": "9686",
				"disable_tls": true
			},
			"settings": {
				"max_tx_fee": 1.0
			}
		}
	}`
)

func ExampleCheck() {
	cfg := &Config{
		Core:   NewDefCoreConfig(),
		Wallet: NewDefWalletConfig(),
	}
	json.Unmarshal([]byte(cfgjson), cfg)
	CheckConfig(cfg)

	data, _ := json.MarshalIndent(cfg, "", "    ")

	fmt.Println(string(data))

	// Output:
	// {
	//     "core": {
	//         "chain": {
	//             "disable_checkpoints": false,
	//             "add_checkpoints": null
	//         },
	//         "metrics": {
	//             "profile_port": ""
	//         },
	//         "p2p": {
	//             "seeds": "",
	//             "add_peer": [],
	//             "skip_upnp": false,
	//             "handshake_timeout": 30,
	//             "dial_timeout": 3,
	//             "vault_mode": false,
	//             "listen_address": "tcp://0.0.0.0:43453"
	//         },
	//         "log": {
	//             "log_dir": "logs",
	//             "log_level": "info",
	//             "disable_cprint": false
	//         },
	//         "datastore": {
	//             "dir": "chain",
	//             "db_type": "leveldb"
	//         },
	//         "influxdb": {
	//             "run": false,
	//             "url": "",
	//             "database": "",
	//             "username": "",
	//             "password": "",
	//             "hostname": "",
	//             "tags": null
	//         }
	//     },
	//     "wallet": {
	//         "pub_pass": "1234567890",
	//         "api": {
	//             "host": "localhost",
	//             "grpc_port": "9687",
	//             "http_port": "9686",
	//             "http_cors_addr": [
	//                 "localhost"
	//             ],
	//             "disable_tls": true,
	//             "rpc_cert": "",
	//             "rpc_key": ""
	//         },
	//         "settings": {
	//             "address_gap_limit": 20,
	//             "max_unused_staking_address": 8,
	//             "max_tx_fee": "1.0"
	//         }
	//     }
	// }

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
				"wallet": {
					"settings": {}
				}
			}`,
			DefaultMaxTxFee,
		},
		{
			"valid",
			`{
				"wallet": {
					"settings": {
						"max_tx_fee": "5.555"
					}
				}
			}`,
			"5.555",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := &Config{
				Core:   NewDefCoreConfig(),
				Wallet: NewDefWalletConfig(),
			}
			err := json.Unmarshal([]byte(test.cfgjson), cfg)
			assert.Nil(t, err)
			CheckConfig(cfg)
			assert.True(t, cfg.Wallet.Settings.MaxTxFee == test.expect)

		})
	}
}
