package logging

import (
	"log"
)

// Infof logs an informational message with a required hard-coded UUID identifier.
func Infof(id, format string, args ...any) {
	log.Printf(withID(id, format), args...)
}

// Errorf logs an error message with a required hard-coded UUID identifier.
func Errorf(id, format string, args ...any) {
	log.Printf(withID(id, "ERROR: "+format), args...)
}

// Fatalf logs a fatal message with a required hard-coded UUID identifier and exits.
func Fatalf(id, format string, args ...any) {
	log.Fatalf(withID(id, "FATAL: "+format), args...)
}

// withID prefixes each log format string so all logs include a searchable Id=... token.
func withID(id, format string) string {
	return "Id=" + id + " " + format
}
