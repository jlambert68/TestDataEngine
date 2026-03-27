package logging

import (
	"os"
	"strings"
	"testing"
)

// logUnitCall standardizes verbose test call logging and can be disabled via LOG_UNIT_CALL.
func logUnitCall(t *testing.T, id, fn string, sent, expected, received any) {
	t.Helper()
	if !unitCallLoggingEnabled() {
		return
	}
	t.Logf("id=%s call=%s sent=%#v expected=%#v received=%#v", id, fn, sent, expected, received)
}

// unitCallLoggingEnabled returns false only for explicit "off" style env var values.
func unitCallLoggingEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_UNIT_CALL")))
	switch v {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}
