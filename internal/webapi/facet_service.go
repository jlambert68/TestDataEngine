package webapi

import (
	"fmt"

	"TestDataEngine/internal/filters"
)

type FacetService interface {
	Values(source SourceType, cfg DataSourceConfig, req filters.FilterRequest, field string, limit int, q string) ([]FacetValue, bool, error)
}

type facetService struct{}

func NewFacetService() FacetService {
	return &facetService{}
}

func (s *facetService) Values(source SourceType, cfg DataSourceConfig, req filters.FilterRequest, field string, limit int, q string) ([]FacetValue, bool, error) {
	var (
		items     []filters.FacetValueCount
		truncated bool
		err       error
	)

	switch source {
	case SourceCSV:
		items, truncated, err = filters.FacetCSVDataSource(req, cfg.CSVPath, field, limit, q)
	case SourceSQLite:
		items, truncated, err = filters.FacetSQLiteDataSource(req, cfg.SQLiteDB, cfg.SQLiteTable, field, limit, q)
	case SourcePostgres:
		items, truncated, err = filters.FacetPostgresDataSource(req, cfg.PostgresDSN, cfg.PostgresTable, cfg.PostgresSchema, field, limit, q)
	default:
		return nil, false, errUnsupportedSource(source)
	}
	if err != nil {
		return nil, false, err
	}

	out := make([]FacetValue, 0, len(items))
	for _, item := range items {
		out = append(out, FacetValue{
			Value:  item.Value,
			Label:  facetLabel(item.Value),
			Count:  item.Count,
			IsNull: item.IsNull,
		})
	}
	return out, truncated, nil
}

func facetLabel(v interface{}) string {
	if v == nil {
		return "(blank)"
	}
	return fmt.Sprintf("%v", v)
}
