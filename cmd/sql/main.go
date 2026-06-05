package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/Stupnikjs/liquid/pkg/api"
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

func GetLiquidations(chainint int64) []api.LiquidationItem {
	liquidations, err := api.FetchAllLiquidations(context.Background(), uint32(chainint))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(liquidations)

	borrowers := []string{}
	for _, liquidation := range liquidations {
		borrowers = append(borrowers, liquidation.Data.Liquidator)
	}

	if len(borrowers) == 0 {
		fmt.Println("No borrowers found")
		return []api.LiquidationItem{}
	}
	return liquidations
}

func main() {
	chainid := os.Args[1]

	chainint, err := strconv.ParseInt(chainid, 10, 64)
	if err != nil {
		log.Fatal(err)
	}

	dbname := fmt.Sprintf("./newdata/%d.db", chainint)
	fmt.Println(dbname)

	db, err := sql.Open("sqlite", dbname)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	query := "SELECT * FROM position_snapshots WHERE borrower_address = ?"
	rows, err := db.Query(query, "0x7F9A2903E4fb8f8E20aca2941CCd1857c60Fc013")
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
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d entries\n", len(positions))
	for _, p := range positions {
		fmt.Printf("%+v\n", p)
	}
}
