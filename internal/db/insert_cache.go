package db

import (
	"hash/fnv"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

type InsertCache struct {
	state map[string]uint64
}

func NewInsertCache() *InsertCache {
	return &InsertCache{state: make(map[string]uint64)}
}

func hash(borrowShares, collateral, oraclePrice string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(borrowShares))
	h.Write([]byte(collateral))
	h.Write([]byte(oraclePrice))
	return h.Sum64()
}

func (c *InsertCache) Filter(entries []Entry) []Entry {
	var toInsert []Entry
	for _, e := range entries {
		key := e.Pos.Address.String() + hexutil.Encode(e.Pos.MarketID[:])
		h := hash(e.Pos.BorrowShares.String(), e.Pos.CollateralAssets.String(), e.OraclePrice.String())
		if c.state[key] != h {
			c.state[key] = h
			toInsert = append(toInsert, e)
		}
	}
	return toInsert
}
