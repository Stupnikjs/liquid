package config

import (
	"fmt"
	"log"
	"os"

	"github.com/Stupnikjs/liquid/internal/lqtypes"
	"github.com/Stupnikjs/liquid/internal/swap"
	"github.com/joho/godotenv"
)

// ---------------------------------------------------------------------------
// Chain descriptor — one per supported chain
// ---------------------------------------------------------------------------

type chainDesc struct {
	chainID   int64
	addresses lqtypes.Addresses
	rpc       struct{ http, ws [2]string }
	dexs      []lqtypes.Dex
	newSigner func(int64) (lqtypes.Signer, error)
}

// ---------------------------------------------------------------------------
// Chain descriptors
// ---------------------------------------------------------------------------

func baseDesc() chainDesc {
	return chainDesc{
		chainID:   8453,
		newSigner: lqtypes.NewBaseSigner,
		addresses: lqtypes.Addresses{
			UniSwapRouter:      BaseUniswapV3Router,
			UniSwapQuoter:      BaseUniswapQuoterV2Addr,
			LiquidatorContract: BaseLiquidatorNew,
			Morpho:             MorphoMain,
			Wallet:             BaseWalletAddr,
		},
		rpc: struct{ http, ws [2]string }{
			http: [2]string{os.Getenv("BASE_HTTP_RPC_DRPC"), os.Getenv("BASE_HTTP_RPC_ALCH")},
			ws:   [2]string{os.Getenv("BASE_WS_RPC_DRPC"), os.Getenv("BASE_WS_RPC_ALCH")},
		},
		dexs: []lqtypes.Dex{
			{QuoterAddr: BasePankakeSwapQuoterV2Addr, RouterAddr: BasePankakeSwapV3Router, Name: "PANCAKE"},
			{QuoterAddr: BaseUniswapQuoterV2Addr, RouterAddr: BaseUniswapV3Router, Name: "UNIV3"},
		},
	}
}

func arbitrumDesc() chainDesc {
	return chainDesc{
		chainID:   42161,
		newSigner: lqtypes.NewArbitrumSigner,
		addresses: lqtypes.Addresses{
			Wallet:             ArbitrumWalletAddress,
			UniSwapRouter:      ArbitrumUniswapV3Router,
			UniSwapQuoter:      ArbitrumUniswapQuoterV2Addr,
			LiquidatorContract: ArbitrumLiquidatorAddr,
			Morpho:             ArbitrumMorphoBlueAddr,
		},
		rpc: struct{ http, ws [2]string }{
			http: [2]string{os.Getenv("ARB_HTTP_RPC_ALCH"), os.Getenv("ARB_HTTP_RPC_ALCH")},
			ws:   [2]string{os.Getenv("ARB_WS_RPC_ALCH"), os.Getenv("ARB_WS_RPC_ALCH")},
		},
		dexs: []lqtypes.Dex{
			{QuoterAddr: ArbitrumUniswapQuoterV2Addr, RouterAddr: ArbitrumUniswapV3Router, Name: "UNIV3"},
		},
	}
}

func optimismDesc() chainDesc {
	return chainDesc{
		chainID:   10,
		newSigner: lqtypes.NewOptimismSigner,
		addresses: lqtypes.Addresses{
			Wallet:             OptimismWalletAddress,
			UniSwapRouter:      OptimismUniswapV3Router,
			UniSwapQuoter:      OptimismUniswapQuoterV2Addr,
			LiquidatorContract: OptimismLiquidatorAddr,
			Morpho:             OptimismMorphoBlueAddr,
		},
		rpc: struct{ http, ws [2]string }{
			http: [2]string{os.Getenv("OPT_HTTP_RPC_ALCH"), os.Getenv("OPT_HTTP_RPC_ALCH")},
			ws:   [2]string{os.Getenv("OPT_WS_RPC_ALCH"), os.Getenv("OPT_WS_RPC_ALCH")},
		},
		dexs: []lqtypes.Dex{
			{QuoterAddr: OptimismUniswapQuoterV2Addr, RouterAddr: OptimismUniswapV3Router, Name: "UNIV3"},
		},
	}
}

func unichainDesc() chainDesc {
	return chainDesc{
		chainID:   130,
		newSigner: lqtypes.NewUniChainSigner,
		addresses: lqtypes.Addresses{
			Wallet:             UnichainWalletAddress,
			UniSwapRouter:      UnichainUniswapV3Router,
			UniSwapQuoter:      UnichainUniswapQuoterV2Addr,
			LiquidatorContract: UnichainLiquidatorAddr,
			Morpho:             UnichainMorphoBlueAddr,
		},
		rpc: struct{ http, ws [2]string }{
			http: [2]string{os.Getenv("UNICHAIN_HTTP_RPC_ALCH"), os.Getenv("UNICHAIN_HTTP_RPC_ALCH")},
			ws:   [2]string{os.Getenv("UNICHAIN_WS_RPC_ALCH"), os.Getenv("UNICHAIN_WS_RPC_ALCH")},
		},
	}
}

func worldChainDesc() chainDesc {
	return chainDesc{
		chainID:   480,
		newSigner: lqtypes.NewWorldChainSigner,
		addresses: lqtypes.Addresses{
			Wallet:             WorldChainWalletAddress,
			UniSwapRouter:      WorldChainUniswapV3Router,
			UniSwapQuoter:      WorldChainUniswapQuoterV2Addr,
			LiquidatorContract: WorldChainLiquidatorAddr,
			Morpho:             WorldChainMorphoBlueAddr,
		},
		rpc: struct{ http, ws [2]string }{
			http: [2]string{os.Getenv("WORLDCHAIN_HTTP_RPC_DRPC"), os.Getenv("WORLDCHAIN_HTTP_RPC_ALCH")},
			ws:   [2]string{os.Getenv("WORLDCHAIN_WS_RPC_ALCH"), os.Getenv("WORLDCHAIN_WS_RPC_ALCH")},
		},
	}
}

func katanaDesc() chainDesc {
	return chainDesc{
		chainID:   747474,
		newSigner: lqtypes.NewKatanaSigner,
		addresses: lqtypes.Addresses{
			Wallet:             KatanaWalletAddress,
			UniSwapRouter:      KatanaUniswapV3Router,
			UniSwapQuoter:      KatanaUniswapQuoterV2Addr,
			LiquidatorContract: KatanaLiquidatorAddr,
			Morpho:             KatanaMorphoBlueAddr,
		},
		rpc: struct{ http, ws [2]string }{
			http: [2]string{os.Getenv("KATANA_HTTP_RPC_DRPC"), os.Getenv("KATANA_HTTP_RPC_ALCH")},
			ws:   [2]string{os.Getenv("KATANA_WS_RPC_ALCH"), os.Getenv("KATANA_WS_RPC_ALCH")},
		},
		dexs: []lqtypes.Dex{
			{QuoterAddr: KatanaUniswapQuoterV2Addr, RouterAddr: KatanaUniswapV3Router, Name: "SUSHIV3"},
		},
	}
}

// ---------------------------------------------------------------------------
// Generic loader — all Load* functions collapse into this
// ---------------------------------------------------------------------------

func loadConfig(desc chainDesc) lqtypes.Config {
	signer, err := desc.newSigner(desc.chainID)
	if err != nil {
		fmt.Printf("signer error for chainID %d: %v\n", desc.chainID, err)
	}

	// Attach UniQuoter to every dex that doesn't have one set
	dexs := make([]lqtypes.Dex, len(desc.dexs))
	for i, d := range desc.dexs {
		if d.Quoter == nil {
			d.Quoter = swap.UniQuoter
		}
		dexs[i] = d
	}

	return lqtypes.Config{
		Signer:    signer,
		Addresses: desc.addresses,
		ChainID:   uint32(desc.chainID),
		RPC: struct {
			HTTP []string
			WS   []string
		}{
			HTTP: desc.rpc.http[:],
			WS:   desc.rpc.ws[:],
		},
		Dexs: dexs,
	}
}

// ---------------------------------------------------------------------------
// Public API — unchanged signatures, zero breakage
// ---------------------------------------------------------------------------

func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, using system env")
	}
}

func LoadBaseConfig() lqtypes.Config      { return loadConfig(baseDesc()) }
func LoadArbitrumConfig() lqtypes.Config  { return loadConfig(arbitrumDesc()) }
func LoadOptimismConfig() lqtypes.Config  { return loadConfig(optimismDesc()) }
func LoadUnichainConfig() lqtypes.Config  { return loadConfig(unichainDesc()) }
func LoadWorldChainConfig() lqtypes.Config { return loadConfig(worldChainDesc()) }
func LoadKatanaConfig() lqtypes.Config    { return loadConfig(katanaDesc()) }
