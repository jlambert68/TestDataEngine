-- Table: public.data_items
-- Purpose: Stores JSON payload rows grouped by datasource metadata.
create table public.data_items
(
    DataSourceUuid             uuid        not null,
    DataSourceName             varchar     not null,
    TestDataDomainUuid         uuid        not null,
    TestDataDomainName         varchar     not null,
    TestDataSourceTemplateName varchar     not null,
    DataUuid                   uuid        not null,
    DataUpdateTimeStamp        timestamptz not null,
    JsonDataUuid               uuid        not null
        primary key,
    JsonData                   jsonb       not null
);

create index idx_data_items_json_data_uuid
    on public.data_items (JsonDataUuid);

create index idx_data_items_source_name
    on public.data_items (DataSourceName);

create index idx_data_items_source_uuid
    on public.data_items (DataSourceUuid);

create index idx_data_items_updated
    on public.data_items (DataUpdateTimeStamp);

-- Table: public.testdataset_response_schemas
-- Purpose: Stores response JSON-schema metadata per test datasource.
create table public.testdataset_response_schemas
(
    TestDataSourceName varchar     not null,
    TestDataSourceUuid uuid        not null,
    JsonSchemaName     varchar     not null,
    JsonSchema         jsonb       not null,
    UpdatedDateTime    timestamptz not null
);

create unique index udx_testdataset_response_schemas_source_schema
    on public.testdataset_response_schemas (TestDataSourceName, TestDataSourceUuid, JsonSchemaName);
