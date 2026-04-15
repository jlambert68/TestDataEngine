package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"TestDataEngine/internal/filters"
	"TestDataEngine/internal/logging"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

const (
	requestSchemaPath      = "internal/json/TestDataSet_Request_Filter_To_TestDataEngine.json-schema.json"
	responseSchemaDir      = "internal/json"
	specificResponseSchema = filters.SpecificDatasourceResponseSchemaName
)

type sourceResult struct {
	Source      string
	Compiled    filters.CompiledFilter
	AllowedResp filters.AllowedFieldResponse
	DataResp    filters.DataSetResponse
}

// main runs a sample filter request and prints both metadata and matching data rows.
func main() {
	// Embedded example request used by default when running the binary directly.
	filterReqJSON := []byte(`{
	  "SchemaVersion": "1.0",
	  "RequestUuid": "6e6e17c4-6cc0-4ef0-a1cf-e96f0c5f8b8f",
	  "DataSourceUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
	  "DataSourceName": "SubCustody",
	  "RequestFilter": {
	    "and": [
	      {
	        "field": "AccountCurrency",
	        "op": "eq",
	        "value": "SEK"
	      },
	      {
	        "field": "AccountEnvironment",
	        "op": "eq",
	        "value": "SysTest"
	      },
	      {
	        "field": "ClientJuristictionCountryCode",
	        "op": "eq",
	        "value": "SE"
	      }
	    ]
	  }
	}`)

	var req filters.FilterRequest
	if err := json.Unmarshal(filterReqJSON, &req); err != nil {
		logging.Fatalf("7e7e323b-a072-4af3-8608-95316cce6fe2", "failed to unmarshal filter request: %v", err)
	}

	var (
		sourceType          string
		csvPath             string
		sqliteDB            string
		sqliteTable         string
		postgresDSN         string
		postgresTable       string
		postgresSchemaTable string
		maxItems            int
		randomGUID          string
	)
	flag.StringVar(&sourceType, "source", "csv", "Data source type: csv, sqlite, postgres, or all")
	flag.StringVar(&csvPath, "csv", "p26_2/FenixRawTestdata_646rows_211220_stripped.csv", "Path to CSV input file")
	flag.StringVar(&sqliteDB, "sqlite-db", "testdata/SQLiteDB/identifier.sqlite", "Path to SQLite DB file")
	flag.StringVar(&sqliteTable, "sqlite-table", "main.data_items", "SQLite table containing JsonData")
	flag.StringVar(&postgresDSN, "postgres-dsn", "", "Postgres connection string")
	flag.StringVar(&postgresTable, "postgres-table", "public.data_items", "Postgres table containing JsonData")
	flag.StringVar(&postgresSchemaTable, "postgres-schema-table", "public.testdataset_response_schemas", "Postgres table containing response schema metadata")
	flag.IntVar(&maxItems, "max-items", 2, "Maximum number of matched rows to return (0=all)")
	flag.StringVar(&randomGUID, "random-seed-guid", "", "Optional GUID used as deterministic shuffle seed")
	flag.Parse()

	// Negative values are treated as unbounded to avoid surprising hard failures.
	if maxItems < 0 {
		maxItems = 0
	}

	if err := validateRequestSchema(filterReqJSON); err != nil {
		logging.Fatalf("9f8abf12-6d9c-4f47-a932-35e6ac4e0db6", "request schema validation failed: %v", err)
	}

	sources, err := resolveSourceTypes(sourceType)
	if err != nil {
		logging.Fatalf("a72f852f-bf0a-40de-bbe7-b54225095f20", "%v", err)
	}

	for _, source := range sources {
		result, err := querySource(req, source, csvPath, sqliteDB, sqliteTable, postgresDSN, postgresTable, postgresSchemaTable, maxItems, randomGUID)
		if err != nil {
			logging.Fatalf("eb34afc2-53bb-4bcb-bd0d-1a4918453442", "failed to query %s datasource: %v", source, err)
		}
		logSourceResult(req, result)
	}
}

func resolveSourceTypes(sourceType string) ([]string, error) {
	normalized := strings.ToLower(strings.TrimSpace(sourceType))
	switch normalized {
	case "all":
		return []string{"csv", "sqlite", "postgres"}, nil
	case "csv", "sqlite", "postgres":
		return []string{normalized}, nil
	default:
		return nil, fmt.Errorf("unsupported source type %q (expected csv, sqlite, postgres, or all)", sourceType)
	}
}

func querySource(
	req filters.FilterRequest,
	source string,
	csvPath string,
	sqliteDB string,
	sqliteTable string,
	postgresDSN string,
	postgresTable string,
	postgresSchemaTable string,
	maxItems int,
	randomGUID string,
) (sourceResult, error) {
	switch source {
	case "csv":
		// Keep backward compatibility with both lowercase and uppercase data folders.
		if _, statErr := os.Stat(csvPath); statErr != nil && csvPath == "p26_2/FenixRawTestdata_646rows_211220_stripped.csv" {
			csvPath = "P26_2/FenixRawTestdata_646rows_211220_stripped.csv"
		}
		compiled, allowedResp, dataResp, err := filters.QueryCSVDataSourceWithSeed(req, csvPath, maxItems, randomGUID)
		if err != nil {
			return sourceResult{}, err
		}
		dataResp, err = applyLocalResponseSchemaMetadata(req, dataResp, specificResponseSchema)
		if err != nil {
			return sourceResult{}, fmt.Errorf("response schema metadata enrichment failed: %w", err)
		}
		if err := validateCSVResponseSchema(dataResp); err != nil {
			return sourceResult{}, fmt.Errorf("response schema validation failed: %w", err)
		}
		logging.Infof("8428b438-123c-40d8-a7ca-ff5b4e87f832", "Source=csv CSV=%s RandomSeedGuid=%s", csvPath, randomGUID)
		return sourceResult{
			Source:      source,
			Compiled:    compiled,
			AllowedResp: allowedResp,
			DataResp:    dataResp,
		}, nil

	case "sqlite":
		compiled, allowedResp, dataResp, err := filters.QuerySQLiteDataSourceWithSeed(req, sqliteDB, sqliteTable, maxItems, randomGUID)
		if err != nil {
			return sourceResult{}, err
		}
		if err := validateSQLiteResponseSchema(dataResp); err != nil {
			return sourceResult{}, fmt.Errorf("response schema validation failed: %w", err)
		}
		logging.Infof("3fd182f4-3d81-4225-b89f-f2dc959fc8ba", "Source=sqlite DB=%s Table=%s RandomSeedGuid=%s", sqliteDB, sqliteTable, randomGUID)
		return sourceResult{
			Source:      source,
			Compiled:    compiled,
			AllowedResp: allowedResp,
			DataResp:    dataResp,
		}, nil

	case "postgres":
		compiled, allowedResp, dataResp, err := filters.QueryPostgresDataSourceWithSeed(
			req,
			postgresDSN,
			postgresTable,
			postgresSchemaTable,
			maxItems,
			randomGUID,
		)
		if err != nil {
			return sourceResult{}, err
		}
		if err := validateSQLiteResponseSchema(dataResp); err != nil {
			return sourceResult{}, fmt.Errorf("response schema validation failed: %w", err)
		}
		logging.Infof(
			"5f795883-5799-48d3-ae27-8d491111d9c0",
			"Source=postgres Table=%s SchemaTable=%s RandomSeedGuid=%s",
			postgresTable,
			postgresSchemaTable,
			randomGUID,
		)
		return sourceResult{
			Source:      source,
			Compiled:    compiled,
			AllowedResp: allowedResp,
			DataResp:    dataResp,
		}, nil
	default:
		return sourceResult{}, fmt.Errorf("unsupported source type %q", source)
	}
}

func logSourceResult(req filters.FilterRequest, result sourceResult) {
	// Log compiled SQL representation and response payloads for traceability.
	logging.Infof("35579f2f-4de2-4cc2-bf0a-bf579f31cf64", "Source=%s WHERE=%s", result.Source, result.Compiled.WhereSQL)
	logging.Infof("37f52f2f-fb8f-47dd-bf14-17c3a194ddbc", "Source=%s ARGS=%v", result.Source, result.Compiled.Args)

	allowedWithInputFilter := struct {
		Source                string                       `json:"Source"`
		InputFilter           json.RawMessage              `json:"InputFilter"`
		AllowedFieldsResponse filters.AllowedFieldResponse `json:"AllowedFieldsResponse"`
	}{
		Source:                result.Source,
		InputFilter:           req.RequestFilter,
		AllowedFieldsResponse: result.AllowedResp,
	}
	allowedPretty, err := json.MarshalIndent(allowedWithInputFilter, "", "  ")
	if err != nil {
		logging.Fatalf("2e8c5ee6-241d-4f65-b82a-0877eef3644d", "failed to marshal allowed fields response: %v", err)
	}
	logging.Infof("15f177af-c4dc-4e86-a4e5-4f20fdf001d3", "AllowedFieldsResponse=%s", string(allowedPretty))

	dataWithInputFilter := struct {
		Source          string                  `json:"Source"`
		InputFilter     json.RawMessage         `json:"InputFilter"`
		DataSetResponse filters.DataSetResponse `json:"DataSetResponse"`
	}{
		Source:          result.Source,
		InputFilter:     req.RequestFilter,
		DataSetResponse: result.DataResp,
	}
	dataPretty, err := json.MarshalIndent(dataWithInputFilter, "", "  ")
	if err != nil {
		logging.Fatalf("9efef5d2-f500-450f-929f-890f4d89f777", "failed to marshal data response: %v", err)
	}
	logging.Infof("70e0f6f2-72fd-42bf-9f0e-f9afca6ebc52", "DataSetResponse=%s", string(dataPretty))
}

func validateSQLiteRequestSchema(raw []byte) error {
	return validateRequestSchema(raw)
}

func validateRequestSchema(raw []byte) error {
	return validateJSONAgainstSchemaFile(requestSchemaPath, raw)
}

func validateSQLiteResponseSchema(resp filters.DataSetResponse) error {
	schemaName := filters.CanonicalResponseSchemaName(resp.JsonSchemaName)
	if schemaName == "" {
		return fmt.Errorf("missing JsonSchemaName in DataSetResponse")
	}

	schemaPath := filepath.Join(responseSchemaDir, schemaName)
	payload, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("marshal response payload for schema validation: %w", err)
	}
	return validateJSONAgainstSchemaFile(schemaPath, payload)
}

func validateCSVResponseSchema(resp filters.DataSetResponse) error {
	return validateSQLiteResponseSchema(resp)
}

func validateJSONAgainstSchemaFile(schemaPath string, payload []byte) error {
	resolvedSchemaPath, err := resolveSchemaPath(schemaPath)
	if err != nil {
		return err
	}

	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(resolvedSchemaPath)
	if err != nil {
		return fmt.Errorf("compile schema %q: %w", schemaPath, err)
	}

	var doc interface{}
	if err := json.Unmarshal(payload, &doc); err != nil {
		return fmt.Errorf("decode payload JSON: %w", err)
	}
	if err := schema.Validate(doc); err != nil {
		return fmt.Errorf("validate payload against %q: %w", schemaPath, err)
	}
	return nil
}

func resolveSchemaPath(schemaPath string) (string, error) {
	candidates := []string{
		schemaPath,
		filepath.Join("..", "..", schemaPath),
	}

	if _, thisFile, _, ok := runtime.Caller(0); ok {
		baseDir := filepath.Dir(thisFile)
		candidates = append(candidates, filepath.Join(baseDir, "..", "..", schemaPath))
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

func applyLocalResponseSchemaMetadata(req filters.FilterRequest, resp filters.DataSetResponse, schemaName string) (filters.DataSetResponse, error) {
	schemaBase := filters.CanonicalResponseSchemaName(schemaName)
	schemaPath := filepath.Join(responseSchemaDir, schemaBase)
	resolved, err := resolveSchemaPath(schemaPath)
	if err != nil {
		return resp, err
	}

	rawSchema, err := os.ReadFile(resolved)
	if err != nil {
		return resp, fmt.Errorf("read response schema %q: %w", schemaPath, err)
	}
	if !json.Valid(rawSchema) {
		return resp, fmt.Errorf("response schema %q does not contain valid JSON", schemaPath)
	}

	resp.TestDataSourceName = req.DataSourceName
	resp.TestDataSourceUUID = req.DataSourceUUID
	resp.JsonSchemaName = schemaBase
	resp.JsonSchema = json.RawMessage(rawSchema)
	resp.UpdatedDateTime = time.Now().UTC().Format(time.RFC3339)
	return resp, nil
}
