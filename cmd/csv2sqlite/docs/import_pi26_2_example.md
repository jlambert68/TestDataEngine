# Import Example: PI26_2 CSV Into SQLite

This example imports:

`/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/testdata/pi26_2/FenixRawTestdata_646rows_211220_stripped.csv`

into the `main.data_items` table in:

`/home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/testdata/SQLiteDB/identifier.sqlite`

## Run from project root

```bash
go run ./cmd/csv2sqlite \
  -db /home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/testdata/SQLiteDB/identifier.sqlite \
  -csv /home/jlambert/egen_kod/go/go_workspace/src/jlambert/TestDataEngine/testdata/pi26_2/FenixRawTestdata_646rows_211220_stripped.csv \
  -datasource-uuid 110cc994-a913-4041-96fe-a96d7e0c97e8 \
  -datasource-name SubCustody \
  -table main.data_items \
  -delimiter ';'
```

Expected output:

```text
Import completed. RowsInserted=<number>
```
