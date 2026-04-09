-- Table: main.data_items
-- Purpose: Stores JSON payload rows grouped by datasource metadata.
create table main.data_items
(
    DataSourceUuid      TEXT not null, -- UUID of the source system.
    DataSourceName      TEXT not null, -- Human-readable name of the source system.
    DataUuid            TEXT not null, -- UUID of the logical data entity in the source. The same UUID for all rows(json data) inserted at the same time.
    DataUpdateTimeStamp TEXT not null, -- Last update timestamp from the source.
    JsonDataUuid        TEXT not null  -- Unique UUID for this JSON payload row.
        primary key,
    JsonData            TEXT not null, -- JSON payload as text.
    check (json_valid(JsonData))       -- Ensures JsonData is valid JSON.
);

-- Index for direct lookup by unique JSON payload UUID.
create index main.idx_data_items_json_data_uuid
    on main.data_items (JsonDataUuid);

-- Index for filtering by datasource name.
create index main.idx_data_items_source_name
    on main.data_items (DataSourceName);

-- Index for filtering by datasource UUID.
create index main.idx_data_items_source_uuid
    on main.data_items (DataSourceUuid);

-- Index for sorting/filtering by source update time.
create index main.idx_data_items_updated
    on main.data_items (DataUpdateTimeStamp);


-- Table: main.testdataset_response_schemas
-- Purpose: Stores response JSON-schema metadata per test datasource.
create table main.testdataset_response_schemas
(
    TestDataSourceName TEXT not null, -- Logical datasource name.
    TestDataSourceUuid TEXT not null, -- Logical datasource UUID.
    JsonSchemaName     TEXT not null, -- Name of the schema document.
    JsonSchema         TEXT not null, -- JSON-schema document as JSON text.
    UpdatedDateTime    TEXT not null, -- Last metadata update timestamp (RFC3339).
    check (json_valid(JsonSchema))
);

-- Ensures one active schema entry per datasource + schema name.
create unique index main.udx_testdataset_response_schemas_source_schema
    on main.testdataset_response_schemas (TestDataSourceName, TestDataSourceUuid, JsonSchemaName);

-- Seed metadata for the specific datasource response schema.
insert into main.testdataset_response_schemas
    (TestDataSourceName, TestDataSourceUuid, JsonSchemaName, JsonSchema, UpdatedDateTime)
values ('SubCustody',
        '110cc994-a913-4041-96fe-a96d7e0c97e8',
        'TestDataSet_Response_For_Specific_DatasourceFrom_TestDataEngine.json-schema.json',
        '{"$schema":"https://json-schema.org/draft/2020-12/schema","title":"TestDataSet for a specific TestDataSource","type":"object","required":["TestDataSourceName","TestDataSourceUuid","Data"],"properties":{"DataSourceName":{"type":"string"},"DataSourceUuid":{"type":"string","format":"uuid"},"Data":{"type":"array","minItems":0,"maxItems":1,"items":{"type":"object","required":["TestDataId","AccountId","DirectClientCustodyAccountId","AccountCurrency","AccountEnvironment","ClientJurisdictionCountryCode","DebitOrCredit","c","MarketCountry","MarketName","MarketSubType","AccountType","PooledCashAccount","Nostro","MarketCurrency","ISIN","SecurityType","ProvisionalIncome"],"properties":{"TestDataId":{"type":"integer"},"AccountId":{"type":"integer"},"DirectClientCustodyAccountId":{"type":"integer"},"AccountCurrency":{"type":"string","minLength":3,"maxLength":3},"AccountEnvironment":{"type":"string"},"ClientJuristictionCountryCode":{"type":"string","minLength":2,"maxLength":2},"DebitOrCredit":{"type":"string"},"c":{"type":"string"},"MarketCountry":{"type":"string","minLength":2,"maxLength":2},"MarketName":{"type":"string"},"MarketSubType":{"type":"string"},"AccountType":{"type":"string"},"PooledCashAccount":{"type":"integer"},"Nostro":{"type":"integer"},"MarketCurrency":{"type":"string","minLength":3,"maxLength":3},"ISIN":{"type":"string","minLength":12,"maxLength":12},"SecurityType":{"type":"string"},"ProvisionalIncome":{"type":"string"}},"additionalProperties":false}}},"additionalProperties":false}',
        '2026-04-09T00:00:00Z');
