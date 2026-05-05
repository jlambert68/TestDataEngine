using System;
using System.Collections.Generic;
using System.Text.Json;
using System.Text.Json.Serialization;

namespace TestDataEngine.Requirements.Models;

[JsonConverter(typeof(SourceTypeJsonConverter))]
public enum SourceType
{
    Csv,
    Sqlite,
    Postgres
}

public sealed class SourceTypeJsonConverter : JsonConverter<SourceType>
{
    public override SourceType Read(ref Utf8JsonReader reader, Type typeToConvert, JsonSerializerOptions options)
    {
        var value = reader.GetString();
        return value switch
        {
            "csv" => SourceType.Csv,
            "sqlite" => SourceType.Sqlite,
            "postgres" => SourceType.Postgres,
            _ => throw new JsonException($"Unsupported source type: {value}")
        };
    }

    public override void Write(Utf8JsonWriter writer, SourceType value, JsonSerializerOptions options)
    {
        writer.WriteStringValue(value switch
        {
            SourceType.Csv => "csv",
            SourceType.Sqlite => "sqlite",
            SourceType.Postgres => "postgres",
            _ => throw new JsonException($"Unsupported source type: {value}")
        });
    }
}

public sealed record FilterRequest
{
    [JsonPropertyName("SchemaVersion")]
    public required string SchemaVersion { get; init; }

    [JsonPropertyName("RequestUuid")]
    public required string RequestUuid { get; init; }

    [JsonPropertyName("DataSourceUuid")]
    public required string DataSourceUuid { get; init; }

    [JsonPropertyName("DataSourceName")]
    public required string DataSourceName { get; init; }

    [JsonPropertyName("RequestFilter")]
    public required FilterExpression RequestFilter { get; init; }
}

public sealed record APIError
{
    [JsonPropertyName("error")]
    public required string Error { get; init; }

    [JsonPropertyName("details")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? Details { get; init; }
}

public sealed record DataSourceListItem
{
    [JsonPropertyName("id")]
    public required string Id { get; init; }

    [JsonPropertyName("label")]
    public required string Label { get; init; }

    [JsonPropertyName("dataSourceName")]
    public required string DataSourceName { get; init; }

    [JsonPropertyName("dataSourceUuid")]
    public required string DataSourceUuid { get; init; }

    [JsonPropertyName("supportedSources")]
    public required IReadOnlyList<SourceType> SupportedSources { get; init; }

    [JsonPropertyName("defaultSource")]
    public required SourceType DefaultSource { get; init; }
}

public sealed record DataSourceConfig
{
    [JsonPropertyName("id")]
    public required string Id { get; init; }

    [JsonPropertyName("label")]
    public required string Label { get; init; }

    [JsonPropertyName("dataSourceName")]
    public required string DataSourceName { get; init; }

    [JsonPropertyName("dataSourceUuid")]
    public required string DataSourceUuid { get; init; }

    [JsonPropertyName("supportedSources")]
    public required IReadOnlyList<SourceType> SupportedSources { get; init; }

    [JsonPropertyName("defaultSource")]
    public required SourceType DefaultSource { get; init; }

    [JsonIgnore]
    public string CsvPath { get; init; } = "";

    [JsonIgnore]
    public string SQLiteDB { get; init; } = "";

    [JsonIgnore]
    public string SQLiteTable { get; init; } = "main.data_items";

    [JsonIgnore]
    public string PostgresDSN { get; init; } = "";

    [JsonIgnore]
    public string PostgresTable { get; init; } = "public.data_items";

    [JsonIgnore]
    public string PostgresSchemaTable { get; init; } = "public.testdataset_response_schemas";

    public DataSourceListItem ToListItem() => new()
    {
        Id = Id,
        Label = Label,
        DataSourceName = DataSourceName,
        DataSourceUuid = DataSourceUuid,
        SupportedSources = SupportedSources,
        DefaultSource = DefaultSource
    };
}

public sealed record ListDataSourcesResponse
{
    [JsonPropertyName("items")]
    public required IReadOnlyList<DataSourceListItem> Items { get; init; }
}

// The runtime JSON does not contain a discriminator property.
// A .NET implementation should deserialize this hierarchy with a custom shape-based converter:
// comparison => { "field": "...", "op": "...", "value": ... }
// and        => { "and": [ ... ] }
// or         => { "or": [ ... ] }
// not        => { "not": { ... } }
public abstract record FilterExpression;

// The typed SQL compiler package in the Go code uses a separate request type and separate
// operator model even though the JSON shape is very similar to the runtime request.
public sealed record TypedSqlRequest
{
    [JsonPropertyName("SchemaVersion")]
    public required string SchemaVersion { get; init; }

    [JsonPropertyName("RequestUuid")]
    public required string RequestUuid { get; init; }

    [JsonPropertyName("DataSourceUuid")]
    public required string DataSourceUuid { get; init; }

    [JsonPropertyName("DataSourceName")]
    public required string DataSourceName { get; init; }

    [JsonPropertyName("RequestFilter")]
    public required TypedSqlExpression RequestFilter { get; init; }
}

public enum TypedSqlPlaceholderStyle
{
    Question,
    Dollar
}

public sealed record TypedSqlCompilerOptions
{
    public TypedSqlPlaceholderStyle Placeholder { get; init; } = TypedSqlPlaceholderStyle.Question;
    public bool QuoteIdent { get; init; } = true;
}

public sealed record ComparisonExpression : FilterExpression
{
    [JsonPropertyName("field")]
    public required string Field { get; init; }

    [JsonPropertyName("op")]
    public required string Op { get; init; }

    [JsonPropertyName("value")]
    public JsonElement? Value { get; init; }
}

public sealed record AndExpression : FilterExpression
{
    [JsonPropertyName("and")]
    public required IReadOnlyList<FilterExpression> And { get; init; }
}

public sealed record OrExpression : FilterExpression
{
    [JsonPropertyName("or")]
    public required IReadOnlyList<FilterExpression> Or { get; init; }
}

public sealed record NotExpression : FilterExpression
{
    [JsonPropertyName("not")]
    public required FilterExpression Not { get; init; }
}

public abstract record TypedSqlExpression;

[JsonConverter(typeof(TypedSqlOperatorJsonConverter))]
public enum TypedSqlOperator
{
    Eq,
    Neq,
    Gt,
    Gte,
    Lt,
    Lte,
    In,
    Nin,
    Contains,
    StartsWith,
    EndsWith,
    Exists,
    IsNull
}

public sealed class TypedSqlOperatorJsonConverter : JsonConverter<TypedSqlOperator>
{
    public override TypedSqlOperator Read(ref Utf8JsonReader reader, Type typeToConvert, JsonSerializerOptions options)
    {
        var value = reader.GetString();
        return value switch
        {
            "eq" => TypedSqlOperator.Eq,
            "neq" => TypedSqlOperator.Neq,
            "gt" => TypedSqlOperator.Gt,
            "gte" => TypedSqlOperator.Gte,
            "lt" => TypedSqlOperator.Lt,
            "lte" => TypedSqlOperator.Lte,
            "in" => TypedSqlOperator.In,
            "nin" => TypedSqlOperator.Nin,
            "contains" => TypedSqlOperator.Contains,
            "startsWith" => TypedSqlOperator.StartsWith,
            "endsWith" => TypedSqlOperator.EndsWith,
            "exists" => TypedSqlOperator.Exists,
            "isNull" => TypedSqlOperator.IsNull,
            _ => throw new JsonException($"Unsupported operator: {value}")
        };
    }

    public override void Write(Utf8JsonWriter writer, TypedSqlOperator value, JsonSerializerOptions options)
    {
        writer.WriteStringValue(value switch
        {
            TypedSqlOperator.Eq => "eq",
            TypedSqlOperator.Neq => "neq",
            TypedSqlOperator.Gt => "gt",
            TypedSqlOperator.Gte => "gte",
            TypedSqlOperator.Lt => "lt",
            TypedSqlOperator.Lte => "lte",
            TypedSqlOperator.In => "in",
            TypedSqlOperator.Nin => "nin",
            TypedSqlOperator.Contains => "contains",
            TypedSqlOperator.StartsWith => "startsWith",
            TypedSqlOperator.EndsWith => "endsWith",
            TypedSqlOperator.Exists => "exists",
            TypedSqlOperator.IsNull => "isNull",
            _ => throw new JsonException($"Unsupported operator: {value}")
        });
    }
}

public sealed record TypedSqlComparisonExpression : TypedSqlExpression
{
    [JsonPropertyName("field")]
    public required string Field { get; init; }

    [JsonPropertyName("op")]
    public required TypedSqlOperator Op { get; init; }

    [JsonPropertyName("value")]
    public object? Value { get; init; }
}

public sealed record TypedSqlAndExpression : TypedSqlExpression
{
    [JsonPropertyName("and")]
    public required IReadOnlyList<TypedSqlExpression> And { get; init; }
}

public sealed record TypedSqlOrExpression : TypedSqlExpression
{
    [JsonPropertyName("or")]
    public required IReadOnlyList<TypedSqlExpression> Or { get; init; }
}

public sealed record TypedSqlNotExpression : TypedSqlExpression
{
    [JsonPropertyName("not")]
    public required TypedSqlExpression Not { get; init; }
}

public sealed record CompiledFilter
{
    public required string WhereSql { get; init; }
    public required IReadOnlyList<object?> Args { get; init; }
}

public sealed record FieldDefinition
{
    public required string FieldType { get; init; }
    public required bool Nullable { get; init; }
    public required IReadOnlyCollection<string> SupportedOperators { get; init; }
    public string? Description { get; init; }
}

public sealed record DataSourceDefinition
{
    public required string Uuid { get; init; }
    public required IReadOnlyDictionary<string, FieldDefinition> Fields { get; init; }
}

public sealed record AllowedFieldResult
{
    [JsonPropertyName("FieldName")]
    public required string FieldName { get; init; }

    [JsonPropertyName("FieldType")]
    public required string FieldType { get; init; }

    [JsonPropertyName("Nullable")]
    public required bool Nullable { get; init; }

    [JsonPropertyName("SupportedOperators")]
    public required IReadOnlyList<string> SupportedOperators { get; init; }

    [JsonPropertyName("Description")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? Description { get; init; }
}

public sealed record AllowedFieldResponse
{
    [JsonPropertyName("SchemaVersion")]
    public required string SchemaVersion { get; init; }

    [JsonPropertyName("RequestUuid")]
    public required string RequestUuid { get; init; }

    [JsonPropertyName("DataSourceUuid")]
    public required string DataSourceUuid { get; init; }

    [JsonPropertyName("DataSourceName")]
    public required string DataSourceName { get; init; }

    [JsonPropertyName("AllowedFields")]
    public required IReadOnlyList<AllowedFieldResult> AllowedFields { get; init; }
}

public sealed record FieldDescriptor
{
    [JsonPropertyName("field")]
    public required string Field { get; init; }

    [JsonPropertyName("fieldType")]
    public required string FieldType { get; init; }

    [JsonPropertyName("nullable")]
    public required bool Nullable { get; init; }

    [JsonPropertyName("supportedOperators")]
    public required IReadOnlyList<string> SupportedOperators { get; init; }

    [JsonPropertyName("widget")]
    public required string Widget { get; init; }

    [JsonPropertyName("facetEligible")]
    public required bool FacetEligible { get; init; }

    [JsonPropertyName("description")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? Description { get; init; }
}

public sealed record GetFieldsResponse
{
    [JsonPropertyName("datasourceId")]
    public required string DatasourceId { get; init; }

    [JsonPropertyName("source")]
    public required SourceType Source { get; init; }

    [JsonPropertyName("fields")]
    public required IReadOnlyList<FieldDescriptor> Fields { get; init; }
}

public sealed record FacetValue
{
    [JsonPropertyName("value")]
    public object? Value { get; init; }

    [JsonPropertyName("label")]
    public required string Label { get; init; }

    [JsonPropertyName("count")]
    public required int Count { get; init; }

    [JsonPropertyName("isNull")]
    public required bool IsNull { get; init; }
}

public sealed record GetFacetsResponse
{
    [JsonPropertyName("datasourceId")]
    public required string DatasourceId { get; init; }

    [JsonPropertyName("source")]
    public required SourceType Source { get; init; }

    [JsonPropertyName("field")]
    public required string Field { get; init; }

    [JsonPropertyName("values")]
    public required IReadOnlyList<FacetValue> Values { get; init; }

    [JsonPropertyName("truncated")]
    public required bool Truncated { get; init; }
}

public sealed record DataSetSchemaMetadata
{
    public required string TestDataSourceName { get; init; }
    public required string TestDataSourceUuid { get; init; }
    public required string JsonSchemaName { get; init; }
    public required JsonDocument JsonSchema { get; init; }
    public required string UpdatedDateTime { get; init; }
}

public sealed record SpecificDatasourceTestData
{
    [JsonPropertyName("SpecificSourceSchemaVersion")]
    public required string SpecificSourceSchemaVersion { get; init; }

    [JsonPropertyName("TestDataSet")]
    public required IReadOnlyList<IReadOnlyDictionary<string, object?>> TestDataSet { get; init; }
}

public sealed record DataSetResponse
{
    [JsonPropertyName("SchemaVersion")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? SchemaVersion { get; init; }

    [JsonPropertyName("TestDataSourceName")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? TestDataSourceName { get; init; }

    [JsonPropertyName("TestDataSourceUuid")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? TestDataSourceUuid { get; init; }

    [JsonPropertyName("JsonSchemaName")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? JsonSchemaName { get; init; }

    [JsonPropertyName("TestData")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public SpecificDatasourceTestData? TestData { get; init; }

    [JsonIgnore]
    public JsonDocument? JsonSchema { get; init; }

    [JsonIgnore]
    public string? UpdatedDateTime { get; init; }

    [JsonIgnore]
    public string? DataSourceName { get; init; }

    [JsonIgnore]
    public string? DataSourceUuid { get; init; }

    [JsonIgnore]
    public IReadOnlyList<IReadOnlyDictionary<string, object?>>? Data { get; init; }
}

public sealed record QueryPreviewRequest
{
    [JsonPropertyName("source")]
    public required SourceType Source { get; init; }

    [JsonPropertyName("maxItems")]
    public required int MaxItems { get; init; }

    [JsonPropertyName("randomSeedGuid")]
    [JsonIgnore(Condition = JsonIgnoreCondition.WhenWritingNull)]
    public string? RandomSeedGuid { get; init; }

    [JsonPropertyName("request")]
    public required FilterRequest Request { get; init; }
}

public sealed record QueryPreviewResponse
{
    [JsonPropertyName("source")]
    public required SourceType Source { get; init; }

    [JsonPropertyName("compiledWhereSql")]
    public required string CompiledWhereSql { get; init; }

    [JsonPropertyName("compiledArgs")]
    public required IReadOnlyList<object?> CompiledArgs { get; init; }

    [JsonPropertyName("allowedFields")]
    public required AllowedFieldResponse AllowedFields { get; init; }

    [JsonPropertyName("dataSet")]
    public required DataSetResponse DataSet { get; init; }
}

public sealed record MarshaledDataSetResponse
{
    [JsonPropertyName("SchemaVersion")]
    public required string SchemaVersion { get; init; }

    [JsonPropertyName("TestDataSourceName")]
    public required string TestDataSourceName { get; init; }

    [JsonPropertyName("TestDataSourceUuid")]
    public required string TestDataSourceUuid { get; init; }

    [JsonPropertyName("JsonSchemaName")]
    public required string JsonSchemaName { get; init; }

    [JsonPropertyName("TestData")]
    public required SpecificDatasourceTestData TestData { get; init; }
}

public sealed record AllowedFieldsLogEnvelope
{
    [JsonPropertyName("Source")]
    public required SourceType Source { get; init; }

    [JsonPropertyName("InputFilter")]
    public required JsonElement InputFilter { get; init; }

    [JsonPropertyName("AllowedFieldsResponse")]
    public required AllowedFieldResponse AllowedFieldsResponse { get; init; }
}

public sealed record DataSetLogEnvelope
{
    [JsonPropertyName("Source")]
    public required SourceType Source { get; init; }

    [JsonPropertyName("InputFilter")]
    public required JsonElement InputFilter { get; init; }

    [JsonPropertyName("DataSetResponse")]
    public required MarshaledDataSetResponse DataSetResponse { get; init; }
}

public sealed record ImportOptions
{
    public required string DbPath { get; init; }
    public required string CsvPath { get; init; }
    public required string DataSourceUuid { get; init; }
    public required string DataSourceName { get; init; }
    public string? TestDataDomainUuid { get; init; }
    public string? TestDataDomainName { get; init; }
    public string? TestDataSourceTemplateName { get; init; }
    public string TableName { get; init; } = "main.data_items";
    public char Delimiter { get; init; } = ';';
    public int BatchSize { get; init; } = 500;
}

public sealed record ImportResult
{
    public required int RowsInserted { get; init; }
}
