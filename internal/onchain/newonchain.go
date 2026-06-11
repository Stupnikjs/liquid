package onchain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	market "github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/pkg/connector"
	"github.com/Stupnikjs/liquid/pkg/ether"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

const rpcTimeout = 20 * time.Second

type OnChainResult struct {
	ID          [32]byte
	Stats       market.MarketStats
	OraclePrice *big.Int
}

func newResult(id [32]byte) *OnChainResult {
	return &OnChainResult{
		ID:          id,
		Stats:       market.MarketStats{},
		OraclePrice: new(big.Int),
	}
}

// newABICall encode l'appel ABI. decode reçoit les bytes bruts de la réponse
// et est responsable du Unpack vers les variables cibles.

// run appelle decode avec les bytes reçus. À appeler après BatchCallContext.

// ---------------------------------------------------------------------------
// Constructeurs spécifiques
// ---------------------------------------------------------------------------

func marketCall(morphoAddr common.Address, res *OnChainResult, method *abi.Method) (*ether.AbiCall, error) {
	return ether.NewABICall(
		morphoAddr,
		method,
		[]any{res.ID},
		func(outputs abi.Arguments, data []byte) error {
			values, err := outputs.Unpack(data)
			if err != nil {
				return err
			}
			// Morpho market() retourne : totalSupplyAssets, totalSupplyShares,
			// totalBorrowAssets, totalBorrowShares, lastUpdate, fee
			res.Stats.TotalBorrowAssets = values[2].(*big.Int)
			res.Stats.TotalBorrowShares = values[3].(*big.Int)
			return nil
		},
	)
}

func oracleCall(oracle common.Address, res *OnChainResult, method *abi.Method) (*ether.AbiCall, error) {
	return ether.NewABICall(
		oracle,
		method,
		nil,
		func(outputs abi.Arguments, data []byte) error {
			values, err := outputs.Unpack(data)
			if err != nil {
				return err
			}
			res.OraclePrice = values[0].(*big.Int)
			return nil
		},
	)
}

func refresh(conn connector.Connector, ctx context.Context, calls []*ether.AbiCall, fastMode bool) error {
	ctx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	// extract btch elem
	elems := make([]rpc.BatchElem, len(calls))
	for i, c := range calls {
		elems[i] = c.Elem
	}

	// rpc call
	if fastMode {
		if err := conn.CallCtx(ctx, elems); err != nil {
			return err
		}
	} else {
		for i := range elems {
			if err := conn.SecondCallCtx(ctx, elems[i]); err != nil {
				return err
			}

		}
	}

	// decoding
	for i, c := range calls {
		c.Elem.Error = elems[i].Error
		if err := c.Run(); err != nil {
			return err
		}
	}
	return nil
}

// OnChainOracleRefresh fetches only the oracle price.
func OnChainOracleRefresh(
	conn connector.Connector,
	ctx context.Context,
	c *cache.Cache,
	mParam morpho.MarketParams,
	oracleMethod *abi.Method,
	fastMode bool,
) error {
	res := newResult(mParam.ID)

	call, err := oracleCall(mParam.Oracle, res, oracleMethod)
	if err != nil {
		return fmt.Errorf("OnChainOracleCallFunc err : %w", err)
	}

	if err := refresh(conn, ctx, []*ether.AbiCall{call}, fastMode); err != nil {
		return fmt.Errorf("OnChainOracleRefresh: %w", err)
	}

	c.Markets.Update(res.ID, func(m *market.Market) {
		m.Oracle.Price = res.OraclePrice
	})

	return nil
}

// OnChainOracleRefresh fetches only the oracle price.
func OnChainRefresh(
	conn connector.Connector,
	morphoAddr common.Address,
	ctx context.Context,
	c *cache.Cache,
	mParam morpho.MarketParams,
	oracleMethod *abi.Method,
	marketMethod *abi.Method,
	fastMode bool,
) error {
	res := newResult(mParam.ID)
	oraclecall, err := oracleCall(mParam.Oracle, res, oracleMethod)
	if err != nil {
		return fmt.Errorf("OnChainOracleRefresh: %w", err)
	}
	marketcall, err := marketCall(morphoAddr, res, marketMethod)
	if err != nil {
		return fmt.Errorf("OnChainOracleRefresh: %w", err)
	}

	if err := refresh(conn, ctx, []*ether.AbiCall{oraclecall, marketcall}, fastMode); err != nil {

		return fmt.Errorf("OnChainRefresh: %w", err)
	}

	c.Markets.Update(res.ID, func(m *market.Market) {
		m.Oracle.Price = res.OraclePrice
		m.Stats.TotalBorrowAssets = res.Stats.TotalBorrowAssets
		m.Stats.TotalBorrowShares = res.Stats.TotalBorrowShares
	})

	return nil
}
