package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

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

/*

This cmd is for intering over liquidation on a chainid that occurs
=> call graphql api
=> gets []Liquidation
=> compare matching borrow id and market id function SELECT Entry WHERE borrow ==
=> prepare fine output if match
*/

func main() {

	chainid := os.Args[1]

	chainint, err := strconv.ParseInt(chainid, 10, 64)
	liquidations, err := api.FetchAllLiquidations(context.Background(), uint32(chainint))
	fmt.Println(liquidations, err)
	borrowers := []string{}
	for _, liquidation := range liquidations {
		borrowers = append(borrowers, liquidation.Data.Liquidator)
	}

	dbname := fmt.Sprintf("./data/%d.db", chainint)
	fmt.Println(dbname)
	db, err := sql.Open("sqlite", dbname)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Construire les placeholders : (?, ?, ?)
	placeholders := make([]string, len(borrowers))
	for i := range borrowers {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf(
		"SELECT * FROM position_snapshots WHERE borrower_address IN (%s)",
		strings.Join(placeholders, ", "),
	)

	// Convertir []string en []any pour db.Query
	args := make([]any, len(borrowers))
	for i, b := range borrowers {
		args[i] = b
	}

	rows, err := db.Query(query, args...)

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
