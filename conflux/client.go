package conflux

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/cfxclient/bulk"
	cfxSdkTypes "github.com/Conflux-Chain/go-conflux-sdk/types"
	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

func (ec *Client) Status(_ context.Context) (*RosettaTypes.BlockIdentifier, int64, *RosettaTypes.SyncStatus, []*RosettaTypes.Peer, error) {
	panic("not implemented") // TODO: Implement
}

func (ec *Client) Block(ctx context.Context, blockIdentifier *RosettaTypes.PartialBlockIdentifier) (*RosettaTypes.Block, error) {
	return ec.getParsedBlock(ctx, blockIdentifier)
}

func (ec *Client) getParsedBlock(
	ctx context.Context,
	_blockIdentifier *RosettaTypes.PartialBlockIdentifier,
) (
	*RosettaTypes.Block,
	error,
) {
	block, loadedTransactions, err := ec.getBlock(ctx, _blockIdentifier)
	if err != nil {
		return nil, fmt.Errorf("%w: could not get block", err)
	}

	blockIdentifier := &RosettaTypes.BlockIdentifier{
		Hash:  block.Hash.String(),
		Index: block.BlockNumber.ToInt().Int64(),
	}

	parentBlockIdentifier := blockIdentifier
	if blockIdentifier.Index != GenesisBlockIndex {
		parentBlockIdentifier = &RosettaTypes.BlockIdentifier{
			Hash:  block.ParentHash.String(),
			Index: blockIdentifier.Index - 1,
		}
	}

	txs, err := ec.populateTransactions(block, loadedTransactions)
	if err != nil {
		return nil, err
	}

	return &RosettaTypes.Block{
		BlockIdentifier:       blockIdentifier,
		ParentBlockIdentifier: parentBlockIdentifier,
		Timestamp:             convertTime(block.Timestamp.ToInt().Uint64()),
		Transactions:          txs,
	}, nil
}

func (ec *Client) populateTransactions(
	block *cfxSdkTypes.Block,
	loadedTransactions []*loadedTransaction) ([]*RosettaTypes.Transaction, error) {
	txs := make([]*RosettaTypes.Transaction, len(loadedTransactions)+1)
	rewardTx, err := ec.blockRewardTransaction(block)
	if err != nil {
		return nil, err
	}
	txs[0] = rewardTx

	for i, tx := range loadedTransactions {
		populated, err := ec.populateTransaction(tx)
		if err != nil {
			return nil, errors.Wrapf(err, "%w: cannot parse %s", tx.Transaction.Hash)
		}
		txs[i+1] = populated
	}
	return txs, nil
}

func (ec *Client) blockRewardTransaction(block *cfxSdkTypes.Block) (*RosettaTypes.Transaction, error) {
	reward, err := ec.getBlockReward(block)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get block reward")
	}

	rewardExceptTxfee := big.NewInt(0).Sub(reward.TotalReward.ToInt(), reward.TxFee.ToInt())

	var ops []*RosettaTypes.Operation
	miningRewardOp := &RosettaTypes.Operation{
		OperationIdentifier: &RosettaTypes.OperationIdentifier{
			Index: 0,
		},
		Type:   MinerRewardOpType,
		Status: RosettaTypes.String(SuccessStatus),
		Account: &RosettaTypes.AccountIdentifier{
			Address: block.Miner.String(),
		},
		Amount: &RosettaTypes.Amount{
			Value:    rewardExceptTxfee.String(),
			Currency: Currency,
		},
	}
	ops = append(ops, miningRewardOp)

	return &RosettaTypes.Transaction{
		TransactionIdentifier: &RosettaTypes.TransactionIdentifier{
			Hash: block.Hash.String(),
		},
		Operations: ops,
	}, nil
}

func (ec *Client) populateTransaction(
	tx *loadedTransaction,
) (*RosettaTypes.Transaction, error) {
	ops := []*RosettaTypes.Operation{}

	// Compute fee operations
	feeOps := feeOps(tx)
	ops = append(ops, feeOps...)

	// Compute trace operations
	// traces := flattenTraces(tx.Trace, []*flatCall{})
	traceOps, err := traceOps(tx.Trace, int64(len(ops)))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	ops = append(ops, traceOps...)

	receiptMap, err := objToMap(tx.Receipt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert receipt to mapping")
	}

	traceMap, err := objToMap(tx.Trace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert trace to mapping")
	}

	populatedTransaction := &RosettaTypes.Transaction{
		TransactionIdentifier: &RosettaTypes.TransactionIdentifier{
			Hash: tx.Transaction.Hash.String(),
		},
		Operations: ops,
		Metadata: map[string]interface{}{
			"gas_limit": tx.Transaction.Gas,
			"gas_price": tx.Transaction.GasPrice,
			"receipt":   receiptMap,
			"trace":     traceMap,
		},
	}

	return populatedTransaction, nil
}

func objToMap(obj interface{}) (val map[string]interface{}, err error) {
	b, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(b, &val)
	return
}

// ============= gen operates =============

func feeOps(ltx *loadedTransaction) []*RosettaTypes.Operation {
	header, tx, receipt := ltx.BlockHeader, ltx.Transaction, ltx.Receipt
	gasFee := (*big.Int)(receipt.GasFee)
	return []*RosettaTypes.Operation{
		{
			OperationIdentifier: &RosettaTypes.OperationIdentifier{
				Index: 0,
			},
			Type:   FeeOpType,
			Status: RosettaTypes.String(SuccessStatus),
			Account: &RosettaTypes.AccountIdentifier{
				Address: tx.From.String(),
			},
			Amount: &RosettaTypes.Amount{
				Value:    new(big.Int).Neg(gasFee).String(),
				Currency: Currency,
			},
		},

		{
			OperationIdentifier: &RosettaTypes.OperationIdentifier{
				Index: 1,
			},
			RelatedOperations: []*RosettaTypes.OperationIdentifier{
				{
					Index: 0,
				},
			},
			Type:   FeeOpType,
			Status: RosettaTypes.String(SuccessStatus),
			Account: &RosettaTypes.AccountIdentifier{
				Address: header.Miner.String(),
			},
			Amount: &RosettaTypes.Amount{
				Value:    gasFee.String(),
				Currency: Currency,
			},
		},
	}
}

func traceOps(traces []cfxSdkTypes.LocalizedTrace, startIndex int64) ([]*RosettaTypes.Operation, error) {
	var ops []*RosettaTypes.Operation
	if len(traces) == 0 {
		return ops, nil
	}

	tree, err := cfxSdkTypes.TraceInTree(traces)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	flattened := tree.Flatten()
	for _, trace := range flattened {
		from, to, opType, value, status, err := getOpElems(*trace)
		metadata := getOpMetadata(*trace)

		if err != nil {
			return nil, errors.WithStack(err)
		}

		// if call and value is 0 skip
		if trace.Raw.Type == cfxSdkTypes.CALL_TYPE && value.Cmp(big.NewInt(0)) == 0 {
			continue
		}

		fromOp := &RosettaTypes.Operation{
			OperationIdentifier: &RosettaTypes.OperationIdentifier{
				Index: startIndex,
			},
			Type:   opType,
			Status: RosettaTypes.String(status),
			Account: &RosettaTypes.AccountIdentifier{
				Address: from.String(),
			},
			Amount: &RosettaTypes.Amount{
				Value:    new(big.Int).Neg(value).String(),
				Currency: Currency,
			},
			Metadata: metadata,
		}

		toOp := &RosettaTypes.Operation{
			OperationIdentifier: &RosettaTypes.OperationIdentifier{
				Index: startIndex + 1,
			},
			RelatedOperations: []*RosettaTypes.OperationIdentifier{
				{
					Index: startIndex,
				},
			},
			Type:   opType,
			Status: RosettaTypes.String(status),
			Account: &RosettaTypes.AccountIdentifier{
				Address: to.String(),
			},
			Amount: &RosettaTypes.Amount{
				Value:    value.String(),
				Currency: Currency,
			},
			Metadata: metadata,
		}

		ops = append(ops, fromOp, toOp)

		startIndex += 2
	}

	// FIXME:
	// create two: current same to create?
	// No Type: suiside and internal transfer action need detail type?

	return ops, nil
}

func getOpElems(node cfxSdkTypes.LocalizedTraceNode) (
	from cfxSdkTypes.Address, to cfxSdkTypes.Address,
	traceType string, value *big.Int, status string, err error) {
	switch node.Raw.Type {
	case cfxSdkTypes.CALL_TYPE:
		r := node.CallWithResult
		return r.From, r.To, strings.ToUpper(r.CallType), r.Value.ToInt(), getOpStatus(r.Outcome), nil

	case cfxSdkTypes.CREATE_TYPE:
		r := node.CreateWithResult
		return r.From, r.Addr, CreateOpType, r.Value.ToInt(), getOpStatus(r.Outcome), nil

	case cfxSdkTypes.INTERNAL_TRANSFER_ACTIION_TYPE:
		r := node.InternalTransferAction
		return r.From, r.To, InternalTransferActionOpType, r.Value.ToInt(), SuccessStatus, nil
	}
	err = errors.New("unsupported trace type")
	return
}

func getOpStatus(outcome string) string {
	if outcome == "success" {
		return SuccessStatus
	}
	return FailureStatus
}

func getOpMetadata(node cfxSdkTypes.LocalizedTraceNode) map[string]interface{} {
	if node.Raw.Type == cfxSdkTypes.CALL_TYPE {
		if node.CallWithResult.Outcome == "success" {
			return nil
		}

		return map[string]interface{}{
			"error": node.CallWithResult.ReturnData,
		}
	}

	if node.Raw.Type == cfxSdkTypes.CREATE_TYPE {
		if node.CreateWithResult.Outcome == "success" {
			return nil
		}

		return map[string]interface{}{
			"error": node.CreateWithResult.ReturnData,
		}
	}
	return nil
}

// ====================================================

func (ec *Client) getBlock(
	ctx context.Context,
	_blockIdentifier *RosettaTypes.PartialBlockIdentifier,

) (
	*cfxSdkTypes.Block,
	[]*loadedTransaction,
	error,
) {
	// get block
	rpcBlock, err := ec.getRpcBlock(_blockIdentifier)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get raw block by rpc")
	}

	rpcTxs := rpcBlock.Transactions
	// get receipts
	bulkCaller := bulk.NewBulkCaller(ec.c)
	receipts := make([]*cfxSdkTypes.TransactionReceipt, len(rpcTxs))
	errs := make([]*error, len(rpcTxs))
	for i, v := range rpcTxs {
		receipts[i], errs[i] = bulkCaller.GetTransactionReceipt(v.Hash)
	}

	err = bulkCaller.Execute()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get receipts")
	}

	for i := range receipts {
		if *errs[i] != nil {
			return nil, nil, *errs[i]
		}
		if receipts[i] == nil {
			return nil, nil, fmt.Errorf("got empty receipt for %x", rpcTxs[i].Hash)
		}
	}

	// get traces
	bulkCaller.Clear()
	txsTraces := make([][]cfxSdkTypes.LocalizedTrace, len(rpcTxs))
	errs = make([]*error, len(rpcTxs))
	for i, v := range rpcTxs {
		txsTraces[i], errs[i] = bulkCaller.Trace().GetTransactionTraces(v.Hash)
	}
	err = bulkCaller.Execute()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get traces")
	}

	for i := range errs {
		if *errs[i] != nil {
			return nil, nil, *errs[i]
		}
	}

	// Convert all txs to loaded txs
	txs := make([]*cfxSdkTypes.Transaction, len(rpcTxs))
	loadedTxs := make([]*loadedTransaction, len(rpcTxs))
	for i, tx := range rpcTxs {
		txs[i] = &tx
		receipt := receipts[i]

		loadedTxs[i].BlockHeader = &rpcBlock.BlockHeader
		loadedTxs[i] = &loadedTransaction{}
		loadedTxs[i].Transaction = txs[i]
		loadedTxs[i].Receipt = receipt

		// Continue if calls does not exist (occurs at genesis)
		// if !addTraces {
		// 	continue
		// }

		loadedTxs[i].Trace = txsTraces[i]
		// loadedTxs[i].RawTrace = rawTraces[i].Result
	}
	return rpcBlock, loadedTxs, nil
}

func (ec *Client) getRpcBlock(_blockIdentifier *RosettaTypes.PartialBlockIdentifier) (
	*cfxSdkTypes.Block, error,
) {
	if _blockIdentifier.Hash != nil {
		return ec.c.GetBlockByHash(cfxSdkTypes.Hash(*_blockIdentifier.Hash))
	}
	if _blockIdentifier.Index != nil {
		blockNum := uint64(*_blockIdentifier.Index)
		return ec.c.GetBlockByBlockNumber(hexutil.Uint64(blockNum))
	}
	return ec.c.GetBlockByEpoch(cfxSdkTypes.EpochLatestState)
}

type loadedTransaction struct {
	BlockHeader *cfxSdkTypes.BlockHeader
	Transaction *cfxSdkTypes.Transaction

	Trace   []cfxSdkTypes.LocalizedTrace
	Receipt *cfxSdkTypes.TransactionReceipt
}

func (ec *Client) Balance(ctx context.Context,
	account *RosettaTypes.AccountIdentifier,
	block *RosettaTypes.PartialBlockIdentifier) (*RosettaTypes.AccountBalanceResponse, error) {

	// get rpc block
	rpcBlock, err := ec.getRpcBlock(block)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get raw block by rpc")
	}

	// get account balance
	addr := ec.c.MustNewAddress(account.Address)
	epoch := cfxSdkTypes.NewEpochNumber(rpcBlock.EpochNumber)
	balance, err := ec.c.GetBalance(addr, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get balance")
	}

	nonce, err := ec.c.GetNextNonce(addr, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get nonce")
	}

	code, err := ec.c.GetCode(addr, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get code")
	}

	return &RosettaTypes.AccountBalanceResponse{
		Balances: []*RosettaTypes.Amount{
			{
				Value:    balance.String(),
				Currency: Currency,
			},
		},
		BlockIdentifier: &RosettaTypes.BlockIdentifier{
			Hash:  rpcBlock.Hash.String(),
			Index: rpcBlock.BlockNumber.ToInt().Int64(),
		},
		Metadata: map[string]interface{}{
			"nonce": nonce.ToInt(),
			"code":  code.String(),
		},
	}, nil
}

func (ec *Client) PendingNonceAt(ctx context.Context, account cfxSdkTypes.Address) (*big.Int, error) {
	nonce, err := ec.c.TxPool().NextNonce(account)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return nonce.ToInt(), nil
}

func (ec *Client) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	gasPrice, err := ec.c.GetGasPrice()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return gasPrice.ToInt(), nil
}

func (ec *Client) SendTransaction(ctx context.Context, tx *cfxSdkTypes.SignedTransaction) error {
	encoded, err := tx.Encode()
	if err != nil {
		return errors.WithStack(err)
	}
	_, err = ec.c.SendRawTransaction(encoded)
	return errors.WithStack(err)
}

func (ec *Client) Call(ctx context.Context, request *RosettaTypes.CallRequest) (*RosettaTypes.CallResponse, error) {
	var result interface{}
	err := ec.c.CallRPC(&result, request.Method, request.Parameters)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &RosettaTypes.CallResponse{}, nil
}

// Close shuts down the RPC client connection.
func (ec *Client) Close() {
	ec.c.Close()
}

func convertTime(time uint64) int64 {
	return int64(time) * 1000
}

// ================== wrapper methods =========================
func (ec *Client) getBlockReward(rpcBlock *cfxSdkTypes.Block) (*cfxSdkTypes.RewardInfo, error) {
	epochReward, err := ec.c.GetBlockRewardInfo(*cfxSdkTypes.NewEpochNumber(rpcBlock.EpochNumber))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get epoch reward")
	}
	for _, blockReward := range epochReward {
		if blockReward.BlockHash == rpcBlock.Hash {
			return &blockReward, nil
		}
	}
	return nil, errors.Wrap(err, "not found block reward in epoch rewards")
}
