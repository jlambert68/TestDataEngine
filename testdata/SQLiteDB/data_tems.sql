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
        'TestDataSet_Response_For_Specific_Datasource_From_TestDataEngine.json-schema.json',
        '{"$schema":"https://json-schema.org/draft/2020-12/schema","title":"TestDataSet response for a specific datasource","type":"object","additionalProperties":false,"required":["SchemaVersion","TestDataSourceName","TestDataSourceUuid","JsonSchemaName","TestData"],"properties":{"SchemaVersion":{"type":"string","title":"Specific schema version","description":"Version of the specific response schema.","enum":["1.0"]},"TestDataSourceName":{"type":"string"},"TestDataSourceUuid":{"type":"string","format":"uuid"},"JsonSchemaName":{"type":"string"},"TestData":{"$ref":"#/$defs/TestData"}},"$defs":{"TestData":{"type":"object","additionalProperties":false,"required":["SpecificSourceSchemaVersion","TestDataSet"],"properties":{"SpecificSourceSchemaVersion":{"type":"string","title":"Specific source schema version","description":"Version of the datasource-specific payload schema.","enum":["1.0"]},"TestDataSet":{"type":"array","items":{"$ref":"#/$defs/TestDataSetItem"}}}},"TestDataSetItem":{"type":"object","additionalProperties":false,"required":["TestDataId","AccountId","DirectClientCustodyAccountId","AccountCurrency","AccountEnvironment","ClientJurisdictionCountryCode","DebitOrCredit","c","MarketCountry","MarketName","MarketSubType","AccountType","PooledCashAccount","Nostro","MarketCurrency","ISIN","SecurityType","ProvisionalIncome","ContraCurrency","InterimCurrency","PrincipalOrIncome","Random","SecProgram"],"properties":{"TestDataId":{"type":"integer"},"AccountId":{"type":"integer"},"DirectClientCustodyAccountId":{"type":"integer"},"AccountCurrency":{"type":"string","minLength":3,"maxLength":3,"pattern":"^[A-Z]{3}$"},"AccountEnvironment":{"type":"string"},"ClientJurisdictionCountryCode":{"type":"string","minLength":2,"maxLength":2,"pattern":"^[A-Z]{2}$"},"DebitOrCredit":{"type":"string"},"c":{"type":"string"},"MarketCountry":{"type":"string","minLength":2,"maxLength":2,"pattern":"^[A-Z]{2}$"},"MarketName":{"type":"string"},"MarketSubType":{"type":"string"},"AccountType":{"type":"string"},"PooledCashAccount":{"type":"integer"},"Nostro":{"type":"integer"},"MarketCurrency":{"type":"string","minLength":3,"maxLength":3,"pattern":"^[A-Z]{3}$"},"ISIN":{"type":"string","minLength":12,"maxLength":12},"SecurityType":{"type":"string"},"ProvisionalIncome":{"type":"string"},"ContraCurrency":{"oneOf":[{"type":"null"},{"type":"string","minLength":3,"maxLength":3,"pattern":"^[A-Z]{3}$"}]},"InterimCurrency":{"oneOf":[{"type":"null"},{"type":"string","minLength":3,"maxLength":3,"pattern":"^[A-Z]{3}$"}]},"PrincipalOrIncome":{"oneOf":[{"type":"null"},{"type":"string"}]},"Random":{"oneOf":[{"type":"null"},{"type":"string"}]},"SecProgram":{"oneOf":[{"type":"null"},{"type":"string"}]}}}}}',
        '2026-04-09T00:00:00Z');
