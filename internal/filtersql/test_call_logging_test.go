package filtersql

import (
	"os"
	"strings"
	"testing"
)

func logUnitCall(t *testing.T, id, fn string, sent, expected, received any) {
	t.Helper()
	if !unitCallLoggingEnabled() {
		return
	}
	t.Logf("id=%s call=%s sent=%#v expected=%#v received=%#v", id, fn, sent, expected, received)
}

func unitCallLoggingEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_UNIT_CALL")))
	switch v {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}
