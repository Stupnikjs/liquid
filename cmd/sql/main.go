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

// for every liquidation we try to find matching borrower
func GetLiquidations(chainint int64) []api.LiquidationItem {
	liquidations, err := api.FetchAllLiquidations(context.Background(), uint32(chainint))
	if err != nil {
		log.Fatal(err)
	}

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

func BorrowersFromLiquidations(liqItems []api.LiquidationItem) []string {
	borrowers := []string{}
	for _, item := range liqItems {
		borrowers = append(borrowers, item.Data.Position.User.Address)
	}
	return borrowers
}

func main() {
	chainid := os.Args[1]
	chainint, err := strconv.ParseInt(chainid, 10, 64)
	if err != nil {
		log.Fatal(err)
	}

	dbname := fmt.Sprintf("./data/%d.db", chainint)
	fmt.Println(dbname)

	db, err := sql.Open("sqlite", dbname)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	liquidations, err := CheckLiquidations(db, chainint)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("found %d matching liquidation \n", len(liquidations))

	if len(liquidations) > 0 {
		for _, l := range liquidations {
			fmt.Println(l)
		}
	}
}

func CheckLiquidations(db *sql.DB, chainint int64) ([]PositionSnapshot, error) {

	liquidations := GetLiquidations(chainint)
	borrowers := BorrowersFromLiquidations(liquidations)

	if len(borrowers) == 0 {
		fmt.Println("No borrowers found")
		return []PositionSnapshot{}, nil
	}

	// Build IN clause
	placeholders := make([]string, len(borrowers))
	args := make([]any, len(borrowers))
	for i, b := range borrowers {
		placeholders[i] = "?"
		args[i] = b
	}

	query := fmt.Sprintf(
		"SELECT * FROM position_snapshots WHERE borrower_address IN (%s)",
		strings.Join(placeholders, ","),
	)

	rows, err := db.Query(query, args...)
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
	return positions, err
}
