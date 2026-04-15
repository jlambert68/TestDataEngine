# 2. Contract Details

Schema authority note:

- Only root-level files directly under `internal/json` are authoritative.
- Do not treat `internal/json/old`, `P26_2`, `testdata/pi26_2`, or `dot-net-requirements/json` as contract-defining schema sources.

## 2.1 Runtime `filters` Contract

This is the contract used by the CSV and SQLite query engine.

### Request model

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

### Allowed-fields response model

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
      "SupportedOperators": ["eq", "neq", "in", "nin", "contains", "startsWith", "endsWith", "exists", "isNull"],
      "Description": "Inferred from CSV column \"AccountCurrency\"."
    }
  ]
}
```

### Dataset response model

```json
{
  "SchemaVersion": "1.0",
  "TestDataSourceName": "SubCustody",
  "TestDataSourceUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "JsonSchemaName": "TestDataSet_Response_For_Specific_Datasource_From_TestDataEngine.json-schema.json",
  "TestData": {
    "SpecificSourceSchemaVersion": "1.0",
    "TestDataSet": [
      {
        "AccountCurrency": "SEK"
      }
    ]
  }
}
```

Important:

- `JsonSchema`, `UpdatedDateTime`, `DataSourceName`, `DataSourceUuid`, and the raw `Data` slice exist in the Go runtime as internal state.
- Those fields are not part of the emitted JSON contract after `DataSetResponse` is marshaled.

## 2.2 Expression Shape Rules

Implementation note for .NET:

- The JSON contract is shape-based, not discriminator-based
- A custom `JsonConverter` or equivalent parser is required to map JSON objects to the correct expression type
- Node selection must follow the current Go behavior: inspect the object shape first, then choose comparison / `and` / `or` / `not`
- The typed `filtersql` package and the runtime `filters` package should have separate model layers in .NET even when they serialize to very similar JSON

### Comparison

```json
{ "field": "Amount", "op": "gt", "value": 100 }
```

### And

```json
{
  "and": [
    { "field": "AccountCurrency", "op": "eq", "value": "SEK" },
    { "field": "Flag", "op": "eq", "value": true }
  ]
}
```

### Or

```json
{
  "or": [
    { "field": "AccountCurrency", "op": "eq", "value": "SEK" },
    { "field": "AccountCurrency", "op": "eq", "value": "NOK" }
  ]
}
```

### Not

```json
{
  "not": {
    "field": "AccountEnvironment",
    "op": "eq",
    "value": "Prod"
  }
}
```

## 2.3 Operator-by-Operator Requirements

### `eq`

- Runtime `filters`: requires a scalar of the field's type
- Typed `filtersql`: public validation requires a non-null scalar
- Internal typed compiler helper also supports `eq null => IS NULL`

Example:

```json
{ "field": "status", "op": "eq", "value": "active" }
```

### `neq`

- Runtime `filters`: requires a scalar of the field's type
- Typed `filtersql`: public validation requires a non-null scalar
- Internal typed compiler helper also supports `neq null => IS NOT NULL`

Example:

```json
{ "field": "status", "op": "neq", "value": "closed" }
```

### `gt`, `gte`, `lt`, `lte`

- Allowed only on `number`, `integer`, `date`, `datetime`
- CSV runtime compares numbers numerically
- CSV runtime compares strings, dates, and datetimes lexicographically
- SQLite and CSV inference currently produce `boolean`, `integer`, `number`, or `string`; they do not auto-infer `date` or `datetime`

Example:

```json
{ "field": "Amount", "op": "gte", "value": 100 }
```

### `in`, `nin`

- Require a non-empty array
- Every item must be valid for the field type

Example:

```json
{ "field": "AccountCurrency", "op": "in", "value": ["SEK", "NOK"] }
```

### `contains`, `startsWith`, `endsWith`

- Require a string value
- Evaluated only against string row values

Example:

```json
{ "field": "customerEmail", "op": "contains", "value": "@example.com" }
```

### `exists`

- Requires a boolean value
- `true` means row field must be non-null
- `false` means row field must be null

Example:

```json
{ "field": "customerEmail", "op": "exists", "value": true }
```

### `isNull`

- Requires a boolean value
- `true` means row field must be null
- `false` means row field must be non-null

Example:

```json
{ "field": "customerEmail", "op": "isNull", "value": false }
```

## 2.4 Evaluation Semantics

Row evaluation rules in the runtime engine:

- Comparison nodes reuse the same validation rules as SQL compilation
- `and` short-circuits on first false
- `or` short-circuits on first true
- `not` negates the nested expression result
- Comparing any ordered operator against a null row value returns `false`
- Internal equality helper logic treats `null == null` as true, but the public runtime comparison path rejects `eq` and `neq` with null filter values before evaluation

Examples:

```text
row value = null
filter = {"field":"x","op":"isNull","value":true}
result = true
```

```text
row value = null
filter = {"field":"x","op":"gt","value":1}
result = false
```

## 2.5 Runtime and Typed-Compiler Differences

These are required compatibility details for the .NET rewrite.

### Difference A: UUID validation

- Runtime `filters`: loose UUID format check
- Typed `filtersql`: strict UUID version/variant check

### Difference B: Datasource knowledge

- Runtime `filters` can validate against a hard-coded datasource catalog or inferred datasource definitions
- Typed `filtersql` validates expression shape and operator/value compatibility only
- Typed `filtersql` does not validate field names against a datasource catalog

### Difference C: Null handling

- Runtime `filters` public contract uses `exists` and `isNull` for null checks
- Typed `filtersql` internal compiler helper supports `eq null` and `neq null`
- Public typed request validation rejects `eq` and `neq` with null

### Difference D: Header normalization

- Runtime CSV query path uses `TrimSpace` and then removes BOM from the first header
- Importer path removes BOM from the first header cell and then trims
- This leads to different first-header behavior when BOM and leading spaces exist

### Difference E: Typed compiler options

- The typed `filtersql` compiler has explicit options for placeholder style and identifier quoting
- The runtime `filters` compiler always uses `?` placeholders and quoted identifiers

## 2.6 Database Tables Required by the Runtime

### SQLite main data table

Required columns:

- `DataSourceUuid`
- `DataSourceName`
- `TestDataDomainUuid`
- `TestDataDomainName`
- `TestDataSourceTemplateName`
- `DataUuid`
- `DataUpdateTimeStamp`
- `JsonDataUuid`
- `JsonData`

### SQLite response schema metadata table

Required table name:

- `main.testdataset_response_schemas`

Required columns:

- `TestDataSourceName`
- `TestDataSourceUuid`
- `JsonSchemaName`
- `JsonSchema`
- `UpdatedDateTime`

Selection rule:

- Use the newest matching row by `UpdatedDateTime DESC`

### Postgres main data table

Required default table name:

- `public.data_items`

Required columns:

- `DataSourceUuid`
- `DataSourceName`
- `TestDataDomainUuid`
- `TestDataDomainName`
- `TestDataSourceTemplateName`
- `DataUuid`
- `DataUpdateTimeStamp`
- `JsonDataUuid`
- `JsonData`

### Postgres response schema metadata table

Required default table name:

- `public.testdataset_response_schemas`

Required columns:

- `TestDataSourceName`
- `TestDataSourceUuid`
- `JsonSchemaName`
- `JsonSchema`
- `UpdatedDateTime`

## 2.7 Error Requirements

The .NET rewrite does not need to reproduce Go error text byte-for-byte, but it must preserve failure categories and enough wording for unit tests to assert the same intent.

Minimum required error categories:

- unsupported schema version
- invalid request UUID
- invalid datasource UUID
- missing datasource name
- missing request filter
- unknown datasource name
- datasource UUID mismatch
- invalid expression shape
- empty `and`
- empty `or`
- empty `not`
- invalid operator
- invalid operator value type
- unsafe field name
- unsafe table name
- missing CSV path
- missing DB path
- missing schema file
- invalid JSON schema metadata
- invalid JSON payload in SQLite `JsonData`
- invalid response schema name
- invalid random seed GUID
- CSV empty file or missing header
- no SQLite rows found for the datasource

## 2.8 Required .NET Architecture

A clean rewrite should have at least these logical components:

- request/response contract models
- runtime filter compiler
- typed `filtersql` compiler
- CSV datasource adapter
- SQLite datasource adapter
- JSON schema validator
- CSV-to-SQLite importer
- logging wrapper
- executable entry point

Recommended implementation split:

- `Contracts`
- `Filtering.Runtime`
- `Filtering.TypedSql`
- `DataSources.Csv`
- `DataSources.Sqlite`
- `SchemaValidation`
- `Importing`
- `Logging`
- `Cli`
