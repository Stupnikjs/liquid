package config

import (
	"github.com/lmittmann/w3"
)

var (

	// change by address of deployment
	// ------------------------ FUNC --------------------------------------------------
	// La (pool)
	// market(id) → (totalSupplyAssets, totalSupplyShares, totalBorrowAssets, totalBorrowShares, lastUpdate, fee)

	FuncBorrowRate = w3.MustNewFunc("borrowRate((address,address,address,address,uint256), (uint128,uint128,uint128,uint128,uint128,uint128))", "uint256")
	MarketFunc     = w3.MustNewFunc(
		"market(bytes32)",
		"uint128,uint128,uint128,uint128,uint128,uint128",
	)

	// position(id, user) → (supplyShares, borrowShares, collateral)
	PositionFunc = w3.MustNewFunc(
		"position(bytes32,address)",
		"uint256,uint128,uint128",
	)

	// idToMarketParams(id) → (loanToken, collateralToken, oracle, irm, lltv)
	IdToMarketParamsFunc = w3.MustNewFunc(
		"idToMarketParams(bytes32)",
		"address,address,address,address,uint256",
	)

	OraclePriceFunc = w3.MustNewFunc("price()", "uint256")

	// Borrow a 6 params (caller n'est pas indexé mais est bien là)
	EventSupply           = w3.MustNewEvent("Supply(bytes32 indexed id,address,address,uint256,uint256)")
	EventBorrow           = w3.MustNewEvent("Borrow(bytes32 indexed id,address,address indexed,address indexed,uint256,uint256)")
	EventRepay            = w3.MustNewEvent("Repay(bytes32 indexed id,address indexed,address indexed,uint256,uint256)")
	EventLiquidate        = w3.MustNewEvent("Liquidate(bytes32 indexed id,address indexed,address indexed,uint256,uint256,uint256,uint256, uint256)")
	EventAccrueInterest   = w3.MustNewEvent("AccrueInterest(bytes32 indexed id,uint256,uint256,uint256)")
	EventSupplyCollateral = w3.MustNewEvent("SupplyCollateral(bytes32 indexed id,address indexed,address indexed,uint256)")

	FuncLiquidate = w3.MustNewFunc(`liquidate(
		(address loanToken, address collateralToken, address oracle, address irm, uint256 lltv) marketParams,
		address borrower,
		uint256 seizedAssets,
		uint256 repaidShares,
		bytes odosCalldata
	)`, "")

	// Chainlink v2
	FuncLatestAnswer = w3.MustNewFunc("latestAnswer()", "int256")
	FuncDecimals     = w3.MustNewFunc("decimals()", "uint8")
	// Chainlink v3
	FuncLatestRoundData = w3.MustNewFunc(
		"latestRoundData()",
		"uint80 roundId, int256 answer, uint256 startedAt, uint256 updatedAt, uint80 answeredInRound",
	)

	FuncPrice                 = w3.MustNewFunc("price()", "uint256")
	FuncQuoteExactInputSingle = w3.MustNewFunc(
		"quoteExactInputSingle((address tokenIn, address tokenOut, uint256 amountIn, uint24 fee, uint160 sqrtPriceLimitX96))",
		"uint256 amountOut, uint160 sqrtPriceX96After, uint32 initializedTicksCrossed, uint256 gasEstimate",
	)
)
