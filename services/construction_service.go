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

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/Conflux-Chain/go-conflux-sdk/types/cfxaddress"
	"github.com/Conflux-Chain/go-conflux-sdk/utils/addressutil"
	"github.com/conflux-fans/rosetta-conflux/common"
	"github.com/conflux-fans/rosetta-conflux/configuration"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/sirupsen/logrus"

	"github.com/coinbase/rosetta-sdk-go/parser"
	"github.com/coinbase/rosetta-sdk-go/types"

	cfxSdkTypes "github.com/Conflux-Chain/go-conflux-sdk/types"
	"github.com/conflux-fans/rosetta-conflux/conflux"
)

// ConstructionAPIService implements the server.ConstructionAPIServicer interface.
type ConstructionAPIService struct {
	config *configuration.Configuration
	client Client
}

// NewConstructionAPIService creates a new instance of a ConstructionAPIService.
func NewConstructionAPIService(
	cfg *configuration.Configuration,
	client Client,
) *ConstructionAPIService {
	return &ConstructionAPIService{
		config: cfg,
		client: client,
	}
}

// ConstructionDerive implements the /construction/derive endpoint.
func (s *ConstructionAPIService) ConstructionDerive(
	ctx context.Context,
	request *types.ConstructionDeriveRequest,
) (*types.ConstructionDeriveResponse, *types.Error) {
	pubkey, err := crypto.DecompressPubkey(request.PublicKey.Bytes)
	if err != nil {
		return nil, wrapErr(ErrUnableToDecompressPubkey, err)
	}

	ethAddr := crypto.PubkeyToAddress(*pubkey)

	networkId, err := s.client.NetworkID()
	if err != nil {
		return nil, wrapErr(ErrUnrecognizedNetwork, err)
	}

	cfxAddr := addressutil.EtherAddressToCfxAddress(ethAddr, false, networkId)
	return &types.ConstructionDeriveResponse{
		AccountIdentifier: &types.AccountIdentifier{
			Address: cfxAddr.String(),
		},
	}, nil
}

// ConstructionPreprocess implements the /construction/preprocess
// endpoint.
func (s *ConstructionAPIService) ConstructionPreprocess(
	ctx context.Context,
	request *types.ConstructionPreprocessRequest,
) (*types.ConstructionPreprocessResponse, *types.Error) {
	descriptions := &parser.Descriptions{
		OperationDescriptions: []*parser.OperationDescription{
			{
				Type: conflux.CallOpType,
				Account: &parser.AccountDescription{
					Exists: true,
				},
				Amount: &parser.AmountDescription{
					Exists:   true,
					Sign:     parser.NegativeAmountSign,
					Currency: conflux.Currency,
				},
			},
			{
				Type: conflux.CallOpType,
				Account: &parser.AccountDescription{
					Exists: true,
				},
				Amount: &parser.AmountDescription{
					Exists:   true,
					Sign:     parser.PositiveAmountSign,
					Currency: conflux.Currency,
				},
			},
		},
		ErrUnmatched: true,
	}

	matches, err := parser.MatchOperations(descriptions, request.Operations)
	if err != nil {
		return nil, wrapErr(ErrUnclearIntent, err)
	}

	fromOp, _ := matches[0].First()
	fromAdd := fromOp.Account.Address
	toOp, _ := matches[1].First()
	toAdd := toOp.Account.Address

	// Ensure valid from address
	checkFrom, err := cfxaddress.New(fromAdd)
	if err != nil {
		return nil, wrapErr(ErrInvalidAddress, fmt.Errorf("%s is not a valid address", fromAdd))
	}

	// Ensure valid to address
	_, err = cfxaddress.New(toAdd)
	if err != nil {
		return nil, wrapErr(ErrInvalidAddress, fmt.Errorf("%s is not a valid address", toAdd))
	}

	preprocessOutput := &options{
		From: checkFrom.String(),
	}

	marshaled, err := common.MarshalToMap(preprocessOutput)
	if err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}

	return &types.ConstructionPreprocessResponse{
		Options: marshaled,
	}, nil
}

// ConstructionMetadata implements the /construction/metadata endpoint.
func (s *ConstructionAPIService) ConstructionMetadata(
	ctx context.Context,
	request *types.ConstructionMetadataRequest,
) (*types.ConstructionMetadataResponse, *types.Error) {

	var input options
	if err := common.UnmarshalMap(request.Options, &input); err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}

	addr, err := cfxaddress.New(input.From)
	if err != nil {
		return nil, wrapErr(ErrInvalidAddress, err)
	}

	nonce, err := s.client.PendingNonceAt(ctx, addr)
	if err != nil {
		return nil, wrapErr(ErrGeth, err)
	}
	gasPrice, err := s.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, wrapErr(ErrGeth, err)
	}
	epochNum, err := s.client.EpochNumber(ctx)
	if err != nil {
		return nil, wrapErr(ErrGeth, err)
	}

	metadata := &metadata{
		Nonce:       nonce,
		GasPrice:    gasPrice,
		EpochHeight: epochNum.Uint64(),
	}

	metadataMap, err := common.MarshalToMap(metadata)
	if err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}

	// Find suggested gas usage
	suggestedFee := metadata.GasPrice.Int64() * conflux.TransferGasLimit

	return &types.ConstructionMetadataResponse{
		Metadata: metadataMap,
		SuggestedFee: []*types.Amount{
			{
				Value:    strconv.FormatInt(suggestedFee, 10),
				Currency: conflux.Currency,
			},
		},
	}, nil
}

// ConstructionPayloads implements the /construction/payloads endpoint.
func (s *ConstructionAPIService) ConstructionPayloads(
	ctx context.Context,
	request *types.ConstructionPayloadsRequest,
) (*types.ConstructionPayloadsResponse, *types.Error) {
	descriptions := &parser.Descriptions{
		OperationDescriptions: []*parser.OperationDescription{
			{
				Type: conflux.CallOpType,
				Account: &parser.AccountDescription{
					Exists: true,
				},
				Amount: &parser.AmountDescription{
					Exists:   true,
					Sign:     parser.NegativeAmountSign,
					Currency: conflux.Currency,
				},
			},
			{
				Type: conflux.CallOpType,
				Account: &parser.AccountDescription{
					Exists: true,
				},
				Amount: &parser.AmountDescription{
					Exists:   true,
					Sign:     parser.PositiveAmountSign,
					Currency: conflux.Currency,
				},
			},
		},
		ErrUnmatched: true,
	}
	matches, err := parser.MatchOperations(descriptions, request.Operations)
	if err != nil {
		return nil, wrapErr(ErrUnclearIntent, err)
	}

	// Convert map to Metadata struct
	var metadata metadata
	if err := common.UnmarshalMap(request.Metadata, &metadata); err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}

	chainID, err := s.client.ChainID()
	if err := common.UnmarshalMap(request.Metadata, &metadata); err != nil {
		return nil, wrapErr(ErrGeth, err)
	}
	// Required Fields for constructing a real Ethereum transaction
	toOp, amount := matches[1].First()
	toAdd := toOp.Account.Address
	nonce := metadata.Nonce
	gasPrice := metadata.GasPrice

	transferGasLimit := uint64(conflux.TransferGasLimit)
	transferStorageLimit := uint64(conflux.TransferStorageLimit)
	transferData := []byte{}

	// Additional Fields for constructing custom Ethereum tx struct
	fromOp, _ := matches[0].First()
	fromAdd := fromOp.Account.Address

	// Ensure valid from address
	checkFrom, err := cfxaddress.New(fromAdd)
	if err != nil {
		return nil, wrapErr(ErrInvalidAddress, fmt.Errorf("%s is not a valid address", fromAdd))
	}

	// Ensure valid to address
	checkTo, err := cfxaddress.New(toAdd)
	if err != nil {
		return nil, wrapErr(ErrInvalidAddress, fmt.Errorf("%s is not a valid address", toAdd))
	}

	utx := cfxSdkTypes.UnsignedTransaction{
		UnsignedTransactionBase: cfxSdkTypes.UnsignedTransactionBase{
			From:         &checkFrom,
			Nonce:        (*hexutil.Big)(nonce),
			GasPrice:     (*hexutil.Big)(gasPrice),
			Gas:          cfxSdkTypes.NewBigInt(transferGasLimit),
			Value:        (*hexutil.Big)(amount),
			StorageLimit: cfxSdkTypes.NewUint64(transferStorageLimit),
			EpochHeight:  cfxSdkTypes.NewUint64(metadata.EpochHeight),
			ChainID:      cfxSdkTypes.NewUint(uint(chainID)),
		},
		To:   &checkTo,
		Data: transferData,
	}

	// Construct SigningPayload
	hash, err := utx.Hash()
	if err != nil {
		return nil, wrapErr(ErrgGetTxHash, err)
	}

	payload := &types.SigningPayload{
		AccountIdentifier: &types.AccountIdentifier{Address: checkFrom.String()},
		Bytes:             hash,
		SignatureType:     types.EcdsaRecovery,
	}

	unsignedTxJSON, err := json.Marshal(utx)
	if err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}

	return &types.ConstructionPayloadsResponse{
		UnsignedTransaction: string(unsignedTxJSON),
		Payloads:            []*types.SigningPayload{payload},
	}, nil
}

// ConstructionCombine implements the /construction/combine
// endpoint.
func (s *ConstructionAPIService) ConstructionCombine(
	ctx context.Context,
	request *types.ConstructionCombineRequest,
) (*types.ConstructionCombineResponse, *types.Error) {
	utx := cfxSdkTypes.UnsignedTransaction{}
	if err := json.Unmarshal([]byte(request.UnsignedTransaction), &utx); err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}

	sig := request.Signatures[0].Bytes
	V, R, S := sig[64], sig[0:32], sig[32:64]

	tx := cfxSdkTypes.SignedTransaction{utx, V, R, S}
	txJson, err := json.Marshal(tx)
	if err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}

	return &types.ConstructionCombineResponse{
		SignedTransaction: string(txJson),
	}, nil
}

// ConstructionHash implements the /construction/hash endpoint.
func (s *ConstructionAPIService) ConstructionHash(
	ctx context.Context,
	request *types.ConstructionHashRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	signedTx := cfxSdkTypes.SignedTransaction{}
	if err := json.Unmarshal([]byte(request.SignedTransaction), &signedTx); err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}

	txhash, err := signedTx.Hash()
	if err != nil {
		return nil, wrapErr(ErrgGetTxHash, err)
	}

	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: &types.TransactionIdentifier{
			Hash: hexutil.Bytes(txhash).String(),
		},
	}, nil
}

// ConstructionParse implements the /construction/parse endpoint.
func (s *ConstructionAPIService) ConstructionParse(
	ctx context.Context,
	request *types.ConstructionParseRequest,
) (*types.ConstructionParseResponse, *types.Error) {
	utx := cfxSdkTypes.UnsignedTransaction{}
	if !request.Signed {
		err := json.Unmarshal([]byte(request.Transaction), &utx)
		if err != nil {
			return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
		}
	} else {
		tx := cfxSdkTypes.SignedTransaction{}
		err := json.Unmarshal([]byte(request.Transaction), &tx)
		logrus.Debugf("request.Transaction %v\n", request.Transaction)
		if err != nil {
			return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
		}

		chainID, err := s.client.ChainID()
		if err != nil {
			return nil, wrapErr(ErrGeth, err)
		}

		from, err := tx.Sender(chainID)
		if err != nil {
			return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
		}

		utx = tx.UnsignedTransaction
		utx.From = &from
	}

	ops := []*types.Operation{
		{
			Type: conflux.CallOpType,
			OperationIdentifier: &types.OperationIdentifier{
				Index: 0,
			},
			Account: &types.AccountIdentifier{
				Address: utx.From.String(),
			},
			Amount: &types.Amount{
				Value:    new(big.Int).Neg(utx.Value.ToInt()).String(),
				Currency: conflux.Currency,
			},
		},
		{
			Type: conflux.CallOpType,
			OperationIdentifier: &types.OperationIdentifier{
				Index: 1,
			},
			RelatedOperations: []*types.OperationIdentifier{
				{
					Index: 0,
				},
			},
			Account: &types.AccountIdentifier{
				Address: utx.To.String(),
			},
			Amount: &types.Amount{
				Value:    utx.Value.ToInt().String(),
				Currency: conflux.Currency,
			},
		},
	}

	metadata := &parseMetadata{
		Nonce:       utx.Nonce.ToInt().Uint64(),
		GasPrice:    utx.GasPrice.ToInt(),
		ChainID:     big.NewInt(int64(*utx.ChainID)),
		EpochHeight: uint64(*utx.EpochHeight),
	}
	metaMap, err := common.MarshalToMap(metadata)
	if err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}

	var resp *types.ConstructionParseResponse
	if request.Signed {
		resp = &types.ConstructionParseResponse{
			Operations: ops,
			AccountIdentifierSigners: []*types.AccountIdentifier{
				{
					Address: utx.From.String(),
				},
			},
			Metadata: metaMap,
		}
	} else {
		resp = &types.ConstructionParseResponse{
			Operations:               ops,
			AccountIdentifierSigners: []*types.AccountIdentifier{},
			Metadata:                 metaMap,
		}
	}
	return resp, nil
}

// ConstructionSubmit implements the /construction/submit endpoint.
func (s *ConstructionAPIService) ConstructionSubmit(
	ctx context.Context,
	request *types.ConstructionSubmitRequest,
) (*types.TransactionIdentifierResponse, *types.Error) {
	if s.config.Mode != configuration.Online {
		return nil, ErrUnavailableOffline
	}

	tx := cfxSdkTypes.SignedTransaction{}
	err := json.Unmarshal([]byte(request.SignedTransaction), &tx)
	if err != nil {
		return nil, wrapErr(ErrUnableToParseIntermediateResult, err)
	}

	_txhash, _err := tx.Hash()
	fmt.Printf("tx hash %x error %v\n", _txhash, _err)

	if err := s.client.SendTransaction(ctx, &tx); err != nil {
		return nil, wrapErr(ErrBroadcastFailed, err)
	}

	txhash, err := tx.Hash()
	if err != nil {
		return nil, wrapErr(ErrgGetTxHash, err)
	}

	txIdentifier := &types.TransactionIdentifier{
		Hash: hexutil.Bytes(txhash).String(),
	}
	return &types.TransactionIdentifierResponse{
		TransactionIdentifier: txIdentifier,
	}, nil
}
