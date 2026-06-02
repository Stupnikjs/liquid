package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

type PositionSnapshot struct {
	ID                int
	BorrowerAddress   string
	MarketID          string
	BorrowShares      string
	TotalBorrowShares string
	TotalBorrowAssets string
	CollateralAssets  string
	OraclePrice       string
	HealthFactor      sql.NullString
	SnapshotTS        int64
}

func main() {

	db, err := sql.Open("sqlite", "./data/999.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM position_snapshots")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var positions []PositionSnapshot

	for rows.Next() {
		var p PositionSnapshot
		err := rows.Scan(
			&p.ID,
			&p.BorrowerAddress,
			&p.MarketID,
			&p.BorrowShares,
			&p.TotalBorrowShares,
			&p.TotalBorrowAssets,
			&p.CollateralAssets,
			&p.OraclePrice,
			&p.HealthFactor,
			&p.SnapshotTS,
		)
		if err != nil {
			log.Fatal(err)
		}
		positions = append(positions, p)
	}

	for _, p := range positions {

		fmt.Printf("%+v\n", p)
	}
}
