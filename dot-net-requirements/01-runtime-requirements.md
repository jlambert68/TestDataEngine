# 1. Runtime Requirements

## 1.1 Goal

The .NET solution must reproduce the current project as a test-data query engine with these capabilities:

- Accept a filter request with schema version `1.0`
- Compile the filter into a SQL-like expression
- Return allowed-field metadata
- Read rows from CSV, SQLite, or Postgres
- Filter rows in memory using the request filter
- Randomize result order, optionally with a deterministic GUID seed
- Limit the number of returned rows
- Enrich dataset responses with JSON schema metadata
- Validate request and response payloads against JSON schema files
- Log every runtime event with a hard-coded UUID search key
- Import raw CSV rows into SQLite as JSON payload rows
- Serve the same HTTP API contract consumed by `ui/`
- Serve `ui/dist` static assets with SPA fallback routing

## 1.2 Main Flows

### Flow A: Query CSV data

1. Read a `FilterRequest`.
2. Validate the request JSON against the request JSON schema file.
3. Validate the request envelope.
4. Load the CSV file using `;` as delimiter.
5. Normalize headers and rows.
6. Build field definitions from schema first (`internal/json/TestDataSet_Response_For_Specific_Datasource_From_TestDataEngine.json-schema.json`) and fall back to legacy inference when schema-driven loading is unavailable.
7. Canonicalize row field names to schema field names when aliases or misspellings are close matches.
8. Compile the request filter for the resolved datasource definition.
9. Build the allowed-fields response.
10. Evaluate the request filter against each row in memory.
11. Sort matching rows canonically before shuffle.
12. Shuffle matching rows.
13. Apply `maxItems` limit after shuffle.
14. Enrich the dataset response with local JSON schema metadata from file.
15. Validate the final dataset response against the response schema file.
16. Log compiled SQL, args, allowed-fields response, and dataset response.

### Flow B: Query SQLite data

1. Read a `FilterRequest`.
2. Validate the request JSON against the request JSON schema file.
3. Validate the request envelope.
4. Open the SQLite database.
5. Read dataset-response schema metadata from `main.testdataset_response_schemas`.
6. Read `JsonData` rows from the configured data table filtered by `DataSourceUuid` and `DataSourceName`.
7. Deserialize each `JsonData` value to an object.
8. Build field definitions from schema metadata JSON first (`JsonSchema` from metadata table) and fall back to local authoritative schema file, then legacy inference.
9. Canonicalize row field names to schema field names when aliases or misspellings are close matches.
10. Compile the request filter for the resolved datasource definition.
11. Build the allowed-fields response.
12. Evaluate the request filter against each row in memory.
13. Sort matching rows canonically before shuffle.
14. Shuffle matching rows.
15. Apply `maxItems` limit after shuffle.
16. Attach schema metadata from the metadata table.
17. Validate the final dataset response against the schema named by `JsonSchemaName`.
18. Log compiled SQL, args, allowed-fields response, and dataset response.

### Flow C: Query Postgres data

1. Read a `FilterRequest`.
2. Validate the request JSON against the request JSON schema file.
3. Validate the request envelope.
4. Open the Postgres database using the configured DSN.
5. Read dataset-response schema metadata from `public.testdataset_response_schemas` or the provided schema table.
6. Read `JsonData` rows from the configured data table filtered by `DataSourceUuid` and `DataSourceName`.
7. Deserialize each `JsonData` value to an object.
8. Build field definitions from schema metadata JSON first (`JsonSchema` from metadata table) and fall back to local authoritative schema file, then legacy inference.
9. Canonicalize row field names to schema field names when aliases or misspellings are close matches.
10. Compile the request filter for the resolved datasource definition.
11. Build the allowed-fields response.
12. Evaluate the request filter against each row in memory.
13. Sort matching rows canonically before shuffle.
14. Shuffle matching rows.
15. Apply `maxItems` limit after shuffle.
16. Attach schema metadata from the metadata table.
17. Validate the final dataset response against the schema named by `JsonSchemaName`.
18. Log compiled SQL, args, allowed-fields response, and dataset response.

### Flow D: Import raw CSV into SQLite

1. Validate import options.
2. Open the CSV file.
3. Normalize headers and rows.
4. Convert each row into a JSON object with string values.
5. Generate `DataUuid` and `JsonDataUuid`.
6. Insert rows into SQLite table `main.data_items` or the provided table name.
7. Commit the transaction.

Current compatibility details:

- If `DataSourceUuid` is `110cc994-a913-4041-96fe-a96d7e0c97e8` and domain metadata is omitted, default:
  - `TestDataDomainUuid` to `7edf2269-a8d3-472c-aed6-8cdcc4a8b6ae`
  - `TestDataDomainName` to `Sub Custody`
  - `TestDataSourceTemplateName` to `SubCustodyMain`
- Values inserted into `JsonData` are strings trimmed with `TrimSpace`.
- The importer currently interpolates the provided table name into the insert statement after applying only blank-default handling. It does not apply the same safe-table-name validation used by the SQLite query path.

### Flow E: Serve UI-compatible web API

1. Start an HTTP server, default bind address `:8080`, override by `HTTP_ADDR`.
2. Expose API endpoints:
   - `GET /api/v1/datasources`
   - `GET /api/v1/datasources/{id}/fields?source=...`
   - `GET /api/v1/datasources/{id}/facets?source=...&field=...&limit=...&q=...`
   - `POST /api/v1/query/preview`
   - `GET /api/v1/healthz`
3. Return JSON errors in the shape `{ "error": "...", "details": "..." }`.
4. Decode JSON request bodies with unknown-field rejection for preview requests.
5. For non-API paths, serve files from `ui/dist` and fall back to `ui/dist/index.html`.

The UI-facing web API and the runtime filter contract intentionally use different JSON naming conventions:

- UI/web API wrapper DTOs use lower camel case, for example `dataSourceUuid`, `supportedSources`, `compiledWhereSql`, and `dataSet`.
- Runtime filter DTOs embedded inside preview requests keep the current Pascal-case schema contract, for example `SchemaVersion`, `RequestUuid`, `DataSourceUuid`, and `RequestFilter`.
- Source values on the web API wire are lowercase strings: `csv`, `sqlite`, and `postgres`.
- The C# implementation must enforce these names explicitly with serializer attributes, dedicated serializer options, or custom converters. It must not rely on default .NET enum or property-name serialization.

## 1.3 Request Envelope Rules

These rules apply to the runtime `filters` path:

- `SchemaVersion` must equal `1.0`
- `RequestUuid` must match the runtime UUID regex
- `DataSourceUuid` must match the runtime UUID regex
- `DataSourceName` must be non-empty
- `RequestFilter` must be present and non-empty

Metadata-only request rule:

- The describe/facet paths use metadata requests that validate `SchemaVersion`, `RequestUuid`, `DataSourceUuid`, and `DataSourceName`.
- Metadata validation does not require `RequestFilter`.
- The web metadata request factory still supplies a probe filter so downstream code can reuse the same request model.

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
- Prefer schema-driven field definitions from the authoritative response schema and fall back to inference when schema loading or matching is not possible
- Canonicalize CSV header names to schema field names when aliases/misspellings are close matches

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
- Prefer schema-driven field definitions from metadata-table `JsonSchema` and fall back to local authoritative schema, then to inference when schema-driven loading is unavailable
- Infer fields across all JSON keys found in all matching rows
- Sort field names deterministically
- Coerce raw JSON values to inferred field types
- Canonicalize JSON field names to schema field names when aliases/misspellings are close matches
- Remain compatible with canonical `data_items` rows that also include:
  - `TestDataDomainUuid`
  - `TestDataDomainName`
  - `TestDataSourceTemplateName`

Important table-name rule:

- Safe table names may contain letters, digits, `_`, and `.`
- Semicolons and other punctuation must be rejected

## 1.8A Postgres Rules

The Postgres datasource implementation must:

- Require a non-empty DSN
- Default blank data table name to `public.data_items`
- Default blank schema metadata table name to `public.testdataset_response_schemas`
- Reject unsafe table names
- Query only rows where both `DataSourceUuid` and `DataSourceName` match the request
- Deserialize `JsonData` from each matching row
- Prefer schema-driven field definitions from metadata-table `JsonSchema` and fall back to local authoritative schema, then to inference when schema-driven loading is unavailable
- Infer fields across all JSON keys found in all matching rows
- Sort field names deterministically
- Coerce raw JSON values to inferred field types
- Canonicalize JSON field names to schema field names when aliases/misspellings are close matches
- Remain compatible with the same canonical `data_items` column set as SQLite, including:
  - `TestDataDomainUuid`
  - `TestDataDomainName`
  - `TestDataSourceTemplateName`

Postgres identifier and metadata compatibility:

- Qualified table identifiers are quoted by splitting on `.` and wrapping every part in double quotes, for example `public.data_items` becomes `"public"."data_items"`.
- If response-schema metadata lookup fails with an error containing `does not exist`, `undefined table`, or `relation`, treat it as missing metadata and continue at the low-level datasource layer.

## 1.9 Result Randomization and Limiting

These rules apply to CSV, SQLite, and Postgres flows:

- If fewer than two rows match, no shuffle work is required
- If no random seed GUID is supplied, use non-deterministic randomness
- If a random seed GUID is supplied, validate it and derive a deterministic seed from it
- Sort matching rows canonically before shuffle so seeded selection remains stable across backends
- Shuffle before applying `maxItems`
- `maxItems == 0` means unlimited
- `maxItems < 0` must behave as unlimited

Seed derivation detail:

- Strip hyphens from the GUID, lowercase it, hex-decode it, and read the first 8 bytes as a big-endian signed 64-bit seed.
- If exact cross-language row order is required, the .NET implementation must reproduce Go's `math/rand` source and `Shuffle` behavior, not only use any deterministic .NET RNG.
- If tests only require repeatability inside the .NET implementation, they may assert deterministic same-seed behavior without byte-for-byte Go order.

## 1.10 Response Metadata Rules

The runtime keeps two response layers:

- Internal response state:
  - `DataSourceName`
  - `DataSourceUuid`
  - `Data`
- Response-schema metadata loaded or attached before validation:
  - `TestDataSourceName`
  - `TestDataSourceUuid`
  - `JsonSchemaName`
  - `JsonSchema`
  - `UpdatedDateTime`

When the final response is marshaled to JSON, the emitted wire contract is:

- `SchemaVersion`
- `TestDataSourceName`
- `TestDataSourceUuid`
- `JsonSchemaName`
- `TestData`
  - `SpecificSourceSchemaVersion`
  - `TestDataSet`

CSV behavior:

- The main program enriches CSV responses from local JSON schema files
- `JsonSchemaName` is set to the basename of the configured schema file and canonicalized to the current response schema filename
- `UpdatedDateTime` is set to current UTC time in RFC3339 format

SQLite behavior:

- Schema metadata is loaded from `main.testdataset_response_schemas`
- The newest row by `UpdatedDateTime DESC` is used
- If the metadata table does not exist, low-level query code stays backward compatible and returns no metadata
- The main program still requires metadata because response-schema validation needs `JsonSchemaName`
- If the metadata row exists but `JsonSchema` is not valid JSON, the SQLite query must fail

Postgres behavior:

- Schema metadata is loaded from `public.testdataset_response_schemas` by default
- The newest row by `UpdatedDateTime DESC` is used
- If the metadata table does not exist, low-level query code stays backward compatible and returns no metadata
- The main program still requires metadata because response-schema validation needs `JsonSchemaName`
- If the metadata row exists but `JsonSchema` is not valid JSON, the Postgres query must fail

Row normalization behavior before final JSON marshal:

- Canonicalize row keys to schema field names where applicable
- For schema fields typed as `string`/`date`/`datetime`, normalize values to strings
- For nullable string-like schema fields, normalize empty or `NULL` (case-insensitive) to JSON `null`
- Numeric values normalized into string schema fields are emitted with `,` as decimal separator

## 1.11 Schema Validation Rules

The main program must validate:

- request JSON against `internal/json/TestDataSet_Request_Filter_To_TestDataEngine.json-schema.json`
- final response JSON against `internal/json/TestDataSet_Response_For_Specific_Datasource_From_TestDataEngine.json-schema.json`

Authoritative schema rule:

- Only root-level files directly under `internal/json` are authoritative.
- Do not use `internal/json/old`, `P26_2`, `testdata/pi26_2`, or `dot-net-requirements/json` as the schema source of truth.
- Files under `dot-net-requirements/json` are snapshots for documentation only.

Rules:

- Resolve schema path from local working directory and source-relative fallback paths
- For response validation, only the basename of `JsonSchemaName` is trusted and it must resolve to a root-level file directly under `internal/json`
- Missing schema files must fail
- Invalid payload JSON must fail
- Path-traversal-style schema names must resolve by basename only
- The legacy response schema filename without `_From_` may be accepted as an alias, but it must canonicalize to `TestDataSet_Response_For_Specific_Datasource_From_TestDataEngine.json-schema.json`

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

- `-source` with values `csv`, `sqlite`, `postgres`, or `all`
- `-csv`
- `-sqlite-db`
- `-sqlite-table`
- `-postgres-dsn`
- `-postgres-table`
- `-postgres-schema-table`
- `-max-items`
- `-random-seed-guid`

Default behavior:

- Default source is `csv`
- Default CSV path is `p26_2/FenixRawTestdata_646rows_211220_stripped.csv`
- If that default path does not exist, try `P26_2/FenixRawTestdata_646rows_211220_stripped.csv`
- Default SQLite DB path is `testdata/SQLiteDB/identifier.sqlite`
- Default SQLite table is `main.data_items`
- Default Postgres data table is `public.data_items`
- Default Postgres schema metadata table is `public.testdataset_response_schemas`
- Default max items is `2`

## 1.14 Runtime Output Shape

The main program logs two response objects, each wrapped with the original input filter:

The main program also logs:

- `WHERE=<compiled where sql>`
- `ARGS=<compiled args>`

Allowed-fields output shape:

```json
{
  "Source": "csv|sqlite|postgres",
  "InputFilter": { "...": "..." },
  "AllowedFieldsResponse": { "...": "..." }
}
```

Dataset output shape:

```json
{
  "Source": "csv|sqlite|postgres",
  "InputFilter": { "...": "..." },
  "DataSetResponse": { "...": "..." }
}
```

The log message payload keys are:

- `AllowedFieldsResponse=...`
- `DataSetResponse=...`

## 1.15 Web API Contract for UI

The .NET rewrite must expose an API contract compatible with the existing Vue UI under `ui/`.

Routing and hosting requirements:

- API prefix is `/api/v1`
- `GET /api/v1/healthz` returns `{ "status": "ok" }`
- Non-API requests are served from `ui/dist`
- Unknown client-side routes return `ui/dist/index.html` (SPA fallback)
- Unknown paths under `/api/` return a plain HTTP `404`, not the JSON error envelope.
- If `ui/dist/index.html` is missing, non-API fallback returns HTTP `503` with the JSON error envelope.

Static catalog requirements:

- The built-in catalog must include exactly the current UI datasource unless the product intentionally adds more:
  - `id`: `subcustody`
  - `label`: `SubCustody`
  - `dataSourceName`: `SubCustody`
  - `dataSourceUuid`: `110cc994-a913-4041-96fe-a96d7e0c97e8`
  - `supportedSources`: `csv`, `sqlite`
  - `defaultSource`: `sqlite`
  - hidden CSV path: `testdata/pi26_2/FenixRawTestdata_646rows_211220_stripped.csv`
  - hidden SQLite DB path: `testdata/SQLiteDB/identifier.sqlite`
  - hidden SQLite table: `main.data_items`
  - hidden Postgres data table: `public.data_items`
  - hidden Postgres schema table: `public.testdataset_response_schemas`
- Hidden catalog fields are required by query, describe, and facet services, but must not be emitted in the datasource-list response.

Source parsing requirements:

- Accepted web source values are exact lowercase `csv`, `sqlite`, and `postgres`.
- Fields and facets default missing or unrecognized `source` query parameters to the datasource `defaultSource`.
- Fields and facets reject recognized but unsupported sources with HTTP `400`.
- Preview requests require a non-empty `source`; missing source is HTTP `400`.
- Preview rejects unsupported sources with HTTP `400`.

Datasource list endpoint:

- `GET /api/v1/datasources`
- Response shape:
  - `items[]` with:
    - `id`
    - `label`
    - `dataSourceName`
    - `dataSourceUuid`
    - `supportedSources` (`csv`/`sqlite`/`postgres`)
    - `defaultSource`

Fields endpoint:

- `GET /api/v1/datasources/{id}/fields?source=...`
- If `source` is missing, use datasource `defaultSource`
- Response shape:
  - `datasourceId`
  - `source`
  - `fields[]` with:
    - `field`
    - `fieldType`
    - `nullable`
    - `supportedOperators`
    - `widget`
    - `facetEligible`
    - `description` (optional)

Field UI mapping:

- `boolean` fields use widget `boolean-toggle`.
- `number` and `integer` fields use widget `searchable-checkbox-group`.
- All other current field types also use widget `searchable-checkbox-group`.
- `facetEligible` is `true` only for `string`, `boolean`, `number`, and `integer`; it is `false` for `date`, `datetime`, and any other type.

Facets endpoint:

- `GET /api/v1/datasources/{id}/facets?source=...&field=...&limit=...&q=...`
- `field` is required
- `limit` defaults to `100`
- Response shape:
  - `datasourceId`
  - `source`
  - `field`
  - `values[]` with:
    - `value`
    - `label`
    - `count`
    - `isNull`
  - `truncated`

Facet behavior:

- Null values are returned as `value: null`, `label: "(blank)"`, `isNull: true`.
- Distinct values are keyed by runtime type plus value; null uses a dedicated `null` key.
- The `q` parameter is trimmed and matched as a case-insensitive substring against the display label.
- Results sort by `count` descending, then `label` ascending.
- `limit <= 0` means unlimited.
- `truncated` is `true` only when a positive `limit` removes results.
- Unknown fields fail with HTTP `400`.

Preview endpoint:

- `POST /api/v1/query/preview`
- Request shape:
  - `source`
  - `maxItems`
  - `randomSeedGuid` (optional)
  - `request` (`FilterRequest`)
- Response shape:
  - `source`
  - `compiledWhereSql`
  - `compiledArgs`
  - `allowedFields`
  - `dataSet`

Error envelope:

- Error responses must use:
  - `error` (short category)
  - `details` (optional detail string)

HTTP status requirements:

- Unknown datasource id in fields/facets returns `404`.
- Unsupported source returns `400`.
- Missing facet `field` returns `400`.
- Invalid facet `limit` returns `400`.
- Failed metadata request construction returns `500`.
- Invalid preview JSON, missing preview source, unknown preview datasource, unsupported preview source, and preview query failures return `400`.
