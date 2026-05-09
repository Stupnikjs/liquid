package testutil



var (
	weth = common.HexToAddress("0x4200000000000000000000000000000000000006")
	me   = common.HexToAddress(FundedAccounts[0])

	usdc       = common.HexToAddress("0x833589fCD6EDB6E08f4c7C32D4f71b54bdA02913")
	morphoBlue = common.HexToAddress("0xBBBBBbbBBb9cC5e90e3b3Af64bdAF62C37EEFFCb")

	// approve(address,uint256)
	approveSelector = []byte{0x09, 0x5e, 0xa7, 0xb3}

	// supplyCollateral((address,address,address,address,uint256),uint256,address,bytes)
	supplyCollateralSelector = []byte{0x23, 0x8d, 0x65, 0x79}

	// borrow((address,address,address,address,uint256),uint256,uint256,address,address)
	borrowSelector = []byte{0x50, 0xd8, 0xcd, 0x4b}

	balanceOfSelector = []byte{0x70, 0xa0, 0x82, 0x31}
)

// depositData retourne le calldata de deposit() : sélecteur 0xd0e30db0
var depositCalldata = []byte{0xd0, 0xe3, 0x0d, 0xb0}

var market = MarketParams{
	LoanToken:       usdc,
	CollateralToken: weth,
	Oracle:          common.HexToAddress("0x2DC205F24BCb6B311E5cdf0745B0741648Aebd3d"),
	Irm:             common.HexToAddress("0x4647B8FfC145fF3D82BDAf0222B869bDAa6072bE"),
	LLTV:            big.NewInt(860000000000000000),
}

type MarketParams struct {
	LoanToken       common.Address
	CollateralToken common.Address
	Oracle          common.Address
	Irm             common.Address
	LLTV            *big.Int
}


func encodeMarketParams(m MarketParams) []byte {
	buf := make([]byte, 5*32)
	copy(buf[12:32], m.LoanToken.Bytes())
	copy(buf[32+12:64], m.CollateralToken.Bytes())
	copy(buf[64+12:96], m.Oracle.Bytes())
	copy(buf[96+12:128], m.Irm.Bytes())
	m.LLTV.FillBytes(buf[128:160])
	return buf
}

func sendAndWait(t *testing.T, client *ethclient.Client, ctx context.Context,
	nonce uint64, gasPrice *big.Int, chainID *big.Int,
	to common.Address, value *big.Int, data []byte,
	privKey *ecdsa.PrivateKey) *types.Receipt {

	t.Helper()

	gas, err := client.EstimateGas(ctx, ethereum.CallMsg{
		From:  me,
		To:    &to,
		Value: value,
		Data:  data,
	})
	if err != nil {
		t.Fatalf("estimateGas: %v", err)
	}

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    value,
		Gas:      gas,
		GasPrice: gasPrice,
		Data:     data,
	})

	signer := types.LatestSignerForChainID(chainID)
	signed, err := types.SignTx(tx, signer, privKey)
	if err != nil {
		t.Fatalf("signTx: %v", err)
	}

	if err := client.SendTransaction(ctx, signed); err != nil {
		t.Fatalf("sendTransaction: %v", err)
	}

	t.Logf("txHash: %s", signed.Hash())

	var receipt *types.Receipt
	for i := 0; i < 20; i++ {
		time.Sleep(300 * time.Millisecond)
		receipt, err = client.TransactionReceipt(ctx, signed.Hash())
		if err == nil {
			break
		}
	}
	if receipt == nil {
		t.Fatalf("receipt non trouvé après timeout")
	}
	if receipt.Status != types.ReceiptStatusSuccessful {
		t.Fatalf("tx revertée, status: %d", receipt.Status)
	}
	return receipt
}

