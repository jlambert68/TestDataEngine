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
- `maxItems == 0` means unlimited
- seeded SQLite query with the same GUID is deterministic
- blank DB path fails
- unsafe table name fails
- safe table-name helper accepts `main.data_items`
- safe table-name helper rejects injected names
- stable field-order collection
- raw JSON value to inference-string conversion
- raw value coercion to inferred types

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
- stored JSON payload values are correct
- validation failure for empty options
- missing CSV file failure
- importer header normalization helper
- record normalization helper
- payload builder helper
- generated UUID format

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
- runtime CSV and importer header normalization remain intentionally different
- main-program log output includes `WHERE=` and `ARGS=` lines before wrapped response payload logs

## 3.10 Test Fixture Requirements

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

## 3.11 Minimum Acceptance Criteria

The .NET rewrite is acceptable only if:

- request/response contracts match the documented shapes
- CSV and SQLite queries produce equivalent filtering behavior
- seeded randomization is deterministic
- schema validation failures happen in the same situations
- logging always includes UUID markers
- importer writes valid rows to SQLite
- the test suite covers both happy paths and guard rails
