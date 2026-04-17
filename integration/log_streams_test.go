package integration_test

import (
	"bytes"
	"context"
	"os/exec"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestLogStreamsSplit pins the split-stream invariant: info-level logrus
// lines must appear on stdout and warn+error-level lines on stderr. The
// other integration tests merge both streams into a single buffer via
// runDeckschrubber, so they pass regardless of which fd a line lands on.
// This test is the only place the routing contract is exercised, so if it
// regresses the rest of the suite stays green and the regression ships.
//
// The invocation triggers both streams deterministically in a single run:
//
//   - Info on stdout:  "Successfully fetched repositories." is logged
//     unconditionally after the (empty) catalog fetch in main.go's repo
//     loop, so an empty registry is enough to produce at least one
//     info line.
//
//   - Warn on stderr:  main.go logs "Pagination enabled but page size is
//     larger than repo count" when -paginate is set AND -repos <= -page-size.
//     -repos=1 -page-size=100 hits that branch on every run.
func TestLogStreamsSplit(t *testing.T) {
	r := startRegistry(t)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx, binaryPath(t),
		"-registry", r.url,
		"-paginate",
		"-repos", "1",
		"-page-size", "100",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("deckschrubber failed: %v\nstdout:\n%s\nstderr:\n%s", err, stdout.String(), stderr.String())
	}

	stdoutStr := stdout.String()
	stderrStr := stderr.String()

	// Info line: on stdout, not on stderr.
	infoMsg := `msg="Successfully fetched repositories."`
	if !strings.Contains(stdoutStr, infoMsg) {
		t.Errorf("expected info line %q on stdout, got stdout:\n%s\nstderr:\n%s", infoMsg, stdoutStr, stderrStr)
	}
	if strings.Contains(stderrStr, infoMsg) {
		t.Errorf("info line %q leaked to stderr:\n%s", infoMsg, stderrStr)
	}

	// Warn line: on stderr, not on stdout. The full message contains
	// formatted flag values that vary with flag handling; assert the
	// stable prefix plus the level tag to prove routing without coupling
	// to the exact wording.
	warnLevelLine := regexp.MustCompile(`(?m)^.*\blevel=warning\b.*\bmsg="Pagination enabled but page size is larger than repo count`)
	if !warnLevelLine.MatchString(stderrStr) {
		t.Errorf("expected warn line on stderr, got stdout:\n%s\nstderr:\n%s", stdoutStr, stderrStr)
	}
	if warnLevelLine.MatchString(stdoutStr) {
		t.Errorf("warn line leaked to stdout:\n%s", stdoutStr)
	}

	// Belt-and-braces: no level=warning or level=error line on stdout at
	// all, and no level=info line on stderr. Cheap regression fence in
	// case a future edit adds new logs in main.go that skip initLogging.
	if regexp.MustCompile(`(?m)^.*\blevel=(warning|error|fatal|panic)\b`).MatchString(stdoutStr) {
		t.Errorf("diagnostic-level line found on stdout:\n%s", stdoutStr)
	}
	if regexp.MustCompile(`(?m)^.*\blevel=info\b`).MatchString(stderrStr) {
		t.Errorf("info-level line found on stderr:\n%s", stderrStr)
	}
}
