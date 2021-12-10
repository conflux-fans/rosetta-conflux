package conflux

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	sdk "github.com/Conflux-Chain/go-conflux-sdk"
	"github.com/Conflux-Chain/go-conflux-sdk/cfxclient/bulk"
	"github.com/Conflux-Chain/go-conflux-sdk/types"
	cfxSdkTypes "github.com/Conflux-Chain/go-conflux-sdk/types"
	RosettaTypes "github.com/coinbase/rosetta-sdk-go/types"
	"github.com/conflux-fans/rosetta-conflux/common"
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
	// c.UseBatchCallRpcMiddleware(middleware.BatchCallRpcConsoleMiddleware)
	// c.UseCallRpcMiddleware(middleware.CallRpcConsoleMiddleware)
	return &Client{c}, nil
}

func (ec *Client) Status(_ context.Context) (*RosettaTypes.BlockIdentifier, int64, *RosettaTypes.SyncStatus, []*RosettaTypes.Peer, error) {
	block, err := ec.getRpcBlock(nil, false)
	if err != nil {
		return nil, -1, nil, nil, err
	}

	return &RosettaTypes.BlockIdentifier{
			Hash:  block.Hash.String(),
			Index: block.BlockNumber.ToInt().Int64(),
		},
		convertTime(block.Timestamp.ToInt().Uint64()),
		// TODO: require rpc implement sync status and peers methods
		nil,
		nil,
		nil
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
	if block.BlockNumber.ToInt().Int64() == GenesisBlockIndex {
		return &RosettaTypes.Transaction{
			TransactionIdentifier: &RosettaTypes.TransactionIdentifier{
				Hash: block.Hash.String(),
			},
			Operations: []*RosettaTypes.Operation{},
		}, nil
	}

	txIdentifier := &RosettaTypes.TransactionIdentifier{
		Hash: block.Hash.String(),
	}

	// if block is not pivot, return
	isPivot, err := ec.IsPivotBlock(block)
	if err != nil {
		return nil, err
	}

	if !isPivot {
		return &RosettaTypes.Transaction{
			TransactionIdentifier: txIdentifier,
		}, nil
	}

	// if block is pivot block, include epoch-12 rewards
	rewardEpochNum := new(big.Int).Sub(block.EpochNumber.ToInt(), big.NewInt(12))
	if rewardEpochNum.Sign() < 0 {
		return &RosettaTypes.Transaction{
			TransactionIdentifier: txIdentifier,
		}, nil
	}

	blockRewards, err := ec.getEpochReward(*cfxSdkTypes.NewEpochNumberBig(rewardEpochNum))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get epoch reward")
	}

	var ops []*RosettaTypes.Operation
	for _, reward := range blockRewards {
		// totalReward := big.NewInt(0).Sub(reward.TotalReward.ToInt(), reward.TxFee.ToInt())
		totalReward := reward.TotalReward.ToInt()
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
				Value:    totalReward.String(),
				Currency: Currency,
			},
		}
		ops = append(ops, miningRewardOp)
	}

	return &RosettaTypes.Transaction{
		TransactionIdentifier: txIdentifier,
		Operations:            ops,
	}, nil
}

func (ec *Client) populateTransaction(
	tx *loadedTransaction,
) (*RosettaTypes.Transaction, error) {
	ops := []*RosettaTypes.Operation{}

	// Compute fee operations
	ops = appendFeeOps(tx, ops)
	ops = appendStorageUsedOps(tx, ops)
	ops = appendStorageRelaseOps(tx, ops)

	// Compute trace operations
	// traces := flattenTraces(tx.Trace, []*flatCall{})
	traceOps, err := traceOps(tx.Trace, int64(len(ops)))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	ops = append(ops, traceOps...)

	receiptMap, err := common.MarshalToMap(tx.Receipt)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert receipt to mapping")
	}

	traceMap, err := common.MarshalToMap(tx.Trace)
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

// ============= gen operates =============

func appendFeeOps(ltx *loadedTransaction, ops []*RosettaTypes.Operation) []*RosettaTypes.Operation {
	fmt.Printf("header %+v, tx %+v, receipt %+v\n", *ltx.BlockHeader, *ltx.Transaction, *ltx.Receipt)
	_, tx, receipt := ltx.BlockHeader, ltx.Transaction, ltx.Receipt
	gasFee := (*big.Int)(receipt.GasFee)

	// 如果是gas sponsored，跳过；否则 from gasOp 余额减少，无接收者

	var gasOp *RosettaTypes.Operation

	// var ops []*RosettaTypes.Operation
	if !receipt.GasCoveredBySponsor {
		gasOp = &RosettaTypes.Operation{
			OperationIdentifier: &RosettaTypes.OperationIdentifier{
				Index: int64(len(ops)),
			},
			Type:   GasFeeOpType,
			Status: RosettaTypes.String(SuccessStatus),
			Account: &RosettaTypes.AccountIdentifier{
				Address: tx.From.String(),
			},
			Amount: &RosettaTypes.Amount{
				Value:    new(big.Int).Neg(gasFee).String(),
				Currency: Currency,
			},
		}
		ops = append(ops, gasOp)
	}
	return ops
}

func appendStorageUsedOps(ltx *loadedTransaction, ops []*RosettaTypes.Operation) []*RosettaTypes.Operation {
	tx, receipt := ltx.Transaction, ltx.Receipt
	storageUsed := new(big.Int).SetUint64(uint64(receipt.StorageCollateralized))
	storageFee := new(big.Int).Mul(StorageUint, storageUsed)

	// 如果是storage sponsored，跳过；否则 from storageOp 余额减少，无接收者
	// var ops []*RosettaTypes.Operation
	if !receipt.StorageCoveredBySponsor {
		storageSpendOp := &RosettaTypes.Operation{
			OperationIdentifier: &RosettaTypes.OperationIdentifier{
				Index: int64(len(ops)),
			},
			Type:   StorageCollaterlOpType,
			Status: RosettaTypes.String(SuccessStatus),
			Account: &RosettaTypes.AccountIdentifier{
				Address: tx.From.String(),
			},
			Amount: &RosettaTypes.Amount{
				Value:    new(big.Int).Neg(storageFee).String(),
				Currency: Currency,
			},
		}
		ops = append(ops, storageSpendOp)
	}
	return ops
}

// FIXME: 没有release后是否release给sponsor的标志，无法准确生成ops
func appendStorageRelaseOps(ltx *loadedTransaction, ops []*RosettaTypes.Operation) []*RosettaTypes.Operation {
	// var ops []*RosettaTypes.Operation
	// storage release，目标地址是否sponsor，非sponsor，targe地址余额增加；sponsored如何处理？
	for _, sc := range ltx.Receipt.StorageReleased {
		// TODO:判断目标地址是否sponsored
		storageReleaseOp := &RosettaTypes.Operation{
			OperationIdentifier: &RosettaTypes.OperationIdentifier{
				Index: int64(len(ops)),
			},
			Type:   StorageReleaseOpType,
			Status: RosettaTypes.String(SuccessStatus),
			Account: &RosettaTypes.AccountIdentifier{
				Address: sc.Address.String(),
			},
			Amount: &RosettaTypes.Amount{
				Value:    StorageFee(uint64(sc.Collaterals)).String(),
				Currency: Currency,
			},
		}
		// startIdx++
		ops = append(ops, storageReleaseOp)
	}
	return ops
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
		if trace.Type == cfxSdkTypes.CALL_TYPE && value.Cmp(big.NewInt(0)) == 0 {
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
	switch node.Type {
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
	if node.Type == cfxSdkTypes.CALL_TYPE {
		if node.CallWithResult.Outcome == "success" {
			return nil
		}

		return map[string]interface{}{
			"error": node.CallWithResult.ReturnData,
		}
	}

	if node.Type == cfxSdkTypes.CREATE_TYPE {
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
	rpcBlock, err := ec.getRpcBlock(_blockIdentifier, true)
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

		fmt.Printf("\nreceipt %+v\n\n", *receipts[i])

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
			return nil, nil, errors.Wrapf(*errs[i], "failed to get the %vth trace", i)
		}
	}

	// fmt.Printf("rpcBlock %v\n", rpcBlock)
	// Convert all txs to loaded txs
	txs := make([]*cfxSdkTypes.Transaction, len(rpcTxs))
	loadedTxs := make([]*loadedTransaction, len(rpcTxs))
	for i, tx := range rpcTxs {
		txs[i] = &tx
		receipt := receipts[i]
		loadedTxs[i] = &loadedTransaction{}
		loadedTxs[i].BlockHeader = &rpcBlock.BlockHeader
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

func (ec *Client) getRpcBlock(_blockIdentifier *RosettaTypes.PartialBlockIdentifier, isContainTxs bool) (
	*cfxSdkTypes.Block, error,
) {
	getRawRpcBlock := func(bockIdtf *RosettaTypes.PartialBlockIdentifier) (*cfxSdkTypes.Block, error) {
		// return latest rewarded block when identifier invalid
		if bockIdtf == nil || (bockIdtf.Hash == nil && bockIdtf.Index == nil) {
			epochNum, err := ec.c.GetEpochNumber()
			if err != nil {
				return nil, errors.WithStack(err)
			}

			latestRewarded := new(big.Int).Sub(epochNum.ToInt(), big.NewInt(12))
			if latestRewarded.Sign() == -1 {
				return nil, errors.New("block has no reward yet")
			}

			return ec.c.GetBlockByEpoch(types.NewEpochNumberBig(latestRewarded))
		}

		if bockIdtf.Hash != nil {
			if isContainTxs {
				sb, err := ec.c.GetBlockSummaryByHash(cfxSdkTypes.Hash(*bockIdtf.Hash))
				if err != nil {
					return nil, errors.WithStack(err)
				}
				return &cfxSdkTypes.Block{
					BlockHeader: sb.BlockHeader,
				}, nil
			}
			return ec.c.GetBlockByHash(cfxSdkTypes.Hash(*bockIdtf.Hash))
		}

		blockNum := uint64(*bockIdtf.Index)
		if blockNum != uint64(GenesisBlockIndex) {
			if !isContainTxs {
				return ec.c.GetBlockByBlockNumber(hexutil.Uint64(blockNum))
			}

			sb, err := ec.c.GetBlockSummaryByBlockNumber(hexutil.Uint64(blockNum))
			if err != nil {
				return nil, err
			}
			return &cfxSdkTypes.Block{
				BlockHeader: sb.BlockHeader,
			}, nil
		}

		// genesis block
		gensisiBlock, err := ec.c.GetBlockByEpoch(cfxSdkTypes.EpochEarliest)
		if err != nil {
			return nil, err
		}
		// set genesis block transactions to empty becuase of no receipt and no trace exist for them.
		gensisiBlock.Transactions = nil
		gensisiBlock.Timestamp = types.NewBigInt(0x5f9998b9)

		return gensisiBlock, nil
	}

	raw, err := getRawRpcBlock(_blockIdentifier)
	if err != nil {
		return nil, err
	}

	if raw == nil {
		return nil, errors.New("The block has not been produced yet")
	}

	// replace parent hash to previours block hash instead of what in tree graph
	parentNum := raw.BlockNumber.ToInt().Int64() - 1
	if parentNum >= 0 {
		parent, err := getRawRpcBlock(&RosettaTypes.PartialBlockIdentifier{
			Index: &parentNum,
		})
		if err != nil {
			return nil, err
		}

		raw.ParentHash = parent.Hash
	}

	return raw, nil
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
	rpcBlock, err := ec.getRpcBlock(block, false)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get raw block by rpc")
	}

	// get account balance
	// 1. get pre epoch balance
	// 2. iterate all transactions in till this block of this epoch
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
				Value:    balance.ToInt().String(),
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

func (ec *Client) EpochNumber(ctx context.Context) (*big.Int, error) {
	epoch, err := ec.c.GetEpochNumber()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return epoch.ToInt(), nil
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

func (ec *Client) getEpochReward(epockNum cfxSdkTypes.Epoch) ([]cfxSdkTypes.RewardInfo, error) {
	return ec.c.GetBlockRewardInfo(epockNum)
}

func (ec *Client) getBlockReward(rpcBlock *cfxSdkTypes.Block) (*cfxSdkTypes.RewardInfo, error) {
	if rpcBlock.EpochNumber.ToInt().Int64() == GenesisBlockIndex {
		return &cfxSdkTypes.RewardInfo{
			BlockHash:   rpcBlock.Hash,
			Author:      rpcBlock.Miner,
			TotalReward: types.NewBigInt(0),
			BaseReward:  types.NewBigInt(0),
			TxFee:       types.NewBigInt(0),
		}, nil
	}
	epochReward, err := ec.c.GetBlockRewardInfo(*cfxSdkTypes.NewEpochNumber(rpcBlock.EpochNumber))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get epoch reward")
	}
	for _, blockReward := range epochReward {
		if blockReward.BlockHash == rpcBlock.Hash {
			return &blockReward, nil
		}
	}
	return nil, errors.New("not found block reward in epoch rewards")
}

func (ec *Client) IsPivotBlock(block *cfxSdkTypes.Block) (bool, error) {
	pivotBlock, err := ec.c.GetBlockByEpoch(cfxSdkTypes.NewEpochNumberBig(block.EpochNumber.ToInt()))
	if err != nil {
		return false, errors.Wrap(err, "failed to get pivot block")
	}
	return pivotBlock.Hash == block.Hash, nil
}

func StorageFee(amount uint64) *big.Int {
	storageUsed := new(big.Int).SetUint64(amount)
	return new(big.Int).Mul(StorageUint, storageUsed)
}
