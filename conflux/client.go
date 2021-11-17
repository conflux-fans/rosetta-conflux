package conflux

import (
	"context"
	"math/big"

	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	cfxSdkTypes "github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

type Client struct {
	// p  *params.ChainConfig
	// tc *tracers.TraceConfig

	c sdk.ClientOperator
	// g GraphQL

	// traceSemaphore *semaphore.Weighted

	// skipAdminCalls bool
}

func NewClient(url string) (*Client, error) {
	c, err := sdk.NewClient(url, sdk.ClientOption{RetryCount: 3})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create client")
	}
	return &Client{c}, nil
}

func (ec *Client) Status(_ context.Context) (*types.BlockIdentifier, int64, *types.SyncStatus, []*types.Peer, error) {
	panic("not implemented") // TODO: Implement
}

func (ec *Client) Block(_ context.Context, _ *types.PartialBlockIdentifier) (*types.Block, error) {
	panic("not implemented") // TODO: Implement
}

func (ec *Client) Balance(_ context.Context, _ *types.AccountIdentifier, _ *types.PartialBlockIdentifier) (*types.AccountBalanceResponse, error) {
	panic("not implemented") // TODO: Implement
}

func (ec *Client) PendingNonceAt(_ context.Context, _ common.Address) (uint64, error) {
	panic("not implemented") // TODO: Implement
}

func (ec *Client) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	panic("not implemented") // TODO: Implement
}

func (ec *Client) SendTransaction(ctx context.Context, tx *cfxSdkTypes.SignedTransaction) error {
	panic("not implemented") // TODO: Implement
}

func (ec *Client) Call(ctx context.Context, request *types.CallRequest) (*types.CallResponse, error) {
	panic("not implemented") // TODO: Implement
}

// Close shuts down the RPC client connection.
func (ec *Client) Close() {
	ec.c.Close()
}
