package evm_test

import (
	"github.com/HarryBin2002/kairoschain/v12/constants"
	"math"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	ethante "github.com/HarryBin2002/kairoschain/v12/app/ante/evm"
	"github.com/HarryBin2002/kairoschain/v12/server/config"
	"github.com/HarryBin2002/kairoschain/v12/testutil"
	testutiltx "github.com/HarryBin2002/kairoschain/v12/testutil/tx"
	"github.com/HarryBin2002/kairoschain/v12/types"
	"github.com/HarryBin2002/kairoschain/v12/x/evm/statedb"
	evmtypes "github.com/HarryBin2002/kairoschain/v12/x/evm/types"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

func (suite *AnteTestSuite) TestNewEthAccountVerificationDecorator() {
	dec := ethante.NewEthAccountVerificationDecorator(
		suite.app.AccountKeeper, suite.app.EvmKeeper,
	)

	addr := testutiltx.GenerateAddress()

	ethContractCreationTxParams := &evmtypes.EvmTxArgs{
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: 1000,
		GasPrice: big.NewInt(1),
	}

	tx := evmtypes.NewTx(ethContractCreationTxParams)
	tx.From = addr.Hex()

	var vmdb *statedb.StateDB

	testCases := []struct {
		name     string
		tx       sdk.Tx
		malleate func()
		checkTx  bool
		expPass  bool
	}{
		{"not CheckTx", nil, func() {}, false, true},
		{"invalid transaction type", &testutiltx.InvalidTx{}, func() {}, true, false},
		{
			"sender not set to msg",
			tx,
			func() {},
			true,
			false,
		},
		{
			"sender not EOA",
			tx,
			func() {
				// set not as an EOA
				vmdb.SetCode(addr, []byte("1"))
			},
			true,
			false,
		},
		{
			"not enough balance to cover tx cost",
			tx,
			func() {
				// reset back to EOA
				vmdb.SetCode(addr, nil)
			},
			true,
			false,
		},
		{
			"success new account",
			tx,
			func() {
				vmdb.AddBalance(addr, big.NewInt(1000000))
			},
			true,
			true,
		},
		{
			"success existing account",
			tx,
			func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

				vmdb.AddBalance(addr, big.NewInt(1000000))
			},
			true,
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			vmdb = testutil.NewStateDB(suite.ctx, suite.app.EvmKeeper)
			tc.malleate()
			suite.Require().NoError(vmdb.Commit())

			_, err := dec.AnteHandle(suite.ctx.WithIsCheckTx(tc.checkTx), tc.tx, false, testutil.NextFn)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *AnteTestSuite) TestEthNonceVerificationDecorator() {
	suite.SetupTest()
	dec := ethante.NewEthIncrementSenderSequenceDecorator(suite.app.AccountKeeper)

	addr := testutiltx.GenerateAddress()

	ethContractCreationTxParams := &evmtypes.EvmTxArgs{
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: 1000,
		GasPrice: big.NewInt(1),
	}

	tx := evmtypes.NewTx(ethContractCreationTxParams)
	tx.From = addr.Hex()

	testCases := []struct {
		name      string
		tx        sdk.Tx
		malleate  func()
		reCheckTx bool
		expPass   bool
	}{
		{"ReCheckTx", &testutiltx.InvalidTx{}, func() {}, true, false},
		{"invalid transaction type", &testutiltx.InvalidTx{}, func() {}, false, false},
		{"sender account not found", tx, func() {}, false, false},
		{
			"sender nonce missmatch",
			tx,
			func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			false,
			false,
		},
		{
			"success",
			tx,
			func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
				suite.Require().NoError(acc.SetSequence(1))
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			false,
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.malleate()
			_, err := dec.AnteHandle(suite.ctx.WithIsReCheckTx(tc.reCheckTx), tc.tx, false, testutil.NextFn)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *AnteTestSuite) TestEthGasConsumeDecorator() {
	chainID := suite.app.EvmKeeper.ChainID()
	dec := ethante.NewEthGasConsumeDecorator(suite.app.BankKeeper, suite.app.DistrKeeper, suite.app.EvmKeeper, suite.app.StakingKeeper, config.DefaultMaxTxGasWanted)

	addr := testutiltx.GenerateAddress()

	txGasLimit := uint64(1000)

	ethContractCreationTxParams := &evmtypes.EvmTxArgs{
		ChainID:  chainID,
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: txGasLimit,
		GasPrice: big.NewInt(1),
	}

	tx := evmtypes.NewTx(ethContractCreationTxParams)
	tx.From = addr.Hex()

	ethCfg := suite.app.EvmKeeper.GetParams(suite.ctx).
		ChainConfig.EthereumConfig(chainID)
	baseFee := suite.app.EvmKeeper.GetBaseFee(suite.ctx, ethCfg)
	suite.Require().Equal(int64(1000000000), baseFee.Int64())

	gasPrice := new(big.Int).Add(baseFee, evmtypes.DefaultPriorityReduction.BigInt())

	tx2GasLimit := uint64(1000000)
	eth2TxContractParams := &evmtypes.EvmTxArgs{
		ChainID:  chainID,
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: tx2GasLimit,
		GasPrice: gasPrice,
		Accesses: &ethtypes.AccessList{{Address: addr, StorageKeys: nil}},
	}
	tx2 := evmtypes.NewTx(eth2TxContractParams)
	tx2.From = addr.Hex()
	tx2Priority := int64(1)

	tx3GasLimit := types.BlockGasLimit(suite.ctx) + uint64(1)
	eth3TxContractParams := &evmtypes.EvmTxArgs{
		ChainID:  chainID,
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: tx3GasLimit,
		GasPrice: gasPrice,
		Accesses: &ethtypes.AccessList{{Address: addr, StorageKeys: nil}},
	}
	tx3 := evmtypes.NewTx(eth3TxContractParams)

	dynamicTxContractParams := &evmtypes.EvmTxArgs{
		ChainID:   chainID,
		Nonce:     1,
		Amount:    big.NewInt(10),
		GasLimit:  tx2GasLimit,
		GasFeeCap: new(big.Int).Add(baseFee, big.NewInt(evmtypes.DefaultPriorityReduction.Int64()*2)),
		GasTipCap: evmtypes.DefaultPriorityReduction.BigInt(),
		Accesses:  &ethtypes.AccessList{{Address: addr, StorageKeys: nil}},
	}
	dynamicFeeTx := evmtypes.NewTx(dynamicTxContractParams)
	dynamicFeeTx.From = addr.Hex()
	dynamicFeeTxPriority := int64(1)

	zeroBalanceAddr := testutiltx.GenerateAddress()
	zeroBalanceAcc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, zeroBalanceAddr.Bytes())
	suite.app.AccountKeeper.SetAccount(suite.ctx, zeroBalanceAcc)
	zeroFeeLegacyTx := evmtypes.NewTx(&evmtypes.EvmTxArgs{
		ChainID:  chainID,
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: 1_000_000,
		GasPrice: big.NewInt(0),
	})
	zeroFeeLegacyTx.From = zeroBalanceAddr.Hex()
	zeroFeeAccessListTx := evmtypes.NewTx(&evmtypes.EvmTxArgs{
		ChainID:  chainID,
		Nonce:    1,
		Amount:   big.NewInt(10),
		GasLimit: 1_000_000,
		GasPrice: big.NewInt(0),
		Accesses: &ethtypes.AccessList{{Address: zeroBalanceAddr, StorageKeys: nil}},
	})
	zeroFeeAccessListTx.From = zeroBalanceAddr.Hex()

	var vmdb *statedb.StateDB

	testCases := []struct {
		name        string
		tx          sdk.Tx
		gasLimit    uint64
		malleate    func(ctx sdk.Context) sdk.Context
		expPass     bool
		expPanic    bool
		expPriority int64
		postCheck   func(ctx sdk.Context)
	}{
		{
			"invalid transaction type",
			&testutiltx.InvalidTx{},
			math.MaxUint64,
			func(ctx sdk.Context) sdk.Context { return ctx },
			false,
			false,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"sender not found",
			evmtypes.NewTx(&evmtypes.EvmTxArgs{
				ChainID:  chainID,
				Nonce:    1,
				Amount:   big.NewInt(10),
				GasLimit: 1000,
				GasPrice: big.NewInt(1),
			}),
			math.MaxUint64,
			func(ctx sdk.Context) sdk.Context { return ctx },
			false, false,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"gas limit too low",
			tx,
			math.MaxUint64,
			func(ctx sdk.Context) sdk.Context { return ctx },
			false, false,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"gas limit above block gas limit",
			tx3,
			math.MaxUint64,
			func(ctx sdk.Context) sdk.Context { return ctx },
			false, false,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"not enough balance for fees",
			tx2,
			math.MaxUint64,
			func(ctx sdk.Context) sdk.Context { return ctx },
			false, false,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"not enough tx gas",
			tx2,
			0,
			func(ctx sdk.Context) sdk.Context {
				vmdb.AddBalance(addr, big.NewInt(1e6))
				return ctx
			},
			false, true,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"not enough block gas",
			tx2,
			0,
			func(ctx sdk.Context) sdk.Context {
				vmdb.AddBalance(addr, big.NewInt(1e6))
				return ctx.WithBlockGasMeter(sdk.NewGasMeter(1))
			},
			false, true,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"success - legacy tx",
			tx2,
			tx2GasLimit, // it's capped
			func(ctx sdk.Context) sdk.Context {
				vmdb.AddBalance(addr, big.NewInt(1e16))
				return ctx.WithBlockGasMeter(sdk.NewGasMeter(1e19))
			},
			true, false,
			tx2Priority,
			func(ctx sdk.Context) {},
		},
		{
			"success - dynamic fee tx",
			dynamicFeeTx,
			tx2GasLimit, // it's capped
			func(ctx sdk.Context) sdk.Context {
				vmdb.AddBalance(addr, big.NewInt(1e16))
				return ctx.WithBlockGasMeter(sdk.NewGasMeter(1e19))
			},
			true, false,
			dynamicFeeTxPriority,
			func(ctx sdk.Context) {},
		},
		{
			"success - gas limit on gasMeter is set on ReCheckTx mode",
			dynamicFeeTx,
			0, // for reCheckTX mode, gas limit should be set to 0
			func(ctx sdk.Context) sdk.Context {
				vmdb.AddBalance(addr, big.NewInt(1e15))
				return ctx.WithIsReCheckTx(true)
			},
			true, false,
			0,
			func(ctx sdk.Context) {},
		},
		{
			"success - legacy tx - insufficient funds but enough staking rewards",
			tx2,
			tx2GasLimit, // it's capped
			func(ctx sdk.Context) sdk.Context {
				ctx, err := testutil.PrepareAccountsForDelegationRewards(
					suite.T(), ctx, suite.app, sdk.AccAddress(addr.Bytes()), sdk.ZeroInt(), sdk.NewInt(1e16),
				)
				suite.Require().NoError(err, "error while preparing accounts for delegation rewards")
				return ctx.
					WithBlockGasMeter(sdk.NewGasMeter(1e19)).
					WithBlockHeight(ctx.BlockHeight() + 1)
			},
			true, false,
			tx2Priority,
			func(ctx sdk.Context) {
				balance := suite.app.BankKeeper.GetBalance(ctx, sdk.AccAddress(addr.Bytes()), constants.BaseDenom)
				suite.Require().False(
					balance.Amount.IsZero(),
					"the fees are paid after withdrawing (a surplus amount of) staking rewards, so it should be higher than the initial balance",
				)

				rewards, err := testutil.GetTotalDelegationRewards(ctx, suite.app.DistrKeeper, sdk.AccAddress(addr.Bytes()))
				suite.Require().NoError(err, "error while querying delegation total rewards")
				suite.Require().Nil(rewards, "the total rewards should be nil after withdrawing all of them")
			},
		},
		{
			"success - legacy tx - enough funds so no staking rewards should be used",
			tx2,
			tx2GasLimit, // it's capped
			func(ctx sdk.Context) sdk.Context {
				ctx, err := testutil.PrepareAccountsForDelegationRewards(
					suite.T(), ctx, suite.app, sdk.AccAddress(addr.Bytes()), sdk.NewInt(1e16), sdk.NewInt(1e16),
				)
				suite.Require().NoError(err, "error while preparing accounts for delegation rewards")
				return ctx.
					WithBlockGasMeter(sdk.NewGasMeter(1e19)).
					WithBlockHeight(ctx.BlockHeight() + 1)
			},
			true, false,
			tx2Priority,
			func(ctx sdk.Context) {
				balance := suite.app.BankKeeper.GetBalance(ctx, sdk.AccAddress(addr.Bytes()), constants.BaseDenom)
				suite.Require().True(
					balance.Amount.LT(sdk.NewInt(1e16)),
					"the fees are paid using the available balance, so it should be lower than the initial balance",
				)

				rewards, err := testutil.GetTotalDelegationRewards(ctx, suite.app.DistrKeeper, sdk.AccAddress(addr.Bytes()))
				suite.Require().NoError(err, "error while querying delegation total rewards")

				// NOTE: the total rewards should be the same as after the setup, since
				// the fees are paid using the account balance
				suite.Require().Equal(
					sdk.NewDecCoins(sdk.NewDecCoin(constants.BaseDenom, sdk.NewInt(1e16))),
					rewards,
					"the total rewards should be the same as after the setup, since the fees are paid using the account balance",
				)
			},
		},
		{
			name:     "success - zero fees (disabled base fee + min gas price)",
			tx:       zeroFeeLegacyTx,
			gasLimit: zeroFeeLegacyTx.GetGas(),
			malleate: func(ctx sdk.Context) sdk.Context {
				suite.disableBaseFee(ctx)
				suite.disableMinGasPrice(ctx)
				return ctx
			},
			expPass:     true,
			expPanic:    false,
			expPriority: 0,
			postCheck: func(ctx sdk.Context) {
				finalBalance := suite.app.BankKeeper.GetBalance(ctx, zeroBalanceAddr.Bytes(), constants.BaseDenom)
				suite.Require().True(finalBalance.IsZero())
			},
		},
		{
			name:     "success - zero fees (disabled base fee + min gas price) - legacy tx",
			tx:       zeroFeeLegacyTx,
			gasLimit: zeroFeeLegacyTx.GetGas(),
			malleate: func(ctx sdk.Context) sdk.Context {
				suite.disableBaseFee(ctx)
				suite.disableMinGasPrice(ctx)
				return ctx
			},
			expPass:     true,
			expPanic:    false,
			expPriority: 0,
			postCheck: func(ctx sdk.Context) {
				finalBalance := suite.app.BankKeeper.GetBalance(ctx, zeroBalanceAddr.Bytes(), constants.BaseDenom)
				suite.Require().True(finalBalance.IsZero())
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			cacheCtx, _ := suite.ctx.CacheContext()
			// Create new stateDB for each test case from the cached context
			vmdb = testutil.NewStateDB(cacheCtx, suite.app.EvmKeeper)
			cacheCtx = tc.malleate(cacheCtx)
			suite.Require().NoError(vmdb.Commit())

			if tc.expPanic {
				suite.Require().Panics(func() {
					_, _ = dec.AnteHandle(cacheCtx.WithIsCheckTx(true).WithGasMeter(sdk.NewGasMeter(1)), tc.tx, false, testutil.NextFn)
				})
				return
			}

			ctx, err := dec.AnteHandle(cacheCtx.WithIsCheckTx(true).WithGasMeter(sdk.NewInfiniteGasMeter()), tc.tx, false, testutil.NextFn)
			if tc.expPass {
				suite.Require().NoError(err)
				suite.Require().Equal(tc.expPriority, ctx.Priority())
			} else {
				suite.Require().Error(err)
			}
			suite.Require().Equal(tc.gasLimit, ctx.GasMeter().Limit())

			// check state after the test case
			tc.postCheck(ctx)
		})
	}
}

func (suite *AnteTestSuite) TestCanTransferDecorator() {
	dec := ethante.NewCanTransferDecorator(suite.app.EvmKeeper)

	addr, privKey := testutiltx.NewAddrKey()

	suite.app.FeeMarketKeeper.SetBaseFee(suite.ctx, big.NewInt(100))
	ethContractCreationTxParams := &evmtypes.EvmTxArgs{
		ChainID:   suite.app.EvmKeeper.ChainID(),
		Nonce:     1,
		Amount:    big.NewInt(10),
		GasLimit:  1000,
		GasPrice:  big.NewInt(1),
		GasFeeCap: big.NewInt(150),
		GasTipCap: big.NewInt(200),
		Accesses:  &ethtypes.AccessList{},
	}

	tx := evmtypes.NewTx(ethContractCreationTxParams)
	tx2 := evmtypes.NewTx(ethContractCreationTxParams)

	tx.From = addr.Hex()

	err := tx.Sign(suite.ethSigner, testutiltx.NewSigner(privKey))
	suite.Require().NoError(err)

	var vmdb *statedb.StateDB

	testCases := []struct {
		name     string
		tx       sdk.Tx
		malleate func()
		expPass  bool
	}{
		{"invalid transaction type", &testutiltx.InvalidTx{}, func() {}, false},
		{"AsMessage failed", tx2, func() {}, false},
		{
			"evm CanTransfer failed",
			tx,
			func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			false,
		},
		{
			"success",
			tx,
			func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

				vmdb.AddBalance(addr, big.NewInt(1000000))
			},
			true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			vmdb = testutil.NewStateDB(suite.ctx, suite.app.EvmKeeper)
			tc.malleate()
			suite.Require().NoError(vmdb.Commit())

			_, err := dec.AnteHandle(suite.ctx.WithIsCheckTx(true), tc.tx, false, testutil.NextFn)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *AnteTestSuite) TestEthIncrementSenderSequenceDecorator() {
	dec := ethante.NewEthIncrementSenderSequenceDecorator(suite.app.AccountKeeper)
	addr, privKey := testutiltx.NewAddrKey()

	ethTxContractParamsNonce0 := &evmtypes.EvmTxArgs{
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    0,
		Amount:   big.NewInt(10),
		GasLimit: 1000,
		GasPrice: big.NewInt(1),
	}
	contract := evmtypes.NewTx(ethTxContractParamsNonce0)
	contract.From = addr.Hex()
	err := contract.Sign(suite.ethSigner, testutiltx.NewSigner(privKey))
	suite.Require().NoError(err)

	to := testutiltx.GenerateAddress()
	ethTxParamsNonce0 := &evmtypes.EvmTxArgs{
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    0,
		To:       &to,
		Amount:   big.NewInt(10),
		GasLimit: 1000,
		GasPrice: big.NewInt(1),
	}
	tx := evmtypes.NewTx(ethTxParamsNonce0)
	tx.From = addr.Hex()
	err = tx.Sign(suite.ethSigner, testutiltx.NewSigner(privKey))
	suite.Require().NoError(err)

	ethTxParamsNonce1 := &evmtypes.EvmTxArgs{
		ChainID:  suite.app.EvmKeeper.ChainID(),
		Nonce:    1,
		To:       &to,
		Amount:   big.NewInt(10),
		GasLimit: 1000,
		GasPrice: big.NewInt(1),
	}
	tx2 := evmtypes.NewTx(ethTxParamsNonce1)
	tx2.From = addr.Hex()
	err = tx2.Sign(suite.ethSigner, testutiltx.NewSigner(privKey))
	suite.Require().NoError(err)

	testCases := []struct {
		name     string
		tx       sdk.Tx
		malleate func()
		expPass  bool
		expPanic bool
	}{
		{
			"invalid transaction type",
			&testutiltx.InvalidTx{},
			func() {},
			false, false,
		},
		{
			"no signers",
			evmtypes.NewTx(ethTxParamsNonce1),
			func() {},
			false, false,
		},
		{
			"account not set to store",
			tx,
			func() {},
			false, false,
		},
		{
			"success - create contract",
			contract,
			func() {
				acc := suite.app.AccountKeeper.NewAccountWithAddress(suite.ctx, addr.Bytes())
				suite.app.AccountKeeper.SetAccount(suite.ctx, acc)
			},
			true, false,
		},
		{
			"success - call",
			tx2,
			func() {},
			true, false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.malleate()

			if tc.expPanic {
				suite.Require().Panics(func() {
					_, _ = dec.AnteHandle(suite.ctx, tc.tx, false, testutil.NextFn)
				})
				return
			}

			_, err := dec.AnteHandle(suite.ctx, tc.tx, false, testutil.NextFn)

			if tc.expPass {
				suite.Require().NoError(err)
				msg := tc.tx.(*evmtypes.MsgEthereumTx)

				txData, err := evmtypes.UnpackTxData(msg.Data)
				suite.Require().NoError(err)

				nonce := suite.app.EvmKeeper.GetNonce(suite.ctx, addr)
				suite.Require().Equal(txData.GetNonce()+1, nonce)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *AnteTestSuite) TestValidateBasicDecorator() {
	dec := ethante.NewEthBasicValidationDecorator()

	getTx := func(f func(args *evmtypes.EvmTxArgs)) *evmtypes.MsgEthereumTx {
		evmTxArgs := &evmtypes.EvmTxArgs{
			ChainID:   suite.app.EvmKeeper.ChainID(),
			Nonce:     1,
			Amount:    nil,
			GasLimit:  1000,
			GasPrice:  big.NewInt(1),
			GasFeeCap: big.NewInt(150),
			GasTipCap: big.NewInt(200),
			Accesses:  &ethtypes.AccessList{},
		}
		f(evmTxArgs)
		return evmtypes.NewTx(evmTxArgs)
	}

	testCases := []struct {
		name     string
		tx       func() sdk.Tx
		expPass  bool
		expPanic bool
	}{
		{
			name: "invalid transaction type",
			tx: func() sdk.Tx {
				return &testutiltx.InvalidTx{}
			},
			expPass: false,
		},
		{
			name: "accept positive value",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.Amount = big.NewInt(10)
				})
			},
			expPass: true,
		},
		{
			name: "accept zero value",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.Amount = big.NewInt(0)
				})
			},
			expPass: true,
		},
		{
			name: "accept nil value",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.Amount = nil
				})
			},
			expPass: true,
		},
		{
			name: "reject negative value",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.Amount = big.NewInt(-10)
				})
			},
			expPass: false,
		},
		{
			name: "reject value which more than 256 bits",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					bz := make([]byte, 257)
					bz[0] = 0xFF
					args.Amount = new(big.Int).SetBytes(bz)
				})
			},
			expPanic: true,
		},
		{
			name: "accept positive gas price",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = big.NewInt(10)
				})
			},
			expPass: true,
		},
		{
			name: "not reject zero gas price",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = big.NewInt(0)
				})
			},
			expPass: true,
		},
		{
			name: "not reject nil gas price",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil
				})
			},
			expPass: true,
		},
		{
			name: "reject negative gas price",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = big.NewInt(-10)
					args.GasFeeCap = nil
					args.GasTipCap = nil
				})
			},
			expPass: false,
		},
		{
			name: "reject gas price which more than 256 bits",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					bz := make([]byte, 257)
					bz[0] = 0xFF
					args.GasPrice = new(big.Int).SetBytes(bz)
				})
			},
			expPanic: true,
		},
		{
			name: "accept positive gas fee cap",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					args.GasFeeCap = big.NewInt(10)
				})
			},
			expPass: true,
		},
		{
			name: "not reject zero gas fee cap",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					args.GasFeeCap = big.NewInt(0)
				})
			},
			expPass: true,
		},
		{
			name: "not reject nil gas fee cap",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					args.GasFeeCap = nil
				})
			},
			expPass: true,
		},
		{
			name: "reject negative gas fee cap",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					args.GasFeeCap = big.NewInt(-10)
				})
			},
			expPass: false,
		},
		{
			name: "reject gas fee cap which more than 256 bits",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					bz := make([]byte, 257)
					bz[0] = 0xFF
					args.GasFeeCap = new(big.Int).SetBytes(bz)
				})
			},
			expPanic: true,
		},
		{
			name: "accept positive gas tip cap",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					args.GasTipCap = big.NewInt(10)
				})
			},
			expPass: true,
		},
		{
			name: "not reject zero gas tip cap",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					args.GasTipCap = big.NewInt(0)
				})
			},
			expPass: true,
		},
		{
			name: "not reject nil gas tip cap",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					args.GasTipCap = nil
				})
			},
			expPass: true,
		},
		{
			name: "reject negative gas tip cap",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					args.GasTipCap = big.NewInt(-10)
				})
			},
			expPass: false,
		},
		{
			name: "reject gas tip cap which more than 256 bits",
			tx: func() sdk.Tx {
				return getTx(func(args *evmtypes.EvmTxArgs) {
					args.GasPrice = nil

					bz := make([]byte, 257)
					bz[0] = 0xFF
					args.GasTipCap = new(big.Int).SetBytes(bz)
				})
			},
			expPanic: true,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			if tc.expPanic {
				suite.Require().Panics(func() {
					_, _ = dec.AnteHandle(suite.ctx, tc.tx(), false, testutil.NextFn)
				})
				return
			}

			_, err := dec.AnteHandle(suite.ctx, tc.tx(), false, testutil.NextFn)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}
