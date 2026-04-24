package filters

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const specificDatasourceSchemaPath = "internal/json/TestDataSet_Response_For_Specific_Datasource_From_TestDataEngine.json-schema.json"

type SchemaField struct {
	CanonicalName      string
	FieldType          string
	Nullable           bool
	SupportedOperators map[string]struct{}
	Description        string
}

type SchemaFieldCatalog struct {
	SchemaName string
	Fields     map[string]SchemaField
	Order      []string
}

func LoadSchemaFieldCatalog(schemaPath string) (SchemaFieldCatalog, error) {
	resolved, err := resolveFilterSchemaPath(schemaPath)
	if err != nil {
		return SchemaFieldCatalog{}, err
	}
	payload, err := os.ReadFile(resolved)
	if err != nil {
		return SchemaFieldCatalog{}, fmt.Errorf("read schema %q: %w", schemaPath, err)
	}
	return ParseSchemaFieldCatalog(payload, filepath.Base(resolved))
}

func ParseSchemaFieldCatalog(schemaJSON []byte, schemaName string) (SchemaFieldCatalog, error) {
	var root map[string]interface{}
	if err := json.Unmarshal(schemaJSON, &root); err != nil {
		return SchemaFieldCatalog{}, fmt.Errorf("decode schema JSON: %w", err)
	}

	defs, ok := root["$defs"].(map[string]interface{})
	if !ok {
		return SchemaFieldCatalog{}, fmt.Errorf("schema %q missing $defs", schemaName)
	}
	itemDef, ok := defs["TestDataSetItem"].(map[string]interface{})
	if !ok {
		return SchemaFieldCatalog{}, fmt.Errorf("schema %q missing $defs.TestDataSetItem", schemaName)
	}
	properties, ok := itemDef["properties"].(map[string]interface{})
	if !ok {
		return SchemaFieldCatalog{}, fmt.Errorf("schema %q missing TestDataSetItem.properties", schemaName)
	}

	requiredSet := make(map[string]struct{})
	requiredOrder := make([]string, 0)
	if required, ok := itemDef["required"].([]interface{}); ok {
		for _, item := range required {
			name, ok := item.(string)
			if !ok {
				continue
			}
			requiredSet[name] = struct{}{}
			requiredOrder = append(requiredOrder, name)
		}
	}

	fields := make(map[string]SchemaField, len(properties))
	extraOrder := make([]string, 0)
	for name, raw := range properties {
		prop, ok := raw.(map[string]interface{})
		if !ok {
			return SchemaFieldCatalog{}, fmt.Errorf("schema %q property %q is not an object", schemaName, name)
		}
		fieldType, nullable, err := parseSchemaFieldType(prop)
		if err != nil {
			return SchemaFieldCatalog{}, fmt.Errorf("schema %q field %q: %w", schemaName, name, err)
		}
		description, _ := prop["description"].(string)
		fields[name] = SchemaField{
			CanonicalName:      name,
			FieldType:          fieldType,
			Nullable:           nullable,
			SupportedOperators: SupportedOperatorsForSchemaType(fieldType),
			Description:        description,
		}
		if _, isRequired := requiredSet[name]; !isRequired {
			extraOrder = append(extraOrder, name)
		}
	}
	sort.Strings(extraOrder)

	order := make([]string, 0, len(requiredOrder)+len(extraOrder))
	order = append(order, requiredOrder...)
	order = append(order, extraOrder...)

	return SchemaFieldCatalog{
		SchemaName: schemaName,
		Fields:     fields,
		Order:      order,
	}, nil
}

func BuildFieldDefinitionsFromCatalog(catalog SchemaFieldCatalog) map[string]FieldDefinition {
	fields := make(map[string]FieldDefinition, len(catalog.Fields))
	for name, field := range catalog.Fields {
		fields[name] = FieldDefinition{
			FieldType:          field.FieldType,
			Nullable:           field.Nullable,
			SupportedOperators: cloneOperatorSet(field.SupportedOperators),
			Description:        field.Description,
		}
	}
	return fields
}

func NormalizeRowToCanonical(row map[string]interface{}, catalog SchemaFieldCatalog) map[string]interface{} {
	if row == nil {
		return nil
	}
	out := make(map[string]interface{}, len(row))
	for key, value := range row {
		out[CanonicalFieldName(key, catalog)] = value
	}
	return out
}

func CanonicalFieldName(name string, catalog SchemaFieldCatalog) string {
	if _, ok := catalog.Fields[name]; ok {
		return name
	}

	normalized := normalizeSchemaFieldLookup(name)
	if normalized == "" {
		return name
	}

	var exactMatch string
	for fieldName := range catalog.Fields {
		if normalizeSchemaFieldLookup(fieldName) == normalized {
			if exactMatch != "" {
				return name
			}
			exactMatch = fieldName
		}
	}
	if exactMatch != "" {
		return exactMatch
	}

	bestName := ""
	bestDistance := -1
	secondBestDistance := -1
	for fieldName := range catalog.Fields {
		distance := levenshteinDistance(normalized, normalizeSchemaFieldLookup(fieldName))
		if bestDistance == -1 || distance < bestDistance {
			secondBestDistance = bestDistance
			bestDistance = distance
			bestName = fieldName
			continue
		}
		if secondBestDistance == -1 || distance < secondBestDistance {
			secondBestDistance = distance
		}
	}

	if bestDistance >= 0 && bestDistance <= 2 && (secondBestDistance == -1 || bestDistance < secondBestDistance) {
		return bestName
	}
	return name
}

func SupportedOperatorsForSchemaType(fieldType string) map[string]struct{} {
	return supportedOperatorsForType(fieldType)
}

func parseSchemaFieldType(prop map[string]interface{}) (string, bool, error) {
	if rawType, ok := prop["type"].(string); ok {
		if _, exists := validFieldTypes[rawType]; !exists {
			return "", false, fmt.Errorf("unsupported schema type %q", rawType)
		}
		return rawType, false, nil
	}

	oneOf, ok := prop["oneOf"].([]interface{})
	if !ok || len(oneOf) == 0 {
		return "", false, fmt.Errorf("missing supported type declaration")
	}

	nullable := false
	fieldType := ""
	for _, rawOption := range oneOf {
		option, ok := rawOption.(map[string]interface{})
		if !ok {
			return "", false, fmt.Errorf("invalid oneOf option")
		}
		t, _ := option["type"].(string)
		switch t {
		case "null":
			nullable = true
		case "":
			continue
		default:
			if _, exists := validFieldTypes[t]; !exists {
				return "", false, fmt.Errorf("unsupported schema type %q", t)
			}
			fieldType = t
		}
	}
	if fieldType == "" {
		return "", false, fmt.Errorf("no non-null field type found")
	}
	return fieldType, nullable, nil
}

func cloneOperatorSet(in map[string]struct{}) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for key := range in {
		out[key] = struct{}{}
	}
	return out
}

func normalizeSchemaFieldLookup(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + ('a' - 'A'))
		}
	}
	return b.String()
}

func levenshteinDistance(left, right string) int {
	if left == right {
		return 0
	}
	if len(left) == 0 {
		return len(right)
	}
	if len(right) == 0 {
		return len(left)
	}

	prev := make([]int, len(right)+1)
	curr := make([]int, len(right)+1)
	for j := 0; j <= len(right); j++ {
		prev[j] = j
	}

	for i := 1; i <= len(left); i++ {
		curr[0] = i
		for j := 1; j <= len(right); j++ {
			cost := 0
			if left[i-1] != right[j-1] {
				cost = 1
			}
			insertion := curr[j-1] + 1
			deletion := prev[j] + 1
			substitution := prev[j-1] + cost
			curr[j] = minInt(insertion, minInt(deletion, substitution))
		}
		prev, curr = curr, prev
	}

	return prev[len(right)]
}

func resolveFilterSchemaPath(schemaPath string) (string, error) {
	candidates := []string{
		schemaPath,
		filepath.Join("..", "..", schemaPath),
	}

	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates,
			filepath.Join(wd, schemaPath),
			filepath.Join(wd, "..", "..", schemaPath),
		)
	}

	for _, candidate := range candidates {
		clean := filepath.Clean(candidate)
		if _, err := os.Stat(clean); err == nil {
			abs, err := filepath.Abs(clean)
			if err != nil {
				return "", fmt.Errorf("resolve schema path %q: %w", clean, err)
			}
			return abs, nil
		}
	}

	return "", fmt.Errorf("resolve schema path %q: no matching file found", schemaPath)
}

func loadSpecificDatasourceSchemaCatalog() (SchemaFieldCatalog, error) {
	return LoadSchemaFieldCatalog(specificDatasourceSchemaPath)
}

func schemaCatalogForDataSource(schemaMeta *DataSetSchemaMetadata) (SchemaFieldCatalog, error) {
	var (
		catalog SchemaFieldCatalog
		err     error
	)

	if schemaMeta != nil && len(schemaMeta.JsonSchema) > 0 {
		catalog, err = schemaCatalogFromJSON(schemaMeta.JsonSchemaName, schemaMeta.JsonSchema)
		if err == nil && len(catalog.Fields) == 0 {
			err = fmt.Errorf("schema metadata did not define any fields")
		}
	}
	if err != nil || len(catalog.Fields) == 0 {
		catalog, err = loadSpecificDatasourceSchemaCatalog()
	}
	if err != nil {
		return SchemaFieldCatalog{}, err
	}
	return catalog, nil
}

func schemaDataSourceDefinition(catalog SchemaFieldCatalog, dataSourceUUID string) DataSourceDefinition {
	return DataSourceDefinition{
		UUID:   dataSourceUUID,
		Fields: BuildFieldDefinitionsFromCatalog(catalog),
	}
}

func normalizeRowsToCanonical(rows []map[string]interface{}, catalog SchemaFieldCatalog) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		out = append(out, NormalizeRowToCanonical(row, catalog))
	}
	return out
}

func schemaCatalogFromJSON(schemaName string, schemaJSON []byte) (SchemaFieldCatalog, error) {
	base := filepath.Base(strings.TrimSpace(schemaName))
	if base == "" {
		base = filepath.Base(specificDatasourceSchemaPath)
	}
	return ParseSchemaFieldCatalog(schemaJSON, base)
}
