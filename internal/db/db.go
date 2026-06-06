package db

import (
	"context"
	"database/sql"
	"log"
	"math/big"
	"path"
	"sync"

	"github.com/Stupnikjs/liquid/internal/cache"
	"github.com/ethereum/go-ethereum/common/hexutil"
	_ "modernc.org/sqlite"
)

type Store struct {
	DB           *sql.DB
	EntryToFlush []Entry
	Mu           sync.Mutex
}

type Entry struct {
	Pos               cache.BorrowPosition
	TotalBorrowShares *big.Int
	TotalBorrowAssets *big.Int
	OraclePrice       *big.Int
	Ts                int64
}

func PosToEntry(pos cache.BorrowPosition, totalBorrowShares, totalBorrowAssets, oraclePrice *big.Int, ts int64) Entry {
	return Entry{
		Pos:               pos,
		TotalBorrowShares: totalBorrowShares,
		TotalBorrowAssets: totalBorrowAssets,
		OraclePrice:       oraclePrice,
		Ts:                ts,
	}

}

func OpenDb(filename string) (*sql.DB, error) {
	path := path.Join("data", filename)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	err = createTable(db)
	if err != nil {
		log.Fatal(err)
	}
	return db, err

}

func createTable(db *sql.DB) error {
	if _, err := db.ExecContext(context.Background(), `PRAGMA journal_mode=WAL`); err != nil {
		return err
	}
	_, err := db.ExecContext(context.Background(), createPositionsTable)
	return err
}

var createPositionsTable = `CREATE TABLE IF NOT EXISTS position_snapshots (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    borrower_address    TEXT    NOT NULL,
    market_id           TEXT    NOT NULL,
    borrow_shares       TEXT    NOT NULL,
    total_borrow_shares TEXT    NOT NULL,
    total_borrow_assets TEXT    NOT NULL,
    collateral_assets   TEXT    NOT NULL,
    oracle_price        TEXT    NOT NULL,
    health_factor       TEXT,
    snapshot_ts         INTEGER NOT NULL
);`

func InsertEntries(db *sql.DB, entries []Entry) error {
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(context.Background(), `
		INSERT INTO position_snapshots
			(borrower_address, market_id, borrow_shares, total_borrow_shares,
			 total_borrow_assets, collateral_assets, oracle_price, health_factor,
			  snapshot_ts)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, e := range entries {
		_, err := stmt.ExecContext(context.Background(),
			e.Pos.Address.String(),
			hexutil.Encode(e.Pos.MarketID[:]),
			e.Pos.BorrowShares.String(),
			e.TotalBorrowShares.String(),
			e.TotalBorrowAssets.String(),
			e.Pos.CollateralAssets.String(),
			e.OraclePrice.String(),
			e.Pos.CachedHF.String(),
			e.Ts,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

// need mutex here
func (s *Store) FlushEntries() {
	if len(s.EntryToFlush) == 0 {
		return
	}
	err := InsertEntries(s.DB, s.EntryToFlush)
	if err != nil {
		log.Printf("error flushing entries: %v", err)
	}

	s.EntryToFlush = s.EntryToFlush[:0] // Clear the slice

}

/*
func GetPositionAtTS(db *sql.DB, addr, marketID string, ts int64) (*Snapshot, error) {
    var s Snapshot
    err := db.QueryRow(`
        SELECT borrower_address, market_id, borrow_shares, total_borrow_shares,
               total_borrow_assets, collateral_assets, oracle_price, health_factor, snapshot_ts
        FROM position_snapshots
        WHERE borrower_address = ?
          AND market_id = ?
          AND snapshot_ts <= ?
        ORDER BY snapshot_ts DESC
        LIMIT 1
    `, addr, marketID, ts).Scan(
        &s.BorrowerAddress, &s.MarketID, &s.BorrowShares,
        &s.TotalBorrowShares, &s.TotalBorrowAssets, &s.CollateralAssets,
        &s.OraclePrice, &s.HealthFactor, &s.SnapshotTS,
    )
    if err != nil {
        return nil, err
    }
    return &s, nil
}*/
