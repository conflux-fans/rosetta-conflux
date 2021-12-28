package configuration

import "math/big"

var (
	MainnetChainConfig *ChainConfig = &ChainConfig{
		ChainID: big.NewInt(1029),
	}

	TestnetChainConfig *ChainConfig = &ChainConfig{
		ChainID: big.NewInt(1),
	}
)
