package filters

import (
	"encoding/binary"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

type DataSetResponse struct {
	DataSourceName string                   `json:"DataSourceName"`
	DataSourceUUID string                   `json:"DataSourceUuid"`
	Data           []map[string]interface{} `json:"Data"`
}

// QueryCSVDataSource loads rows from a CSV file, applies RequestFilter, and returns matches.
func QueryCSVDataSource(req FilterRequest, csvPath string, maxItems int) (CompiledFilter, AllowedFieldResponse, DataSetResponse, error) {
	return QueryCSVDataSourceWithSeed(req, csvPath, maxItems, "")
}

// QueryCSVDataSourceWithSeed behaves like QueryCSVDataSource but allows deterministic shuffling
// when randomSeedGUID is set.
func QueryCSVDataSourceWithSeed(
	req FilterRequest,
	csvPath string,
	maxItems int,
	randomSeedGUID string,
) (CompiledFilter, AllowedFieldResponse, DataSetResponse, error) {
	if err := validateRequestEnvelope(req); err != nil {
		return CompiledFilter{}, AllowedFieldResponse{}, DataSetResponse{}, err
	}
	if csvPath == "" {
		return CompiledFilter{}, AllowedFieldResponse{}, DataSetResponse{}, errors.New("csv path is required")
	}
	if maxItems < 0 {
		maxItems = 0
	}

	ds, rows, err := loadCSVDataSource(req.DataSourceUUID, csvPath)
	if err != nil {
		return CompiledFilter{}, AllowedFieldResponse{}, DataSetResponse{}, err
	}

	return queryDataRows(req, ds, rows, "csv", maxItems, randomSeedGUID)
}

// queryDataRows runs the shared filtering pipeline used by CSV and SQLite sources.
func queryDataRows(
	req FilterRequest,
	ds DataSourceDefinition,
	rows []map[string]interface{},
	sourceLabel string,
	maxItems int,
	randomSeedGUID string,
) (CompiledFilter, AllowedFieldResponse, DataSetResponse, error) {
	compiled, err := compileRequestForDataSource(req, ds)
	if err != nil {
		return CompiledFilter{}, AllowedFieldResponse{}, DataSetResponse{}, err
	}

	allowed, err := allowedFieldsForDataSource(req, ds)
	if err != nil {
		return CompiledFilter{}, AllowedFieldResponse{}, DataSetResponse{}, err
	}

	filtered := make([]map[string]interface{}, 0, len(rows))
	for i, row := range rows {
		matched, err := evaluateExpression(req.RequestFilter, row, ds.Fields)
		if err != nil {
			return CompiledFilter{}, AllowedFieldResponse{}, DataSetResponse{}, fmt.Errorf("evaluate %s row %d: %w", sourceLabel, i+1, err)
		}
		if !matched {
			continue
		}
		filtered = append(filtered, row)
	}

	// Shuffle before limiting so repeated requests do not always return identical first rows.
	if err := shuffleRowsWithGUID(filtered, randomSeedGUID); err != nil {
		return CompiledFilter{}, AllowedFieldResponse{}, DataSetResponse{}, err
	}
	if maxItems > 0 && len(filtered) > maxItems {
		filtered = filtered[:maxItems]
	}

	return compiled, allowed, DataSetResponse{
		DataSourceName: req.DataSourceName,
		DataSourceUUID: req.DataSourceUUID,
		Data:           filtered,
	}, nil
}

// shuffleRows randomizes row order in-place.
func shuffleRows(rows []map[string]interface{}) {
	_ = shuffleRowsWithGUID(rows, "")
}

// shuffleRowsWithGUID randomizes row order in-place and can use a deterministic GUID seed.
func shuffleRowsWithGUID(rows []map[string]interface{}, randomSeedGUID string) error {
	if len(rows) < 2 {
		return nil
	}
	r, err := seededRand(randomSeedGUID)
	if err != nil {
		return err
	}
	r.Shuffle(len(rows), func(i, j int) {
		rows[i], rows[j] = rows[j], rows[i]
	})
	return nil
}

// seededRand returns deterministic randomness when randomSeedGUID is set.
func seededRand(randomSeedGUID string) (*rand.Rand, error) {
	guid := strings.TrimSpace(randomSeedGUID)
	if guid == "" {
		return rand.New(rand.NewSource(time.Now().UnixNano())), nil
	}
	if !isUUID(guid) {
		return nil, fmt.Errorf("invalid random seed guid: %q", randomSeedGUID)
	}

	normalized := strings.ReplaceAll(strings.ToLower(guid), "-", "")
	decoded, err := hex.DecodeString(normalized)
	if err != nil || len(decoded) < 8 {
		h := fnv.New64a()
		_, _ = h.Write([]byte(guid))
		return rand.New(rand.NewSource(int64(h.Sum64()))), nil
	}
	seed := int64(binary.BigEndian.Uint64(decoded[:8]))
	return rand.New(rand.NewSource(seed)), nil
}

// compileRequestForDataSource compiles the request against a concrete inferred datasource.
func compileRequestForDataSource(req FilterRequest, ds DataSourceDefinition) (CompiledFilter, error) {
	if err := validateRequestEnvelope(req); err != nil {
		return CompiledFilter{}, err
	}
	if !strings.EqualFold(ds.UUID, req.DataSourceUUID) {
		return CompiledFilter{}, fmt.Errorf("DataSourceUuid mismatch for %q", req.DataSourceName)
	}
	sql, args, err := compileExpression(req.RequestFilter, ds.Fields)
	if err != nil {
		return CompiledFilter{}, err
	}
	return CompiledFilter{
		WhereSQL: sql,
		Args:     args,
	}, nil
}

// allowedFieldsForDataSource builds the response payload with sorted field metadata.
func allowedFieldsForDataSource(req FilterRequest, ds DataSourceDefinition) (AllowedFieldResponse, error) {
	if req.SchemaVersion != "1.0" {
		return AllowedFieldResponse{}, fmt.Errorf("unsupported SchemaVersion: %q", req.SchemaVersion)
	}
	if !isUUID(req.RequestUUID) {
		return AllowedFieldResponse{}, fmt.Errorf("invalid RequestUuid: %q", req.RequestUUID)
	}
	if !isUUID(req.DataSourceUUID) {
		return AllowedFieldResponse{}, fmt.Errorf("invalid DataSourceUuid: %q", req.DataSourceUUID)
	}
	if !strings.EqualFold(ds.UUID, req.DataSourceUUID) {
		return AllowedFieldResponse{}, fmt.Errorf("DataSourceUuid mismatch for %q", req.DataSourceName)
	}

	items := make([]AllowedFieldResult, 0, len(ds.Fields))
	for fieldName, def := range ds.Fields {
		items = append(items, AllowedFieldResult{
			FieldName:          fieldName,
			FieldType:          def.FieldType,
			Nullable:           def.Nullable,
			SupportedOperators: sortedOperators(def.SupportedOperators),
			Description:        def.Description,
		})
	}
	sortAllowedFields(items)

	return AllowedFieldResponse{
		SchemaVersion:  req.SchemaVersion,
		RequestUUID:    req.RequestUUID,
		DataSourceUUID: req.DataSourceUUID,
		DataSourceName: req.DataSourceName,
		AllowedFields:  items,
	}, nil
}

// loadCSVDataSource infers field types/operators and parses all CSV rows into typed values.
func loadCSVDataSource(dataSourceUUID, csvPath string) (DataSourceDefinition, []map[string]interface{}, error) {
	f, err := os.Open(csvPath)
	if err != nil {
		return DataSourceDefinition{}, nil, fmt.Errorf("open csv: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.Comma = ';'
	reader.FieldsPerRecord = -1

	records, err := reader.ReadAll()
	if err != nil {
		return DataSourceDefinition{}, nil, fmt.Errorf("read csv: %w", err)
	}
	if len(records) == 0 {
		return DataSourceDefinition{}, nil, errors.New("csv is empty")
	}

	headers := normalizeHeaders(records[0])
	if len(headers) == 0 {
		return DataSourceDefinition{}, nil, errors.New("csv has no headers")
	}

	columnValues := make([][]string, len(headers))
	for _, rec := range records[1:] {
		rec = normalizeRecord(rec, len(headers))
		for i := range headers {
			columnValues[i] = append(columnValues[i], rec[i])
		}
	}

	fields := make(map[string]FieldDefinition, len(headers))
	for i, header := range headers {
		fieldType := inferFieldType(columnValues[i])
		fields[header] = FieldDefinition{
			FieldType:          fieldType,
			Nullable:           hasNullValue(columnValues[i]),
			SupportedOperators: supportedOperatorsForType(fieldType),
			Description:        fmt.Sprintf("Inferred from CSV column %q.", header),
		}
	}

	rows := make([]map[string]interface{}, 0, len(records)-1)
	for rowIdx, rec := range records[1:] {
		rec = normalizeRecord(rec, len(headers))
		row := make(map[string]interface{}, len(headers))
		for colIdx, header := range headers {
			fieldDef := fields[header]
			value, err := parseCSVValue(rec[colIdx], fieldDef.FieldType)
			if err != nil {
				return DataSourceDefinition{}, nil, fmt.Errorf("row %d column %q: %w", rowIdx+2, header, err)
			}
			row[header] = value
		}
		rows = append(rows, row)
	}

	return DataSourceDefinition{
		UUID:   dataSourceUUID,
		Fields: fields,
	}, rows, nil
}

// evaluateExpression recursively evaluates comparison/and/or/not expressions against one row.
func evaluateExpression(raw json.RawMessage, row map[string]interface{}, fields map[string]FieldDefinition) (bool, error) {
	var cmpProbe struct {
		Field *string `json:"field"`
		Op    *string `json:"op"`
	}
	if err := json.Unmarshal(raw, &cmpProbe); err == nil && cmpProbe.Field != nil && cmpProbe.Op != nil {
		var cmp Comparison
		if err := json.Unmarshal(raw, &cmp); err != nil {
			return false, fmt.Errorf("invalid comparison: %w", err)
		}
		return evaluateComparison(cmp, row, fields)
	}

	var andProbe struct {
		And *[]json.RawMessage `json:"and"`
	}
	if err := json.Unmarshal(raw, &andProbe); err == nil && andProbe.And != nil {
		if len(*andProbe.And) == 0 {
			return false, errors.New("and expression must contain at least one item")
		}
		for _, part := range *andProbe.And {
			ok, err := evaluateExpression(part, row, fields)
			if err != nil {
				return false, err
			}
			if !ok {
				return false, nil
			}
		}
		return true, nil
	}

	var orProbe struct {
		Or *[]json.RawMessage `json:"or"`
	}
	if err := json.Unmarshal(raw, &orProbe); err == nil && orProbe.Or != nil {
		if len(*orProbe.Or) == 0 {
			return false, errors.New("or expression must contain at least one item")
		}
		for _, part := range *orProbe.Or {
			ok, err := evaluateExpression(part, row, fields)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	}

	var notProbe struct {
		Not *json.RawMessage `json:"not"`
	}
	if err := json.Unmarshal(raw, &notProbe); err == nil && notProbe.Not != nil {
		ok, err := evaluateExpression(*notProbe.Not, row, fields)
		if err != nil {
			return false, err
		}
		return !ok, nil
	}

	return false, errors.New("invalid expression: must be comparison, and, or, or not")
}

// evaluateComparison applies one operator to one row value.
func evaluateComparison(c Comparison, row map[string]interface{}, fields map[string]FieldDefinition) (bool, error) {
	if _, _, err := compileComparison(c, fields); err != nil {
		return false, err
	}

	fieldDef := fields[c.Field]
	rowValue := row[c.Field]

	switch c.Op {
	case "eq":
		return valuesEqual(fieldDef.FieldType, rowValue, c.Value)
	case "neq":
		equal, err := valuesEqual(fieldDef.FieldType, rowValue, c.Value)
		return !equal, err
	case "gt":
		return compareOrdered(fieldDef.FieldType, rowValue, c.Value, func(left, right int) bool { return left > right })
	case "gte":
		return compareOrdered(fieldDef.FieldType, rowValue, c.Value, func(left, right int) bool { return left >= right })
	case "lt":
		return compareOrdered(fieldDef.FieldType, rowValue, c.Value, func(left, right int) bool { return left < right })
	case "lte":
		return compareOrdered(fieldDef.FieldType, rowValue, c.Value, func(left, right int) bool { return left <= right })
	case "contains":
		s, ok := rowValue.(string)
		if !ok {
			return false, nil
		}
		needle := c.Value.(string)
		return strings.Contains(s, needle), nil
	case "startsWith":
		s, ok := rowValue.(string)
		if !ok {
			return false, nil
		}
		prefix := c.Value.(string)
		return strings.HasPrefix(s, prefix), nil
	case "endsWith":
		s, ok := rowValue.(string)
		if !ok {
			return false, nil
		}
		suffix := c.Value.(string)
		return strings.HasSuffix(s, suffix), nil
	case "exists":
		exists := rowValue != nil
		want := c.Value.(bool)
		return exists == want, nil
	case "isNull":
		isNull := rowValue == nil
		wantNull := c.Value.(bool)
		return isNull == wantNull, nil
	case "in", "nin":
		items, _ := normalizeArray(c.Value)
		found := false
		for _, item := range items {
			eq, err := valuesEqual(fieldDef.FieldType, rowValue, item)
			if err != nil {
				return false, err
			}
			if eq {
				found = true
				break
			}
		}
		if c.Op == "in" {
			return found, nil
		}
		return !found, nil
	default:
		return false, fmt.Errorf("operator not implemented: %s", c.Op)
	}
}

// valuesEqual compares row/filter values using field-type aware conversions.
func valuesEqual(fieldType string, rowValue, filterValue interface{}) (bool, error) {
	if rowValue == nil || filterValue == nil {
		return rowValue == nil && filterValue == nil, nil
	}

	switch fieldType {
	case "number", "integer":
		left, ok := toFloat(rowValue)
		if !ok {
			return false, fmt.Errorf("row value is not numeric: %v", rowValue)
		}
		right, ok := toFloat(filterValue)
		if !ok {
			return false, fmt.Errorf("filter value is not numeric: %v", filterValue)
		}
		return left == right, nil
	case "boolean":
		left, ok := rowValue.(bool)
		if !ok {
			return false, fmt.Errorf("row value is not boolean: %v", rowValue)
		}
		right, ok := filterValue.(bool)
		if !ok {
			return false, fmt.Errorf("filter value is not boolean: %v", filterValue)
		}
		return left == right, nil
	default:
		left, ok := rowValue.(string)
		if !ok {
			return false, fmt.Errorf("row value is not string: %v", rowValue)
		}
		right, ok := filterValue.(string)
		if !ok {
			return false, fmt.Errorf("filter value is not string: %v", filterValue)
		}
		return left == right, nil
	}
}

// compareOrdered handles gt/gte/lt/lte comparisons for numeric and string-like values.
func compareOrdered(fieldType string, rowValue, filterValue interface{}, okFn func(left, right int) bool) (bool, error) {
	if rowValue == nil {
		return false, nil
	}

	switch fieldType {
	case "number", "integer":
		left, ok := toFloat(rowValue)
		if !ok {
			return false, fmt.Errorf("row value is not numeric: %v", rowValue)
		}
		right, ok := toFloat(filterValue)
		if !ok {
			return false, fmt.Errorf("filter value is not numeric: %v", filterValue)
		}
		if left < right {
			return okFn(-1, 0), nil
		}
		if left > right {
			return okFn(1, 0), nil
		}
		return okFn(0, 0), nil
	case "date", "datetime", "string":
		left, ok := rowValue.(string)
		if !ok {
			return false, fmt.Errorf("row value is not string: %v", rowValue)
		}
		right, ok := filterValue.(string)
		if !ok {
			return false, fmt.Errorf("filter value is not string: %v", filterValue)
		}
		return okFn(strings.Compare(left, right), 0), nil
	default:
		return false, fmt.Errorf("field type %q is not order-comparable", fieldType)
	}
}

// validateRequestEnvelope validates request metadata without requiring catalog lookup.
func validateRequestEnvelope(req FilterRequest) error {
	if req.SchemaVersion != "1.0" {
		return fmt.Errorf("unsupported SchemaVersion: %q", req.SchemaVersion)
	}
	if !isUUID(req.RequestUUID) {
		return fmt.Errorf("invalid RequestUuid: %q", req.RequestUUID)
	}
	if !isUUID(req.DataSourceUUID) {
		return fmt.Errorf("invalid DataSourceUuid: %q", req.DataSourceUUID)
	}
	if req.DataSourceName == "" {
		return errors.New("DataSourceName is required")
	}
	if len(req.RequestFilter) == 0 {
		return errors.New("RequestFilter is required")
	}
	return nil
}

func normalizeHeaders(rawHeaders []string) []string {
	out := make([]string, 0, len(rawHeaders))
	for i, h := range rawHeaders {
		h = strings.TrimSpace(h)
		if i == 0 {
			h = strings.TrimPrefix(h, "\ufeff")
		}
		out = append(out, h)
	}
	return out
}

func normalizeRecord(rec []string, wantLen int) []string {
	if len(rec) == wantLen {
		return rec
	}
	out := make([]string, wantLen)
	copy(out, rec)
	return out
}

func inferFieldType(values []string) string {
	hasNonNull := false
	allBoolean := true
	allInteger := true
	allNumber := true

	for _, raw := range values {
		if isCSVNull(raw) {
			continue
		}
		hasNonNull = true

		if _, err := parseBoolean(raw); err != nil {
			allBoolean = false
		}
		if _, err := parseInteger(raw); err != nil {
			allInteger = false
		}
		if _, err := parseNumber(raw); err != nil {
			allNumber = false
		}
	}

	if !hasNonNull {
		return "string"
	}
	if allBoolean {
		return "boolean"
	}
	if allInteger {
		return "integer"
	}
	if allNumber {
		return "number"
	}
	return "string"
}

func hasNullValue(values []string) bool {
	for _, v := range values {
		if isCSVNull(v) {
			return true
		}
	}
	return false
}

func supportedOperatorsForType(fieldType string) map[string]struct{} {
	switch fieldType {
	case "number", "integer", "date", "datetime":
		return toSet("eq", "neq", "gt", "gte", "lt", "lte", "in", "nin", "exists", "isNull")
	case "boolean":
		return toSet("eq", "neq", "exists", "isNull")
	default:
		return toSet("eq", "neq", "in", "nin", "contains", "startsWith", "endsWith", "exists", "isNull")
	}
}

func parseCSVValue(raw, fieldType string) (interface{}, error) {
	if isCSVNull(raw) {
		return nil, nil
	}
	switch fieldType {
	case "boolean":
		return parseBoolean(raw)
	case "integer":
		return parseInteger(raw)
	case "number":
		return parseNumber(raw)
	default:
		return strings.TrimSpace(raw), nil
	}
}

func isCSVNull(raw string) bool {
	s := strings.TrimSpace(raw)
	return s == "" || strings.EqualFold(s, "NULL")
}

func parseBoolean(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("not a boolean: %q", raw)
	}
}

func parseInteger(raw string) (int64, error) {
	s := strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	if strings.ContainsAny(s, ".eE") {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, err
		}
		if f != float64(int64(f)) {
			return 0, fmt.Errorf("not an integer: %q", raw)
		}
		return int64(f), nil
	}
	return strconv.ParseInt(s, 10, 64)
}

func parseNumber(raw string) (float64, error) {
	s := strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
	return strconv.ParseFloat(s, 64)
}

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	default:
		return 0, false
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
