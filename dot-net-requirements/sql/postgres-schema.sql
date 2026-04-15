CREATE TABLE public.data_items
(
    DataSourceUuid             uuid        NOT NULL,
    DataSourceName             varchar     NOT NULL,
    TestDataDomainUuid         uuid        NOT NULL,
    TestDataDomainName         varchar     NOT NULL,
    TestDataSourceTemplateName varchar     NOT NULL,
    DataUuid                   uuid        NOT NULL,
    DataUpdateTimeStamp        timestamptz NOT NULL,
    JsonDataUuid               uuid        NOT NULL PRIMARY KEY,
    JsonData                   jsonb       NOT NULL
);

CREATE TABLE public.testdataset_response_schemas
(
    TestDataSourceName varchar     NOT NULL,
    TestDataSourceUuid uuid        NOT NULL,
    JsonSchemaName     varchar     NOT NULL,
    JsonSchema         jsonb       NOT NULL,
    UpdatedDateTime    timestamptz NOT NULL
);
