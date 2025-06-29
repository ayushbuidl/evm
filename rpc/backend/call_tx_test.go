package backend

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"google.golang.org/grpc/metadata"

	"github.com/cosmos/evm/rpc/backend/mocks"
	rpctypes "github.com/cosmos/evm/rpc/types"
	utiltx "github.com/cosmos/evm/testutil/tx"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"cosmossdk.io/math"

	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
)

func (suite *BackendTestSuite) TestResend() {
	txNonce := (hexutil.Uint64)(1)
	baseFee := math.NewInt(1)
	gasPrice := new(hexutil.Big)
	toAddr := utiltx.GenerateAddress()
	chainID := (*hexutil.Big)(suite.backend.chainID)
	callArgs := evmtypes.TransactionArgs{
		From:                 nil,
		To:                   &toAddr,
		Gas:                  nil,
		GasPrice:             nil,
		MaxFeePerGas:         gasPrice,
		MaxPriorityFeePerGas: gasPrice,
		Value:                gasPrice,
		Nonce:                &txNonce,
		Input:                nil,
		Data:                 nil,
		AccessList:           nil,
		ChainID:              chainID,
	}

	testCases := []struct {
		name         string
		registerMock func()
		args         evmtypes.TransactionArgs
		gasPrice     *hexutil.Big
		gasLimit     *hexutil.Uint64
		expHash      common.Hash
		expPass      bool
	}{
		{
			"fail - Missing transaction nonce",
			func() {},
			evmtypes.TransactionArgs{
				Nonce: nil,
			},
			nil,
			nil,
			common.Hash{},
			false,
		},
		{
			"pass - Can't set Tx defaults BaseFee disabled",
			func() {
				var header metadata.MD
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParams(queryClient, &header, 1)
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFeeDisabled(queryClient)
			},
			evmtypes.TransactionArgs{
				Nonce:   &txNonce,
				ChainID: callArgs.ChainID,
			},
			nil,
			nil,
			common.Hash{},
			true,
		},
		{
			"pass - Can't set Tx defaults",
			func() {
				var header metadata.MD
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				feeMarketClient := suite.backend.queryClient.FeeMarket.(*mocks.FeeMarketQueryClient)
				RegisterParams(queryClient, &header, 1)
				RegisterFeeMarketParams(feeMarketClient, 1)
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)
			},
			evmtypes.TransactionArgs{
				Nonce: &txNonce,
			},
			nil,
			nil,
			common.Hash{},
			true,
		},
		{
			"pass - MaxFeePerGas is nil",
			func() {
				var header metadata.MD
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParams(queryClient, &header, 1)
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFeeDisabled(queryClient)
			},
			evmtypes.TransactionArgs{
				Nonce:                &txNonce,
				MaxPriorityFeePerGas: nil,
				GasPrice:             nil,
				MaxFeePerGas:         nil,
			},
			nil,
			nil,
			common.Hash{},
			true,
		},
		{
			"fail - GasPrice and (MaxFeePerGas or MaxPriorityPerGas specified)",
			func() {},
			evmtypes.TransactionArgs{
				Nonce:                &txNonce,
				MaxPriorityFeePerGas: nil,
				GasPrice:             gasPrice,
				MaxFeePerGas:         gasPrice,
			},
			nil,
			nil,
			common.Hash{},
			false,
		},
		{
			"fail - Block error",
			func() {
				var header metadata.MD
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParams(queryClient, &header, 1)
				RegisterBlockError(client, 1)
			},
			evmtypes.TransactionArgs{
				Nonce: &txNonce,
			},
			nil,
			nil,
			common.Hash{},
			false,
		},
		{
			"pass - MaxFeePerGas is nil",
			func() {
				var header metadata.MD
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParams(queryClient, &header, 1)
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)
			},
			evmtypes.TransactionArgs{
				Nonce:                &txNonce,
				GasPrice:             nil,
				MaxPriorityFeePerGas: gasPrice,
				MaxFeePerGas:         gasPrice,
				ChainID:              callArgs.ChainID,
			},
			nil,
			nil,
			common.Hash{},
			true,
		},
		{
			"pass - Chain Id is nil",
			func() {
				var header metadata.MD
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				RegisterParams(queryClient, &header, 1)
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)
			},
			evmtypes.TransactionArgs{
				Nonce:                &txNonce,
				MaxPriorityFeePerGas: gasPrice,
				ChainID:              nil,
			},
			nil,
			nil,
			common.Hash{},
			true,
		},
		{
			"fail - Pending transactions error",
			func() {
				var header metadata.MD
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)
				RegisterEstimateGas(queryClient, callArgs)
				RegisterParams(queryClient, &header, 1)
				RegisterUnconfirmedTxsError(client, nil)
			},
			evmtypes.TransactionArgs{
				Nonce:                &txNonce,
				To:                   &toAddr,
				MaxFeePerGas:         gasPrice,
				MaxPriorityFeePerGas: gasPrice,
				Value:                gasPrice,
				Gas:                  nil,
				ChainID:              callArgs.ChainID,
			},
			gasPrice,
			nil,
			common.Hash{},
			false,
		},
		{
			"fail - Not Ethereum txs",
			func() {
				var header metadata.MD
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, baseFee)
				RegisterEstimateGas(queryClient, callArgs)
				RegisterParams(queryClient, &header, 1)

				RegisterUnconfirmedTxsEmpty(client, nil)
			},
			evmtypes.TransactionArgs{
				Nonce:                &txNonce,
				To:                   &toAddr,
				MaxFeePerGas:         gasPrice,
				MaxPriorityFeePerGas: gasPrice,
				Value:                gasPrice,
				Gas:                  nil,
				ChainID:              callArgs.ChainID,
			},
			gasPrice,
			nil,
			common.Hash{},
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("case %s", tc.name), func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			hash, err := suite.backend.Resend(tc.args, tc.gasPrice, tc.gasLimit)

			if tc.expPass {
				suite.Require().Equal(tc.expHash, hash)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestSendRawTransaction() {
	ethTx, bz := suite.buildEthereumTx()

	emptyChainIDTx := suite.buildEthereumTxWithChainID(nil)
	invalidChainIDTx := suite.buildEthereumTxWithChainID(big.NewInt(1))

	// Sign the ethTx
	ethSigner := ethtypes.LatestSigner(suite.backend.ChainConfig())
	err := ethTx.Sign(ethSigner, suite.signer)
	suite.Require().NoError(err)

	rlpEncodedBz, _ := rlp.EncodeToBytes(ethTx.AsTransaction())
	evmDenom := evmtypes.GetEVMCoinDenom()

	testCases := []struct {
		name         string
		registerMock func()
		rawTx        func() []byte
		expHash      common.Hash
		expError     string
		expPass      bool
	}{
		{
			"fail - empty bytes",
			func() {},
			func() []byte { return []byte{} },
			common.Hash{},
			"",
			false,
		},
		{
			"fail - no RLP encoded bytes",
			func() {},
			func() []byte { return bz },
			common.Hash{},
			"",
			false,
		},
		{
			"fail - invalid chain-id",
			func() {
				suite.backend.allowUnprotectedTxs = false
			},
			func() []byte {
				from, priv := utiltx.NewAddrKey()
				signer := utiltx.NewSigner(priv)
				invalidChainIDTx.From = from.String()
				err := invalidChainIDTx.Sign(ethSigner, signer)
				suite.Require().NoError(err)
				bytes, _ := rlp.EncodeToBytes(invalidChainIDTx.AsTransaction())
				return bytes
			},
			common.Hash{},
			fmt.Errorf("incorrect chain-id; expected %d, got %d", 3456, big.NewInt(1)).Error(),
			false,
		},
		{
			"fail - unprotected tx",
			func() {
				suite.backend.allowUnprotectedTxs = false
			},
			func() []byte {
				bytes, _ := rlp.EncodeToBytes(emptyChainIDTx.AsTransaction())
				return bytes
			},
			common.Hash{},
			errors.New("only replay-protected (EIP-155) transactions allowed over RPC").Error(),
			false,
		},
		{
			"fail - failed to broadcast transaction",
			func() {
				cosmosTx, _ := ethTx.BuildTx(suite.backend.clientCtx.TxConfig.NewTxBuilder(), evmDenom)
				txBytes, _ := suite.backend.clientCtx.TxConfig.TxEncoder()(cosmosTx)

				client := suite.backend.clientCtx.Client.(*mocks.Client)
				suite.backend.allowUnprotectedTxs = true
				RegisterBroadcastTxError(client, txBytes)
			},
			func() []byte {
				bytes, _ := rlp.EncodeToBytes(ethTx.AsTransaction())
				return bytes
			},
			common.HexToHash(ethTx.Hash),
			errortypes.ErrInvalidRequest.Error(),
			false,
		},
		{
			"pass - Gets the correct transaction hash of the eth transaction",
			func() {
				cosmosTx, _ := ethTx.BuildTx(suite.backend.clientCtx.TxConfig.NewTxBuilder(), evmDenom)
				txBytes, _ := suite.backend.clientCtx.TxConfig.TxEncoder()(cosmosTx)

				client := suite.backend.clientCtx.Client.(*mocks.Client)
				suite.backend.allowUnprotectedTxs = true
				RegisterBroadcastTx(client, txBytes)
			},
			func() []byte { return rlpEncodedBz },
			common.HexToHash(ethTx.Hash),
			"",
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("case %s", tc.name), func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			hash, err := suite.backend.SendRawTransaction(tc.rawTx())

			if tc.expPass {
				suite.Require().Equal(tc.expHash, hash)
			} else {
				suite.Require().Errorf(err, tc.expError)
			}
		})
	}
}

func (suite *BackendTestSuite) TestDoCall() {
	_, bz := suite.buildEthereumTx()
	gasPrice := (*hexutil.Big)(big.NewInt(1))
	toAddr := utiltx.GenerateAddress()
	chainID := (*hexutil.Big)(suite.backend.chainID)
	callArgs := evmtypes.TransactionArgs{
		From:                 nil,
		To:                   &toAddr,
		Gas:                  nil,
		GasPrice:             nil,
		MaxFeePerGas:         gasPrice,
		MaxPriorityFeePerGas: gasPrice,
		Value:                gasPrice,
		Input:                nil,
		Data:                 nil,
		AccessList:           nil,
		ChainID:              chainID,
	}
	argsBz, err := json.Marshal(callArgs)
	suite.Require().NoError(err)

	testCases := []struct {
		name         string
		registerMock func()
		blockNum     rpctypes.BlockNumber
		callArgs     evmtypes.TransactionArgs
		expEthTx     *evmtypes.MsgEthereumTxResponse
		expPass      bool
	}{
		{
			"fail - Invalid request",
			func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, bz)
				suite.Require().NoError(err)
				RegisterEthCallError(queryClient, &evmtypes.EthCallRequest{Args: argsBz, ChainId: suite.backend.chainID.Int64()})
			},
			rpctypes.BlockNumber(1),
			callArgs,
			&evmtypes.MsgEthereumTxResponse{},
			false,
		},
		{
			"pass - Returned transaction response",
			func() {
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				_, err := RegisterBlock(client, 1, bz)
				suite.Require().NoError(err)
				RegisterEthCall(queryClient, &evmtypes.EthCallRequest{Args: argsBz, ChainId: suite.backend.chainID.Int64()})
			},
			rpctypes.BlockNumber(1),
			callArgs,
			&evmtypes.MsgEthereumTxResponse{},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("case %s", tc.name), func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			msgEthTx, err := suite.backend.DoCall(tc.callArgs, tc.blockNum)

			if tc.expPass {
				suite.Require().Equal(tc.expEthTx, msgEthTx)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *BackendTestSuite) TestGasPrice() {
	defaultGasPrice := (*hexutil.Big)(big.NewInt(1))

	testCases := []struct {
		name         string
		registerMock func()
		expGas       *hexutil.Big
		expPass      bool
	}{
		{
			"pass - get the default gas price",
			func() {
				var header metadata.MD
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				feeMarketClient := suite.backend.queryClient.FeeMarket.(*mocks.FeeMarketQueryClient)
				RegisterFeeMarketParams(feeMarketClient, 1)
				RegisterParams(queryClient, &header, 1)
				RegisterGlobalMinGasPrice(queryClient, 1)
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, math.NewInt(1))
			},
			defaultGasPrice,
			true,
		},
		{
			"fail - can't get gasFee, FeeMarketParams error",
			func() {
				var header metadata.MD
				client := suite.backend.clientCtx.Client.(*mocks.Client)
				queryClient := suite.backend.queryClient.QueryClient.(*mocks.EVMQueryClient)
				feeMarketClient := suite.backend.queryClient.FeeMarket.(*mocks.FeeMarketQueryClient)
				RegisterFeeMarketParamsError(feeMarketClient, 1)
				RegisterParams(queryClient, &header, 1)
				_, err := RegisterBlock(client, 1, nil)
				suite.Require().NoError(err)
				_, err = RegisterBlockResults(client, 1)
				suite.Require().NoError(err)
				RegisterBaseFee(queryClient, math.NewInt(1))
			},
			defaultGasPrice,
			false,
		},
	}

	for _, tc := range testCases {
		suite.Run(fmt.Sprintf("case %s", tc.name), func() {
			suite.SetupTest() // reset test and queries
			tc.registerMock()

			gasPrice, err := suite.backend.GasPrice()
			if tc.expPass {
				suite.Require().Equal(tc.expGas, gasPrice)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
