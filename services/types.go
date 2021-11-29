package services

import (
	"context"
	"math/big"

	cfxSdkTypes "github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/coinbase/rosetta-sdk-go/types"
)

// Client is used by the servicers to get block
// data and to submit transactions.
type Client interface {
	Status(context.Context) (
		*types.BlockIdentifier,
		int64,
		*types.SyncStatus,
		[]*types.Peer,
		error,
	)

	Block(
		context.Context,
		*types.PartialBlockIdentifier,
	) (*types.Block, error)

	Balance(
		context.Context,
		*types.AccountIdentifier,
		*types.PartialBlockIdentifier,
	) (*types.AccountBalanceResponse, error)

	PendingNonceAt(context.Context, cfxSdkTypes.Address) (*big.Int, error)

	SuggestGasPrice(ctx context.Context) (*big.Int, error)

	EpochNumber(ctx context.Context) (*big.Int, error)

	SendTransaction(ctx context.Context, tx *cfxSdkTypes.SignedTransaction) error

	Call(
		ctx context.Context,
		request *types.CallRequest,
	) (*types.CallResponse, error)
}

type options struct {
	From string `json:"from"`
}

type metadata struct {
	Nonce       *big.Int `json:"nonce"`
	GasPrice    *big.Int `json:"gas_price"`
	EpochHeight uint64   `json:"epoch_height"`
}

type parseMetadata struct {
	Nonce       uint64   `json:"nonce"`
	GasPrice    *big.Int `json:"gas_price"`
	EpochHeight uint64   `json:"epoch_height"`
	ChainID     *big.Int `json:"chain_id"`
}
