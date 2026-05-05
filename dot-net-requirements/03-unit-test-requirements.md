# 3. Unit-Test Requirements

The .NET rewrite must ship with unit tests that prove behavioral equivalence with the current Go repository.

Recommended framework:

- `xUnit`

Recommended assertion style:

- exact result checks for returned objects
- substring checks for error categories
- deterministic comparisons for seeded randomization

## 3.1 Runtime Filter-Core Tests

Must cover:

- `toSet`-equivalent returns unique operators
- operator sorting order is:
  - `eq`, `neq`, `gt`, `gte`, `lt`, `lte`, `in`, `nin`, `contains`, `startsWith`, `endsWith`, `exists`, `isNull`
- allowed fields are sorted by field name
- request validation success path
- request validation failures:
  - bad schema version
  - bad request UUID
  - bad datasource UUID
  - missing datasource name
  - missing filter
  - unknown datasource
  - datasource UUID mismatch
- allowed-fields response success path
- allowed-fields response invalid schema version failure
- logical expression compilation for nested `and` and `not`
- empty logical expression failure
- SQL compilation for every supported operator
- unsafe field-name rejection
- scalar/comparable helper behavior
- identifier quoting behavior
- UUID helper behavior
- numeric/integer JSON helper behavior

## 3.2 CSV Datasource Tests

Must cover:

- full happy-path CSV query
- inferred allowed fields are non-empty
- compiled SQL is non-empty
- filtered result count is correct
- returned row content is correct
- marshaled dataset response uses the wrapped `TestData` object with:
  - `SpecificSourceSchemaVersion`
  - `TestDataSet`
- runtime CSV header normalization exact behavior:
  - BOM is removed from the first header cell
  - non-first headers are trimmed normally
  - first-header BOM-plus-space behavior matches current Go output
- schema-first CSV loading with fallback to inference
- canonical schema-field mapping for CSV headers (including legacy misspelling compatibility)
- row normalization pads short rows
- type inference:
  - boolean
  - integer
  - number
  - string
  - all-null column becomes string
  - date and datetime are not auto-inferred
- null handling for empty string and `NULL`
- operator evaluation:
  - `eq`
  - `neq`
  - `gt`
  - `gte`
  - `lt`
  - `lte`
  - `contains`
  - `startsWith`
  - `endsWith`
  - `exists`
  - `isNull`
  - `in`
  - `nin`
- `valuesEqual` behavior by type
- ordered comparison behavior by type
- unsupported ordered comparison on non-comparable type fails

## 3.3 CSV Randomization and Limit Tests

Must cover:

- `maxItems == 0` means unlimited
- `maxItems < 0` behaves like unlimited
- positive `maxItems` truncates after shuffle
- shuffle preserves all original elements
- shuffle on one element leaves it unchanged
- seeded shuffle with the same GUID is deterministic
- seeded CSV query with the same GUID returns the same rows
- invalid seed GUID fails
- seed derivation uses the first 8 decoded GUID bytes as a big-endian signed 64-bit value
- if Go byte-for-byte parity is required, seeded shuffle fixtures match Go `math/rand.Shuffle` output for fixed GUIDs

## 3.4 SQLite Datasource Tests

Must cover:

- full happy-path SQLite query using a temp DB
- inferred allowed fields are non-empty
- compiled SQL is non-empty
- filtered row count is correct
- returned row content is correct
- response metadata from `main.testdataset_response_schemas` is attached
- `JsonSchema` in response metadata is valid JSON
- `UpdatedDateTime` is populated
- schema-first SQLite loading from metadata `JsonSchema` with fallback behavior
- canonical schema-field mapping for SQLite `JsonData` keys
- `maxItems == 0` means unlimited
- seeded SQLite query with the same GUID is deterministic
- blank DB path fails
- unsafe table name fails
- safe table-name helper accepts `main.data_items`
- safe table-name helper rejects injected names
- stable field-order collection
- raw JSON value to inference-string conversion
- raw value coercion to inferred types

## 3.4A Postgres Datasource Tests

Must cover:

- blank Postgres DSN fails
- unsafe Postgres data table name fails
- unsafe Postgres schema metadata table name fails
- canonical schema-field mapping for Postgres `JsonData` keys
- quoted qualified identifier helper behavior (`schema.table` -> `"schema"."table"`)
- Postgres schema fixture contains the same canonical `data_items` metadata columns as SQLite:
  - `TestDataDomainUuid`
  - `TestDataDomainName`
  - `TestDataSourceTemplateName`

## 3.5 Typed `filtersql` Tests

Must cover:

- every supported operator is marked valid
- unknown operator is invalid
- expression marker interfaces/classes are valid
- `hasOnlyKey` helper behavior
- parse expression as:
  - comparison
  - `and`
  - `or`
  - `not`
- parse invalid empty object fails
- parse non-object JSON fails
- custom request JSON deserialization success path
- invalid `RequestFilter` deserialization failure
- typed request validation success path
- typed request validation failure cases:
  - bad schema version
  - bad request UUID
  - bad datasource UUID
  - missing datasource name
  - missing request filter
- typed expression validation:
  - valid `and`
  - valid `or`
  - valid `not`
  - empty `and` fails
  - empty `or` fails
  - empty `not` fails
  - unsupported expression type fails
- comparison validation success cases for:
  - scalar operators
  - array operators
  - boolean null operators
- comparison validation failure cases for:
  - missing field
  - bad operator
  - wrong value type for `exists`
  - wrong value type for `isNull`
  - empty array for `in`
  - non-string value for `contains`
  - null value for `eq`
  - non-scalar object value
- scalar helper behavior
- compiler placeholder mode:
  - `?`
  - `$n`
- identifier quoting on and off
- invalid identifier rejection
- compile nested logical expression tree
- internal compiler helper checks:
  - `eq null => IS NULL`
  - `neq null => IS NOT NULL`
  - `in`
  - `isNull`
  - unsupported operator failure
  - unsupported expression failure

## 3.6 Logging Tests

Must cover:

- `Infof` equivalent includes the supplied UUID in each line
- different log calls keep different UUIDs
- `Errorf` equivalent prefixes `ERROR: `
- internal format helper builds `Id=<uuid> <message>`
- `Fatalf` equivalent terminates the process
- fatal logging test should run through a subprocess so the test runner process does not exit

## 3.7 Importer Tests

Must cover:

- importing a small CSV into a temp SQLite DB
- inserted row count is correct
- stored datasource metadata is correct
- stored domain metadata is correct:
  - `TestDataDomainUuid`
  - `TestDataDomainName`
  - `TestDataSourceTemplateName`
- stored JSON payload values are correct
- validation failure for empty options
- missing CSV file failure
- importer header normalization helper
- record normalization helper
- payload builder helper
- generated UUID format
- SubCustody import defaults missing domain metadata to:
  - `7edf2269-a8d3-472c-aed6-8cdcc4a8b6ae`
  - `Sub Custody`
  - `SubCustodyMain`
- imported JSON payload values are trimmed strings
- importer table-name behavior is tested separately from query safe-table-name validation

## 3.8 Main-Program Schema Tests

Must cover:

- valid request schema passes
- request without `RequestFilter` fails schema validation
- valid response schema passes against the root-level file in `internal/json`
- empty `JsonSchemaName` fails response validation
- path-traversal-style `JsonSchemaName` resolves by basename
- unknown `JsonSchemaName` fails
- legacy response schema filename without `_From_` canonicalizes to the current filename
- validating invalid JSON payload fails
- missing schema file fails
- local schema metadata enrichment succeeds
- local schema metadata enrichment fails for missing file
- local metadata enrichment stores basename only

## 3.9 Additional Behavior Tests

Must also cover:

- SQLite query remains backward compatible when `main.testdataset_response_schemas` does not exist
- SQLite query fails when schema metadata row contains invalid JSON in `JsonSchema`
- Postgres metadata lookup treats `does not exist`, `undefined table`, and `relation` errors as missing metadata
- Postgres qualified identifier quoting splits on `.` and quotes every part
- runtime CSV and importer header normalization remain intentionally different
- schema-catalog parsing and canonicalization behavior:
  - required-field-first ordering
  - extra fields sorted alphabetically after required fields
  - nullable-field parsing from `oneOf`
  - exact normalized field-name match only when unique
  - canonical alias resolution from near-miss field names
  - ambiguous exact/fuzzy matches preserve the original field name
- main-program log output includes `WHERE=` and `ARGS=` lines before wrapped response payload logs

## 3.10 Web API and Request-Factory Tests

Must cover:

- metadata request factory builds a probe filter from schema-derived first field
- metadata request factory uses `11111111-1111-4111-8111-111111111111`
- metadata request factory emits probe filter:
  - `op == "exists"`
  - `value == true`
- metadata request validation accepts requests without `RequestFilter`
- API route coverage:
  - list datasources
  - fields endpoint success and failure modes
  - facets endpoint success and failure modes
  - preview endpoint success and failure modes
  - health endpoint
- static catalog exact `subcustody` values:
  - `id`
  - `label`
  - `dataSourceName`
  - `dataSourceUuid`
  - `supportedSources`
  - `defaultSource`
  - hidden CSV/SQLite/Postgres config values
- source parsing:
  - exact lowercase values are accepted
  - fields/facets default missing source to default source
  - fields/facets default unrecognized source to default source
  - recognized unsupported source fails
  - preview missing source fails
- field descriptor UI mapping:
  - `boolean` => `boolean-toggle`
  - `number` and `integer` => `searchable-checkbox-group`
  - facet eligibility only for `string`, `boolean`, `number`, `integer`
- facet behavior:
  - null value label is `(blank)`
  - search is case-insensitive substring on label
  - sort by count descending then label ascending
  - `limit <= 0` means unlimited
  - `truncated` only when a positive limit removes values
  - unknown field fails
- HTTP status codes:
  - unknown datasource id is `404`
  - unsupported source is `400`
  - missing facet field is `400`
  - invalid facet limit is `400`
  - metadata factory failure is `500`
  - missing UI build is `503`
  - unknown `/api/` route is plain `404`
- API error envelope shape:
  - `error`
  - `details` (optional)
- preview request JSON body unknown fields are rejected

## 3.10A C# Serialization and Model Tests

Must cover:

- web DTO serialization uses lower camel case:
  - `dataSourceUuid`
  - `supportedSources`
  - `defaultSource`
  - `compiledWhereSql`
  - `compiledArgs`
  - `allowedFields`
  - `dataSet`
- runtime filter DTO serialization keeps Pascal case:
  - `SchemaVersion`
  - `RequestUuid`
  - `DataSourceUuid`
  - `DataSourceName`
  - `RequestFilter`
- `SourceType` serializes and deserializes as lowercase string values `csv`, `sqlite`, and `postgres`
- `SourceType` rejects unknown source strings during preview-body deserialization
- `DataSetResponse` wire JSON contains wrapped `TestData.SpecificSourceSchemaVersion` and `TestData.TestDataSet`
- `DataSetResponse` wire JSON omits internal properties:
  - `JsonSchema`
  - `UpdatedDateTime`
  - `DataSourceName`
  - `DataSourceUuid`
  - `Data`
- public `DataSourceListItem` JSON omits hidden datasource config
- internal `DataSourceConfig` retains hidden CSV/SQLite/Postgres paths and tables
- CLI log envelopes serialize `Source`, `InputFilter`, and the appropriate response object

## 3.11 Test Fixture Requirements

The .NET test project should include or generate:

- temporary SQLite databases
- small inline CSV files for focused tests
- repository sample CSV files for broader integration-style tests
- request schema file
- specific response schema file
- deterministic GUIDs reused across tests

Examples of fixed GUIDs already used by the Go tests:

- `6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f`
- `110cc994-a913-4041-96fe-a96d7e0c97e8`
- `aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa`
- `bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb`
- `cccccccc-cccc-4ccc-8ccc-cccccccccccc`

## 3.12 Minimum Acceptance Criteria

The .NET rewrite is acceptable only if:

- request/response contracts match the documented shapes
- C# serialization preserves both the UI lower-camel web contract and the Pascal-case runtime schema contract
- CSV, SQLite, and Postgres queries produce equivalent filtering behavior
- seeded randomization is deterministic
- schema validation failures happen in the same situations
- logging always includes UUID markers
- importer writes valid rows to SQLite
- UI-facing HTTP API contract matches the documented endpoint and payload shapes
- the test suite covers both happy paths and guard rails
