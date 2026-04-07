package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"TestDataEngine/internal/sqlitecsv"
)

// main imports a raw CSV file into the SQLite data_items table.
func main() {
	var (
		dbPath         string
		csvPath        string
		dataSourceUUID string
		dataSourceName string
		tableName      string
		delimiter      string
	)

	flag.StringVar(&dbPath, "db", "testdata/SQLiteDB/identifier.sqlite", "Path to SQLite database file")
	flag.StringVar(&csvPath, "csv", "", "Path to raw CSV file to import")
	flag.StringVar(&dataSourceUUID, "datasource-uuid", "", "Datasource UUID to store in data_items")
	flag.StringVar(&dataSourceName, "datasource-name", "", "Datasource name to store in data_items")
	flag.StringVar(&tableName, "table", "main.data_items", "Target table name")
	flag.StringVar(&delimiter, "delimiter", ";", "CSV delimiter character")
	flag.Parse()

	// Validate required CLI arguments before opening files and DB connections.
	if csvPath == "" {
		log.Fatal("-csv is required")
	}
	if dataSourceUUID == "" {
		log.Fatal("-datasource-uuid is required")
	}
	if dataSourceName == "" {
		log.Fatal("-datasource-name is required")
	}
	if len(delimiter) != 1 {
		log.Fatal("-delimiter must be a single character")
	}

	// Run the import in one call so the command stays a thin wrapper around the library.
	result, err := sqlitecsv.ImportRawCSV(context.Background(), sqlitecsv.ImportOptions{
		DBPath:         dbPath,
		CSVPath:        csvPath,
		DataSourceUUID: dataSourceUUID,
		DataSourceName: dataSourceName,
		TableName:      tableName,
		Delimiter:      rune(delimiter[0]),
	})
	if err != nil {
		log.Fatalf("import failed: %v", err)
	}

	// Keep output script-friendly and easy to grep.
	fmt.Printf("Import completed. RowsInserted=%d\n", result.RowsInserted)
}
