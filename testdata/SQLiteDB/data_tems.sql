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
