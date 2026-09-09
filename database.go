package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "modernc.org/sqlite"
)

func openDB(path string) (*sql.DB, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("database %q does not exist. create it? [y/N] ", path)
		var answer string
		fmt.Scanln(&answer)
		if answer != "y" && answer != "Y" {
			return nil, fmt.Errorf("aborted")
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	return db, nil
}

func executeQueryWithRows(db *sql.DB, query string) (columns []string, rows [][]string, err error) {
	rowsResult, err := db.Query(query)
	if err != nil {
		return nil, nil, fmt.Errorf("query: %w", err)
	}
	defer rowsResult.Close()

	columns, err = rowsResult.Columns()
	if err != nil {
		return nil, nil, fmt.Errorf("columns: %w", err)
	}

	for rowsResult.Next() {
		values := make([]string, len(columns))
		scanArgs := make([]interface{}, len(columns))
		for i := range scanArgs {
			scanArgs[i] = &values[i]
		}
		if err := rowsResult.Scan(scanArgs...); err != nil {
			return nil, nil, fmt.Errorf("scan: %w", err)
		}
		rows = append(rows, values)
	}

	return columns, rows, rowsResult.Err()
}
