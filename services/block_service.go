package services

import (
	"context"
	"errors"

	"github.com/coinbase/rosetta-sdk-go/types"
	"github.com/conflux-fans/rosetta-conflux/configuration"
	"github.com/conflux-fans/rosetta-conflux/conflux"
)

// BlockAPIService implements the server.BlockAPIServicer interface.
type BlockAPIService struct {
	config *configuration.Configuration
	client Client
}

// NewBlockAPIService creates a new instance of a BlockAPIService.
func NewBlockAPIService(
	cfg *configuration.Configuration,
	client Client,
) *BlockAPIService {
	return &BlockAPIService{
		config: cfg,
		client: client,
	}
}

// Block implements the /block endpoint.
func (s *BlockAPIService) Block(
	ctx context.Context,
	request *types.BlockRequest,
) (*types.BlockResponse, *types.Error) {
	if s.config.Mode != configuration.Online {
		return nil, ErrUnavailableOffline
	}

	block, err := s.client.Block(ctx, request.BlockIdentifier)
	// TODO: Need change ErrBlockOrphaned name
	if errors.Is(err, conflux.ErrBlockOrphaned) {
		return nil, wrapErr(ErrBlockOrphaned, err)
	}
	if err != nil {
		return nil, wrapErr(ErrGeth, err)
	}

	return &types.BlockResponse{
		Block: block,
	}, nil
}

// BlockTransaction implements the /block/transaction endpoint.
func (s *BlockAPIService) BlockTransaction(
	ctx context.Context,
	request *types.BlockTransactionRequest,
) (*types.BlockTransactionResponse, *types.Error) {
	return nil, wrapErr(ErrUnimplemented, nil)
}
