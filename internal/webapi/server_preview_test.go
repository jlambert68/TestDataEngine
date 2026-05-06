package webapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"TestDataEngine/internal/filters"
)

type previewStubQueryService struct {
	previewCalled bool
	lastCfg       DataSourceConfig
	lastReq       QueryPreviewRequest
	previewResp   QueryPreviewResponse
	previewErr    error
}

func (s *previewStubQueryService) Describe(source SourceType, cfg DataSourceConfig, req filters.FilterRequest) (filters.AllowedFieldResponse, error) {
	return filters.AllowedFieldResponse{}, nil
}

func (s *previewStubQueryService) Preview(cfg DataSourceConfig, in QueryPreviewRequest) (QueryPreviewResponse, error) {
	s.previewCalled = true
	s.lastCfg = cfg
	s.lastReq = in
	if s.previewErr != nil {
		return QueryPreviewResponse{}, s.previewErr
	}
	return s.previewResp, nil
}

type previewStubFacetService struct{}

func (s *previewStubFacetService) Values(source SourceType, cfg DataSourceConfig, req filters.FilterRequest, field string, limit int, q string) ([]FacetValue, bool, error) {
	return nil, false, nil
}

func TestHandlePreviewRejectsNegativeRandomSeedOffset(t *testing.T) {
	t.Parallel()

	queryStub := &previewStubQueryService{}
	server := NewServer(ServerDeps{
		Catalog:      StaticCatalog(),
		QueryService: queryStub,
		FacetService: &previewStubFacetService{},
		StaticDir:    t.TempDir(),
	})

	payload := map[string]any{
		"source":           "sqlite",
		"maxItems":         10,
		"randomSeedGuid":   "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		"randomSeedOffset": -1,
		"request": map[string]any{
			"SchemaVersion":  "1.0",
			"RequestUuid":    "11111111-1111-4111-8111-111111111111",
			"DataSourceUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
			"DataSourceName": "SubCustody",
			"RequestFilter": map[string]any{
				"field": "AccountCurrency",
				"op":    "eq",
				"value": "SEK",
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/query/preview", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	if queryStub.previewCalled {
		t.Fatal("expected query service preview to not be called")
	}

	var apiErr APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("unmarshal APIError: %v", err)
	}
	if apiErr.Error != "invalid randomSeedOffset" {
		t.Fatalf("unexpected error code: %#v", apiErr)
	}
}

func TestHandlePreviewRejectsOffsetWithoutGUID(t *testing.T) {
	t.Parallel()

	queryStub := &previewStubQueryService{}
	server := NewServer(ServerDeps{
		Catalog:      StaticCatalog(),
		QueryService: queryStub,
		FacetService: &previewStubFacetService{},
		StaticDir:    t.TempDir(),
	})

	payload := map[string]any{
		"source":           "sqlite",
		"maxItems":         10,
		"randomSeedOffset": 3,
		"request": map[string]any{
			"SchemaVersion":  "1.0",
			"RequestUuid":    "11111111-1111-4111-8111-111111111111",
			"DataSourceUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
			"DataSourceName": "SubCustody",
			"RequestFilter": map[string]any{
				"field": "AccountCurrency",
				"op":    "eq",
				"value": "SEK",
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/query/preview", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
	if queryStub.previewCalled {
		t.Fatal("expected query service preview to not be called")
	}

	var apiErr APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("unmarshal APIError: %v", err)
	}
	if apiErr.Error != "invalid randomSeedOffset" {
		t.Fatalf("unexpected error code: %#v", apiErr)
	}
}

func TestHandlePreviewForwardsGuidAndOffsetToQueryService(t *testing.T) {
	t.Parallel()

	queryStub := &previewStubQueryService{
		previewResp: QueryPreviewResponse{
			Source:           SourceSQLite,
			CompiledWhereSQL: "(\"AccountCurrency\" = ?)",
			CompiledArgs:     []interface{}{"SEK"},
		},
	}
	server := NewServer(ServerDeps{
		Catalog:      StaticCatalog(),
		QueryService: queryStub,
		FacetService: &previewStubFacetService{},
		StaticDir:    t.TempDir(),
	})

	payload := map[string]any{
		"source":           "sqlite",
		"maxItems":         10,
		"randomSeedGuid":   "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		"randomSeedOffset": 7,
		"request": map[string]any{
			"SchemaVersion":  "1.0",
			"RequestUuid":    "11111111-1111-4111-8111-111111111111",
			"DataSourceUuid": "110cc994-a913-4041-96fe-a96d7e0c97e8",
			"DataSourceName": "SubCustody",
			"RequestFilter": map[string]any{
				"field": "AccountCurrency",
				"op":    "eq",
				"value": "SEK",
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/query/preview", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !queryStub.previewCalled {
		t.Fatal("expected query service preview to be called")
	}
	if queryStub.lastReq.RandomSeedGUID != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" {
		t.Fatalf("unexpected forwarded randomSeedGuid: %q", queryStub.lastReq.RandomSeedGUID)
	}
	if queryStub.lastReq.RandomSeedOffset != 7 {
		t.Fatalf("unexpected forwarded randomSeedOffset: %d", queryStub.lastReq.RandomSeedOffset)
	}
}
