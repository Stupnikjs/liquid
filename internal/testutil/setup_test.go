package testutil

/*
var (
	weth = common.HexToAddress("0x4200000000000000000000000000000000000006")
	usdc = common.HexToAddress("0x833589fCD6EDB6E08f4c7C32D4f71b54bdA02913")

	morphoBlue = common.HexToAddress("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb")

	me = common.HexToAddress(FundedAccounts[0])
)

var market = morpho.MarketParams{
	LoanToken:       usdc,
	CollateralToken: weth,
	Oracle:          common.HexToAddress("0x2DC205F24BCb6B311E5cdf0745B0741648Aebd3d"),
	Irm:             common.HexToAddress("0x4647B8FfC145fF3D82BDAf0222B869bDAa6072bE"),
	LLTV:            big.NewInt(860000000000000000),
}

var (
	depositFunc = w3.MustNewFunc("deposit()", "")

	approveFunc = w3.MustNewFunc("approve(address,uint256)", "bool")

	balanceOfFunc = w3.MustNewFunc("balanceOf(address)", "uint256")

	supplyCollateralFunc = w3.MustNewFunc(
		"supplyCollateral((address,address,address,address,uint256),uint256,address,bytes)",
		"",
	)

	borrowFunc = w3.MustNewFunc(
		"borrow((address,address,address,address,uint256),uint256,uint256,address,address)",
		"(uint256,uint256)",
	)
)

func (a *AnvilInstance) BorrowUSDCAgainstWETH(t *testing.T) {
	t.Helper()

	client := a.DialHTTP(t)
	ctx := context.Background()

	// ------------------------------------------------------------
	// Wrap ETH -> WETH
	// ------------------------------------------------------------

	tx := &w3types.Tx{
		To:    &weth,
		From:  me,
		Value: big.NewInt(2e18),
	}
	err := client.CallCtx(
		ctx,
		eth.SendTx(&types.Transaction{}).
			From(me).
			To(&weth).
			Value(big.NewInt(2e18)).
			Func(depositFunc).
			Returns(&txHash),
	)
	if err != nil {
		t.Fatalf("wrap ETH failed: %v", err)
	}
	err := depositFunc.EncodeArgs(tx)
	if err != nil {
		t.Fatalf("encode deposit failed: %v", err)
	}

	err = client.CallCtx(ctx, eth.SendTransaction(tx))
	if err != nil {
		t.Fatalf("wrap ETH failed: %v", err)
	}
	depositData, err := depositFunc.EncodeArgs()
	if err != nil {
		t.Fatalf("encode deposit args failed: %v", err)
	}

	var wrapTxHash common.Hash
	err = client.CallCtx(ctx,
		eth.SendTx(types.NewTx(&types.LegacyTx{
			To:    &weth,
			Value: big.NewInt(2e18),
			Data:  depositData,
		})).Returns(&wrapTxHash),
	)
	if err != nil {
		t.Fatalf("wrap ETH failed: %v", err)
	}
	t.Log("wrapped 2 ETH into WETH")

	// ------------------------------------------------------------
	// Approve Morpho
	// ------------------------------------------------------------

	approveData, err := approveFunc.EncodeArgs(morphoBlue, big.NewInt(1e18))
	if err != nil {
		t.Fatalf("encode approve args failed: %v", err)
	}

	var approveTxHash common.Hash
	err = client.CallCtx(ctx,
		eth.SendTx(types.NewTx(&types.LegacyTx{
			To:   &weth,
			Data: approveData,
		})).Returns(&approveTxHash),
	)
	if err != nil {
		t.Fatalf("approve failed: %v", err)
	}
	t.Log("approved Morpho to spend WETH")

	// ------------------------------------------------------------
	// Supply collateral
	// ------------------------------------------------------------

	supplyData, err := supplyCollateralFunc.EncodeArgs(market, big.NewInt(1e18), me, []byte{})
	if err != nil {
		t.Fatalf("encode supplyCollateral args failed: %v", err)
	}

	var supplyTxHash common.Hash
	err = client.CallCtx(ctx,
		eth.SendTx(types.NewTx(&types.LegacyTx{
			To:   &morphoBlue,
			Data: supplyData,
		})).Returns(&supplyTxHash),
	)
	if err != nil {
		t.Fatalf("supply collateral failed: %v", err)
	}
	t.Log("supplied 1 WETH as collateral")

	// ------------------------------------------------------------
	// Borrow USDC
	// ------------------------------------------------------------

	borrowData, err := borrowFunc.EncodeArgs(market, big.NewInt(500_000000), big.NewInt(0), me, me)
	if err != nil {
		t.Fatalf("encode borrow args failed: %v", err)
	}

	var borrowTxHash common.Hash
	err = client.CallCtx(ctx,
		eth.SendTx(types.NewTx(&types.LegacyTx{
			To:   &morphoBlue,
			Data: borrowData,
		})).Returns(&borrowTxHash),
	)
	if err != nil {
		t.Fatalf("borrow failed: %v", err)
	}
	t.Log("borrowed 500 USDC")

	// ------------------------------------------------------------
	// Read USDC balance
	// ------------------------------------------------------------

	var balance *big.Int
	err = client.CallCtx(ctx,
		eth.CallFunc(usdc, balanceOfFunc, me).Returns(&balance),
	)
	if err != nil {
		t.Fatalf("read USDC balance failed: %v", err)
	}
	t.Logf("USDC balance: %s", balance.String())

}

func TestBorrowUSDCWETH(t *testing.T) {

	a := StartAnvil(t)
	a.BorrowUSDCAgainstWETH(t)
}
*/
