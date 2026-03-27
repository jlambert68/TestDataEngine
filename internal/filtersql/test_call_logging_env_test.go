package filtersql

import "testing"

// TestUnitCallLoggingEnabled verifies LOG_UNIT_CALL on/off parsing behavior.
func TestUnitCallLoggingEnabled(t *testing.T) {
	t.Setenv("LOG_UNIT_CALL", "off")
	if unitCallLoggingEnabled() {
		t.Fatal("expected logging disabled for LOG_UNIT_CALL=off")
	}

	t.Setenv("LOG_UNIT_CALL", "no")
	if unitCallLoggingEnabled() {
		t.Fatal("expected logging disabled for LOG_UNIT_CALL=no")
	}

	t.Setenv("LOG_UNIT_CALL", "true")
	if !unitCallLoggingEnabled() {
		t.Fatal("expected logging enabled for LOG_UNIT_CALL=true")
	}
}
