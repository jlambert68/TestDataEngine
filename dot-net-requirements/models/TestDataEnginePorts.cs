using System.Collections.Generic;

namespace TestDataEngine.Requirements.Models;

public interface IRuntimeFilterCompiler
{
    CompiledFilter Compile(FilterRequest request, DataSourceDefinition dataSource);
}

public interface ITypedSqlCompiler
{
    CompiledFilter CompileTyped(TypedSqlRequest request, TypedSqlCompilerOptions? options = null);
}

public interface IAllowedFieldsService
{
    AllowedFieldResponse Build(FilterRequest request, DataSourceDefinition dataSource);
}

public interface ICsvQueryService
{
    QueryResult Query(FilterRequest request, string csvPath, int maxItems, string? randomSeedGuid);
}

public interface ISqliteQueryService
{
    QueryResult Query(FilterRequest request, string dbPath, string tableName, int maxItems, string? randomSeedGuid);
}

public interface IPostgresQueryService
{
    QueryResult Query(FilterRequest request, string dsn, string dataTable, string schemaTable, int maxItems, string? randomSeedGuid);
}

public interface IJsonSchemaValidator
{
    void ValidateJsonFile(string schemaPath, string payloadJson);
}

public interface IImportService
{
    ImportResult ImportRawCsv(ImportOptions options);
}

public interface IDataSourceCatalog
{
    IReadOnlyList<DataSourceConfig> List();
    DataSourceConfig? Get(string id);
}

public interface IRequestFactory
{
    FilterRequest BuildMetadataRequest(DataSourceConfig dataSource);
}

public interface IFacetService
{
    (IReadOnlyList<FacetValue> Values, bool Truncated) Values(
        SourceType source,
        DataSourceConfig dataSource,
        FilterRequest metadataRequest,
        string field,
        int limit,
        string? queryText
    );
}

public interface IWebQueryService
{
    AllowedFieldResponse Describe(SourceType source, DataSourceConfig dataSource, FilterRequest metadataRequest);
    QueryPreviewResponse Preview(DataSourceConfig dataSource, QueryPreviewRequest request);
}

public interface IWebApiServer
{
    // Should expose:
    // GET  /api/v1/datasources
    // GET  /api/v1/datasources/{id}/fields
    // GET  /api/v1/datasources/{id}/facets
    // POST /api/v1/query/preview
    // GET  /api/v1/healthz
    // and serve ui/dist with SPA fallback.
    object BuildRoutes();
}

public interface ILoggerWithId
{
    void Info(string id, string messageTemplate, params object?[] args);
    void Error(string id, string messageTemplate, params object?[] args);
    void Fatal(string id, string messageTemplate, params object?[] args);
}

public sealed record QueryResult
{
    public required CompiledFilter CompiledFilter { get; init; }
    public required AllowedFieldResponse AllowedFieldResponse { get; init; }
    public required DataSetResponse DataSetResponse { get; init; }
}
