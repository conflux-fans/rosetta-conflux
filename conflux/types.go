// Copyright 2020 Coinbase, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package conflux

import (
	"context"
	"math/big"

	cfxSdkTypes "github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/rpc"
)

const (
	// NodeVersion is the version of conflux full node we are using.
	NodeVersion = "1.1.6"

	// Blockchain is conflux.
	Blockchain string = "Conflux"

	// MainnetNetwork is the value of the network
	// in MainnetNetworkIdentifier.
	MainnetNetwork string = "Mainnet"

	TestnetNetwork string = "Testnet"

	// // RopstenNetwork is the value of the network
	// // in RopstenNetworkIdentifier.
	// RopstenNetwork string = "Ropsten"

	// // RinkebyNetwork is the value of the network
	// // in RinkebyNetworkNetworkIdentifier.
	// RinkebyNetwork string = "Rinkeby"

	// // GoerliNetwork is the value of the network
	// // in GoerliNetworkNetworkIdentifier.
	// GoerliNetwork string = "Goerli"

	// Symbol is the symbol value
	// used in Currency.
	Symbol = "CFX"

	// Decimals is the decimals value
	// used in Currency.
	Decimals = 18

	// MinerRewardOpType is used to describe
	// a miner block reward.
	MinerRewardOpType = "MINER_REWARD"
	PosRewardOpType   = "POS_REWARD"

	// UncleRewardOpType is used to describe
	// an uncle block reward.
	// UncleRewardOpType = "UNCLE_REWARD"

	// FeeOpType is used to represent fee operations.
	FeeOpType              = "FEE"
	GasFeeOpType           = "GAS_FEE"
	StorageCollaterlOpType = "STORAGE_COLLATERAL_FEE"
	StorageReleaseOpType   = "STORAGE_RELEASE"

	// SelfDestructOpType is used to represent SELFDESTRUCT trace operations.
	SelfDestructOpType = "SELFDESTRUCT"

	// CallOpType is used to represent CALL trace operations.
	CallOpType = "CALL"

	// CallCodeOpType is used to represent CALLCODE trace operations.
	CallCodeOpType = "CALLCODE"

	// DelegateCallOpType is used to represent DELEGATECALL trace operations.
	DelegateCallOpType = "DELEGATECALL"

	// StaticCallOpType is used to represent STATICCALL trace operations.
	StaticCallOpType = "STATICCALL"

	// CreateOpType is used to represent CREATE trace operations.
	CreateOpType = cfxSdkTypes.CREATE_CREATE

	// Create2OpType is used to represent CREATE2 trace operations.
	Create2OpType = cfxSdkTypes.CREATE_CREATE2

	InternalTransferActionOpType = "INTERNALTRANSFERACTION"

	// DestructOpType is a synthetic operation used to represent the
	// deletion of suicided accounts that still have funds at the end
	// of a transaction.
	DestructOpType = "DESTRUCT"

	// SuccessStatus is the status of any
	// conflux operation considered successful.
	SuccessStatus = "SUCCESS"

	// FailureStatus is the status of any
	// conflux operation considered unsuccessful.
	FailureStatus = "FAILURE"

	// HistoricalBalanceSupported is whether
	// historical balance is supported.
	HistoricalBalanceSupported = true

	// UnclesRewardMultiplier is the uncle reward
	// multiplier.
	UnclesRewardMultiplier = 32

	// MaxUncleDepth is the maximum depth for
	// an uncle to be rewarded.
	MaxUncleDepth = 8

	// GenesisBlockIndex is the index of the
	// genesis block.
	GenesisBlockIndex = int64(0)

	// TransferGasLimit is the gas limit
	// of a transfer.
	TransferGasLimit = int64(21000) //nolint:gomnd

	// TransferGasLimit is the gas limit
	// of a transfer.
	TransferStorageLimit = int64(0) //nolint:gomnd

	// MainnetGethArguments are the arguments to start a mainnet geth instance.
	// MainnetGethArguments = `--config=/app/ethereum/geth.toml --gcmode=archive --graphql`

	// IncludeMempoolCoins does not apply to rosetta-ethereum as it is not UTXO-based.
	IncludeMempoolCoins = false
)

var (
	// // RopstenGethArguments are the arguments to start a ropsten geth instance.
	// RopstenGethArguments = fmt.Sprintf("%s --ropsten", MainnetGethArguments)

	// // RinkebyGethArguments are the arguments to start a rinkeby geth instance.
	// RinkebyGethArguments = fmt.Sprintf("%s --rinkeby", MainnetGethArguments)

	// // GoerliGethArguments are the arguments to start a ropsten geth instance.
	// GoerliGethArguments = fmt.Sprintf("%s --goerli", MainnetGethArguments)

	// MainnetGenesisBlockIdentifier is the *types.BlockIdentifier
	// of the mainnet genesis block.
	MainnetGenesisBlockIdentifier = &types.BlockIdentifier{
		Hash:  "0xf6cec5af1ee95f56038fc30589bcfeb4b960075c3c76b8a0eaa6d36e8e623b50", //params.MainnetGenesisHash.Hex(),
		Index: GenesisBlockIndex,
	}

	TestnetGenesisBlockIdentifier = &types.BlockIdentifier{
		Hash:  "0x3fb9d350f95424c3c27fcf248d9a6f633a83c2a75c11ef887ea41540c85e7804", //params.MainnetGenesisHash.Hex(),
		Index: GenesisBlockIndex,
	}

	// // RopstenGenesisBlockIdentifier is the *types.BlockIdentifier
	// // of the Ropsten genesis block.
	// RopstenGenesisBlockIdentifier = &types.BlockIdentifier{
	// 	Hash:  params.RopstenGenesisHash.Hex(),
	// 	Index: GenesisBlockIndex,
	// }

	// // RinkebyGenesisBlockIdentifier is the *types.BlockIdentifier
	// // of the Ropsten genesis block.
	// RinkebyGenesisBlockIdentifier = &types.BlockIdentifier{
	// 	Hash:  params.RinkebyGenesisHash.Hex(),
	// 	Index: GenesisBlockIndex,
	// }

	// // GoerliGenesisBlockIdentifier is the *types.BlockIdentifier
	// // of the Goerli genesis block.
	// GoerliGenesisBlockIdentifier = &types.BlockIdentifier{
	// 	Hash:  params.GoerliGenesisHash.Hex(),
	// 	Index: GenesisBlockIndex,
	// }

	// Currency is the *types.Currency for all
	// conflux networks.
	Currency = &types.Currency{
		Symbol:   Symbol,
		Decimals: Decimals,
	}

	// OperationTypes are all suppoorted operation types.
	OperationTypes = []string{
		MinerRewardOpType,
		PosRewardOpType,
		// UncleRewardOpType,
		// FeeOpType,

		// GasFeeOpType,
		// StorageCollaterlOpType,
		// StorageReleaseOpType,
		// CallOpType,
		string(CreateOpType),
		// CallResultOpType,
		// CreateResultOpType,
		InternalTransferActionOpType,

		// Create2OpType,
		// SelfDestructOpType,
		CallOpType,
		CallCodeOpType,
		DelegateCallOpType,
		StaticCallOpType,
		// DestructOpType,
	}

	// OperationStatuses are all supported operation statuses.
	OperationStatuses = []*types.OperationStatus{
		{
			Status:     SuccessStatus,
			Successful: true,
		},
		{
			Status:     FailureStatus,
			Successful: false,
		},
	}

	// CallMethods are all supported call methods.
	CallMethods = []string{
		// "eth_getBlockByNumber",
		// "eth_getTransactionReceipt",
	}
)

var (
	StorageUint = new(big.Int).Div(big.NewInt(1e18), big.NewInt(1024))
)

// JSONRPC is the interface for accessing go-ethereum's JSON RPC endpoint.
type JSONRPC interface {
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
	BatchCallContext(ctx context.Context, b []rpc.BatchElem) error
	Close()
}

// CallType returns a boolean indicating
// if the provided trace type is a call type.
func CallType(t string) bool {
	callTypes := []string{
		CallOpType,
		CallCodeOpType,
		DelegateCallOpType,
		StaticCallOpType,
	}

	for _, callType := range callTypes {
		if callType == t {
			return true
		}
	}

	return false
}

// CreateType returns a boolean indicating
// if the provided trace type is a create type.
func CreateType(t string) bool {
	createTypes := []string{
		string(CreateOpType),
		string(Create2OpType),
	}

	for _, createType := range createTypes {
		if createType == t {
			return true
		}
	}

	return false
}
