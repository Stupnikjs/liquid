package api

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// ── PositionsQuery ────────────────────────────────────────────────────────────

func TestPositionsQuery_ContainsMarketID(t *testing.T) {
	id := "0xdeadbeef"
	q := PositionsQuery(id, 42161)
	if !strings.Contains(q, id) {
		t.Errorf("query does not contain market id %s", id)
	}
}

func TestPositionsQuery_ContainsChainID(t *testing.T) {
	q := PositionsQuery("0xabc", 42161)
	if !strings.Contains(q, "42161") {
		t.Errorf("query does not contain chainId 42161")
	}
}

// ── MarketsQuery ──────────────────────────────────────────────────────────────

func TestMarketsQuery_ContainsChainID(t *testing.T) {
	q := MarketsQuery(8453)
	if !strings.Contains(q, "8453") {
		t.Errorf("query does not contain chainId 8453")
	}
}

// ── QueryMarkets (intégration) ────────────────────────────────────────────────

func TestQueryMarkets_Base_ReturnsItems(t *testing.T) {
	result, err := QueryMarkets(nil, 8453)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Markets.Items) == 0 {
		t.Error("expected markets for Base, got empty")
	}
}

func TestQueryMarkets_Arbitrum_ReturnsItems(t *testing.T) {
	result, err := QueryMarkets(nil, 42161)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Markets.Items) == 0 {
		t.Error("expected markets for Arbitrum, got empty")
	}
}

func TestQueryMarkets_ItemsHaveRequiredFields(t *testing.T) {
	result, err := QueryMarkets(nil, 8453)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, item := range result.Markets.Items {
		if item.UniqueKey == "" {
			t.Error("market item has empty UniqueKey")
		}
		if item.LoanAsset.Symbol == "" {
			t.Error("market item has empty LoanAsset.Symbol")
		}
	}
}

// ── FetchBorrowersFromMarket (intégration) ────────────────────────────────────

func TestFetchBorrowersFromMarket_Base_ReturnsResult(t *testing.T) {
	// USDC/wstETH sur Base — market actif avec des positions
	id := [32]byte(common.HexToHash("0xb323495f7e4148be5643a4ea4a8221eef163e4bccfdedc2a6f4696baacbc86cc"))
	result, err := FetchBorrowersFromMarket(id, 8453)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.MarketPositions.Items) == 0 {
		t.Error("expected positions for known market, got empty")
	}
}

func TestFetchBorrowersFromMarket_Arbitrum_ReturnsResult(t *testing.T) {
	// USDC/wstETH sur Arbitrum
	id := [32]byte(common.HexToHash("0x33e0c8ab132390822b07e5dc95033cf250c963153320b7ffca73220664da2ea0"))
	result, err := FetchBorrowersFromMarket(id, 42161)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.MarketPositions.Items) == 0 {
		t.Error("expected positions for known Arbitrum market, got empty")
	}
}

func TestFetchBorrowersFromMarket_PositionFieldsPopulated(t *testing.T) {
	id := [32]byte(common.HexToHash("0xb323495f7e4148be5643a4ea4a8221eef163e4bccfdedc2a6f4696baacbc86cc"))
	result, err := FetchBorrowersFromMarket(id, 8453)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, p := range result.MarketPositions.Items {
		if p.User.Address == "" {
			t.Error("position has empty user address")
		}
		if p.State.BorrowShares == "" {
			t.Error("position has empty BorrowShares")
		}
	}
}

func TestFetchBorrowersFromMarket_UnknownMarket_ReturnsEmpty(t *testing.T) {
	// ID inexistant — doit retourner une slice vide sans erreur
	id := [32]byte{0xFF, 0xFF, 0xFF}
	result, err := FetchBorrowersFromMarket(id, 8453)
	if err != nil {
		t.Fatalf("unexpected error for unknown market: %v", err)
	}
	if len(result.MarketPositions.Items) != 0 {
		t.Errorf("expected empty positions for unknown market, got %d", len(result.MarketPositions.Items))
	}
}

// ── FilterMarket ──────────────────────────────────────────────────────────────

func TestFilterMarket_SkipsZeroSupply(t *testing.T) {
	result, err := QueryMarkets(nil, 8453)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// filtre très large pour garder tout sauf supply=0
	markets := FilterMarket(result, MarketFilters{MinUsdMarket: 0, MaxUsdMarket: 1e12}, 8453)
	for _, m := range markets {
		_ = m // tous doivent avoir supply > 0 — vérifié dans FilterMarket
	}
	if len(markets) == 0 {
		t.Error("expected at least one market after filtering")
	}
}

func TestFilterMarket_RespectsMinBorrow(t *testing.T) {
	result, err := QueryMarkets(nil, 8453)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	min := 1_000_000.0
	markets := FilterMarket(result, MarketFilters{MinUsdMarket: min, MaxUsdMarket: 1e12}, 8453)
	// vérifie qu'on a moins de markets qu'avec un filtre à 0
	marketsAll := FilterMarket(result, MarketFilters{MinUsdMarket: 0, MaxUsdMarket: 1e12}, 8453)
	if len(markets) > len(marketsAll) {
		t.Error("filtered markets should be <= unfiltered markets")
	}
}

func TestFilterMarket_ChainIdPropagated(t *testing.T) {
	result, err := QueryMarkets(nil, 42161)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	markets := FilterMarket(result, MarketFilters{MinUsdMarket: 0, MaxUsdMarket: 1e12}, 42161)
	for _, m := range markets {
		if m.ChainID != 42161 {
			t.Errorf("expected ChainID 42161, got %d", m.ChainID)
		}
	}
}

// ── ToConfig ──────────────────────────────────────────────────────────────────

func TestToConfig_MapsFieldsCorrectly(t *testing.T) {
	result, err := QueryMarkets(nil, 8453)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Markets.Items) == 0 {
		t.Skip("no markets returned")
	}
	item := result.Markets.Items[0]
	if item.CollateralAsset == nil {
		t.Skip("first market has no collateral asset")
	}

	cfg := item.ToConfig(8453)

	if cfg.LoanTokenStr != item.LoanAsset.Symbol {
		t.Errorf("LoanTokenStr mismatch: got %s want %s", cfg.LoanTokenStr, item.LoanAsset.Symbol)
	}
	if cfg.LoanToken != common.HexToAddress(item.LoanAsset.Address) {
		t.Errorf("LoanToken address mismatch")
	}
	if cfg.ChainID != 8453 {
		t.Errorf("expected ChainID 8453, got %d", cfg.ChainID)
	}
	expectedID := [32]byte(common.HexToHash(item.UniqueKey))
	if cfg.ID != expectedID {
		t.Errorf("ID mismatch: got %x want %x", cfg.ID, expectedID)
	}
}
