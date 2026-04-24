package webapi

import "TestDataEngine/internal/filters"

type APIError struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

type DataSourceListItem struct {
	ID               string       `json:"id"`
	Label            string       `json:"label"`
	DataSourceName   string       `json:"dataSourceName"`
	DataSourceUUID   string       `json:"dataSourceUuid"`
	SupportedSources []SourceType `json:"supportedSources"`
	DefaultSource    SourceType   `json:"defaultSource"`
}

type ListDataSourcesResponse struct {
	Items []DataSourceListItem `json:"items"`
}

type FieldDescriptor struct {
	Field              string   `json:"field"`
	FieldType          string   `json:"fieldType"`
	Nullable           bool     `json:"nullable"`
	SupportedOperators []string `json:"supportedOperators"`
	Widget             string   `json:"widget"`
	FacetEligible      bool     `json:"facetEligible"`
	Description        string   `json:"description,omitempty"`
}

type GetFieldsResponse struct {
	DatasourceID string            `json:"datasourceId"`
	Source       SourceType        `json:"source"`
	Fields       []FieldDescriptor `json:"fields"`
}

type FacetValue struct {
	Value  interface{} `json:"value"`
	Label  string      `json:"label"`
	Count  int         `json:"count"`
	IsNull bool        `json:"isNull"`
}

type GetFacetsResponse struct {
	DatasourceID string       `json:"datasourceId"`
	Source       SourceType   `json:"source"`
	Field        string       `json:"field"`
	Values       []FacetValue `json:"values"`
	Truncated    bool         `json:"truncated"`
}

type QueryPreviewRequest struct {
	Source         SourceType            `json:"source"`
	MaxItems       int                   `json:"maxItems"`
	RandomSeedGUID string                `json:"randomSeedGuid,omitempty"`
	Request        filters.FilterRequest `json:"request"`
}

type QueryPreviewResponse struct {
	Source           SourceType                   `json:"source"`
	CompiledWhereSQL string                       `json:"compiledWhereSql"`
	CompiledArgs     []interface{}                `json:"compiledArgs"`
	AllowedFields    filters.AllowedFieldResponse `json:"allowedFields"`
	DataSet          filters.DataSetResponse      `json:"dataSet"`
}
