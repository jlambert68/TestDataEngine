package webapi

import "sort"

type SourceType string

const (
	SourceCSV      SourceType = "csv"
	SourceSQLite   SourceType = "sqlite"
	SourcePostgres SourceType = "postgres"
)

type DataSourceConfig struct {
	ID               string       `json:"id"`
	Label            string       `json:"label"`
	DataSourceName   string       `json:"dataSourceName"`
	DataSourceUUID   string       `json:"dataSourceUuid"`
	SupportedSources []SourceType `json:"supportedSources"`
	DefaultSource    SourceType   `json:"defaultSource"`
	CSVPath          string       `json:"-"`
	SQLiteDB         string       `json:"-"`
	SQLiteTable      string       `json:"-"`
	PostgresDSN      string       `json:"-"`
	PostgresTable    string       `json:"-"`
	PostgresSchema   string       `json:"-"`
}

type Catalog interface {
	List() []DataSourceConfig
	Get(id string) (DataSourceConfig, bool)
}

type staticCatalog map[string]DataSourceConfig

func StaticCatalog() Catalog {
	return staticCatalog{
		"subcustody": {
			ID:               "subcustody",
			Label:            "SubCustody",
			DataSourceName:   "SubCustody",
			DataSourceUUID:   "110cc994-a913-4041-96fe-a96d7e0c97e8",
			SupportedSources: []SourceType{SourceCSV, SourceSQLite},
			DefaultSource:    SourceSQLite,
			CSVPath:          "testdata/pi26_2/FenixRawTestdata_646rows_211220_stripped.csv",
			SQLiteDB:         "testdata/SQLiteDB/identifier.sqlite",
			SQLiteTable:      "main.data_items",
			PostgresTable:    "public.data_items",
			PostgresSchema:   "public.testdataset_response_schemas",
		},
	}
}

func (c staticCatalog) List() []DataSourceConfig {
	out := make([]DataSourceConfig, 0, len(c))
	for _, item := range c {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func (c staticCatalog) Get(id string) (DataSourceConfig, bool) {
	item, ok := c[id]
	return item, ok
}
