package logging

import "testing"

// TestUnitCallLoggingEnabled verifies LOG_UNIT_CALL on/off parsing behavior.
func TestUnitCallLoggingEnabled(t *testing.T) {
	t.Setenv("LOG_UNIT_CALL", "FALSE")
	if unitCallLoggingEnabled() {
		t.Fatal("expected logging disabled for LOG_UNIT_CALL=FALSE")
	}

	t.Setenv("LOG_UNIT_CALL", "1")
	if !unitCallLoggingEnabled() {
		t.Fatal("expected logging enabled for LOG_UNIT_CALL=1")
	}
}
