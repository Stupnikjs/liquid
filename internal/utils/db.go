package utils


import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func openDb(filename string) (sqlitedb,error) {
	db, err := sql.Open("sqlite", filename)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	return db,err 

}
