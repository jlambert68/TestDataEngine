export type SourceType = 'csv' | 'sqlite' | 'postgres'

export interface DataSourceListItem {
  id: string
  label: string
  dataSourceName: string
  dataSourceUuid: string
  supportedSources: SourceType[]
  defaultSource: SourceType
}

export interface ListDataSourcesResponse {
  items: DataSourceListItem[]
}

export type Scalar = string | number | boolean | null

export interface ComparisonExpression {
  field: string
  op: 'eq' | 'neq' | 'gt' | 'gte' | 'lt' | 'lte' | 'in' | 'nin' | 'contains' | 'startsWith' | 'endsWith' | 'exists' | 'isNull'
  value: Scalar | Scalar[]
}

export interface AndExpression {
  and: FilterExpression[]
}

export interface OrExpression {
  or: FilterExpression[]
}

export interface NotExpression {
  not: FilterExpression
}

export type FilterExpression = ComparisonExpression | AndExpression | OrExpression | NotExpression

export interface FilterRequest {
  SchemaVersion: '1.0'
  RequestUuid: string
  DataSourceUuid: string
  DataSourceName: string
  RequestFilter: FilterExpression
}

export interface FieldDescriptor {
  field: string
  fieldType: 'string' | 'number' | 'integer' | 'boolean' | 'date' | 'datetime'
  nullable: boolean
  supportedOperators: string[]
  widget: 'checkbox-group' | 'searchable-checkbox-group' | 'text' | 'number-range' | 'boolean-toggle'
  facetEligible: boolean
  description?: string
}

export interface GetFieldsResponse {
  datasourceId: string
  source: SourceType
  fields: FieldDescriptor[]
}

export interface FacetValue {
  value: Scalar
  label: string
  count: number
  isNull: boolean
}

export interface GetFacetsResponse {
  datasourceId: string
  source: SourceType
  field: string
  values: FacetValue[]
  truncated: boolean
}

export interface QueryPreviewRequest {
  source: SourceType
  maxItems: number
  randomSeedGuid?: string
  request: FilterRequest
}

export interface AllowedFieldResult {
  FieldName: string
  FieldType: string
  Nullable: boolean
  SupportedOperators: string[]
  Description?: string
}

export interface AllowedFieldResponse {
  SchemaVersion: string
  RequestUuid: string
  DataSourceUuid: string
  DataSourceName: string
  AllowedFields: AllowedFieldResult[]
}

export interface SpecificTestData {
  SpecificSourceSchemaVersion: string
  TestDataSet: Array<Record<string, unknown>>
}

export interface DataSetResponse {
  SchemaVersion?: string
  TestDataSourceName?: string
  TestDataSourceUuid?: string
  JsonSchemaName?: string
  TestData?: SpecificTestData
}

export interface QueryPreviewResponse {
  source: SourceType
  compiledWhereSql: string
  compiledArgs: unknown[]
  allowedFields: AllowedFieldResponse
  dataSet: DataSetResponse
}

export type GroupCombinator = 'and' | 'or'

export interface FilterRuleState {
  id: string
  field: string
  operator: ComparisonExpression['op']
  values: Scalar[]
  scalarValue: Scalar
  booleanValue: boolean
}

export interface FilterGroupState {
  id: string
  combinator: GroupCombinator
  negated: boolean
  items: FilterNodeState[]
}

export interface FilterRuleNode {
  kind: 'rule'
  rule: FilterRuleState
}

export interface FilterGroupNode {
  kind: 'group'
  group: FilterGroupState
}

export type FilterNodeState = FilterRuleNode | FilterGroupNode

export interface BuilderState {
  datasourceId: string
  source: SourceType
  maxItems: number
  randomSeedGuid: string
  rootGroup: FilterGroupState
}
