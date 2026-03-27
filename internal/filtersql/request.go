package filtersql

import (
	"encoding/json"
	"fmt"
)

const SchemaVersion = "1.0"

// Request is the incoming filter contract used by the SQL compiler package.
type Request struct {
	SchemaVersion  string     `json:"SchemaVersion"`
	RequestUUID    string     `json:"RequestUuid"`
	DataSourceUUID string     `json:"DataSourceUuid"`
	DataSourceName string     `json:"DataSourceName"`
	RequestFilter  Expression `json:"RequestFilter"`
}

// UnmarshalJSON parses RequestFilter as a typed Expression tree.
func (r *Request) UnmarshalJSON(data []byte) error {
	type alias struct {
		SchemaVersion  string          `json:"SchemaVersion"`
		RequestUUID    string          `json:"RequestUuid"`
		DataSourceUUID string          `json:"DataSourceUuid"`
		DataSourceName string          `json:"DataSourceName"`
		RequestFilter  json.RawMessage `json:"RequestFilter"`
	}

	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	expr, err := ParseExpression(aux.RequestFilter)
	if err != nil {
		return fmt.Errorf("invalid RequestFilter: %w", err)
	}

	r.SchemaVersion = aux.SchemaVersion
	r.RequestUUID = aux.RequestUUID
	r.DataSourceUUID = aux.DataSourceUUID
	r.DataSourceName = aux.DataSourceName
	r.RequestFilter = expr

	return nil
}
