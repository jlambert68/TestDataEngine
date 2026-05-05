# 2. Contract Details

Schema authority note:

- Only root-level files directly under `internal/json` are authoritative.
- Do not treat `internal/json/old`, `P26_2`, `testdata/pi26_2`, or `dot-net-requirements/json` as contract-defining schema sources.

JSON serialization note:

- Runtime filter models use Pascal-case JSON names defined by the JSON schemas.
- UI/web API wrapper models use lower camel case JSON names defined by `ui/src/types/api.ts` and `internal/webapi/contracts.go`.
- Source values must serialize as lowercase strings: `csv`, `sqlite`, and `postgres`.
- C# records/classes must use explicit `JsonPropertyName`, dedicated serializer options, or custom converters so both contracts can coexist safely in the same process.

## 2.1 Runtime `filters` Contract

This is the contract used by the CSV, SQLite, and Postgres query engine.

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
- The emitted `TestData` property is an object containing `SpecificSourceSchemaVersion` and `TestDataSet`; it is not the raw internal row list.

Canonical field behavior:

- Runtime loading is schema-first and falls back to inference.
- CSV/SQLite/Postgres rows are normalized to canonical schema field names when aliases or misspellings are close matches.
- This includes current compatibility for legacy misspellings such as `ClientJuristictionCountryCode` mapping to `ClientJurisdictionCountryCode`.

Schema catalog parsing and canonical matching:

- Read fields from `$defs.TestDataSetItem.properties` in the response schema.
- Required fields preserve the order from the schema's `required` array.
- Non-required fields are appended in alphabetical order.
- `type` accepts one current field type; `oneOf` may combine one current field type with `null`.
- Canonical lookup first strips all non-ASCII-alphanumeric characters and lowercases the candidate name.
- A normalized exact match is accepted only when it is unique.
- If no unique normalized exact match exists, use the unique best Levenshtein match only when distance is `<= 2`.
- Ambiguous exact or fuzzy matches leave the original field name unchanged.

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

### Difference F: Schema-first datasource modeling

- Runtime CSV/SQLite/Postgres paths first try schema-driven field definitions.
- If schema-driven loading fails or does not match, runtime falls back to legacy inference.
- Typed `filtersql` is independent of datasource schema loading and operates on expression validity/compilation.

### Difference G: Runtime and metadata request validation

- Normal runtime query validation requires a non-empty `RequestFilter`.
- Metadata validation for describe/facet flows does not require `RequestFilter`.
- The web request factory builds a probe filter anyway:
  - `RequestUuid`: `11111111-1111-4111-8111-111111111111`
  - `field`: first schema-catalog field
  - `op`: `exists`
  - `value`: `true`

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

Postgres lookup details:

- Table names are validated with the same safe-table-name rule as SQLite.
- Qualified table names are quoted by splitting on `.` and quoting each part.
- Missing metadata table compatibility treats errors containing `does not exist`, `undefined table`, or `relation` as no metadata.

## 2.6A Web API Contract Required for UI Compatibility

The .NET rewrite must include an HTTP layer compatible with the existing UI under `ui/`.

### Required routes

- `GET /api/v1/datasources`
- `GET /api/v1/datasources/{id}/fields?source=...`
- `GET /api/v1/datasources/{id}/facets?source=...&field=...&limit=...&q=...`
- `POST /api/v1/query/preview`
- `GET /api/v1/healthz`

### Error envelope

- API errors must return:
  - `error`
  - `details` (optional)

Status code requirements:

- Unknown datasource id on fields/facets: `404`
- Unsupported source: `400`
- Missing facet `field`: `400`
- Invalid facet `limit`: `400`
- Failed metadata request factory: `500`
- Invalid preview body: `400`
- Missing preview source: `400`
- Unknown preview datasource: `400`
- Preview query failure: `400`
- Missing UI build: `503`
- Unknown `/api/` route: plain `404`

### Source values

- Accepted serialized values are exactly `csv`, `sqlite`, and `postgres`.
- Fields/facets default a missing or unrecognized query-string source to `defaultSource`.
- Preview requires an explicit non-empty source.

### Static catalog

The built-in catalog must contain this datasource:

```json
{
  "id": "subcustody",
  "label": "SubCustody",
  "dataSourceName": "SubCustody",
  "dataSourceUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
  "supportedSources": ["csv", "sqlite"],
  "defaultSource": "sqlite"
}
```

Hidden config for the same datasource:

- `CSVPath`: `testdata/pi26_2/FenixRawTestdata_646rows_211220_stripped.csv`
- `SQLiteDB`: `testdata/SQLiteDB/identifier.sqlite`
- `SQLiteTable`: `main.data_items`
- `PostgresTable`: `public.data_items`
- `PostgresSchema`: `public.testdataset_response_schemas`
- `PostgresDSN`: empty unless configured externally

### Datasource list response

Each datasource item must include:

- `id`
- `label`
- `dataSourceName`
- `dataSourceUuid`
- `supportedSources`
- `defaultSource`

### Fields response

- `datasourceId`
- `source`
- `fields[]` entries include:
  - `field`
  - `fieldType`
  - `nullable`
  - `supportedOperators`
  - `widget`
  - `facetEligible`
  - `description` (optional)

Field mapping:

- `widget` is `boolean-toggle` for `boolean`.
- `widget` is `searchable-checkbox-group` for `number`, `integer`, and all other current field types.
- `facetEligible` is `true` only for `string`, `boolean`, `number`, and `integer`.

### Facets response

- `datasourceId`
- `source`
- `field`
- `values[]` entries include:
  - `value`
  - `label`
  - `count`
  - `isNull`
- `truncated`

Facet value semantics:

- Null values use label `(blank)`.
- Search text is matched against the label using case-insensitive substring matching after trimming the query text.
- Results sort by count descending and then label ascending.
- `limit <= 0` is unlimited.
- `truncated` is true only when a positive limit cuts off values.

### Preview request/response

Request:

- `source`
- `maxItems`
- `randomSeedGuid` (optional)
- `request` (`FilterRequest`)

Response:

- `source`
- `compiledWhereSql`
- `compiledArgs`
- `allowedFields`
- `dataSet`

The nested `request` object retains runtime Pascal-case names while the preview wrapper uses lower camel case.

## 2.6C C# Model Requirements

The C# model layer must separate public wire DTOs from internal runtime state:

- Public web DTOs must serialize exactly like `ui/src/types/api.ts`.
- Runtime filter DTOs must serialize exactly like the JSON schemas.
- `DataSetResponse` must keep `JsonSchema`, `UpdatedDateTime`, `DataSourceName`, `DataSourceUuid`, and raw `Data` as internal/non-serialized properties.
- `DataSetResponse.TestData` on the wire must be a `SpecificDatasourceTestData` object, not a raw row list.
- `DataSourceListItem` is the public catalog DTO.
- `DataSourceConfig` or an equivalent internal model is required for hidden CSV/SQLite/Postgres paths and tables.
- Web query, facet, and request-factory services must accept the internal datasource config, not only the public datasource-list DTO.
- CLI log envelopes must include `Source`, `InputFilter`, and the response object.

## 2.6B Static UI Hosting Contract

The same .NET process should serve static UI assets compatible with current Go behavior:

- Bind address defaults to `:8080` and can be overridden by `HTTP_ADDR`.
- Static root is `ui/dist`.
- Non-API unknown paths return `ui/dist/index.html` to support SPA routing.

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
- unsafe schema metadata table name
- missing CSV path
- missing DB path
- missing Postgres DSN
- missing schema file
- invalid JSON schema metadata
- invalid JSON payload in SQLite `JsonData`
- invalid response schema name
- invalid random seed GUID
- CSV empty file or missing header
- no SQLite rows found for the datasource
- unsupported source
- unknown datasource id
- invalid request body
- missing facet field parameter
- invalid facet limit parameter

## 2.8 Required .NET Architecture

A clean rewrite should have at least these logical components:

- request/response contract models
- runtime filter compiler
- typed `filtersql` compiler
- CSV datasource adapter
- SQLite datasource adapter
- Postgres datasource adapter
- JSON schema validator
- CSV-to-SQLite importer
- logging wrapper
- web API server layer
- executable entry point

Recommended implementation split:

- `Contracts`
- `Filtering.Runtime`
- `Filtering.TypedSql`
- `DataSources.Csv`
- `DataSources.Sqlite`
- `DataSources.Postgres`
- `SchemaValidation`
- `Importing`
- `Logging`
- `WebApi`
- `Cli`
