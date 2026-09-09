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

func burnDB(db *sql.DB) error {
	tables, err := listTables(db)
	if err != nil {
		return err
	}
	for _, t := range tables {
		if _, err := db.Exec(`DROP TABLE IF EXISTS "` + t + `"`); err != nil {
			return fmt.Errorf("drop %s: %w", t, err)
		}
	}
	return nil
}

func listTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan table: %w", err)
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}
