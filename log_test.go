package main

import (
	"strings"
	"testing"
)

func TestUpdateCLIPipedLogsPreserveDiagnostics(t *testing.T) {
	root, charts := helmFixture(t, []bool{false})
	installHelm(t, `printf '%s\n' 'fixture rejected � near literal \x1b' >&2
exit 7
`)

	code, stdout, stderr := runInputCLIStreams(t, "--log-level", "error", "update", root)
	if code != 1 || stderr != "" {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q; want a chart failure", code, stdout, stderr)
	}
	if strings.ContainsRune(stdout, '\x1b') {
		t.Errorf("piped log contains terminal escapes: %q", stdout)
	}
	if want := charts[0] + `: error fetching: exit status 7; output: fixture rejected � near literal \\x1b`; !strings.Contains(stdout, want) {
		t.Errorf("piped log = %q; want diagnostic %q", stdout, want)
	}
}
