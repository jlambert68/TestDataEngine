package webapi

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string, details string) {
	writeJSON(w, status, APIError{
		Error:   msg,
		Details: details,
	})
}

func errUnsupportedSource(source SourceType) error {
	return fmt.Errorf("unsupported source: %q", source)
}
