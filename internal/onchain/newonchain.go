package onchain

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/Stupnikjs/liquid/internal/cache"
	market "github.com/Stupnikjs/liquid/internal/cache"
	"github.com/Stupnikjs/liquid/pkg/connector"
	"github.com/Stupnikjs/liquid/pkg/morpho"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
)

const rpcTimeout = 5 * time.Second

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

type callMsg struct {
	To   common.Address `json:"to"`
	Data hexutil.Bytes  `json:"data"`
}

type abiCall struct {
	elem   rpc.BatchElem
	raw    *hexutil.Bytes          // pointe sur Result dans elem
	decode func(data []byte) error // spécifique à chaque call
}

// newABICall encode l'appel ABI. decode reçoit les bytes bruts de la réponse
// et est responsable du Unpack vers les variables cibles.
func newABICall(
	to common.Address,
	method *abi.Method,
	args []interface{},
	decode func(outputs abi.Arguments, data []byte) error,
) (*abiCall, error) {
	input, err := method.Inputs.Pack(args...)
	if err != nil {
		return nil, fmt.Errorf("pack %s: %w", method.Name, err)
	}
	calldata := append(method.ID, input...)

	raw := new(hexutil.Bytes)
	return &abiCall{
		elem: rpc.BatchElem{
			Method: "eth_call",
			Args:   []interface{}{callMsg{To: to, Data: calldata}, "latest"},
			Result: raw,
		},
		raw: raw,
		decode: func(data []byte) error {
			return decode(method.Outputs, data)
		},
	}, nil
}

// run appelle decode avec les bytes reçus. À appeler après BatchCallContext.
func (a *abiCall) run() error {
	if a.elem.Error != nil {
		return fmt.Errorf("rpc error: %w", a.elem.Error)
	}
	return a.decode([]byte(*a.raw))
}

// ---------------------------------------------------------------------------
// Constructeurs spécifiques
// ---------------------------------------------------------------------------

func marketCall(morphoAddr common.Address, res *OnChainResult, method *abi.Method) (*abiCall, error) {
	return newABICall(
		morphoAddr,
		method,
		[]interface{}{res.ID},
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

func oracleCall(oracle common.Address, res *OnChainResult, method *abi.Method) (*abiCall, error) {
	return newABICall(
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

func refresh(conn connector.Connector, ctx context.Context, calls []*abiCall) error {
	ctx, cancel := context.WithTimeout(ctx, rpcTimeout)
	defer cancel()

	// extract btch elem
	elems := make([]rpc.BatchElem, len(calls))
	for i, c := range calls {
		elems[i] = c.elem
	}

	// rpc call
	if err := conn.SecondCallCtx(ctx, elems); err != nil {
		return err
	}

	// decoding
	for _, c := range calls {
		if err := c.run(); err != nil {
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
) error {
	res := newResult(mParam.ID)

	call, err := oracleCall(mParam.Oracle, res, oracleMethod)
	if err != nil {
		return fmt.Errorf("OnChainOracleRefresh: %w", err)
	}

	if err := refresh(conn, ctx, []*abiCall{call}); err != nil {
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
	if err := refresh(conn, ctx, []*abiCall{oraclecall, marketcall}); err != nil {
		return fmt.Errorf("OnChainOracleRefresh: %w", err)
	}

	c.Markets.Update(res.ID, func(m *market.Market) {
		m.Oracle.Price = res.OraclePrice
	})

	return nil
}
