package main

import (
	"bytes"
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
	code, stdout, stderr := runInputCLIStreams(t, args...)
	return code, stdout + stderr
}

func runInputCLIStreams(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestUpdateCLIProcess$", "--"}, args...)...)
	cmd.Env = append(os.Environ(), "K8S_TOOLS_CLI_FIXTURE=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		t.Fatalf("CLI timed out: %v\n%s\n%s", ctx.Err(), &stdout, &stderr)
	}
	if err == nil {
		return 0, stdout.String(), stderr.String()
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatal(err)
	}
	return exitErr.ExitCode(), stdout.String(), stderr.String()
}

func TestCLIHelpUsesStdout(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"--help"}, "Print the application version"},
		{[]string{"help"}, "Print the application version"},
		{[]string{"update", "--help"}, "The directory to search for root Helm charts"},
		{[]string{"help", "update"}, "The directory to search for root Helm charts"},
	} {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			code, stdout, stderr := runInputCLIStreams(t, tc.args...)
			if code != 0 || stderr != "" || !strings.Contains(stdout, "Usage:") || !strings.Contains(stdout, tc.want) {
				t.Errorf("exit = %d, stdout = %q, stderr = %q; want complete help on stdout", code, stdout, stderr)
			}
		})
	}
}

func TestCLIUsageErrorsUseStderr(t *testing.T) {
	for _, tc := range []struct {
		args []string
		code int
		want string
	}{
		{nil, 1, "Print the application version"},
		{[]string{"bogus"}, 1, "Unknown command: bogus"},
		{[]string{"help", "bogus"}, 1, "Unknown command: bogus"},
		{[]string{"--typo"}, 2, "flag provided but not defined"},
		{[]string{"update", "--typo"}, 2, "flag provided but not defined"},
	} {
		t.Run(strings.Join(tc.args, "_"), func(t *testing.T) {
			code, stdout, stderr := runInputCLIStreams(t, tc.args...)
			if code != tc.code || stdout != "" || !strings.Contains(stderr, "Usage:") || !strings.Contains(stderr, tc.want) {
				t.Errorf("exit = %d, stdout = %q, stderr = %q; want exit %d and usage error on stderr", code, stdout, stderr, tc.code)
			}
		})
	}
}

func TestUpdateCLIRejectsExtraArguments(t *testing.T) {
	for _, extra := range [][]string{{"second-directory"}, {"--threads", "0"}, {"--typo"}} {
		t.Run(strings.Join(extra, "_"), func(t *testing.T) {
			root, charts := helmFixture(t, []bool{false})
			args := append([]string{"update", root}, extra...)
			code, stdout, stderr := runInputCLIStreams(t, args...)
			if code != 1 || stdout != "" || !strings.Contains(stderr, "expected at most one directory argument") {
				t.Errorf("exit = %d, stdout = %q, stderr = %q; want argument error on stderr", code, stdout, stderr)
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

func TestUpdateCLIResolvesSymlinkBeforeParentDirectory(t *testing.T) {
	for _, form := range []string{"absolute", "relative"} {
		t.Run(form, func(t *testing.T) {
			root, charts := helmFixture(t, []bool{false, false})
			target := filepath.Join(root, "target")
			child := filepath.Join(target, "child")
			if err := os.MkdirAll(child, 0755); err != nil {
				t.Fatal(err)
			}
			chartName := filepath.Base(charts[0])
			want := filepath.Join(target, chartName)
			if err := os.Rename(charts[1], want); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(child, filepath.Join(root, "link")); err != nil {
				t.Fatal(err)
			}

			// Joining these components would remove the symlink from the operand.
			sep := string(filepath.Separator)
			dir := "link" + sep + ".." + sep + chartName
			if form == "absolute" {
				dir = root + sep + dir
			} else {
				t.Chdir(root)
			}
			code, output := runInputCLI(t, "update", dir)
			if code != 0 {
				t.Fatalf("exit = %d, output = %s", code, output)
			}
			assertAttempted(t, []string{want}, true)
			assertAttempted(t, charts[:1], false)
		})
	}
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
