# TestDataEngine Implemented Requirements

This document describes the behavior that is implemented in code and enforced by tests in this repository as of April 7, 2026.

It is written from the implementation outward. If the code and this document disagree, the code should be treated as the current truth until the document is updated.

## 1. What This System Is

This repository contains three related but different contract layers:

1. `cmd/testdataengine`
   The runnable CLI entry point.
2. `internal/filters`
   The active runtime engine used by `cmd/testdataengine`.
3. `internal/filtersql`
   A typed request parser and SQL compiler library.

The only authoritative JSON schema files are the root-level files directly under `internal/json`.

Files in `internal/json/old`, `P26_2`, `testdata/pi26_2`, and other documentation folders are not authoritative schema sources and must not be treated as the active contract.

## 2. Quick Start

The current product is a CLI, not an HTTP service.

Run the main program with CSV:

```bash
make run-main-csv
```

Run the main program with SQLite:

```bash
make run-main-sqlite
```

Run the main program with SQLite and a deterministic random seed:

```bash
make run-main-sqlite MAX_ITEMS=5 RANDOM_SEED_GUID=bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb
```

Run the main program with Postgres:

```bash
go run ./cmd/testdataengine \
  -source postgres \
  -postgres-dsn 'postgres://user:pass@localhost:5432/dbname?sslmode=disable' \
  -postgres-table public.data_items \
  -postgres-schema-table public.testdataset_response_schemas
```

The request itself is currently embedded in `cmd/testdataengine/main.go`.

## 3. Main Concepts

The runtime engine produces three outputs for each request:

1. `CompiledFilter`
   A SQL-like trace of the filter.
2. `AllowedFieldResponse`
   The inferred fields and their supported operators.
3. `DataSetResponse`
   The matching data rows.

Important: the runtime engine does not push the full filter into SQL for execution. It evaluates the filter in Go row by row.

## 4. Request Contract

## 4.1 Top-level request shape

The runtime request shape is:

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
  "DataSourceUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceName": "SubCustody",
  "RequestFilter": {
    "field": "AccountCurrency",
    "op": "eq",
    "value": "SEK"
  }
}
```

Runtime field meanings:

- `SchemaVersion`
  Must be `"1.0"`.
- `RequestUuid`
  Request identifier.
- `DataSourceUuid`
  Datasource identifier.
- `DataSourceName`
  Datasource name.
- `RequestFilter`
  A comparison or logical expression tree.

## 4.2 How requests are supplied today

Today the request is not loaded from file, stdin, or HTTP.

The request is hard-coded in `cmd/testdataengine/main.go`, and the caller controls only the source-related runtime parameters through CLI flags:

- `-source`
- `-csv`
- `-sqlite-db`
- `-sqlite-table`
- `-postgres-dsn`
- `-postgres-table`
- `-postgres-schema-table`
- `-max-items`
- `-random-seed-guid`

## 5. Filter Expression Contract

The root of `RequestFilter` must be exactly one of:

- a comparison
- an `and` expression
- an `or` expression
- a `not` expression

Each logical expression may recursively contain nested expressions of those same four kinds.

## 5.1 Comparison expression

Example:

```json
{
  "field": "AccountCurrency",
  "op": "eq",
  "value": "SEK"
}
```

Structure:

- `field`
  The field to test.
- `op`
  The operator.
- `value`
  The value for the operator.

## 5.2 Logical expressions

`and` example:

```json
{
  "and": [
    { "field": "AccountCurrency", "op": "eq", "value": "SEK" },
    { "field": "AccountEnvironment", "op": "eq", "value": "SysTest" }
  ]
}
```

`or` example:

```json
{
  "or": [
    { "field": "AccountCurrency", "op": "eq", "value": "SEK" },
    { "field": "AccountCurrency", "op": "eq", "value": "NOK" }
  ]
}
```

`not` example:

```json
{
  "not": { "field": "AccountCurrency", "op": "eq", "value": "NOK" }
}
```

Nested example:

```json
{
  "and": [
    { "field": "AccountEnvironment", "op": "eq", "value": "SysTest" },
    {
      "or": [
        { "field": "AccountCurrency", "op": "eq", "value": "SEK" },
        { "field": "AccountCurrency", "op": "eq", "value": "NOK" }
      ]
    },
    {
      "not": { "field": "ProvisionalIncome", "op": "eq", "value": "Y" }
    }
  ]
}
```

So this is valid:

- root node: `and`
- child 1: comparison
- child 2: `or`
- child 3: `not`

And this is also valid:

```json
{
  "not": {
    "or": [
      { "field": "AccountCurrency", "op": "eq", "value": "NOK" },
      { "field": "AccountCurrency", "op": "eq", "value": "DKK" }
    ]
  }
}
```

## 6. Supported Operators

The active runtime engine supports:

- `eq`
- `neq`
- `gt`
- `gte`
- `lt`
- `lte`
- `in`
- `nin`
- `contains`
- `startsWith`
- `endsWith`
- `exists`
- `isNull`

## 6.1 Operator examples

Equality:

```json
{ "field": "AccountCurrency", "op": "eq", "value": "SEK" }
```

Inequality:

```json
{ "field": "AccountCurrency", "op": "neq", "value": "NOK" }
```

Greater than:

```json
{ "field": "Amount", "op": "gt", "value": 100 }
```

In list:

```json
{ "field": "AccountCurrency", "op": "in", "value": ["SEK", "NOK"] }
```

Contains:

```json
{ "field": "MarketName", "op": "contains", "value": "Stock" }
```

Exists:

```json
{ "field": "ISIN", "op": "exists", "value": true }
```

Is null:

```json
{ "field": "ISIN", "op": "isNull", "value": false }
```

## 6.2 Operator rules

- `eq`, `neq`
  Require a scalar value of the field type.
- `gt`, `gte`, `lt`, `lte`
  Allowed only for comparable field types.
- `in`, `nin`
  Require a non-empty array.
- `contains`, `startsWith`, `endsWith`
  Require a string value.
- `exists`
  Requires a boolean value.
- `isNull`
  Requires a boolean value.

Meaning of boolean operators:

- `exists: true`
  Field must be non-null.
- `exists: false`
  Field must be null.
- `isNull: true`
  Field must be null.
- `isNull: false`
  Field must be non-null.

Equivalent examples:

```json
{ "field": "ISIN", "op": "exists", "value": true }
```

and

```json
{ "field": "ISIN", "op": "isNull", "value": false }
```

express the same runtime condition.

## 7. Request Validation Rules

## 7.1 Envelope rules

The runtime engine validates:

- `SchemaVersion` must be `1.0`
- `RequestUuid` must match the repository UUID shape
- `DataSourceUuid` must match the repository UUID shape
- `DataSourceName` must be non-empty
- `RequestFilter` must be present and non-empty

Invalid example: bad schema version

```json
{
  "SchemaVersion": "2.0",
  "RequestUuid": "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
  "DataSourceUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceName": "SubCustody",
  "RequestFilter": { "field": "AccountCurrency", "op": "eq", "value": "SEK" }
}
```

Result: rejected.

## 7.2 Logical-shape rules

- `and` must have at least one child
- `or` must have at least one child
- `not` must contain exactly one child expression

Invalid example:

```json
{ "and": [] }
```

Result: rejected.

## 7.3 Field and operator rules

At execution time:

- the field name must be safe
- the field must exist in the inferred datasource
- the operator must be supported for that field type
- the value must satisfy the operator rules

Unsafe-field example:

```json
{ "field": "AccountCurrency;drop", "op": "eq", "value": "SEK" }
```

Result: rejected.

Unsupported-operator example:

```json
{ "field": "Amount", "op": "contains", "value": "10" }
```

Result: rejected when `Amount` is inferred as numeric.

## 8. Runtime Execution Model

The runtime engine in `internal/filters` works like this:

1. validate request envelope
2. load source rows from CSV or SQLite
3. infer the datasource fields from the loaded data
4. compile the filter to a SQL-like trace
5. evaluate the filter in Go against each row
6. shuffle matching rows
7. apply `max-items`
8. build responses

Important: filtering is not executed by SQL. The SQL-like output is trace information.

## 9. Source-Specific Behavior

## 9.1 CSV source

CSV behavior:

- delimiter is `;`
- all rows are read into memory
- headers are normalized
- column types are inferred from all rows
- rows are converted into typed maps

CSV example input:

```text
AccountCurrency;Amount;Flag
SEK;100;true
NOK;200;false
```

Possible inferred fields:

- `AccountCurrency` -> `string`
- `Amount` -> `integer`
- `Flag` -> `boolean`

## 9.2 SQLite source

SQLite behavior:

- validates the DB path
- defaults empty table name to `main.data_items`
- validates table name safety
- reads rows using:
  - `DataSourceUuid`
  - `DataSourceName`
- reads `JsonData`
- unmarshals `JsonData`
- infers fields from all JSON payloads

Expected SQLite table columns for runtime use:

- `DataSourceUuid`
- `DataSourceName`
- `JsonData`

Canonical table used by tests and importer:

- `DataSourceUuid TEXT NOT NULL`
- `DataSourceName TEXT NOT NULL`
- `TestDataDomainUuid TEXT NOT NULL`
- `TestDataDomainName TEXT NOT NULL`
- `TestDataSourceTemplateName TEXT NOT NULL`
- `DataUuid TEXT NOT NULL`
- `DataUpdateTimeStamp TEXT NOT NULL`
- `JsonDataUuid TEXT NOT NULL PRIMARY KEY`
- `JsonData TEXT NOT NULL`

## 9.3 Postgres source

Postgres behavior:

- validates the Postgres DSN
- defaults empty data table name to `public.data_items`
- defaults empty schema metadata table name to `public.testdataset_response_schemas`
- validates table name safety
- reads rows using:
  - `DataSourceUuid`
  - `DataSourceName`
- reads `JsonData`
- unmarshals `JsonData`
- infers fields from all JSON payloads

Expected Postgres table columns for runtime use:

- `DataSourceUuid`
- `DataSourceName`
- `JsonData`

Canonical Postgres table should mirror the SQLite `data_items` metadata columns:

- `DataSourceUuid UUID NOT NULL`
- `DataSourceName VARCHAR NOT NULL`
- `TestDataDomainUuid UUID NOT NULL`
- `TestDataDomainName VARCHAR NOT NULL`
- `TestDataSourceTemplateName VARCHAR NOT NULL`
- `DataUuid UUID NOT NULL`
- `DataUpdateTimeStamp TIMESTAMPTZ NOT NULL`
- `JsonDataUuid UUID NOT NULL PRIMARY KEY`
- `JsonData JSONB NOT NULL`

Canonical schema fixture:

- `testdata/PostgresDB/schema.sql`

Example `JsonData` payload stored in SQLite:

```json
{
  "AccountCurrency": "SEK",
  "Amount": "100",
  "Flag": "true"
}
```

## 10. Data Typing Rules

## 10.1 Null rules

CSV treats a value as null when:

- it is empty after trimming
- it equals `NULL`, case-insensitive

Examples:

- `""` -> null
- `"   "` -> null
- `"NULL"` -> null
- `"null"` -> null
- `"SEK"` -> not null

## 10.2 Type inference order

For each field, the runtime inference order is:

1. `boolean`
2. `integer`
3. `number`
4. `string`

Examples:

- `["true", "false", "NULL"]` -> `boolean`
- `["1", "2", ""]` -> `integer`
- `["1,5", "2", ""]` -> `number`
- `["abc", "2"]` -> `string`

If a field has only null values, it becomes `string`.

## 10.3 Number parsing rules

Rules:

- comma decimal separators are accepted
- integer values may appear as `1` or `1.0`
- non-integral values like `1.5` are rejected for integer fields

Examples:

- `"1,5"` as number -> `1.5`
- `"12"` as integer -> `12`
- `"1.0"` as integer -> `1`
- `"1.5"` as integer -> rejected

## 10.4 String rules

- strings are trimmed on parse
- runtime string comparison is exact for `eq` and `neq`

Example:

- raw CSV value `" SEK "` becomes `"SEK"`

## 10.5 Date and datetime note

The runtime engine has operator support definitions for `date` and `datetime`, but the current CSV and SQLite inference logic does not infer those types automatically.

In practice, runtime-inferred field types are usually:

- `string`
- `integer`
- `number`
- `boolean`

## 11. Evaluation Rules

Runtime evaluation is row-by-row and in memory.

Rules:

- string equality compares trimmed strings
- numeric comparisons convert supported numeric Go types to `float64`
- ordered string-like comparisons are lexicographic
- `contains`, `startsWith`, `endsWith` operate only on strings
- if a row value is `nil`, ordered comparisons return false
- `exists` uses `rowValue != nil`
- `isNull` uses `rowValue == nil`

Examples:

If the row is:

```json
{
  "AccountCurrency": "SEK",
  "Amount": 100,
  "ISIN": null
}
```

Then:

```json
{ "field": "AccountCurrency", "op": "eq", "value": "SEK" }
```

matches.

```json
{ "field": "Amount", "op": "gt", "value": 50 }
```

matches.

```json
{ "field": "ISIN", "op": "exists", "value": true }
```

does not match.

```json
{ "field": "ISIN", "op": "isNull", "value": true }
```

matches.

## 12. Response Contract

The runtime engine returns three logical outputs.

## 12.1 CompiledFilter

Shape:

```json
{
  "WhereSQL": "(\"AccountCurrency\" = ?)",
  "Args": ["SEK"]
}
```

This is trace output. It is not the actual execution engine.

## 12.2 AllowedFieldResponse

Shape:

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
  "DataSourceUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "DataSourceName": "SubCustody",
  "AllowedFields": [
    {
      "FieldName": "AccountCurrency",
      "FieldType": "string",
      "Nullable": false,
      "SupportedOperators": [
        "eq",
        "neq",
        "in",
        "nin",
        "contains",
        "startsWith",
        "endsWith",
        "exists",
        "isNull"
      ],
      "Description": "Inferred from CSV column \"AccountCurrency\"."
    }
  ]
}
```

Rules:

- field list is sorted by field name
- operator list is emitted in a fixed order
- fields are inferred from the loaded source data

## 12.3 DataSetResponse

Shape:

```json
{
  "DataSourceName": "SubCustody",
  "DataSourceUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "Data": [
    {
      "AccountCurrency": "SEK",
      "AccountEnvironment": "SysTest"
    }
  ]
}
```

Rules:

- `Data` may contain zero rows
- row shape is dynamic
- row values may be string, integer, number, boolean, or null

## 12.4 What the CLI actually logs

`cmd/testdataengine` logs the responses wrapped with the original input filter.

Allowed-fields log example:

```json
{
  "InputFilter": {
    "field": "AccountCurrency",
    "op": "eq",
    "value": "SEK"
  },
  "AllowedFieldsResponse": {
    "SchemaVersion": "1.0",
    "RequestUuid": "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
    "DataSourceUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
    "DataSourceName": "SubCustody",
    "AllowedFields": []
  }
}
```

Data log example:

```json
{
  "InputFilter": {
    "field": "AccountCurrency",
    "op": "eq",
    "value": "SEK"
  },
  "DataSetResponse": {
    "DataSourceName": "SubCustody",
    "DataSourceUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
    "Data": []
  }
}
```

## 13. Ordering, Limiting, and Randomization

Rules:

- matching rows are shuffled before limiting
- if fewer than two rows match, shuffle has no effect
- `max-items > 0` truncates after shuffle
- `max-items == 0` means return all matches
- `max-items < 0` is treated as `0`

Randomization rules:

- empty `random-seed-guid` -> nondeterministic order
- valid `random-seed-guid` -> deterministic order
- invalid `random-seed-guid` -> request fails

Examples:

- `MAX_ITEMS=1` -> at most one row
- `MAX_ITEMS=0` -> all matching rows
- `MAX_ITEMS=-1` -> same as unbounded

## 14. Safety Rules

## 14.1 Safe request field names

Runtime request fields must be safe identifiers.

Allowed characters:

- first character: letter or underscore
- remaining characters: letter, digit, underscore

Examples:

- `AccountCurrency` -> allowed
- `_internalField` -> allowed
- `client-id` -> rejected
- `main.data_items` -> rejected as request field name

## 14.2 Safe SQLite table names

SQLite table names are allowed to contain:

- letters
- digits
- underscore
- dot

Examples:

- `main.data_items` -> allowed
- `data_items_2026` -> allowed
- `main.data_items;drop table x` -> rejected

## 15. Logging Rules

All runtime logs use `internal/logging`.

Rules:

- every log line is prefixed with `Id=<uuid>`
- `Infof` logs info
- `Errorf` logs with `ERROR:`
- `Fatalf` logs with `FATAL:` and exits

Typical CLI logging sequence:

1. source configuration
2. compiled `WHERE`
3. compiled `ARGS`
4. `AllowedFieldsResponse` with `InputFilter`
5. `DataSetResponse` with `InputFilter`

## 16. CSV-to-SQLite Import Flow

`cmd/csv2sqlite` imports raw CSV rows into SQLite.

Required arguments:

- `-csv`
- `-datasource-uuid`
- `-datasource-name`

Optional arguments:

- `-db`
- `-table`
- `-delimiter`

Defaults:

- DB path defaults to `testdata/SQLiteDB/identifier.sqlite`
- table defaults to `main.data_items`
- delimiter defaults to `;`

Example:

```bash
go run ./cmd/csv2sqlite \
  -db testdata/SQLiteDB/identifier.sqlite \
  -csv testdata/pi26_2/FenixRawTestdata_646rows_211220_stripped.csv \
  -datasource-uuid 110cc994-a913-4041-96fe-a96d7e0c97e8 \
  -datasource-name SubCustody
```

Expected output:

```text
Import completed. RowsInserted=<n>
```

## 17. Practical Runtime Examples

## 17.1 Example: CSV-backed run

Command:

```bash
make run-main-csv
```

Effect:

- loads the embedded request from `cmd/testdataengine/main.go`
- reads the CSV file configured in the `Makefile`
- logs compiled filter, allowed fields, and matching rows

## 17.2 Example: SQLite-backed run

Command:

```bash
make run-main-sqlite
```

Effect:

- loads the embedded request
- reads matching datasource rows from SQLite
- infers fields from `JsonData`
- logs compiled filter, allowed fields, and matching rows

## 17.3 Example: library usage

Programmatic CSV usage:

```go
req := filters.FilterRequest{
    SchemaVersion:  "1.0",
    RequestUUID:    "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
    DataSourceUUID: "110cc994-a913-4041-96fe-a96d7e0c97e8",
    DataSourceName: "SubCustody",
    RequestFilter:  []byte(`{"field":"AccountCurrency","op":"eq","value":"SEK"}`),
}

compiled, allowed, data, err := filters.QueryCSVDataSource(req, "file.csv", 10)
```

Programmatic SQLite usage:

```go
compiled, allowed, data, err := filters.QuerySQLiteDataSource(
    req,
    "testdata/SQLiteDB/identifier.sqlite",
    "main.data_items",
    10,
)
```

## 18. Known Contract Differences

These differences matter if this repository is used as an external contract.

## 18.1 Runtime `internal/filters` vs `internal/filtersql`

Both support:

- `eq`
- `neq`
- `gt`
- `gte`
- `lt`
- `lte`
- `in`
- `nin`
- `contains`
- `startsWith`
- `endsWith`
- `exists`
- `isNull`

Differences:

- runtime executes filtering in memory
- `filtersql` only parses, validates, and compiles
- runtime accepts datasource names outside the static catalog during CSV/SQLite execution
- `filtersql` has no datasource catalog
- runtime uses `json.RawMessage`
- `filtersql` uses typed expression nodes
- runtime UUID validation is looser

Exact example:

```json
{
  "SchemaVersion": "1.0",
  "RequestUuid": "11111111-1111-1111-1111-111111111111",
  "DataSourceUuid": "11111111-1111-1111-1111-111111111111",
  "DataSourceName": "SubCustody",
  "RequestFilter": {
    "field": "AccountCurrency",
    "op": "eq",
    "value": "SEK"
  }
}
```

Result:

- runtime may accept the UUID shape
- `filtersql` rejects it because its UUID validation is stricter

## 18.2 Runtime vs bundled JSON schemas

The request schema is now aligned on `isNull`, but the response schema is still narrower than runtime behavior.

Runtime response example:

```json
{
  "DataSourceName": "SubCustody",
  "DataSourceUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "Data": [
    { "AccountCurrency": "SEK" },
    { "AccountCurrency": "NOK" }
  ]
}
```

Why this matters:

- runtime can return multiple rows
- runtime rows are dynamic
- the CLI logs wrapper objects containing `InputFilter`
- the bundled response schema models a much narrower fixed structure

## 19. Test-Enforced Behavior Summary

The tests enforce all of the following:

- request envelope validation
- logical-expression shape validation
- operator validation
- SQL-like compilation behavior
- CSV inference and parsing behavior
- SQLite datasource loading behavior
- row-by-row evaluation correctness
- deterministic shuffling when seeded
- `max-items` semantics
- safe table-name validation
- importer validation and row insertion
- required log prefix format

## 20. Recommended Clarification

If this repository is intended to define a stable external contract, one contract layer should be declared the source of truth:

- runtime `internal/filters`
- typed `internal/filtersql`
- bundled JSON schemas

At the moment, these three layers are aligned in many areas, but they are still not fully identical.

Recommended position:

- treat the runtime behavior in `internal/filters` as the authoritative contract for current behavior
- align `internal/filtersql` to that behavior where typed validation and SQL compilation are intended to represent the same request model
- align the bundled JSON schemas to that same behavior so documentation and validation match the executable system

Reason:

- `internal/filters` is the layer that actually loads data, validates requests for execution, evaluates filters, and produces responses
- `internal/filtersql` is a supporting contract layer, not the runtime execution layer
- the JSON schema files are documentation and validation artifacts, not the executing implementation
