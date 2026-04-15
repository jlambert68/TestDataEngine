CREATE TABLE main.data_items
(
    DataSourceUuid      TEXT NOT NULL,
    DataSourceName      TEXT NOT NULL,
    TestDataDomainUuid  TEXT NOT NULL,
    TestDataDomainName  TEXT NOT NULL,
    TestDataSourceTemplateName TEXT NOT NULL,
    DataUuid            TEXT NOT NULL,
    DataUpdateTimeStamp TEXT NOT NULL,
    JsonDataUuid        TEXT NOT NULL PRIMARY KEY,
    JsonData            TEXT NOT NULL,
    CHECK (json_valid(JsonData))
);

CREATE TABLE main.testdataset_response_schemas
(
    TestDataSourceName TEXT NOT NULL,
    TestDataSourceUuid TEXT NOT NULL,
    JsonSchemaName     TEXT NOT NULL,
    JsonSchema         TEXT NOT NULL,
    UpdatedDateTime    TEXT NOT NULL
);
