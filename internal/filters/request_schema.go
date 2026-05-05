package filters

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

const requestFilterSchemaPath = "internal/json/TestDataSet_Request_Filter_To_TestDataEngine.json-schema.json"

var (
	requestSchemaVersionOnce sync.Once
	requestSchemaVersion     string
	requestSchemaVersionErr  error
)

func RequestSchemaVersion() (string, error) {
	requestSchemaVersionOnce.Do(func() {
		requestSchemaVersion, requestSchemaVersionErr = loadRequestSchemaVersion(requestFilterSchemaPath)
	})
	if requestSchemaVersionErr != nil {
		return "", requestSchemaVersionErr
	}
	return requestSchemaVersion, nil
}

func loadRequestSchemaVersion(schemaPath string) (string, error) {
	resolved, err := resolveFilterSchemaPath(schemaPath)
	if err != nil {
		return "", err
	}
	payload, err := os.ReadFile(resolved)
	if err != nil {
		return "", fmt.Errorf("read request schema %q: %w", schemaPath, err)
	}

	var doc struct {
		Properties struct {
			SchemaVersion struct {
				Enum []string `json:"enum"`
			} `json:"SchemaVersion"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(payload, &doc); err != nil {
		return "", fmt.Errorf("decode request schema JSON: %w", err)
	}
	if len(doc.Properties.SchemaVersion.Enum) == 0 {
		return "", fmt.Errorf("request schema %q missing properties.SchemaVersion.enum", schemaPath)
	}
	return doc.Properties.SchemaVersion.Enum[0], nil
}

func validateRequestSchemaVersion(schemaVersion string) error {
	expected, err := RequestSchemaVersion()
	if err != nil {
		return fmt.Errorf("load request schema version: %w", err)
	}
	if schemaVersion != expected {
		return fmt.Errorf("unsupported SchemaVersion: %q", schemaVersion)
	}
	return nil
}
