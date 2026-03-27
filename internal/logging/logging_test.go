package logging

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

// TestInfofIncludesProvidedUUIDId verifies that each log line keeps its caller-provided UUID.
func TestInfofIncludesProvidedUUIDId(t *testing.T) {
	var buf bytes.Buffer

	origWriter := log.Writer()
	origFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(origWriter)
		log.SetFlags(origFlags)
	})

	id1 := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	id2 := "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
	Infof(id1, "first")
	Infof(id2, "second")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	logUnitCall(t, "33333333-3333-4333-8333-333333333333", "Infof", []string{id1, id2}, "two log lines with ids", lines)
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %d", len(lines))
	}

	re := regexp.MustCompile(`Id=([0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})`)
	m1 := re.FindStringSubmatch(lines[0])
	m2 := re.FindStringSubmatch(lines[1])
	logUnitCall(t, "33333333-3333-4333-8333-333333333333", "regexp.FindStringSubmatch", lines, "two UUID matches", []any{m1, m2})
	if len(m1) != 2 {
		t.Fatalf("first line missing Id UUID: %q", lines[0])
	}
	if len(m2) != 2 {
		t.Fatalf("second line missing Id UUID: %q", lines[1])
	}
	if m1[1] != id1 {
		t.Fatalf("expected first line Id %q, got %q", id1, m1[1])
	}
	if m2[1] != id2 {
		t.Fatalf("expected second line Id %q, got %q", id2, m2[1])
	}
}

// TestErrorfAndWithID validates error prefixing and ID format construction.
func TestErrorfAndWithID(t *testing.T) {
	var buf bytes.Buffer

	origWriter := log.Writer()
	origFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	t.Cleanup(func() {
		log.SetOutput(origWriter)
		log.SetFlags(origFlags)
	})

	id := "cccccccc-cccc-4ccc-8ccc-cccccccccccc"
	Errorf(id, "boom %d", 42)
	line := strings.TrimSpace(buf.String())
	logUnitCall(t, "33333333-3333-4333-8333-333333333333", "Errorf", map[string]any{"id": id, "format": "boom %d", "args": []any{42}}, "error line with id+message", line)

	if !strings.Contains(line, "Id="+id) {
		t.Fatalf("expected Id in error log, got %q", line)
	}
	if !strings.Contains(line, "ERROR: boom 42") {
		t.Fatalf("expected ERROR message, got %q", line)
	}
	if got := withID(id, "x=%d"); got != "Id="+id+" x=%d" {
		t.Fatalf("unexpected withID output: %q", got)
	}
	logUnitCall(t, "33333333-3333-4333-8333-333333333333", "withID", map[string]any{"id": id, "format": "x=%d"}, "Id="+id+" x=%d", withID(id, "x=%d"))
}

// TestFatalf validates fatal logging behavior through a subprocess to avoid exiting the test process.
func TestFatalf(t *testing.T) {
	if os.Getenv("LOGGING_FATAL_HELPER") == "1" {
		log.SetFlags(0)
		Fatalf("dddddddd-dddd-4ddd-8ddd-dddddddddddd", "fatal %d", 1)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestFatalf")
	cmd.Env = append(os.Environ(), "LOGGING_FATAL_HELPER=1")
	out, err := cmd.CombinedOutput()
	logUnitCall(t, "33333333-3333-4333-8333-333333333333", "Fatalf(subprocess)", map[string]any{"cmd": cmd.String(), "env": "LOGGING_FATAL_HELPER=1"}, "non-nil error and fatal log output", map[string]any{"err": err, "out": string(out)})
	if err == nil {
		t.Fatal("expected subprocess to fail because Fatalf exits")
	}
	s := string(out)
	if !strings.Contains(s, "Id=dddddddd-dddd-4ddd-8ddd-dddddddddddd") {
		t.Fatalf("expected Id in fatal output, got %q", s)
	}
	if !strings.Contains(s, "FATAL: fatal 1") {
		t.Fatalf("expected fatal message in output, got %q", s)
	}
}
