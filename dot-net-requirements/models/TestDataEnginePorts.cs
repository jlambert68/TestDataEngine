namespace TestDataEngine.Requirements.Models;

public interface IRuntimeFilterCompiler
{
    CompiledFilter Compile(FilterRequest request, DataSourceDefinition dataSource);
}

public interface ITypedSqlCompiler
{
    CompiledFilter CompileTyped(FilterRequest request);
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

public interface IJsonSchemaValidator
{
    void ValidateJsonFile(string schemaPath, string payloadJson);
}

public interface IImportService
{
    ImportResult ImportRawCsv(ImportOptions options);
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
