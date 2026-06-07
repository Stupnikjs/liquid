package morpho

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// ---------------------------------------------------------------------------
// ABI JSON
// ---------------------------------------------------------------------------

const MorphoBlueABIJson = `[
    {
        "name": "market",
        "type": "function",
        "inputs": [{"name": "id", "type": "bytes32"}],
        "outputs": [
            {"name": "totalSupplyAssets", "type": "uint128"},
            {"name": "totalSupplyShares", "type": "uint128"},
            {"name": "totalBorrowAssets", "type": "uint128"},
            {"name": "totalBorrowShares", "type": "uint128"},
            {"name": "lastUpdate",        "type": "uint128"},
            {"name": "fee",               "type": "uint128"}
        ],
        "stateMutability": "view"
    }
]`

const MorphoOracleABIJson = `[
    {
        "name": "price",
        "type": "function",
        "inputs": [],
        "outputs": [
            {"name": "", "type": "uint256"}
        ],
        "stateMutability": "view"
    }
]`

const MorphoEventsABIJson = `[
    {
        "name": "Borrow",
        "type": "event",
        "inputs": [
            {"name": "id",       "type": "bytes32", "indexed": true},
            {"name": "caller",   "type": "address", "indexed": false},
            {"name": "onBehalf", "type": "address", "indexed": true},
            {"name": "receiver", "type": "address", "indexed": false},
            {"name": "assets",   "type": "uint256", "indexed": false},
            {"name": "shares",   "type": "uint256", "indexed": false}
        ]
    },
    {
        "name": "Repay",
        "type": "event",
        "inputs": [
            {"name": "id",       "type": "bytes32", "indexed": true},
            {"name": "caller",   "type": "address", "indexed": false},
            {"name": "onBehalf", "type": "address", "indexed": true},
            {"name": "assets",   "type": "uint256", "indexed": false},
            {"name": "shares",   "type": "uint256", "indexed": false}
        ]
    },
    {
        "name": "SupplyCollateral",
        "type": "event",
        "inputs": [
            {"name": "id",       "type": "bytes32", "indexed": true},
            {"name": "caller",   "type": "address", "indexed": false},
            {"name": "onBehalf", "type": "address", "indexed": true},
            {"name": "assets",   "type": "uint256", "indexed": false}
        ]
    },
    {
        "name": "Liquidate",
        "type": "event",
        "inputs": [
            {"name": "id",            "type": "bytes32", "indexed": true},
            {"name": "caller",        "type": "address", "indexed": false},
            {"name": "borrower",      "type": "address", "indexed": true},
            {"name": "repaidAssets",  "type": "uint256", "indexed": false},
            {"name": "repaidShares",  "type": "uint256", "indexed": false},
            {"name": "seizedAssets",  "type": "uint256", "indexed": false},
            {"name": "badDebtAssets", "type": "uint256", "indexed": false},
            {"name": "badDebtShares", "type": "uint256", "indexed": false}
        ]
    },
    {
        "name": "AccrueInterest",
        "type": "event",
        "inputs": [
            {"name": "id",             "type": "bytes32", "indexed": true},
            {"name": "prevBorrowRate", "type": "uint256", "indexed": false},
            {"name": "interest",       "type": "uint256", "indexed": false},
            {"name": "feeShares",      "type": "uint256", "indexed": false}
        ]
    }
]`

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

type MorphoBlueABI struct {
	Market *abi.Method
}

type MorphoOracleABI struct {
	Price *abi.Method
}

type MorphoEvents struct {
	Borrow           abi.Event
	Repay            abi.Event
	SupplyCollateral abi.Event
	Liquidate        abi.Event
	AccrueInterest   abi.Event
}

type MorphoTopics struct {
	Borrow           common.Hash
	Repay            common.Hash
	SupplyCollateral common.Hash
	Liquidate        common.Hash
	AccrueInterest   common.Hash
}

type MorphoABI struct {
	Blue   MorphoBlueABI
	Oracle MorphoOracleABI
	Events MorphoEvents
	Topics MorphoTopics
}

// ---------------------------------------------------------------------------
// Loader
// ---------------------------------------------------------------------------

func LoadMorphoABI() (*MorphoABI, error) {
	blueParsed, err := abi.JSON(strings.NewReader(MorphoBlueABIJson))
	if err != nil {
		return nil, fmt.Errorf("morpho blue abi: %w", err)
	}

	oracleParsed, err := abi.JSON(strings.NewReader(MorphoOracleABIJson))
	if err != nil {
		return nil, fmt.Errorf("morpho oracle abi: %w", err)
	}

	eventsParsed, err := abi.JSON(strings.NewReader(MorphoEventsABIJson))
	if err != nil {
		return nil, fmt.Errorf("morpho events abi: %w", err)
	}

	marketMethod := blueParsed.Methods["market"]
	priceMethod := oracleParsed.Methods["price"]

	m := &MorphoABI{
		Blue:   MorphoBlueABI{Market: &marketMethod},
		Oracle: MorphoOracleABI{Price: &priceMethod},
	}

	m.Events.Borrow = eventsParsed.Events["Borrow"]
	m.Events.Repay = eventsParsed.Events["Repay"]
	m.Events.SupplyCollateral = eventsParsed.Events["SupplyCollateral"]
	m.Events.Liquidate = eventsParsed.Events["Liquidate"]
	m.Events.AccrueInterest = eventsParsed.Events["AccrueInterest"]

	m.Topics.Borrow = m.Events.Borrow.ID
	m.Topics.Repay = m.Events.Repay.ID
	m.Topics.SupplyCollateral = m.Events.SupplyCollateral.ID
	m.Topics.Liquidate = m.Events.Liquidate.ID
	m.Topics.AccrueInterest = m.Events.AccrueInterest.ID

	return m, nil
}
