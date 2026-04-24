package webapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"TestDataEngine/internal/filters"
)

type ServerDeps struct {
	Catalog      Catalog
	QueryService QueryService
	FacetService FacetService
	StaticDir    string
}

type Server struct {
	deps ServerDeps
}

func NewServer(deps ServerDeps) *Server {
	return &Server{deps: deps}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/datasources", s.handleListDataSources)
	mux.HandleFunc("GET /api/v1/datasources/{id}/fields", s.handleGetFields)
	mux.HandleFunc("GET /api/v1/datasources/{id}/facets", s.handleGetFacets)
	mux.HandleFunc("POST /api/v1/query/preview", s.handlePreview)
	mux.HandleFunc("GET /api/v1/healthz", s.handleHealthz)
	mux.Handle("/", s.serveUI())
	return mux
}

func (s *Server) handleListDataSources(w http.ResponseWriter, r *http.Request) {
	items := s.deps.Catalog.List()
	resp := ListDataSourcesResponse{
		Items: make([]DataSourceListItem, 0, len(items)),
	}
	for _, item := range items {
		resp.Items = append(resp.Items, DataSourceListItem{
			ID:               item.ID,
			Label:            item.Label,
			DataSourceName:   item.DataSourceName,
			DataSourceUUID:   item.DataSourceUUID,
			SupportedSources: item.SupportedSources,
			DefaultSource:    item.DefaultSource,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetFields(w http.ResponseWriter, r *http.Request) {
	cfg, source, ok := s.loadConfigAndSource(w, r)
	if !ok {
		return
	}
	req, err := BuildMetadataRequest(cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build metadata request", err.Error())
		return
	}
	allowed, err := s.deps.QueryService.Describe(source, cfg, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to describe datasource", err.Error())
		return
	}

	resp := GetFieldsResponse{
		DatasourceID: cfg.ID,
		Source:       source,
		Fields:       make([]FieldDescriptor, 0, len(allowed.AllowedFields)),
	}
	for _, item := range allowed.AllowedFields {
		resp.Fields = append(resp.Fields, FieldDescriptor{
			Field:              item.FieldName,
			FieldType:          item.FieldType,
			Nullable:           item.Nullable,
			SupportedOperators: item.SupportedOperators,
			Widget:             widgetForField(item.FieldType),
			FacetEligible:      facetEligible(item.FieldType),
			Description:        item.Description,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleGetFacets(w http.ResponseWriter, r *http.Request) {
	cfg, source, ok := s.loadConfigAndSource(w, r)
	if !ok {
		return
	}
	field := strings.TrimSpace(r.URL.Query().Get("field"))
	if field == "" {
		writeError(w, http.StatusBadRequest, "missing field", "query parameter field is required")
		return
	}

	limit := 100
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		var err error
		_, err = fmt.Sscanf(rawLimit, "%d", &limit)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid limit", err.Error())
			return
		}
	}

	req, err := BuildMetadataRequest(cfg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to build metadata request", err.Error())
		return
	}

	values, truncated, err := s.deps.FacetService.Values(source, cfg, req, field, limit, r.URL.Query().Get("q"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to load facets", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, GetFacetsResponse{
		DatasourceID: cfg.ID,
		Source:       source,
		Field:        field,
		Values:       values,
		Truncated:    truncated,
	})
}

func (s *Server) handlePreview(w http.ResponseWriter, r *http.Request) {
	var req QueryPreviewRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body", err.Error())
		return
	}
	if req.Source == "" {
		writeError(w, http.StatusBadRequest, "missing source", "source is required")
		return
	}

	cfg, ok := s.findConfigByRequest(req.Request)
	if !ok {
		writeError(w, http.StatusBadRequest, "unknown datasource", "request datasource does not match the server catalog")
		return
	}
	if !sourceSupported(cfg, req.Source) {
		writeError(w, http.StatusBadRequest, "unsupported source", string(req.Source))
		return
	}

	resp, err := s.deps.QueryService.Preview(cfg, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, "preview failed", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) loadConfigAndSource(w http.ResponseWriter, r *http.Request) (DataSourceConfig, SourceType, bool) {
	id := r.PathValue("id")
	cfg, ok := s.deps.Catalog.Get(id)
	if !ok {
		writeError(w, http.StatusNotFound, "unknown datasource", id)
		return DataSourceConfig{}, "", false
	}
	source := sourceTypeFromString(strings.TrimSpace(r.URL.Query().Get("source")))
	if source == "" {
		source = cfg.DefaultSource
	}
	if !sourceSupported(cfg, source) {
		writeError(w, http.StatusBadRequest, "unsupported source", string(source))
		return DataSourceConfig{}, "", false
	}
	return cfg, source, true
}

func (s *Server) findConfigByRequest(req filters.FilterRequest) (DataSourceConfig, bool) {
	for _, cfg := range s.deps.Catalog.List() {
		if strings.EqualFold(cfg.DataSourceName, req.DataSourceName) && strings.EqualFold(cfg.DataSourceUUID, req.DataSourceUUID) {
			return cfg, true
		}
	}
	return DataSourceConfig{}, false
}

func decodeJSON(body io.ReadCloser, v interface{}) error {
	defer body.Close()
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

func widgetForField(fieldType string) string {
	switch fieldType {
	case "boolean":
		return "boolean-toggle"
	case "number", "integer":
		return "searchable-checkbox-group"
	default:
		return "searchable-checkbox-group"
	}
}

func facetEligible(fieldType string) bool {
	switch fieldType {
	case "string", "boolean", "number", "integer":
		return true
	default:
		return false
	}
}

func (s *Server) serveUI() http.Handler {
	staticDir := s.deps.StaticDir
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		path := strings.TrimPrefix(filepath.Clean(r.URL.Path), "/")
		if path == "" || path == "." {
			path = "index.html"
		}

		fullPath := filepath.Join(staticDir, path)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			http.ServeFile(w, r, fullPath)
			return
		}

		indexPath := filepath.Join(staticDir, "index.html")
		if _, err := os.Stat(indexPath); err != nil {
			writeError(w, http.StatusServiceUnavailable, "ui build missing", "ui/dist/index.html was not found")
			return
		}
		http.ServeFile(w, r, indexPath)
	})
}
