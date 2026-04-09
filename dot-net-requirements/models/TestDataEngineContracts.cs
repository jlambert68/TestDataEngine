using System.Collections.Generic;
using System.Text.Json;

namespace TestDataEngine.Requirements.Models;

public sealed record FilterRequest
{
    public required string SchemaVersion { get; init; }
    public required string RequestUuid { get; init; }
    public required string DataSourceUuid { get; init; }
    public required string DataSourceName { get; init; }
    public required FilterExpression RequestFilter { get; init; }
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
    public required string SchemaVersion { get; init; }
    public required string RequestUuid { get; init; }
    public required string DataSourceUuid { get; init; }
    public required string DataSourceName { get; init; }
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
    public required string Field { get; init; }
    public required string Op { get; init; }
    public JsonElement? Value { get; init; }
}

public sealed record AndExpression : FilterExpression
{
    public required IReadOnlyList<FilterExpression> And { get; init; }
}

public sealed record OrExpression : FilterExpression
{
    public required IReadOnlyList<FilterExpression> Or { get; init; }
}

public sealed record NotExpression : FilterExpression
{
    public required FilterExpression Not { get; init; }
}

public abstract record TypedSqlExpression;

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

public sealed record TypedSqlComparisonExpression : TypedSqlExpression
{
    public required string Field { get; init; }
    public required TypedSqlOperator Op { get; init; }
    public object? Value { get; init; }
}

public sealed record TypedSqlAndExpression : TypedSqlExpression
{
    public required IReadOnlyList<TypedSqlExpression> And { get; init; }
}

public sealed record TypedSqlOrExpression : TypedSqlExpression
{
    public required IReadOnlyList<TypedSqlExpression> Or { get; init; }
}

public sealed record TypedSqlNotExpression : TypedSqlExpression
{
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
    public required string FieldName { get; init; }
    public required string FieldType { get; init; }
    public required bool Nullable { get; init; }
    public required IReadOnlyList<string> SupportedOperators { get; init; }
    public string? Description { get; init; }
}

public sealed record AllowedFieldResponse
{
    public required string SchemaVersion { get; init; }
    public required string RequestUuid { get; init; }
    public required string DataSourceUuid { get; init; }
    public required string DataSourceName { get; init; }
    public required IReadOnlyList<AllowedFieldResult> AllowedFields { get; init; }
}

public sealed record DataSetSchemaMetadata
{
    public required string TestDataSourceName { get; init; }
    public required string TestDataSourceUuid { get; init; }
    public required string JsonSchemaName { get; init; }
    public required JsonDocument JsonSchema { get; init; }
    public required string UpdatedDateTime { get; init; }
}

public sealed record DataSetResponse
{
    public string? TestDataSourceName { get; init; }
    public string? TestDataSourceUuid { get; init; }
    public string? JsonSchemaName { get; init; }
    public JsonDocument? JsonSchema { get; init; }
    public string? UpdatedDateTime { get; init; }
    public required string DataSourceName { get; init; }
    public required string DataSourceUuid { get; init; }
    public required IReadOnlyList<IReadOnlyDictionary<string, object?>> Data { get; init; }
}

public sealed record AllowedFieldsLogEnvelope
{
    public required JsonElement InputFilter { get; init; }
    public required AllowedFieldResponse AllowedFieldsResponse { get; init; }
}

public sealed record DataSetLogEnvelope
{
    public required JsonElement InputFilter { get; init; }
    public required DataSetResponse DataSetResponse { get; init; }
}

public sealed record ImportOptions
{
    public required string DbPath { get; init; }
    public required string CsvPath { get; init; }
    public required string DataSourceUuid { get; init; }
    public required string DataSourceName { get; init; }
    public string TableName { get; init; } = "main.data_items";
    public char Delimiter { get; init; } = ';';
    public int BatchSize { get; init; } = 500;
}

public sealed record ImportResult
{
    public required int RowsInserted { get; init; }
}
