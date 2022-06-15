package keeper_test

import (
	"encoding/binary"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/store/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
)

func (suite *KeeperTestSuite) TestStoreNonce() {
	// store setup, you can ignore this mostly...
	storeKey := sdk.NewKVStoreKey("foo")
	ctx := testutil.DefaultContext(storeKey, sdk.NewTransientStoreKey("transient_test"))

	// this is how you would get the store in your keeper: sdkCtx.KVStore(k.storeKey)
	someAddr := authtypes.NewEmptyModuleAccount("foobar")
	account := someAddr.GetAddress()
	store := ctx.KVStore(storeKey)

	var nonce uint64
	bz := store.Get(account.Bytes())
	if bz == nil {
		// there's no nonce, so we should default it.
		nonce = 0
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, nonce+1)
		store.Set(account.Bytes(), b)

		// okay now lets try to get it again.
		// it should be 1 because we set it to 0+1 = 1.
		bz = store.Get(account.Bytes())
		nonce = binary.LittleEndian.Uint64(bz)
		suite.Require().Equal(uint64(1), nonce)

	} else { // this will never run in the test, but showcases how you can get the nonce from the bytes.
		nonce = binary.LittleEndian.Uint64(bz)
	}
}

func (suite *KeeperTestSuite) TestGrantAllowance() {
	oneYear := suite.sdkCtx.BlockTime().AddDate(1, 0, 0)

	testCases := []struct {
		name      string
		req       func() *feegrant.MsgGrantAllowance
		expectErr bool
		errMsg    string
	}{
		{
			"invalid granter address",
			func() *feegrant.MsgGrantAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{})
				suite.Require().NoError(err)
				return &feegrant.MsgGrantAllowance{
					Granter:   "invalid-granter",
					Grantee:   suite.addrs[1].String(),
					Allowance: any,
				}
			},
			true,
			"decoding bech32 failed",
		},
		{
			"invalid grantee address",
			func() *feegrant.MsgGrantAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{})
				suite.Require().NoError(err)
				return &feegrant.MsgGrantAllowance{
					Granter:   suite.addrs[0].String(),
					Grantee:   "invalid-grantee",
					Allowance: any,
				}
			},
			true,
			"decoding bech32 failed",
		},
		{
			"valid: basic fee allowance",
			func() *feegrant.MsgGrantAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{
					SpendLimit: suite.atom,
					Expiration: &oneYear,
				})
				suite.Require().NoError(err)
				return &feegrant.MsgGrantAllowance{
					Granter:   suite.addrs[0].String(),
					Grantee:   suite.addrs[1].String(),
					Allowance: any,
				}
			},
			false,
			"",
		},
		{
			"fail: fee allowance exists",
			func() *feegrant.MsgGrantAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.BasicAllowance{
					SpendLimit: suite.atom,
					Expiration: &oneYear,
				})
				suite.Require().NoError(err)
				return &feegrant.MsgGrantAllowance{
					Granter:   suite.addrs[0].String(),
					Grantee:   suite.addrs[1].String(),
					Allowance: any,
				}
			},
			true,
			"fee allowance already exists",
		},
		{
			"valid: periodic fee allowance",
			func() *feegrant.MsgGrantAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.PeriodicAllowance{
					Basic: feegrant.BasicAllowance{
						SpendLimit: suite.atom,
						Expiration: &oneYear,
					},
				})
				suite.Require().NoError(err)
				return &feegrant.MsgGrantAllowance{
					Granter:   suite.addrs[1].String(),
					Grantee:   suite.addrs[2].String(),
					Allowance: any,
				}
			},
			false,
			"",
		},
		{
			"error: fee allowance exists",
			func() *feegrant.MsgGrantAllowance {
				any, err := codectypes.NewAnyWithValue(&feegrant.PeriodicAllowance{
					Basic: feegrant.BasicAllowance{
						SpendLimit: suite.atom,
						Expiration: &oneYear,
					},
				})
				suite.Require().NoError(err)
				return &feegrant.MsgGrantAllowance{
					Granter:   suite.addrs[1].String(),
					Grantee:   suite.addrs[2].String(),
					Allowance: any,
				}
			},
			true,
			"fee allowance already exists",
		},
	}
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			_, err := suite.msgSrvr.GrantAllowance(suite.ctx, tc.req())
			if tc.expectErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errMsg)
			}
		})
	}
}

func (suite *KeeperTestSuite) TestRevokeAllowance() {
	oneYear := suite.sdkCtx.BlockTime().AddDate(1, 0, 0)

	testCases := []struct {
		name      string
		request   *feegrant.MsgRevokeAllowance
		preRun    func()
		expectErr bool
		errMsg    string
	}{
		{
			"error: invalid granter",
			&feegrant.MsgRevokeAllowance{
				Granter: "invalid-granter",
				Grantee: suite.addrs[1].String(),
			},
			func() {},
			true,
			"decoding bech32 failed",
		},
		{
			"error: invalid grantee",
			&feegrant.MsgRevokeAllowance{
				Granter: suite.addrs[0].String(),
				Grantee: "invalid-grantee",
			},
			func() {},
			true,
			"decoding bech32 failed",
		},
		{
			"error: fee allowance not found",
			&feegrant.MsgRevokeAllowance{
				Granter: suite.addrs[0].String(),
				Grantee: suite.addrs[1].String(),
			},
			func() {},
			true,
			"fee-grant not found",
		},
		{
			"success: revoke fee allowance",
			&feegrant.MsgRevokeAllowance{
				Granter: suite.addrs[0].String(),
				Grantee: suite.addrs[1].String(),
			},
			func() {
				// removing fee allowance from previous tests if exists
				suite.msgSrvr.RevokeAllowance(suite.ctx, &feegrant.MsgRevokeAllowance{
					Granter: suite.addrs[0].String(),
					Grantee: suite.addrs[1].String(),
				})
				any, err := codectypes.NewAnyWithValue(&feegrant.PeriodicAllowance{
					Basic: feegrant.BasicAllowance{
						SpendLimit: suite.atom,
						Expiration: &oneYear,
					},
				})
				suite.Require().NoError(err)
				req := &feegrant.MsgGrantAllowance{
					Granter:   suite.addrs[0].String(),
					Grantee:   suite.addrs[1].String(),
					Allowance: any,
				}
				_, err = suite.msgSrvr.GrantAllowance(suite.ctx, req)
				suite.Require().NoError(err)
			},
			false,
			"",
		},
		{
			"error: check fee allowance revoked",
			&feegrant.MsgRevokeAllowance{
				Granter: suite.addrs[0].String(),
				Grantee: suite.addrs[1].String(),
			},
			func() {},
			true,
			"fee-grant not found",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.preRun()
			_, err := suite.msgSrvr.RevokeAllowance(suite.ctx, tc.request)
			if tc.expectErr {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.errMsg)
			}
		})
	}
}
