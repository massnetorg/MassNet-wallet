package config

import (
	corecfg "github.com/massnetorg/mass-core/config"
)

type Params = corecfg.Params

var ChainParams *Params

func init() {
	ChainParams = &corecfg.ChainParams
}
