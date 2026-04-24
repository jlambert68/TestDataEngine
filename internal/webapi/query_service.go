package webapi

import "TestDataEngine/internal/filters"

type QueryService interface {
	Describe(source SourceType, cfg DataSourceConfig, req filters.FilterRequest) (filters.AllowedFieldResponse, error)
	Preview(cfg DataSourceConfig, in QueryPreviewRequest) (QueryPreviewResponse, error)
}

type queryService struct{}

func NewQueryService() QueryService {
	return &queryService{}
}

func (s *queryService) Describe(source SourceType, cfg DataSourceConfig, req filters.FilterRequest) (filters.AllowedFieldResponse, error) {
	switch source {
	case SourceCSV:
		return filters.DescribeCSVDataSource(req, cfg.CSVPath)
	case SourceSQLite:
		return filters.DescribeSQLiteDataSource(req, cfg.SQLiteDB, cfg.SQLiteTable)
	case SourcePostgres:
		return filters.DescribePostgresDataSource(req, cfg.PostgresDSN, cfg.PostgresTable, cfg.PostgresSchema)
	default:
		return filters.AllowedFieldResponse{}, errUnsupportedSource(source)
	}
}

func (s *queryService) Preview(cfg DataSourceConfig, in QueryPreviewRequest) (QueryPreviewResponse, error) {
	var (
		compiled filters.CompiledFilter
		allowed  filters.AllowedFieldResponse
		dataSet  filters.DataSetResponse
		err      error
	)

	switch in.Source {
	case SourceCSV:
		compiled, allowed, dataSet, err = filters.QueryCSVDataSourceWithSeed(in.Request, cfg.CSVPath, in.MaxItems, in.RandomSeedGUID)
	case SourceSQLite:
		compiled, allowed, dataSet, err = filters.QuerySQLiteDataSourceWithSeed(in.Request, cfg.SQLiteDB, cfg.SQLiteTable, in.MaxItems, in.RandomSeedGUID)
	case SourcePostgres:
		compiled, allowed, dataSet, err = filters.QueryPostgresDataSourceWithSeed(in.Request, cfg.PostgresDSN, cfg.PostgresTable, cfg.PostgresSchema, in.MaxItems, in.RandomSeedGUID)
	default:
		return QueryPreviewResponse{}, errUnsupportedSource(in.Source)
	}
	if err != nil {
		return QueryPreviewResponse{}, err
	}

	return QueryPreviewResponse{
		Source:           in.Source,
		CompiledWhereSQL: compiled.WhereSQL,
		CompiledArgs:     compiled.Args,
		AllowedFields:    allowed,
		DataSet:          dataSet,
	}, nil
}
