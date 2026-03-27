package sqlitecsv

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type ImportOptions struct {
	DBPath         string
	CSVPath        string
	DataSourceUUID string
	DataSourceName string
	TableName      string
	Delimiter      rune
	BatchSize      int
}

type ImportResult struct {
	RowsInserted int
}

// ImportRawCSV reads a raw CSV file and inserts each row as JsonData into SQLite.
func ImportRawCSV(ctx context.Context, opts ImportOptions) (ImportResult, error) {
	if strings.TrimSpace(opts.DBPath) == "" {
		return ImportResult{}, errors.New("db path is required")
	}
	if strings.TrimSpace(opts.CSVPath) == "" {
		return ImportResult{}, errors.New("csv path is required")
	}
	if strings.TrimSpace(opts.DataSourceUUID) == "" {
		return ImportResult{}, errors.New("data source uuid is required")
	}
	if strings.TrimSpace(opts.DataSourceName) == "" {
		return ImportResult{}, errors.New("data source name is required")
	}
	if strings.TrimSpace(opts.TableName) == "" {
		opts.TableName = "main.data_items"
	}
	if opts.Delimiter == 0 {
		opts.Delimiter = ';'
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 500
	}

	f, err := os.Open(opts.CSVPath)
	if err != nil {
		return ImportResult{}, fmt.Errorf("open csv: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.Comma = opts.Delimiter
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return ImportResult{}, fmt.Errorf("read csv: %w", err)
	}
	if len(records) < 1 {
		return ImportResult{}, errors.New("csv has no header row")
	}

	headers := normalizeHeaders(records[0])
	if len(headers) == 0 {
		return ImportResult{}, errors.New("csv has empty header row")
	}

	db, err := sql.Open("sqlite", opts.DBPath)
	if err != nil {
		return ImportResult{}, fmt.Errorf("open db: %w", err)
	}
	defer db.Close()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return ImportResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	query := fmt.Sprintf(
		"INSERT INTO %s (DataSourceUuid, DataSourceName, DataUuid, DataUpdateTimeStamp, JsonDataUuid, JsonData) VALUES (?, ?, ?, ?, ?, ?)",
		opts.TableName,
	)
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return ImportResult{}, fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	inserted := 0
	for i, raw := range records[1:] {
		row := normalizeRecord(raw, len(headers))
		payload := buildPayload(headers, row)
		jsonBytes, err := json.Marshal(payload)
		if err != nil {
			return ImportResult{}, fmt.Errorf("row %d marshal json: %w", i+2, err)
		}

		dataUUID, err := newUUID()
		if err != nil {
			return ImportResult{}, fmt.Errorf("row %d generate data uuid: %w", i+2, err)
		}
		jsonDataUUID, err := newUUID()
		if err != nil {
			return ImportResult{}, fmt.Errorf("row %d generate json data uuid: %w", i+2, err)
		}

		timestamp := time.Now().UTC().Format(time.RFC3339Nano)
		// Store original row payload as JSON together with datasource metadata.
		if _, err := stmt.ExecContext(
			ctx,
			opts.DataSourceUUID,
			opts.DataSourceName,
			dataUUID,
			timestamp,
			jsonDataUUID,
			string(jsonBytes),
		); err != nil {
			return ImportResult{}, fmt.Errorf("insert row %d: %w", i+2, err)
		}

		inserted++
	}

	if err := tx.Commit(); err != nil {
		return ImportResult{}, fmt.Errorf("final commit: %w", err)
	}

	return ImportResult{RowsInserted: inserted}, nil
}

// normalizeHeaders trims whitespace and removes UTF-8 BOM in the first header cell.
func normalizeHeaders(raw []string) []string {
	out := make([]string, 0, len(raw))
	for i, h := range raw {
		if i == 0 {
			h = strings.TrimPrefix(h, "\ufeff")
		}
		h = strings.TrimSpace(h)
		out = append(out, h)
	}
	return out
}

// normalizeRecord pads short rows so each row has one value per header.
func normalizeRecord(row []string, want int) []string {
	if len(row) == want {
		return row
	}
	out := make([]string, want)
	copy(out, row)
	return out
}

// buildPayload maps headers to row values and trims each value.
func buildPayload(headers, row []string) map[string]string {
	payload := make(map[string]string, len(headers))
	for i := range headers {
		payload[headers[i]] = strings.TrimSpace(row[i])
	}
	return payload
}

// newUUID generates a random RFC 4122 version 4 UUID string.
func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}

	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	var out [36]byte
	hex.Encode(out[0:8], b[0:4])
	out[8] = '-'
	hex.Encode(out[9:13], b[4:6])
	out[13] = '-'
	hex.Encode(out[14:18], b[6:8])
	out[18] = '-'
	hex.Encode(out[19:23], b[8:10])
	out[23] = '-'
	hex.Encode(out[24:36], b[10:16])

	return string(out[:]), nil
}
