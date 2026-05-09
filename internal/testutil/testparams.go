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


