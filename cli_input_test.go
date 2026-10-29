package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func runInputCLI(t *testing.T, args ...string) (int, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestUpdateCLIProcess$", "--"}, args...)...)
	cmd.Env = append(os.Environ(), "K8S_TOOLS_CLI_FIXTURE=1")
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("CLI timed out: %v\n%s", ctx.Err(), output)
	}
	if err == nil {
		return 0, string(output)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatal(err)
	}
	return exitErr.ExitCode(), string(output)
}

func TestUpdateCLIRejectsExtraArguments(t *testing.T) {
	for _, extra := range [][]string{{"second-directory"}, {"--threads", "0"}, {"--typo"}} {
		t.Run(strings.Join(extra, "_"), func(t *testing.T) {
			root, charts := helmFixture(t, []bool{false})
			args := append([]string{"update", root}, extra...)
			code, output := runInputCLI(t, args...)
			if code != 1 || !strings.Contains(output, "expected at most one directory argument") {
				t.Errorf("exit = %d, output = %s; want argument error", code, output)
			}
			assertAttempted(t, charts, false)
		})
	}
}

func TestUpdateCLIPreservesLiteralDollarInDirectory(t *testing.T) {
	root, charts := helmFixture(t, []bool{false, false})
	literal := filepath.Join(root, "$K8S_TOOLS_DIRECTORY")
	expanded := filepath.Join(root, "expanded")
	for i, dest := range []string{literal, expanded} {
		if err := os.Rename(charts[i], dest); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("K8S_TOOLS_DIRECTORY", "expanded")
	code, output := runInputCLI(t, "update", literal)
	if code != 0 {
		t.Fatalf("exit = %d, output = %s", code, output)
	}
	assertAttempted(t, []string{literal}, true)
	assertAttempted(t, []string{expanded}, false)
}

func TestUpdateCLIDefaultDirectory(t *testing.T) {
	root, charts := helmFixture(t, []bool{false})
	home := t.TempDir()
	repos := filepath.Join(home, "repos")
	if err := os.Rename(root, repos); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	code, output := runInputCLI(t, "update")
	if code != 0 {
		t.Fatalf("exit = %d, output = %s", code, output)
	}
	assertAttempted(t, []string{filepath.Join(repos, filepath.Base(charts[0]))}, true)
}
