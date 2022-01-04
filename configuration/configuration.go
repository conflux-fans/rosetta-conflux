package configuration

import (
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/conflux-fans/rosetta-conflux/conflux"
)

type Mode string

const (
	// Online is when the implementation is permitted
	// to make outbound connections.
	Online Mode = "ONLINE"

	// Offline is when the implementation is not permitted
	// to make outbound connections.
	Offline Mode = "OFFLINE"

	// Mainnet is the conflux Mainnet.
	Mainnet string = "MAINNET"

	// Testnet defaults to `Ropsten` for backwards compatibility.
	Testnet string = "TESTNET"

	// DataDirectory is the default location for all
	// persistent data.
	DataDirectory = "/data"

	// ModeEnv is the environment variable read
	// to determine mode.
	ModeEnv = "MODE"

	// NetworkEnv is the environment variable
	// read to determine network.
	NetworkEnv = "NETWORK"

	// PortEnv is the environment variable
	// read to determine the port for the Rosetta
	// implementation.
	PortEnv = "PORT"

	// CFXNodeEnv is an optional environment variable
	// used to connect rosetta-conflux to an already
	// running CFX node.
	CFXNodeEnv = "CFXNODE"

	// DefaultCFXNodeURL is the default URL for
	// a running CFX node. This is used
	// when CFXNodeEnv is not populated.
	DefaultCFXNodeURL = "https://test.confluxrpc.com"

	// SkipGethAdminEnv is an optional environment variable
	// to skip geth `admin` calls which are typically not supported
	// by hosted node services. When not set, defaults to false.
	// SkipGethAdminEnv = "SKIP_GETH_ADMIN"

	// MiddlewareVersion is the version of rosetta-conflux.
	MiddlewareVersion = "0.0.4"
)

type Configuration struct {
	Mode                   Mode
	Network                *types.NetworkIdentifier
	GenesisBlockIdentifier *types.BlockIdentifier
	CFXNodeURL             string
	RemoteCFXNode          bool
	Port                   int
	// GethArguments          string
	// SkipGethAdmin          bool

	// Block Reward Data
	Params *ChainConfig
}

type ChainConfig struct {
	ChainID *big.Int `json:"chainId"` // chainId identifies the current chain and is used for replay protection
}

// LoadConfiguration attempts to create a new Configuration
// using the ENVs in the environment.
func LoadConfiguration() (*Configuration, error) {
	config := &Configuration{}

	modeValue := Mode(os.Getenv(ModeEnv))
	switch modeValue {
	case Online:
		config.Mode = Online
	case Offline:
		config.Mode = Offline
	case "":
		return nil, errors.New("MODE must be populated")
	default:
		return nil, fmt.Errorf("%s is not a valid mode", modeValue)
	}

	networkValue := os.Getenv(NetworkEnv)
	switch networkValue {
	case Mainnet:
		config.Network = &types.NetworkIdentifier{
			Blockchain: conflux.Blockchain,
			Network:    conflux.MainnetNetwork,
		}
		config.GenesisBlockIdentifier = conflux.MainnetGenesisBlockIdentifier
		config.Params = MainnetChainConfig

	case Testnet:
		config.Network = &types.NetworkIdentifier{
			Blockchain: conflux.Blockchain,
			Network:    conflux.TestnetNetwork,
		}
		config.GenesisBlockIdentifier = conflux.TestnetGenesisBlockIdentifier
		config.Params = TestnetChainConfig

	case "":
		return nil, errors.New("NETWORK must be populated")
	default:

		return nil, fmt.Errorf("%s is not a valid network", networkValue)
	}

	config.CFXNodeURL = DefaultCFXNodeURL
	envCFXNodeURL := os.Getenv(CFXNodeEnv)
	if len(envCFXNodeURL) > 0 {
		config.RemoteCFXNode = true
		config.CFXNodeURL = envCFXNodeURL
	}

	// config.SkipGethAdmin = false
	// envSkipGethAdmin := os.Getenv(SkipGethAdminEnv)
	// if len(envSkipGethAdmin) > 0 {
	// 	val, err := strconv.ParseBool(envSkipGethAdmin)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("%w: unable to parse SKIP_GETH_ADMIN %s", err, envSkipGethAdmin)
	// 	}
	// 	config.SkipGethAdmin = val
	// }

	portValue := os.Getenv(PortEnv)
	if len(portValue) == 0 {
		return nil, errors.New("PORT must be populated")
	}

	port, err := strconv.Atoi(portValue)
	if err != nil || len(portValue) == 0 || port <= 0 {
		return nil, fmt.Errorf("%w: unable to parse port %s", err, portValue)
	}
	config.Port = port

	return config, nil
}
