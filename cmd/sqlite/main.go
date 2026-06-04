package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"

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

	chainid := os.Args[1]
	chainint, err := strconv.ParseInt(chainid, 10, 64)
	dbname := fmt.Sprintf("./data/%d.db", chainint)
	fmt.Println(dbname)
	db, err := sql.Open("sqlite", dbname)
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

	fmt.Printf("Found table with %d entries \n", len(positions))
	for _, p := range positions {
		fmt.Printf("%+v\n", p)
	}
}
