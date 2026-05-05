using System.Buffers.Binary;
using System.Globalization;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using System.Text.RegularExpressions;
using Microsoft.Data.Sqlite;
using Microsoft.Extensions.FileProviders;

var repoRoot = RepoPaths.FindRepositoryRoot();
var staticDir = Path.Combine(repoRoot, "ui", "dist");
var schemaPath = Path.Combine(repoRoot, "internal", "json", Constants.SpecificResponseSchemaName);

var builder = WebApplication.CreateBuilder(args);
builder.Services.AddSingleton(new RepoPaths(repoRoot, staticDir, schemaPath));
builder.Services.AddSingleton<SchemaCatalog>();
builder.Services.AddSingleton<DataSourceCatalog>();
builder.Services.AddSingleton<QueryEngine>();
builder.Services.AddSingleton<FacetEngine>();

var app = builder.Build();

app.MapGet("/api/v1/healthz", IResult () => Results.Json(new Dictionary<string, string> { ["status"] = "ok" }, JsonOptions.Web));

app.MapGet("/api/v1/datasources", IResult (DataSourceCatalog catalog) =>
{
    var items = catalog.List().Select(x => x.ToListItem()).ToArray();
    return Results.Json(new ListDataSourcesResponse(items), JsonOptions.Web);
});

app.MapGet("/api/v1/datasources/{id}/fields", IResult (string id, HttpRequest request, DataSourceCatalog catalog, QueryEngine engine, SchemaCatalog schemaCatalog) =>
{
    if (!catalog.TryGet(id, out var cfg))
    {
        return JsonError.NotFound("unknown datasource", id);
    }

    var source = SourceType.ParseOrEmpty((string?)request.Query["source"]);
    if (source == "")
    {
        source = cfg.DefaultSource;
    }
    if (!cfg.SupportedSources.Contains(source, StringComparer.Ordinal))
    {
        return JsonError.BadRequest("unsupported source", source);
    }

    try
    {
        var metadataRequest = RequestFactory.BuildMetadataRequest(cfg, schemaCatalog);
        var allowed = engine.Describe(source, cfg, metadataRequest);
        var fields = allowed.AllowedFields.Select(item => new FieldDescriptor(
            item.FieldName,
            item.FieldType,
            item.Nullable,
            item.SupportedOperators,
            WidgetForField(item.FieldType),
            FacetEligible(item.FieldType),
            string.IsNullOrWhiteSpace(item.Description) ? null : item.Description
        )).ToArray();
        return Results.Json(new GetFieldsResponse(cfg.Id, source, fields), JsonOptions.Web);
    }
    catch (MetadataRequestException ex)
    {
        return JsonError.Internal("failed to build metadata request", ex.Message);
    }
    catch (Exception ex)
    {
        return JsonError.BadRequest("failed to describe datasource", ex.Message);
    }
});

app.MapGet("/api/v1/datasources/{id}/facets", IResult (string id, HttpRequest request, DataSourceCatalog catalog, FacetEngine facets, SchemaCatalog schemaCatalog) =>
{
    if (!catalog.TryGet(id, out var cfg))
    {
        return JsonError.NotFound("unknown datasource", id);
    }

    var source = SourceType.ParseOrEmpty((string?)request.Query["source"]);
    if (source == "")
    {
        source = cfg.DefaultSource;
    }
    if (!cfg.SupportedSources.Contains(source, StringComparer.Ordinal))
    {
        return JsonError.BadRequest("unsupported source", source);
    }

    var field = ((string?)request.Query["field"] ?? "").Trim();
    if (field == "")
    {
        return JsonError.BadRequest("missing field", "query parameter field is required");
    }

    var limit = 100;
    var rawLimit = ((string?)request.Query["limit"] ?? "").Trim();
    if (rawLimit != "" && !int.TryParse(rawLimit, NumberStyles.Integer, CultureInfo.InvariantCulture, out limit))
    {
        return JsonError.BadRequest("invalid limit", $"input string was not in a correct format: {rawLimit}");
    }

    try
    {
        var metadataRequest = RequestFactory.BuildMetadataRequest(cfg, schemaCatalog);
        var result = facets.Values(source, cfg, metadataRequest, field, limit, (string?)request.Query["q"] ?? "");
        return Results.Json(new GetFacetsResponse(cfg.Id, source, field, result.Values, result.Truncated), JsonOptions.Web);
    }
    catch (MetadataRequestException ex)
    {
        return JsonError.Internal("failed to build metadata request", ex.Message);
    }
    catch (Exception ex)
    {
        return JsonError.BadRequest("failed to load facets", ex.Message);
    }
});

app.MapPost("/api/v1/query/preview", async Task<IResult> (HttpRequest http, DataSourceCatalog catalog, QueryEngine engine) =>
{
    QueryPreviewRequest? request;
    try
    {
        request = await JsonSerializer.DeserializeAsync<QueryPreviewRequest>(http.Body, JsonOptions.WebStrict);
    }
    catch (Exception ex)
    {
        return JsonError.BadRequest("invalid request body", ex.Message);
    }

    if (request is null)
    {
        return JsonError.BadRequest("invalid request body", "request body is required");
    }
    if (string.IsNullOrWhiteSpace(request.Source))
    {
        return JsonError.BadRequest("missing source", "source is required");
    }
    if (!catalog.TryFindByRequest(request.Request, out var cfg))
    {
        return JsonError.BadRequest("unknown datasource", "request datasource does not match the server catalog");
    }
    if (!cfg.SupportedSources.Contains(request.Source, StringComparer.Ordinal))
    {
        return JsonError.BadRequest("unsupported source", request.Source);
    }

    try
    {
        return Results.Json(engine.Preview(cfg, request), JsonOptions.Web);
    }
    catch (Exception ex)
    {
        return JsonError.BadRequest("preview failed", ex.Message);
    }
});

if (Directory.Exists(staticDir))
{
    app.UseStaticFiles(new StaticFileOptions
    {
        FileProvider = new PhysicalFileProvider(staticDir)
    });
}

app.MapFallback(async context =>
{
    if (context.Request.Path.StartsWithSegments("/api"))
    {
        context.Response.StatusCode = StatusCodes.Status404NotFound;
        return;
    }

    var indexPath = Path.Combine(staticDir, "index.html");
    if (!File.Exists(indexPath))
    {
        context.Response.StatusCode = StatusCodes.Status503ServiceUnavailable;
        context.Response.ContentType = "application/json";
        await JsonSerializer.SerializeAsync(context.Response.Body, new ApiError("ui build missing", "ui/dist/index.html was not found"), JsonOptions.Web);
        return;
    }

    context.Response.ContentType = "text/html; charset=utf-8";
    await context.Response.SendFileAsync(indexPath);
});

app.Run(NormalizeListenUrl(Environment.GetEnvironmentVariable("HTTP_ADDR") ?? ":8080"));

static string NormalizeListenUrl(string addr)
{
    addr = string.IsNullOrWhiteSpace(addr) ? ":8080" : addr.Trim();
    if (addr.Contains("://", StringComparison.Ordinal))
    {
        return addr;
    }
    if (addr.StartsWith(':'))
    {
        return $"http://0.0.0.0{addr}";
    }
    return $"http://{addr}";
}

static string WidgetForField(string fieldType) => fieldType switch
{
    "boolean" => "boolean-toggle",
    "number" or "integer" => "searchable-checkbox-group",
    _ => "searchable-checkbox-group"
};

static bool FacetEligible(string fieldType) => fieldType is "string" or "boolean" or "number" or "integer";

static class Constants
{
    public const string SpecificResponseSchemaName = "TestDataSet_Response_For_Specific_Datasource_From_TestDataEngine.json-schema.json";
}

static class JsonOptions
{
    public static readonly JsonSerializerOptions Web = new(JsonSerializerDefaults.Web)
    {
        DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
    };

    public static readonly JsonSerializerOptions WebStrict = new(JsonSerializerDefaults.Web)
    {
        DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull,
        UnmappedMemberHandling = JsonUnmappedMemberHandling.Disallow
    };
}

static class SourceType
{
    public const string Csv = "csv";
    public const string Sqlite = "sqlite";
    public const string Postgres = "postgres";

    public static string ParseOrEmpty(string? raw)
    {
        return raw switch
        {
            Csv => Csv,
            Sqlite => Sqlite,
            Postgres => Postgres,
            _ => ""
        };
    }
}

sealed record ApiError(
    [property: JsonPropertyName("error")] string Error,
    [property: JsonPropertyName("details")] string? Details = null
);

static class JsonError
{
    public static IResult BadRequest(string error, string details) => Results.Json(new ApiError(error, details), JsonOptions.Web, statusCode: StatusCodes.Status400BadRequest);
    public static IResult NotFound(string error, string details) => Results.Json(new ApiError(error, details), JsonOptions.Web, statusCode: StatusCodes.Status404NotFound);
    public static IResult Internal(string error, string details) => Results.Json(new ApiError(error, details), JsonOptions.Web, statusCode: StatusCodes.Status500InternalServerError);
}

sealed record DataSourceListItem(
    [property: JsonPropertyName("id")] string Id,
    [property: JsonPropertyName("label")] string Label,
    [property: JsonPropertyName("dataSourceName")] string DataSourceName,
    [property: JsonPropertyName("dataSourceUuid")] string DataSourceUuid,
    [property: JsonPropertyName("supportedSources")] IReadOnlyList<string> SupportedSources,
    [property: JsonPropertyName("defaultSource")] string DefaultSource
);

sealed record DataSourceConfig(
    string Id,
    string Label,
    string DataSourceName,
    string DataSourceUuid,
    IReadOnlyList<string> SupportedSources,
    string DefaultSource,
    string CsvPath,
    string SQLiteDb,
    string SQLiteTable,
    string PostgresDsn,
    string PostgresTable,
    string PostgresSchemaTable
)
{
    public DataSourceListItem ToListItem() => new(Id, Label, DataSourceName, DataSourceUuid, SupportedSources, DefaultSource);
}

sealed record ListDataSourcesResponse([property: JsonPropertyName("items")] IReadOnlyList<DataSourceListItem> Items);

sealed record FilterRequest(
    [property: JsonPropertyName("SchemaVersion")] string SchemaVersion,
    [property: JsonPropertyName("RequestUuid")] string RequestUuid,
    [property: JsonPropertyName("DataSourceUuid")] string DataSourceUuid,
    [property: JsonPropertyName("DataSourceName")] string DataSourceName,
    [property: JsonPropertyName("RequestFilter")] JsonElement RequestFilter
);

sealed record QueryPreviewRequest(
    [property: JsonPropertyName("source")] string Source,
    [property: JsonPropertyName("maxItems")] int MaxItems,
    [property: JsonPropertyName("randomSeedGuid")] string? RandomSeedGuid,
    [property: JsonPropertyName("request")] FilterRequest Request
);

sealed record QueryPreviewResponse(
    [property: JsonPropertyName("source")] string Source,
    [property: JsonPropertyName("compiledWhereSql")] string CompiledWhereSql,
    [property: JsonPropertyName("compiledArgs")] IReadOnlyList<object?> CompiledArgs,
    [property: JsonPropertyName("allowedFields")] AllowedFieldResponse AllowedFields,
    [property: JsonPropertyName("dataSet")] DataSetResponse DataSet
);

sealed record FieldDescriptor(
    [property: JsonPropertyName("field")] string Field,
    [property: JsonPropertyName("fieldType")] string FieldType,
    [property: JsonPropertyName("nullable")] bool Nullable,
    [property: JsonPropertyName("supportedOperators")] IReadOnlyList<string> SupportedOperators,
    [property: JsonPropertyName("widget")] string Widget,
    [property: JsonPropertyName("facetEligible")] bool FacetEligible,
    [property: JsonPropertyName("description")] string? Description
);

sealed record GetFieldsResponse(
    [property: JsonPropertyName("datasourceId")] string DatasourceId,
    [property: JsonPropertyName("source")] string Source,
    [property: JsonPropertyName("fields")] IReadOnlyList<FieldDescriptor> Fields
);

sealed record FacetValue(
    [property: JsonPropertyName("value")] object? Value,
    [property: JsonPropertyName("label")] string Label,
    [property: JsonPropertyName("count")] int Count,
    [property: JsonPropertyName("isNull")] bool IsNull
);

sealed record GetFacetsResponse(
    [property: JsonPropertyName("datasourceId")] string DatasourceId,
    [property: JsonPropertyName("source")] string Source,
    [property: JsonPropertyName("field")] string Field,
    [property: JsonPropertyName("values")] IReadOnlyList<FacetValue> Values,
    [property: JsonPropertyName("truncated")] bool Truncated
);

sealed record AllowedFieldResult(
    [property: JsonPropertyName("FieldName")] string FieldName,
    [property: JsonPropertyName("FieldType")] string FieldType,
    [property: JsonPropertyName("Nullable")] bool Nullable,
    [property: JsonPropertyName("SupportedOperators")] IReadOnlyList<string> SupportedOperators,
    [property: JsonPropertyName("Description")] string? Description
);

sealed record AllowedFieldResponse(
    [property: JsonPropertyName("SchemaVersion")] string SchemaVersion,
    [property: JsonPropertyName("RequestUuid")] string RequestUuid,
    [property: JsonPropertyName("DataSourceUuid")] string DataSourceUuid,
    [property: JsonPropertyName("DataSourceName")] string DataSourceName,
    [property: JsonPropertyName("AllowedFields")] IReadOnlyList<AllowedFieldResult> AllowedFields
);

sealed record SpecificDatasourceTestData(
    [property: JsonPropertyName("SpecificSourceSchemaVersion")] string SpecificSourceSchemaVersion,
    [property: JsonPropertyName("TestDataSet")] IReadOnlyList<Dictionary<string, object?>> TestDataSet
);

sealed record DataSetResponse(
    [property: JsonPropertyName("SchemaVersion"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] string? SchemaVersion,
    [property: JsonPropertyName("TestDataSourceName"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] string? TestDataSourceName,
    [property: JsonPropertyName("TestDataSourceUuid"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] string? TestDataSourceUuid,
    [property: JsonPropertyName("JsonSchemaName"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] string? JsonSchemaName,
    [property: JsonPropertyName("TestData"), JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)] SpecificDatasourceTestData? TestData
);

sealed record CompiledFilter(string WhereSql, IReadOnlyList<object?> Args);
sealed record QueryResult(CompiledFilter Compiled, AllowedFieldResponse Allowed, DataSetResponse DataSet, IReadOnlyList<Dictionary<string, object?>> Rows);

sealed record FieldDefinition(string FieldType, bool Nullable, IReadOnlyList<string> SupportedOperators, string? Description);
sealed record DataSourceDefinition(string Uuid, IReadOnlyDictionary<string, FieldDefinition> Fields);
sealed record DataSetSchemaMetadata(string TestDataSourceName, string TestDataSourceUuid, string JsonSchemaName, string UpdatedDateTime);
sealed record LoadedDataSource(DataSourceDefinition Definition, IReadOnlyList<Dictionary<string, object?>> Rows, DataSetSchemaMetadata? Metadata);

sealed class RepoPaths(string repoRoot, string staticDir, string schemaPath)
{
    public string RepoRoot { get; } = repoRoot;
    public string StaticDir { get; } = staticDir;
    public string SchemaPath { get; } = schemaPath;

    public string ResolveFromRepo(string relativePath) => Path.GetFullPath(Path.Combine(RepoRoot, relativePath));

    public static string FindRepositoryRoot()
    {
        var candidates = new[]
        {
            Directory.GetCurrentDirectory(),
            AppContext.BaseDirectory
        };

        foreach (var candidate in candidates)
        {
            var dir = new DirectoryInfo(candidate);
            while (dir is not null)
            {
                if (File.Exists(Path.Combine(dir.FullName, "internal", "json", Constants.SpecificResponseSchemaName)) &&
                    Directory.Exists(Path.Combine(dir.FullName, "ui")))
                {
                    return dir.FullName;
                }
                dir = dir.Parent;
            }
        }

        throw new InvalidOperationException("Could not locate repository root containing internal/json and ui.");
    }
}

sealed class DataSourceCatalog(RepoPaths paths)
{
    private readonly IReadOnlyList<DataSourceConfig> _items =
    [
        new(
            "subcustody",
            "SubCustody",
            "SubCustody",
            "110cc994-a913-4041-96fe-a96d7e0c97e8",
            [SourceType.Csv, SourceType.Sqlite],
            SourceType.Sqlite,
            paths.ResolveFromRepo("testdata/pi26_2/FenixRawTestdata_646rows_211220_stripped.csv"),
            paths.ResolveFromRepo("testdata/SQLiteDB/identifier.sqlite"),
            "main.data_items",
            "",
            "public.data_items",
            "public.testdataset_response_schemas"
        )
    ];

    public IReadOnlyList<DataSourceConfig> List() => _items.OrderBy(x => x.Id, StringComparer.Ordinal).ToArray();

    public bool TryGet(string id, out DataSourceConfig cfg)
    {
        cfg = _items.FirstOrDefault(x => x.Id == id)!;
        return cfg is not null;
    }

    public bool TryFindByRequest(FilterRequest request, out DataSourceConfig cfg)
    {
        cfg = _items.FirstOrDefault(x =>
            string.Equals(x.DataSourceName, request.DataSourceName, StringComparison.OrdinalIgnoreCase) &&
            string.Equals(x.DataSourceUuid, request.DataSourceUuid, StringComparison.OrdinalIgnoreCase))!;
        return cfg is not null;
    }
}

sealed class MetadataRequestException(string message, Exception? inner = null) : Exception(message, inner);

static class RequestFactory
{
    public static FilterRequest BuildMetadataRequest(DataSourceConfig cfg, SchemaCatalog schemaCatalog)
    {
        try
        {
            var probeField = schemaCatalog.Order.FirstOrDefault();
            if (string.IsNullOrWhiteSpace(probeField))
            {
                throw new InvalidOperationException("schema catalog did not define any fields");
            }

            using var doc = JsonDocument.Parse($$"""
            {"field":"{{probeField}}","op":"exists","value":true}
            """);
            return new FilterRequest(
                "1.0",
                "11111111-1111-4111-8111-111111111111",
                cfg.DataSourceUuid,
                cfg.DataSourceName,
                doc.RootElement.Clone()
            );
        }
        catch (Exception ex)
        {
            throw new MetadataRequestException("failed to build metadata request", ex);
        }
    }
}

sealed class SchemaCatalog
{
    private readonly Lazy<SchemaFieldCatalog> _catalog;

    public SchemaCatalog(RepoPaths paths)
    {
        _catalog = new Lazy<SchemaFieldCatalog>(() => Load(paths.SchemaPath));
    }

    public SchemaFieldCatalog Current => _catalog.Value;

    private static SchemaFieldCatalog Load(string schemaPath)
    {
        using var doc = JsonDocument.Parse(File.ReadAllText(schemaPath));
        var itemDef = doc.RootElement.GetProperty("$defs").GetProperty("TestDataSetItem");
        var properties = itemDef.GetProperty("properties");

        var required = itemDef.TryGetProperty("required", out var requiredElement) && requiredElement.ValueKind == JsonValueKind.Array
            ? requiredElement.EnumerateArray().Select(x => x.GetString()).Where(x => !string.IsNullOrWhiteSpace(x)).Cast<string>().ToArray()
            : [];
        var requiredSet = required.ToHashSet(StringComparer.Ordinal);

        var fields = new Dictionary<string, SchemaField>(StringComparer.Ordinal);
        var extraOrder = new List<string>();

        foreach (var property in properties.EnumerateObject())
        {
            var (fieldType, nullable) = ParseSchemaType(property.Value);
            var description = property.Value.TryGetProperty("description", out var desc) ? desc.GetString() : null;
            fields[property.Name] = new SchemaField(property.Name, fieldType, nullable, SupportedOperatorsForType(fieldType), description);
            if (!requiredSet.Contains(property.Name))
            {
                extraOrder.Add(property.Name);
            }
        }

        extraOrder.Sort(StringComparer.Ordinal);
        return new SchemaFieldCatalog(fields, [.. required, .. extraOrder]);
    }

    private static (string FieldType, bool Nullable) ParseSchemaType(JsonElement property)
    {
        if (property.TryGetProperty("type", out var typeElement) && typeElement.ValueKind == JsonValueKind.String)
        {
            return (typeElement.GetString() ?? "string", false);
        }

        var nullable = false;
        var fieldType = "";
        if (property.TryGetProperty("oneOf", out var oneOf) && oneOf.ValueKind == JsonValueKind.Array)
        {
            foreach (var item in oneOf.EnumerateArray())
            {
                if (!item.TryGetProperty("type", out var t))
                {
                    continue;
                }
                var value = t.GetString();
                if (value == "null")
                {
                    nullable = true;
                }
                else if (!string.IsNullOrWhiteSpace(value))
                {
                    fieldType = value;
                }
            }
        }

        return (fieldType == "" ? "string" : fieldType, nullable);
    }

    public string CanonicalFieldName(string name) => Current.CanonicalFieldName(name);

    public DataSourceDefinition DefinitionFor(string dataSourceUuid) => new(
        dataSourceUuid,
        Current.Fields.ToDictionary(
            x => x.Key,
            x => new FieldDefinition(x.Value.FieldType, x.Value.Nullable, x.Value.SupportedOperators, x.Value.Description),
            StringComparer.Ordinal
        )
    );

    public IReadOnlyList<string> Order => Current.Order;

    public static IReadOnlyList<string> SupportedOperatorsForType(string fieldType) => fieldType switch
    {
        "number" or "integer" or "date" or "datetime" => ["eq", "neq", "gt", "gte", "lt", "lte", "in", "nin", "exists", "isNull"],
        "boolean" => ["eq", "neq", "exists", "isNull"],
        _ => ["eq", "neq", "in", "nin", "contains", "startsWith", "endsWith", "exists", "isNull"]
    };
}

sealed record SchemaField(string CanonicalName, string FieldType, bool Nullable, IReadOnlyList<string> SupportedOperators, string? Description);

sealed record SchemaFieldCatalog(IReadOnlyDictionary<string, SchemaField> Fields, IReadOnlyList<string> Order)
{
    public string CanonicalFieldName(string name)
    {
        if (Fields.ContainsKey(name))
        {
            return name;
        }

        var normalized = NormalizeLookup(name);
        if (normalized == "")
        {
            return name;
        }

        string? exact = null;
        foreach (var field in Fields.Keys)
        {
            if (NormalizeLookup(field) != normalized)
            {
                continue;
            }
            if (exact is not null)
            {
                return name;
            }
            exact = field;
        }
        if (exact is not null)
        {
            return exact;
        }

        string? bestName = null;
        var bestDistance = int.MaxValue;
        var secondBest = int.MaxValue;
        foreach (var field in Fields.Keys)
        {
            var distance = Levenshtein(normalized, NormalizeLookup(field));
            if (distance < bestDistance)
            {
                secondBest = bestDistance;
                bestDistance = distance;
                bestName = field;
            }
            else if (distance < secondBest)
            {
                secondBest = distance;
            }
        }

        return bestName is not null && bestDistance <= 2 && bestDistance < secondBest ? bestName : name;
    }

    private static string NormalizeLookup(string value)
    {
        var b = new StringBuilder(value.Length);
        foreach (var ch in value)
        {
            if ((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9'))
            {
                b.Append(ch);
            }
            else if (ch >= 'A' && ch <= 'Z')
            {
                b.Append((char)(ch + ('a' - 'A')));
            }
        }
        return b.ToString();
    }

    private static int Levenshtein(string left, string right)
    {
        if (left == right) return 0;
        if (left.Length == 0) return right.Length;
        if (right.Length == 0) return left.Length;

        var prev = new int[right.Length + 1];
        var curr = new int[right.Length + 1];
        for (var j = 0; j <= right.Length; j++) prev[j] = j;

        for (var i = 1; i <= left.Length; i++)
        {
            curr[0] = i;
            for (var j = 1; j <= right.Length; j++)
            {
                var cost = left[i - 1] == right[j - 1] ? 0 : 1;
                curr[j] = Math.Min(Math.Min(curr[j - 1] + 1, prev[j] + 1), prev[j - 1] + cost);
            }
            (prev, curr) = (curr, prev);
        }
        return prev[right.Length];
    }
}

sealed class QueryEngine(SchemaCatalog schema)
{
    private static readonly Regex UuidRegex = new(@"^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$", RegexOptions.Compiled);

    public AllowedFieldResponse Describe(string source, DataSourceConfig cfg, FilterRequest request)
    {
        ValidateMetadataRequest(request);
        var loaded = Load(source, cfg, request);
        return AllowedFields(request, loaded.Definition);
    }

    public QueryPreviewResponse Preview(DataSourceConfig cfg, QueryPreviewRequest request)
    {
        var result = Query(request.Source, cfg, request.Request, request.MaxItems, request.RandomSeedGuid ?? "");
        return new QueryPreviewResponse(request.Source, result.Compiled.WhereSql, result.Compiled.Args, result.Allowed, result.DataSet);
    }

    public QueryResult Query(string source, DataSourceConfig cfg, FilterRequest request, int maxItems, string randomSeedGuid)
    {
        ValidateRuntimeRequest(request);
        if (maxItems < 0)
        {
            maxItems = 0;
        }

        var loaded = Load(source, cfg, request);
        var compiled = CompileExpression(request.RequestFilter, loaded.Definition.Fields);
        var allowed = AllowedFields(request, loaded.Definition);

        var filtered = loaded.Rows.Where(row => EvaluateExpression(request.RequestFilter, row, loaded.Definition.Fields)).Select(CloneRow).ToList();
        filtered.Sort((a, b) => string.CompareOrdinal(CanonicalRowKey(a), CanonicalRowKey(b)));
        Shuffle(filtered, randomSeedGuid);
        if (maxItems > 0 && filtered.Count > maxItems)
        {
            filtered = filtered.Take(maxItems).ToList();
        }

        var normalizedRows = filtered.Select(NormalizeForResponse).ToArray();
        var meta = loaded.Metadata;
        var dataSet = new DataSetResponse(
            request.SchemaVersion,
            meta?.TestDataSourceName ?? request.DataSourceName,
            meta?.TestDataSourceUuid ?? request.DataSourceUuid,
            string.IsNullOrWhiteSpace(meta?.JsonSchemaName) ? null : CanonicalResponseSchemaName(meta!.JsonSchemaName),
            new SpecificDatasourceTestData(request.SchemaVersion, normalizedRows)
        );
        return new QueryResult(compiled, allowed, dataSet, normalizedRows);
    }

    public LoadedDataSource Load(string source, DataSourceConfig cfg, FilterRequest request)
    {
        return source switch
        {
            SourceType.Csv => LoadCsv(cfg, request),
            SourceType.Sqlite => LoadSqlite(cfg, request),
            _ => throw new InvalidOperationException($"unsupported source: {source}")
        };
    }

    private LoadedDataSource LoadCsv(DataSourceConfig cfg, FilterRequest request)
    {
        if (string.IsNullOrWhiteSpace(cfg.CsvPath))
        {
            throw new InvalidOperationException("csv path is required");
        }
        var records = CsvRecords.ReadAll(cfg.CsvPath);
        if (records.Count == 0)
        {
            throw new InvalidOperationException("csv is empty");
        }

        var headers = NormalizeHeaders(records[0]);
        if (headers.Count == 0)
        {
            throw new InvalidOperationException("csv has no headers");
        }

        var definition = schema.DefinitionFor(request.DataSourceUuid);
        var headerIndex = new Dictionary<string, int>(StringComparer.Ordinal);
        for (var i = 0; i < headers.Count; i++)
        {
            headerIndex[schema.CanonicalFieldName(headers[i])] = i;
        }

        var matchedHeaders = schema.Order.Count(field => headerIndex.ContainsKey(field));
        var requiredMatches = Math.Min(2, headers.Count);
        if (matchedHeaders < requiredMatches)
        {
            throw new InvalidOperationException("csv headers do not match schema catalog");
        }

        var rows = new List<Dictionary<string, object?>>();
        foreach (var raw in records.Skip(1))
        {
            var rec = NormalizeRecord(raw, headers.Count);
            var row = new Dictionary<string, object?>(StringComparer.Ordinal);
            foreach (var field in schema.Order)
            {
                if (!definition.Fields.TryGetValue(field, out var fieldDef))
                {
                    continue;
                }
                row[field] = headerIndex.TryGetValue(field, out var colIdx) ? ParseValue(rec[colIdx], fieldDef.FieldType) : null;
            }
            rows.Add(row);
        }

        return new LoadedDataSource(definition, rows, null);
    }

    private LoadedDataSource LoadSqlite(DataSourceConfig cfg, FilterRequest request)
    {
        if (string.IsNullOrWhiteSpace(cfg.SQLiteDb))
        {
            throw new InvalidOperationException("db path is required");
        }
        if (!IsSafeTableIdentifier(cfg.SQLiteTable))
        {
            throw new InvalidOperationException($"unsafe table name: {cfg.SQLiteTable}");
        }

        using var connection = new SqliteConnection($"Data Source={cfg.SQLiteDb}");
        connection.Open();

        var metadata = LoadSqliteMetadata(connection, request);
        var definition = schema.DefinitionFor(request.DataSourceUuid);

        using var command = connection.CreateCommand();
        command.CommandText = $"SELECT JsonData FROM {cfg.SQLiteTable} WHERE DataSourceUuid = $uuid AND DataSourceName = $name";
        command.Parameters.AddWithValue("$uuid", request.DataSourceUuid);
        command.Parameters.AddWithValue("$name", request.DataSourceName);

        var rawRows = new List<Dictionary<string, object?>>();
        using var reader = command.ExecuteReader();
        while (reader.Read())
        {
            var rawJson = reader.GetString(0);
            var row = JsonSerializer.Deserialize<Dictionary<string, JsonElement>>(rawJson, JsonOptions.Web)
                ?? throw new InvalidOperationException("unmarshal JsonData failed");
            rawRows.Add(row.ToDictionary(x => schema.CanonicalFieldName(x.Key), x => JsonElementToObject(x.Value), StringComparer.Ordinal));
        }

        if (rawRows.Count == 0)
        {
            throw new InvalidOperationException($"no data rows found for datasource {request.DataSourceName} ({request.DataSourceUuid})");
        }

        var typedRows = new List<Dictionary<string, object?>>();
        foreach (var raw in rawRows)
        {
            var row = new Dictionary<string, object?>(StringComparer.Ordinal);
            foreach (var field in schema.Order)
            {
                if (!definition.Fields.TryGetValue(field, out var fieldDef))
                {
                    continue;
                }
                raw.TryGetValue(field, out var value);
                row[field] = CoerceRawValue(value, fieldDef.FieldType);
            }
            typedRows.Add(row);
        }

        return new LoadedDataSource(definition, typedRows, metadata);
    }

    private static DataSetSchemaMetadata? LoadSqliteMetadata(SqliteConnection connection, FilterRequest request)
    {
        try
        {
            using var command = connection.CreateCommand();
            command.CommandText = """
                SELECT TestDataSourceName, TestDataSourceUuid, JsonSchemaName, JsonSchema, UpdatedDateTime
                FROM main.testdataset_response_schemas
                WHERE lower(TestDataSourceUuid) = lower($uuid) AND TestDataSourceName = $name
                ORDER BY UpdatedDateTime DESC
                LIMIT 1
                """;
            command.Parameters.AddWithValue("$uuid", request.DataSourceUuid);
            command.Parameters.AddWithValue("$name", request.DataSourceName);

            using var reader = command.ExecuteReader();
            if (!reader.Read())
            {
                return null;
            }

            var jsonSchema = reader.GetString(3);
            using var _ = JsonDocument.Parse(jsonSchema);
            return new DataSetSchemaMetadata(
                reader.GetString(0),
                reader.GetString(1),
                CanonicalResponseSchemaName(reader.GetString(2)),
                reader.GetString(4)
            );
        }
        catch (SqliteException ex) when (ex.Message.Contains("no such table", StringComparison.OrdinalIgnoreCase))
        {
            return null;
        }
        catch (JsonException)
        {
            throw new InvalidOperationException($"invalid json schema in main.testdataset_response_schemas for datasource {request.DataSourceName} ({request.DataSourceUuid})");
        }
    }

    private AllowedFieldResponse AllowedFields(FilterRequest request, DataSourceDefinition definition)
    {
        if (!string.Equals(definition.Uuid, request.DataSourceUuid, StringComparison.OrdinalIgnoreCase))
        {
            throw new InvalidOperationException($"DataSourceUuid mismatch for {request.DataSourceName}");
        }

        var items = definition.Fields
            .OrderBy(x => x.Key, StringComparer.Ordinal)
            .Select(x => new AllowedFieldResult(
                x.Key,
                x.Value.FieldType,
                x.Value.Nullable,
                x.Value.SupportedOperators,
                string.IsNullOrWhiteSpace(x.Value.Description) ? null : x.Value.Description
            ))
            .ToArray();
        return new AllowedFieldResponse(request.SchemaVersion, request.RequestUuid, request.DataSourceUuid, request.DataSourceName, items);
    }

    private static List<string> NormalizeHeaders(IReadOnlyList<string> rawHeaders)
    {
        var outHeaders = new List<string>(rawHeaders.Count);
        for (var i = 0; i < rawHeaders.Count; i++)
        {
            var h = rawHeaders[i].Trim();
            if (i == 0)
            {
                h = h.TrimStart('\ufeff');
            }
            outHeaders.Add(h);
        }
        return outHeaders;
    }

    private static List<string> NormalizeRecord(IReadOnlyList<string> raw, int wanted)
    {
        var row = new List<string>(wanted);
        for (var i = 0; i < wanted; i++)
        {
            row.Add(i < raw.Count ? raw[i] : "");
        }
        return row;
    }

    private static object? ParseValue(string raw, string fieldType)
    {
        if (IsCsvNull(raw))
        {
            return null;
        }
        var s = raw.Trim();
        return fieldType switch
        {
            "boolean" => bool.Parse(s.ToLowerInvariant()),
            "integer" => ParseInteger(s),
            "number" => double.Parse(s.Replace(',', '.'), CultureInfo.InvariantCulture),
            _ => s
        };
    }

    private static object? CoerceRawValue(object? value, string fieldType)
    {
        if (value is null)
        {
            return null;
        }
        var raw = RawValueToString(value);
        return ParseValue(raw, fieldType);
    }

    private static string RawValueToString(object? value) => value switch
    {
        null => "NULL",
        bool b => b ? "true" : "false",
        double d => d.ToString("G17", CultureInfo.InvariantCulture),
        float f => f.ToString("G9", CultureInfo.InvariantCulture),
        decimal d => d.ToString(CultureInfo.InvariantCulture),
        JsonElement e => JsonElementToString(e),
        _ => Convert.ToString(value, CultureInfo.InvariantCulture) ?? ""
    };

    private static string JsonElementToString(JsonElement element) => element.ValueKind switch
    {
        JsonValueKind.Null => "NULL",
        JsonValueKind.True => "true",
        JsonValueKind.False => "false",
        JsonValueKind.Number => element.GetRawText(),
        JsonValueKind.String => element.GetString() ?? "",
        _ => element.GetRawText()
    };

    private static object? JsonElementToObject(JsonElement element) => element.ValueKind switch
    {
        JsonValueKind.Null => null,
        JsonValueKind.True => true,
        JsonValueKind.False => false,
        JsonValueKind.Number when element.TryGetInt64(out var i) => i,
        JsonValueKind.Number when element.TryGetDouble(out var d) => d,
        JsonValueKind.String => element.GetString(),
        _ => element.GetRawText()
    };

    private static bool IsCsvNull(string raw)
    {
        var s = raw.Trim();
        return s == "" || string.Equals(s, "NULL", StringComparison.OrdinalIgnoreCase);
    }

    private static long ParseInteger(string raw)
    {
        var s = raw.Trim().Replace(',', '.');
        if (s.Contains('.') || s.Contains('e') || s.Contains('E'))
        {
            var f = double.Parse(s, CultureInfo.InvariantCulture);
            if (f != Math.Truncate(f))
            {
                throw new FormatException($"not an integer: {raw}");
            }
            return (long)f;
        }
        return long.Parse(s, CultureInfo.InvariantCulture);
    }

    private static void ValidateMetadataRequest(FilterRequest request)
    {
        ValidateBasicRequestFields(request);
    }

    private static void ValidateRuntimeRequest(FilterRequest request)
    {
        ValidateBasicRequestFields(request);
        if (request.RequestFilter.ValueKind is JsonValueKind.Undefined or JsonValueKind.Null)
        {
            throw new InvalidOperationException("RequestFilter is required");
        }
    }

    private static void ValidateBasicRequestFields(FilterRequest request)
    {
        if (request.SchemaVersion != "1.0")
        {
            throw new InvalidOperationException($"unsupported SchemaVersion: {request.SchemaVersion}");
        }
        if (!UuidRegex.IsMatch(request.RequestUuid))
        {
            throw new InvalidOperationException($"invalid RequestUuid: {request.RequestUuid}");
        }
        if (!UuidRegex.IsMatch(request.DataSourceUuid))
        {
            throw new InvalidOperationException($"invalid DataSourceUuid: {request.DataSourceUuid}");
        }
        if (string.IsNullOrWhiteSpace(request.DataSourceName))
        {
            throw new InvalidOperationException("DataSourceName is required");
        }
    }

    private static bool IsSafeIdentifier(string value)
    {
        if (value == "") return false;
        for (var i = 0; i < value.Length; i++)
        {
            var ch = value[i];
            if ((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_' || (i > 0 && ch >= '0' && ch <= '9'))
            {
                continue;
            }
            return false;
        }
        return true;
    }

    private static bool IsSafeTableIdentifier(string value)
    {
        return value != "" && value.All(ch =>
            (ch >= 'a' && ch <= 'z') ||
            (ch >= 'A' && ch <= 'Z') ||
            (ch >= '0' && ch <= '9') ||
            ch == '_' ||
            ch == '.');
    }

    private CompiledFilter CompileExpression(JsonElement expression, IReadOnlyDictionary<string, FieldDefinition> fields)
    {
        var (sql, args) = CompileExpressionCore(expression, fields);
        return new CompiledFilter(sql, args);
    }

    private (string Sql, List<object?> Args) CompileExpressionCore(JsonElement expression, IReadOnlyDictionary<string, FieldDefinition> fields)
    {
        if (expression.ValueKind != JsonValueKind.Object)
        {
            throw new InvalidOperationException("invalid expression: must be comparison, and, or, or not");
        }

        if (expression.TryGetProperty("field", out _) && expression.TryGetProperty("op", out _))
        {
            return CompileComparison(expression, fields);
        }
        if (expression.TryGetProperty("and", out var andElement))
        {
            return CompileLogical("AND", andElement, fields);
        }
        if (expression.TryGetProperty("or", out var orElement))
        {
            return CompileLogical("OR", orElement, fields);
        }
        if (expression.TryGetProperty("not", out var notElement))
        {
            var (sql, args) = CompileExpressionCore(notElement, fields);
            return ($"(NOT {sql})", args);
        }
        throw new InvalidOperationException("invalid expression: must be comparison, and, or, or not");
    }

    private (string Sql, List<object?> Args) CompileLogical(string op, JsonElement element, IReadOnlyDictionary<string, FieldDefinition> fields)
    {
        if (element.ValueKind != JsonValueKind.Array || element.GetArrayLength() == 0)
        {
            throw new InvalidOperationException($"{op.ToLowerInvariant()} expression must contain at least one item");
        }

        var clauses = new List<string>();
        var args = new List<object?>();
        foreach (var part in element.EnumerateArray())
        {
            var (sql, partArgs) = CompileExpressionCore(part, fields);
            clauses.Add(sql);
            args.AddRange(partArgs);
        }
        return ($"({string.Join($" {op} ", clauses)})", args);
    }

    private (string Sql, List<object?> Args) CompileComparison(JsonElement expression, IReadOnlyDictionary<string, FieldDefinition> fields)
    {
        var field = expression.GetProperty("field").GetString() ?? "";
        var op = expression.GetProperty("op").GetString() ?? "";
        var value = expression.TryGetProperty("value", out var valueElement) ? valueElement : default;

        if (field == "") throw new InvalidOperationException("comparison.field is required");
        if (op == "") throw new InvalidOperationException("comparison.op is required");
        if (!IsSafeIdentifier(field)) throw new InvalidOperationException($"unsafe field name: {field}");
        if (!fields.TryGetValue(field, out var fieldDef)) throw new InvalidOperationException($"field {field} is not allowed for this datasource");
        if (!fieldDef.SupportedOperators.Contains(op, StringComparer.Ordinal)) throw new InvalidOperationException($"operator {op} is not allowed for field {field}");

        var col = $"\"{field}\"";
        switch (op)
        {
            case "eq":
                ValidateScalar(value, fieldDef.FieldType);
                return ($"({col} = ?)", new List<object?> { JsonValueToObject(value) });
            case "neq":
                ValidateScalar(value, fieldDef.FieldType);
                return ($"({col} <> ?)", new List<object?> { JsonValueToObject(value) });
            case "gt":
            case "gte":
            case "lt":
            case "lte":
                ValidateComparable(value, fieldDef.FieldType);
                var sqlOp = op switch { "gt" => ">", "gte" => ">=", "lt" => "<", _ => "<=" };
                return ($"({col} {sqlOp} ?)", new List<object?> { JsonValueToObject(value) });
            case "contains":
                if (value.ValueKind != JsonValueKind.String) throw new InvalidOperationException("contains requires string value");
                return ($"({col} LIKE ?)", new List<object?> { $"%{value.GetString()}%" });
            case "startsWith":
                if (value.ValueKind != JsonValueKind.String) throw new InvalidOperationException("startsWith requires string value");
                return ($"({col} LIKE ?)", new List<object?> { $"{value.GetString()}%" });
            case "endsWith":
                if (value.ValueKind != JsonValueKind.String) throw new InvalidOperationException("endsWith requires string value");
                return ($"({col} LIKE ?)", new List<object?> { $"%{value.GetString()}" });
            case "exists":
                if (value.ValueKind is not JsonValueKind.True and not JsonValueKind.False) throw new InvalidOperationException("exists requires boolean value");
                return value.GetBoolean() ? ($"({col} IS NOT NULL)", new List<object?>()) : ($"({col} IS NULL)", new List<object?>());
            case "isNull":
                if (value.ValueKind is not JsonValueKind.True and not JsonValueKind.False) throw new InvalidOperationException("isNull requires boolean value");
                return value.GetBoolean() ? ($"({col} IS NULL)", new List<object?>()) : ($"({col} IS NOT NULL)", new List<object?>());
            case "in":
            case "nin":
                if (value.ValueKind != JsonValueKind.Array || value.GetArrayLength() == 0) throw new InvalidOperationException($"{op} requires a non-empty array");
                var args = new List<object?>();
                foreach (var item in value.EnumerateArray())
                {
                    ValidateScalar(item, fieldDef.FieldType);
                    args.Add(JsonValueToObject(item));
                }
                var placeholders = string.Join(", ", Enumerable.Repeat("?", args.Count));
                return ($"({col} {(op == "in" ? "IN" : "NOT IN")} ({placeholders}))", args);
            default:
                throw new InvalidOperationException($"operator not implemented: {op}");
        }
    }

    private static void ValidateScalar(JsonElement value, string fieldType)
    {
        switch (fieldType)
        {
            case "string":
            case "date":
            case "datetime":
                if (value.ValueKind != JsonValueKind.String) throw new InvalidOperationException($"expected string value for field type {fieldType}");
                break;
            case "number":
                if (value.ValueKind != JsonValueKind.Number) throw new InvalidOperationException($"expected numeric value for field type {fieldType}");
                break;
            case "integer":
                if (value.ValueKind != JsonValueKind.Number || !value.TryGetInt64(out _)) throw new InvalidOperationException($"expected integer value for field type {fieldType}");
                break;
            case "boolean":
                if (value.ValueKind is not JsonValueKind.True and not JsonValueKind.False) throw new InvalidOperationException($"expected boolean value for field type {fieldType}");
                break;
            default:
                throw new InvalidOperationException($"unsupported field type {fieldType}");
        }
    }

    private static void ValidateComparable(JsonElement value, string fieldType)
    {
        if (fieldType is not ("number" or "integer" or "date" or "datetime"))
        {
            throw new InvalidOperationException($"operator requires comparable field type, got {fieldType}");
        }
        ValidateScalar(value, fieldType);
    }

    private static object? JsonValueToObject(JsonElement value) => value.ValueKind switch
    {
        JsonValueKind.Null => null,
        JsonValueKind.String => value.GetString(),
        JsonValueKind.True => true,
        JsonValueKind.False => false,
        JsonValueKind.Number when value.TryGetInt64(out var i) => i,
        JsonValueKind.Number when value.TryGetDouble(out var d) => d,
        JsonValueKind.Array => value.EnumerateArray().Select(JsonValueToObject).ToArray(),
        _ => value.GetRawText()
    };

    private bool EvaluateExpression(JsonElement expression, IReadOnlyDictionary<string, object?> row, IReadOnlyDictionary<string, FieldDefinition> fields)
    {
        if (expression.TryGetProperty("field", out _) && expression.TryGetProperty("op", out _))
        {
            return EvaluateComparison(expression, row, fields);
        }
        if (expression.TryGetProperty("and", out var andElement))
        {
            if (andElement.ValueKind != JsonValueKind.Array || andElement.GetArrayLength() == 0) throw new InvalidOperationException("and expression must contain at least one item");
            foreach (var part in andElement.EnumerateArray())
            {
                if (!EvaluateExpression(part, row, fields)) return false;
            }
            return true;
        }
        if (expression.TryGetProperty("or", out var orElement))
        {
            if (orElement.ValueKind != JsonValueKind.Array || orElement.GetArrayLength() == 0) throw new InvalidOperationException("or expression must contain at least one item");
            foreach (var part in orElement.EnumerateArray())
            {
                if (EvaluateExpression(part, row, fields)) return true;
            }
            return false;
        }
        if (expression.TryGetProperty("not", out var notElement))
        {
            return !EvaluateExpression(notElement, row, fields);
        }
        throw new InvalidOperationException("invalid expression: must be comparison, and, or, or not");
    }

    private bool EvaluateComparison(JsonElement expression, IReadOnlyDictionary<string, object?> row, IReadOnlyDictionary<string, FieldDefinition> fields)
    {
        CompileComparison(expression, fields);

        var field = expression.GetProperty("field").GetString() ?? "";
        var op = expression.GetProperty("op").GetString() ?? "";
        var value = expression.GetProperty("value");
        var fieldDef = fields[field];
        row.TryGetValue(field, out var rowValue);

        return op switch
        {
            "eq" => ValuesEqual(fieldDef.FieldType, rowValue, JsonValueToObject(value)),
            "neq" => !ValuesEqual(fieldDef.FieldType, rowValue, JsonValueToObject(value)),
            "gt" => rowValue is not null && CompareOrdered(fieldDef.FieldType, rowValue, JsonValueToObject(value)) > 0,
            "gte" => rowValue is not null && CompareOrdered(fieldDef.FieldType, rowValue, JsonValueToObject(value)) >= 0,
            "lt" => rowValue is not null && CompareOrdered(fieldDef.FieldType, rowValue, JsonValueToObject(value)) < 0,
            "lte" => rowValue is not null && CompareOrdered(fieldDef.FieldType, rowValue, JsonValueToObject(value)) <= 0,
            "contains" => rowValue is string s && s.Contains(value.GetString() ?? "", StringComparison.Ordinal),
            "startsWith" => rowValue is string s && s.StartsWith(value.GetString() ?? "", StringComparison.Ordinal),
            "endsWith" => rowValue is string s && s.EndsWith(value.GetString() ?? "", StringComparison.Ordinal),
            "exists" => (rowValue is not null) == value.GetBoolean(),
            "isNull" => (rowValue is null) == value.GetBoolean(),
            "in" => value.EnumerateArray().Any(item => ValuesEqual(fieldDef.FieldType, rowValue, JsonValueToObject(item))),
            "nin" => !value.EnumerateArray().Any(item => ValuesEqual(fieldDef.FieldType, rowValue, JsonValueToObject(item))),
            _ => throw new InvalidOperationException($"operator not implemented: {op}")
        };
    }

    private static bool ValuesEqual(string fieldType, object? rowValue, object? filterValue)
    {
        if (rowValue is null || filterValue is null)
        {
            return rowValue is null && filterValue is null;
        }

        return fieldType switch
        {
            "number" or "integer" => Convert.ToDouble(rowValue, CultureInfo.InvariantCulture) == Convert.ToDouble(filterValue, CultureInfo.InvariantCulture),
            "boolean" => Convert.ToBoolean(rowValue, CultureInfo.InvariantCulture) == Convert.ToBoolean(filterValue, CultureInfo.InvariantCulture),
            _ => Convert.ToString(rowValue, CultureInfo.InvariantCulture) == Convert.ToString(filterValue, CultureInfo.InvariantCulture)
        };
    }

    private static int CompareOrdered(string fieldType, object? rowValue, object? filterValue)
    {
        if (fieldType is "number" or "integer")
        {
            return Convert.ToDouble(rowValue, CultureInfo.InvariantCulture).CompareTo(Convert.ToDouble(filterValue, CultureInfo.InvariantCulture));
        }
        return string.CompareOrdinal(Convert.ToString(rowValue, CultureInfo.InvariantCulture), Convert.ToString(filterValue, CultureInfo.InvariantCulture));
    }

    private static Dictionary<string, object?> CloneRow(IReadOnlyDictionary<string, object?> row)
    {
        return row.ToDictionary(x => x.Key, x => x.Value, StringComparer.Ordinal);
    }

    private static string CanonicalRowKey(Dictionary<string, object?> row)
    {
        return JsonSerializer.Serialize(row.OrderBy(x => x.Key, StringComparer.Ordinal).ToDictionary(x => x.Key, x => x.Value), JsonOptions.Web);
    }

    private Dictionary<string, object?> NormalizeForResponse(Dictionary<string, object?> row)
    {
        var normalized = new Dictionary<string, object?>(StringComparer.Ordinal);
        foreach (var (key, value) in row)
        {
            var canonical = schema.CanonicalFieldName(key);
            var field = schema.Current.Fields.TryGetValue(canonical, out var f) ? f : null;
            normalized[canonical] = field is null ? value : NormalizeSchemaFieldValue(value, field);
        }
        return normalized;
    }

    private static object? NormalizeSchemaFieldValue(object? value, SchemaField field)
    {
        if (value is null)
        {
            return null;
        }
        if (field.FieldType is not ("string" or "date" or "datetime"))
        {
            return value;
        }

        var s = value switch
        {
            double d => d.ToString("G17", CultureInfo.InvariantCulture).Replace('.', ','),
            float f => f.ToString("G9", CultureInfo.InvariantCulture).Replace('.', ','),
            decimal d => d.ToString(CultureInfo.InvariantCulture).Replace('.', ','),
            bool b => b ? "true" : "false",
            _ => Convert.ToString(value, CultureInfo.InvariantCulture) ?? ""
        };
        s = s.Trim();
        if (field.Nullable && (s == "" || string.Equals(s, "NULL", StringComparison.OrdinalIgnoreCase)))
        {
            return null;
        }
        return s;
    }

    private static void Shuffle(List<Dictionary<string, object?>> rows, string randomSeedGuid)
    {
        if (rows.Count < 2)
        {
            return;
        }

        Random rng;
        if (string.IsNullOrWhiteSpace(randomSeedGuid))
        {
            rng = Random.Shared;
        }
        else
        {
            if (!UuidRegex.IsMatch(randomSeedGuid.Trim()))
            {
                throw new InvalidOperationException($"invalid random seed guid: {randomSeedGuid}");
            }
            var normalized = randomSeedGuid.Trim().Replace("-", "", StringComparison.Ordinal).ToLowerInvariant();
            var bytes = Convert.FromHexString(normalized);
            var seed64 = BinaryPrimitives.ReadInt64BigEndian(bytes.AsSpan(0, 8));
            rng = new Random(unchecked((int)(seed64 ^ (seed64 >> 32))));
        }

        for (var i = rows.Count - 1; i > 0; i--)
        {
            var j = rng.Next(i + 1);
            (rows[i], rows[j]) = (rows[j], rows[i]);
        }
    }

    private static string CanonicalResponseSchemaName(string name)
    {
        var fileName = Path.GetFileName((name ?? "").Trim());
        return fileName == "TestDataSet_Response_For_Specific_DatasourceFrom_TestDataEngine.json-schema.json"
            ? Constants.SpecificResponseSchemaName
            : fileName;
    }
}

sealed class FacetEngine(QueryEngine queryEngine)
{
    public (IReadOnlyList<FacetValue> Values, bool Truncated) Values(string source, DataSourceConfig cfg, FilterRequest metadataRequest, string field, int limit, string queryText)
    {
        var loaded = queryEngine.Load(source, cfg, metadataRequest);
        if (!loaded.Definition.Fields.ContainsKey(field))
        {
            throw new InvalidOperationException($"unknown field: {field}");
        }

        var needle = (queryText ?? "").Trim().ToLowerInvariant();
        var values = new Dictionary<string, (object? Value, bool IsNull, int Count)>(StringComparer.Ordinal);
        foreach (var row in loaded.Rows)
        {
            row.TryGetValue(field, out var value);
            var label = FacetLabel(value);
            if (needle != "" && !label.ToLowerInvariant().Contains(needle, StringComparison.Ordinal))
            {
                continue;
            }

            var key = FacetKey(value);
            values[key] = values.TryGetValue(key, out var existing)
                ? (existing.Value, existing.IsNull, existing.Count + 1)
                : (value, value is null, 1);
        }

        var outValues = values.Values
            .Select(x => new FacetValue(x.Value, FacetLabel(x.Value), x.Count, x.IsNull))
            .OrderByDescending(x => x.Count)
            .ThenBy(x => x.Label, StringComparer.Ordinal)
            .ToList();

        var truncated = false;
        if (limit > 0 && outValues.Count > limit)
        {
            outValues = outValues.Take(limit).ToList();
            truncated = true;
        }

        return (outValues, truncated);
    }

    private static string FacetLabel(object? value) => value switch
    {
        null => "(blank)",
        bool b => b ? "true" : "false",
        _ => Convert.ToString(value, CultureInfo.InvariantCulture) ?? ""
    };
    private static string FacetKey(object? value) => value is null ? "null" : $"{value.GetType().FullName}:{FacetLabel(value)}";
}

static class CsvRecords
{
    public static List<List<string>> ReadAll(string path)
    {
        var records = new List<List<string>>();
        var field = new StringBuilder();
        var record = new List<string>();
        var inQuotes = false;

        using var reader = new StreamReader(path, Encoding.UTF8, detectEncodingFromByteOrderMarks: true);
        while (true)
        {
            var next = reader.Read();
            if (next == -1)
            {
                break;
            }
            var ch = (char)next;
            if (inQuotes)
            {
                if (ch == '"')
                {
                    if (reader.Peek() == '"')
                    {
                        reader.Read();
                        field.Append('"');
                    }
                    else
                    {
                        inQuotes = false;
                    }
                }
                else
                {
                    field.Append(ch);
                }
                continue;
            }

            switch (ch)
            {
                case '"':
                    inQuotes = true;
                    break;
                case ';':
                    record.Add(field.ToString());
                    field.Clear();
                    break;
                case '\r':
                    if (reader.Peek() == '\n')
                    {
                        reader.Read();
                    }
                    FinishRecord(records, record, field);
                    break;
                case '\n':
                    FinishRecord(records, record, field);
                    break;
                default:
                    field.Append(ch);
                    break;
            }
        }

        if (field.Length > 0 || record.Count > 0)
        {
            FinishRecord(records, record, field);
        }

        return records;
    }

    private static void FinishRecord(List<List<string>> records, List<string> record, StringBuilder field)
    {
        record.Add(field.ToString());
        field.Clear();
        records.Add([.. record]);
        record.Clear();
    }
}
