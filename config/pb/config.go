package configpb

func NewConfig() *Config {
	return &Config{
		Common: &CommonConfig{},
		App:    &AppConfig{},
		Network: &NetworkConfig{
			P2P: &P2PConfig{
				AddPeer: make([]string, 0),
			},
			API: new(APIConfig),
		},
		Db:    &DataConfig{},
		Log:   &LogConfig{},
		Chain: &ChainConfig{},
		Pool:  &MemPoolConfig{},
		Wallet: &WalletConfig{},
	}
}
