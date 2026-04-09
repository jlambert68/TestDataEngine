# 1. Runtime Requirements

## 1.1 Goal

The .NET solution must reproduce the current project as a test-data query engine with these capabilities:

- Accept a filter request with schema version `1.0`
- Compile the filter into a SQL-like expression
- Return allowed-field metadata
- Read rows from either CSV or SQLite
- Filter rows in memory using the request filter
- Randomize result order, optionally with a deterministic GUID seed
- Limit the number of returned rows
- Enrich dataset responses with JSON schema metadata
- Validate request and response payloads against JSON schema files
- Log every runtime event with a hard-coded UUID search key
- Import raw CSV rows into SQLite as JSON payload rows

## 1.2 Main Flows

### Flow A: Query CSV data

1. Read a `FilterRequest`.
2. Validate the request JSON against the request JSON schema file.
3. Validate the request envelope.
4. Load the CSV file using `;` as delimiter.
5. Normalize headers and rows.
6. Infer field types and supported operators from CSV content.
7. Compile the request filter for the inferred datasource.
8. Build the allowed-fields response.
9. Evaluate the request filter against each row in memory.
10. Shuffle matching rows.
11. Apply `maxItems` limit after shuffle.
12. Enrich the dataset response with local JSON schema metadata from file.
13. Validate the final dataset response against the response schema file.
14. Log compiled SQL, args, allowed-fields response, and dataset response.

### Flow B: Query SQLite data

1. Read a `FilterRequest`.
2. Validate the request JSON against the request JSON schema file.
3. Validate the request envelope.
4. Open the SQLite database.
5. Read dataset-response schema metadata from `main.testdataset_response_schemas`.
6. Read `JsonData` rows from the configured data table filtered by `DataSourceUuid` and `DataSourceName`.
7. Deserialize each `JsonData` value to an object.
8. Infer field types and supported operators from the discovered JSON fields and values.
9. Compile the request filter for the inferred datasource.
10. Build the allowed-fields response.
11. Evaluate the request filter against each row in memory.
12. Shuffle matching rows.
13. Apply `maxItems` limit after shuffle.
14. Attach schema metadata from the metadata table.
15. Validate the final dataset response against the schema named by `JsonSchemaName`.
16. Log compiled SQL, args, allowed-fields response, and dataset response.

### Flow C: Import raw CSV into SQLite

1. Validate import options.
2. Open the CSV file.
3. Normalize headers and rows.
4. Convert each row into a JSON object with string values.
5. Generate `DataUuid` and `JsonDataUuid`.
6. Insert rows into SQLite table `main.data_items` or the provided table name.
7. Commit the transaction.

## 1.3 Request Envelope Rules

These rules apply to the runtime `filters` path:

- `SchemaVersion` must equal `1.0`
- `RequestUuid` must match the runtime UUID regex
- `DataSourceUuid` must match the runtime UUID regex
- `DataSourceName` must be non-empty
- `RequestFilter` must be present and non-empty

Runtime UUID rule:

- The runtime `filters` package accepts any hex UUID in `8-4-4-4-12` form
- It does not enforce UUID version/variant bits

Typed compiler UUID rule:

- The `filtersql` package uses a stricter UUID regex
- It requires valid RFC-style version and variant bits
- The .NET rewrite must preserve this difference because the tests depend on it

## 1.4 Supported Filter Operators

The engine must support:

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

Operator behavior:

- `exists: true` means field value is not null
- `exists: false` means field value is null
- `isNull: true` means field value is null
- `isNull: false` means field value is not null
- `in` and `nin` require a non-empty array
- `contains`, `startsWith`, and `endsWith` require string values
- `gt`, `gte`, `lt`, and `lte` only work on comparable field types

## 1.5 Expression Rules

`RequestFilter` is recursively nested.

Valid node shapes:

- comparison
- `and`
- `or`
- `not`

Rules:

- A comparison node requires `field` and `op`
- `and` must contain at least one nested expression
- `or` must contain at least one nested expression
- `not` must contain exactly one nested expression
- The runtime parser decides node type by inspecting object shape

Example:

```json
{
  "and": [
    { "field": "AccountCurrency", "op": "eq", "value": "SEK" },
    {
      "not": {
        "field": "AccountEnvironment",
        "op": "eq",
        "value": "Prod"
      }
    }
  ]
}
```

## 1.6 SQL Compilation Rules

The runtime `filters` compiler must:

- Quote field names as `"FieldName"`
- Reject unsafe field names
- Use `?` placeholders
- Return SQL and ordered args

Examples:

```text
{"field":"status","op":"eq","value":"active"}
=> ("status" = ?)
args => ["active"]
```

```text
{"field":"age","op":"in","value":[18,21]}
=> ("age" IN (?, ?))
args => [18, 21]
```

```text
{"field":"deletedAt","op":"isNull","value":true}
=> ("deletedAt" IS NULL)
args => []
```

The typed `filtersql` compiler must support two placeholder modes:

- `?`
- `$1`, `$2`, `$3`

The typed `filtersql` compiler must also support:

- quoted identifiers
- unquoted identifiers

## 1.7 CSV Rules

The CSV datasource implementation must:

- Use `;` as delimiter
- Allow variable field counts per row
- Trim whitespace from non-BOM headers
- Remove UTF-8 BOM from the first header cell after the first trim step
- Pad short rows with empty strings
- Treat empty string and case-insensitive `NULL` as null

Important current-behavior detail:

- In the runtime CSV query path, the first header is processed as `TrimSpace` first and `TrimPrefix(BOM)` second
- Because of that order, a first header like `"\ufeff A "` becomes `" A"`, not `"A"`
- In the CSV-to-SQLite importer path, the order is reversed enough that the first header ends up fully trimmed
- The .NET rewrite should preserve this difference if the goal is exact behavioral parity with the current code and tests

Field type inference order:

1. `boolean`
2. `integer`
3. `number`
4. `string`

Rules:

- If all non-null values parse as booleans, the field type is `boolean`
- Else if all non-null values parse as integers, the field type is `integer`
- Else if all non-null values parse as numbers, the field type is `number`
- Else the field type is `string`
- If all values are null, the field type is `string`
- Dates and datetimes are not inferred from raw CSV content in the current implementation

Supported operators per inferred field type:

- `number`, `integer`, `date`, `datetime`: `eq`, `neq`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`, `exists`, `isNull`
- `boolean`: `eq`, `neq`, `exists`, `isNull`
- `string`: `eq`, `neq`, `in`, `nin`, `contains`, `startsWith`, `endsWith`, `exists`, `isNull`

## 1.8 SQLite Rules

The SQLite datasource implementation must:

- Require a non-empty DB path
- Default blank table name to `main.data_items`
- Reject unsafe table names
- Query only rows where both `DataSourceUuid` and `DataSourceName` match the request
- Deserialize `JsonData` from each matching row
- Infer fields across all JSON keys found in all matching rows
- Sort field names deterministically
- Coerce raw JSON values to inferred field types

Important table-name rule:

- Safe table names may contain letters, digits, `_`, and `.`
- Semicolons and other punctuation must be rejected

## 1.9 Result Randomization and Limiting

These rules apply to both CSV and SQLite flows:

- If fewer than two rows match, no shuffle work is required
- If no random seed GUID is supplied, use non-deterministic randomness
- If a random seed GUID is supplied, validate it and derive a deterministic seed from it
- Shuffle before applying `maxItems`
- `maxItems == 0` means unlimited
- `maxItems < 0` must behave as unlimited

## 1.10 Response Metadata Rules

`DataSetResponse` contains two metadata layers:

- Request datasource identity:
  - `DataSourceName`
  - `DataSourceUuid`
- Response-schema metadata:
  - `TestDataSourceName`
  - `TestDataSourceUuid`
  - `JsonSchemaName`
  - `JsonSchema`
  - `UpdatedDateTime`

CSV behavior:

- The main program enriches CSV responses from local JSON schema files
- `JsonSchemaName` is set to the basename of the configured schema file
- `UpdatedDateTime` is set to current UTC time in RFC3339 format

SQLite behavior:

- Schema metadata is loaded from `main.testdataset_response_schemas`
- The newest row by `UpdatedDateTime DESC` is used
- If the metadata table does not exist, low-level query code stays backward compatible and returns no metadata
- The main program still requires metadata because response-schema validation needs `JsonSchemaName`
- If the metadata row exists but `JsonSchema` is not valid JSON, the SQLite query must fail

## 1.11 Schema Validation Rules

The main program must validate:

- request JSON against `internal/json/TestDataSet_Request_Filter_To_TestDataEngine.json-schema.json`
- final response JSON against the specific response schema

Rules:

- Resolve schema path from local working directory and source-relative fallback paths
- For response validation, only the basename of `JsonSchemaName` is trusted
- Missing schema files must fail
- Invalid payload JSON must fail
- Path-traversal-style schema names must resolve by basename only

## 1.12 Logging Rules

Every log message must include a caller-supplied UUID marker:

```text
Id=<uuid> message
```

Rules:

- `Infof` logs normal messages
- `Errorf` prefixes message text with `ERROR: `
- `Fatalf` prefixes message text with `FATAL: ` and terminates the process
- UUIDs are hard-coded at call sites

## 1.13 Command-Line Requirements

The `.NET` executable equivalent of `cmd/testdataengine/main.go` must support:

- `-source` with values `csv` or `sqlite`
- `-csv`
- `-sqlite-db`
- `-sqlite-table`
- `-max-items`
- `-random-seed-guid`

Default behavior:

- Default source is `csv`
- Default CSV path is `p26_2/FenixRawTestdata_646rows_211220_stripped.csv`
- If that default path does not exist, try `P26_2/FenixRawTestdata_646rows_211220_stripped.csv`
- Default SQLite DB path is `testdata/SQLiteDB/identifier.sqlite`
- Default SQLite table is `main.data_items`
- Default max items is `2`

## 1.14 Runtime Output Shape

The main program logs two response objects, each wrapped with the original input filter:

The main program also logs:

- `WHERE=<compiled where sql>`
- `ARGS=<compiled args>`

Allowed-fields output shape:

```json
{
  "InputFilter": { "...": "..." },
  "AllowedFieldsResponse": { "...": "..." }
}
```

Dataset output shape:

```json
{
  "InputFilter": { "...": "..." },
  "DataSetResponse": { "...": "..." }
}
```

The log message payload keys are:

- `AllowedFieldsResponse=...`
- `DataSetResponse=...`
