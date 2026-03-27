package filters

import "testing"

// TestUnitCallLoggingEnabled verifies LOG_UNIT_CALL on/off parsing behavior.
func TestUnitCallLoggingEnabled(t *testing.T) {
	t.Setenv("LOG_UNIT_CALL", "0")
	if unitCallLoggingEnabled() {
		t.Fatal("expected logging disabled for LOG_UNIT_CALL=0")
	}

	t.Setenv("LOG_UNIT_CALL", "false")
	if unitCallLoggingEnabled() {
		t.Fatal("expected logging disabled for LOG_UNIT_CALL=false")
	}

	t.Setenv("LOG_UNIT_CALL", "")
	if !unitCallLoggingEnabled() {
		t.Fatal("expected logging enabled by default")
	}

	t.Setenv("LOG_UNIT_CALL", "1")
	if !unitCallLoggingEnabled() {
		t.Fatal("expected logging enabled for LOG_UNIT_CALL=1")
	}
}
