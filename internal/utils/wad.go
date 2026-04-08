package utils

import (
	"math/big"
)

var (
	WADX11       = new(big.Int).Mul(WAD, big.NewInt(11))
	WADX13       = new(big.Int).Mul(WAD, big.NewInt(13))
	WAD1DOT1     = new(big.Int).Div(WADX11, big.NewInt(10)) // 1.1 WAD
	WAD1DOT3     = new(big.Int).Div(WADX13, big.NewInt(10)) // 1.3 WAD
	WADON10      = new(big.Int).Div(WAD, big.NewInt(10))    // 0.1 WAD
	WAD          = TenPowInt(18)
	WAD1DOT01    = new(big.Int).Add(WAD, new(big.Int).Div(WAD, big.NewInt(100)))   // 1.01%
	WAD1DOT005   = new(big.Int).Add(WAD, new(big.Int).Div(WAD, big.NewInt(200)))   // 1.005%
	WAD1DOT0005  = new(big.Int).Add(WAD, new(big.Int).Div(WAD, big.NewInt(2000)))  // 1.005%
	WAD1DOT00005 = new(big.Int).Add(WAD, new(big.Int).Div(WAD, big.NewInt(20000))) // 1.0005%
	WAD_2        = new(big.Int).Mul(big.NewInt(2), WAD)
	// Oracle thresholds
	OracleThreshold0DOT1Pct = new(big.Int).Div(WAD, big.NewInt(1000)) // 0.1%
	OracleThreshold0DOT5Pct = new(big.Int).Div(WAD, big.NewInt(200))  // 0.5%
	OracleThreshold0DOT2Pct = new(big.Int).Div(WAD, big.NewInt(500))  // 0.2%
)
