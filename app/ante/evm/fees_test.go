package evm_test

import (
	
	"math"
	"math/big"

	sdkmath "cosmossdk.io/math"
	evmante "github.com/HarryBin2002/kairoschain/v12/app/ante/evm"
	"github.com/HarryBin2002/kairoschain/v12/testutil"
	testutiltx "github.com/HarryBin2002/kairoschain/v12/testutil/tx"
	evmtypes "github.com/HarryBin2002/kairoschain/v12/x/evm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

var execTypes = []struct {
	name      string
	isCheckTx bool
	simulate  bool
}{
	{"deliverTx", false, false},
	{"deliverTxSimulate", false, true},
}

func (suite *AnteTestSuite) TestEthMinGasPriceDecorator() {
	denom := evmtypes.DefaultEVMDenom
	from, privKey := testutiltx.NewAddrKey()
	to := testutiltx.GenerateAddress()
	emptyAccessList := ethtypes.AccessList{}

	testCases := []struct {
		name     string
		malleate func() sdk.Tx
		expPass  bool
		errMsg   string
	}{
		{
			"invalid tx type",
			func() sdk.Tx {
				params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				params.MinGasPrice = sdk.NewDec(10)
				err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)

				return &testutiltx.InvalidTx{}
			},
			false,
			"invalid message type",
		},
		{
			"wrong tx type",
			func() sdk.Tx {
				params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				params.MinGasPrice = sdk.NewDec(10)
				err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
				testMsg := banktypes.MsgSend{
					FromAddress: "kairos1x8fhpj9nmhqk8z9kpgjt95ck2xwyue0p7g88sh",
					ToAddress:   "kairos1dx67l23hz9l0k9hcher8xz04uj7wf3yulsw3v9",
					Amount:      sdk.Coins{sdk.Coin{Amount: sdkmath.NewInt(10), Denom: denom}},
				}
				txBuilder := suite.CreateTestCosmosTxBuilder(sdkmath.NewInt(0), denom, &testMsg)
				return txBuilder.GetTx()
			},
			false,
			"invalid message type",
		},
		{
			"valid: invalid tx type with MinGasPrices = 0",
			func() sdk.Tx {
				params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				params.MinGasPrice = sdk.ZeroDec()
				err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)
				return &testutiltx.InvalidTx{}
			},
			true,
			"",
		},
		{
			"valid legacy tx with MinGasPrices = 0, gasPrice = 0",
			func() sdk.Tx {
				params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				params.MinGasPrice = sdk.ZeroDec()
				err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)

				msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), big.NewInt(0), nil, nil, nil)
				return suite.CreateTestTx(msg, privKey, 1, false)
			},
			true,
			"",
		},
		{
			"valid legacy tx with MinGasPrices = 0, gasPrice > 0",
			func() sdk.Tx {
				params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				params.MinGasPrice = sdk.ZeroDec()
				err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)

				msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), big.NewInt(10), nil, nil, nil)
				return suite.CreateTestTx(msg, privKey, 1, false)
			},
			true,
			"",
		},
		{
			"valid legacy tx with MinGasPrices = 10, gasPrice = 10",
			func() sdk.Tx {
				params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				params.MinGasPrice = sdk.NewDec(10)
				err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)

				msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), big.NewInt(10), nil, nil, nil)
				return suite.CreateTestTx(msg, privKey, 1, false)
			},
			true,
			"",
		},
		{
			"invalid legacy tx with MinGasPrices = 10, gasPrice = 0",
			func() sdk.Tx {
				params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				params.MinGasPrice = sdk.NewDec(10)
				err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)

				msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), big.NewInt(0), nil, nil, nil)
				return suite.CreateTestTx(msg, privKey, 1, false)
			},
			false,
			"provided fee < minimum global fee",
		},
		{
			"valid dynamic tx with MinGasPrices = 0, EffectivePrice = 0",
			func() sdk.Tx {
				params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				params.MinGasPrice = sdk.ZeroDec()
				err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)

				msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), nil, big.NewInt(0), big.NewInt(0), &emptyAccessList)
				return suite.CreateTestTx(msg, privKey, 1, false)
			},
			true,
			"",
		},
		{
			"valid dynamic tx with MinGasPrices = 0, EffectivePrice > 0",
			func() sdk.Tx {
				params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				params.MinGasPrice = sdk.ZeroDec()
				err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)

				msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), nil, big.NewInt(100), big.NewInt(50), &emptyAccessList)
				return suite.CreateTestTx(msg, privKey, 1, false)
			},
			true,
			"",
		},
		{
			"valid dynamic tx with MinGasPrices < EffectivePrice",
			func() sdk.Tx {
				params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				params.MinGasPrice = sdk.NewDec(10)
				err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)

				msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), nil, big.NewInt(100), big.NewInt(100), &emptyAccessList)
				return suite.CreateTestTx(msg, privKey, 1, false)
			},
			true,
			"",
		},
		{
			"invalid dynamic tx with MinGasPrices > EffectivePrice",
			func() sdk.Tx {
				params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				params.MinGasPrice = sdk.NewDec(10)
				err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)

				msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), nil, big.NewInt(0), big.NewInt(0), &emptyAccessList)
				return suite.CreateTestTx(msg, privKey, 1, false)
			},
			false,
			"provided fee < minimum global fee",
		},
		{
			"invalid dynamic tx with MinGasPrices > BaseFee, MinGasPrices > EffectivePrice",
			func() sdk.Tx {
				params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				params.MinGasPrice = sdk.NewDec(100)
				err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)

				feemarketParams := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				feemarketParams.BaseFee = sdkmath.NewInt(10)
				err = suite.app.FeeMarketKeeper.SetParams(suite.ctx, feemarketParams)
				suite.Require().NoError(err)

				msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), nil, big.NewInt(1000), big.NewInt(0), &emptyAccessList)
				return suite.CreateTestTx(msg, privKey, 1, false)
			},
			false,
			"provided fee < minimum global fee",
		},
		{
			"valid dynamic tx with MinGasPrices > BaseFee, MinGasPrices < EffectivePrice (big GasTipCap)",
			func() sdk.Tx {
				params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				params.MinGasPrice = sdk.NewDec(100)
				err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)

				feemarketParams := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				feemarketParams.BaseFee = sdkmath.NewInt(10)
				err = suite.app.FeeMarketKeeper.SetParams(suite.ctx, feemarketParams)
				suite.Require().NoError(err)

				msg := suite.BuildTestEthTx(from, to, nil, make([]byte, 0), nil, big.NewInt(1000), big.NewInt(101), &emptyAccessList)
				return suite.CreateTestTx(msg, privKey, 1, false)
			},
			true,
			"",
		},
		{
			name: "do not panic when tx fee overflow of int64",
			malleate: func() sdk.Tx {
				params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				params.MinGasPrice = sdk.NewDec(math.MaxInt64).MulInt64(2)
				err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)

				gasFeeOverflowInt64 := new(big.Int).Add(big.NewInt(math.MaxInt64), big.NewInt(1))
				msg := suite.BuildTestEthTx(
					from,                // from
					to,                  // to
					nil,                 // amount
					make([]byte, 0),     // input
					nil,                 // gas price
					gasFeeOverflowInt64, // gas fee cap
					gasFeeOverflowInt64, // gas tip cap
					&emptyAccessList,    // access list
				)
				return suite.CreateTestTx(msg, privKey, 1, false)
			},
			expPass: false,
			errMsg:  "provided fee < minimum global fee",
		},
		{
			name: "do not panic when required fee (minimum global fee) overflow of int64",
			malleate: func() sdk.Tx {
				params := suite.app.FeeMarketKeeper.GetParams(suite.ctx)
				params.MinGasPrice = sdk.NewDec(math.MaxInt64).MulInt64(2)
				err := suite.app.FeeMarketKeeper.SetParams(suite.ctx, params)
				suite.Require().NoError(err)

				msg := suite.BuildTestEthTx(
					from,                // from
					to,                  // to
					nil,                 // amount
					make([]byte, 0),     // input
					nil,                 // gas price
					big.NewInt(100_000), // gas fee cap
					big.NewInt(100),     // gas tip cap
					&emptyAccessList,    // access list
				)
				return suite.CreateTestTx(msg, privKey, 1, false)
			},
			expPass: false,
			errMsg:  "provided fee < minimum global fee",
		},
	}

	for _, et := range execTypes {
		for _, tc := range testCases {
			suite.Run(et.name+"_"+tc.name, func() {
				// s.SetupTest(et.isCheckTx)
				suite.SetupTest()
				dec := evmante.NewEthMinGasPriceDecorator(suite.app.FeeMarketKeeper, suite.app.EvmKeeper)
				_, err := dec.AnteHandle(suite.ctx, tc.malleate(), et.simulate, testutil.NextFn)

				if tc.expPass {
					suite.Require().NoError(err, tc.name)
				} else {
					suite.Require().Error(err, tc.name)
					suite.Require().Contains(err.Error(), tc.errMsg, tc.name)
				}
			})
		}
	}
}

func (suite *AnteTestSuite) TestEthMempoolFeeDecorator() {
	// TODO: add test
}