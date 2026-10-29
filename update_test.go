package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Each fixture writes in its own chart directory, so concurrent Helm calls do
// not share an output file. No real Helm or cluster access is needed.
func helmFixture(t *testing.T, failed []bool) (string, []string) {
	t.Helper()
	bin := t.TempDir()
	script := `#!/bin/sh
[ "$*" = "dependency update" ] || exit 99
printf 'attempted\n' > .attempted
[ ! -f .fail ]
`
	if err := os.WriteFile(filepath.Join(bin, "helm"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	root := t.TempDir()
	var charts []string
	for i, fail := range failed {
		chart := filepath.Join(root, fmt.Sprintf("chart-%d", i))
		if err := os.Mkdir(chart, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(chart, "Chart.yaml"), []byte("apiVersion: v2\nname: fixture\nversion: 1.0.0\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if fail {
			if err := os.WriteFile(filepath.Join(chart, ".fail"), nil, 0644); err != nil {
				t.Fatal(err)
			}
		}
		charts = append(charts, chart)
	}
	return root, charts
}

func assertAttempted(t *testing.T, charts []string, want bool) {
	t.Helper()
	for _, chart := range charts {
		_, err := os.Stat(filepath.Join(chart, ".attempted"))
		if want && err != nil {
			t.Errorf("chart %s was not attempted: %v", chart, err)
		} else if !want && !os.IsNotExist(err) {
			t.Errorf("chart %s was attempted or stat failed: %v", chart, err)
		}
	}
}

func TestUpdateAll(t *testing.T) {
	for _, threads := range []int{1, 2} {
		for _, tc := range []struct {
			name   string
			failed []bool
		}{
			{"empty", nil},
			{"success", []bool{false, false, false}},
			{"mixed", []bool{true, false, true}},
			{"all-fail", []bool{true, true, true}},
		} {
			t.Run(fmt.Sprintf("%s/threads-%d", tc.name, threads), func(t *testing.T) {
				root, charts := helmFixture(t, tc.failed)
				err := updateAll(context.Background(), root, threads, defaultCommandTimeout)
				var wantErrors []string
				for i, fail := range tc.failed {
					if fail {
						wantErrors = append(wantErrors, "chart "+charts[i]+": error fetching: exit status 1")
					}
				}
				if len(wantErrors) == 0 {
					if err != nil {
						t.Fatal(err)
					}
				} else {
					if err == nil || err.Error() != strings.Join(wantErrors, "\n") {
						t.Fatalf("error = %v, want %v", err, wantErrors)
					}
					var exitErr *exec.ExitError
					if !errors.As(err, &exitErr) {
						t.Fatal("aggregate does not preserve Helm exit error")
					}
				}
				assertAttempted(t, charts, true)
			})
		}
	}
}

func TestUpdateAllInvalidThreads(t *testing.T) {
	for _, threads := range []int{0, -1} {
		t.Run(fmt.Sprint(threads), func(t *testing.T) {
			root, charts := helmFixture(t, []bool{false})
			for _, dir := range []string{root, filepath.Join(root, "missing")} {
				err := updateAll(context.Background(), dir, threads, defaultCommandTimeout)
				if err == nil || !strings.Contains(err.Error(), "--threads must be greater than zero") {
					t.Fatalf("error = %v, want concurrency validation before discovery", err)
				}
			}
			assertAttempted(t, charts, false)
		})
	}
}

// Run the real CLI entry point in a child test process to observe os.Exit.
func TestUpdateCLIProcess(t *testing.T) {
	if os.Getenv("K8S_TOOLS_CLI_FIXTURE") != "1" {
		return
	}
	for i, arg := range os.Args {
		if arg == "--" {
			os.Args = append([]string{os.Args[0]}, os.Args[i+1:]...)
			main()
			os.Exit(0)
		}
	}
	t.Fatal("missing CLI argument separator")
}

func TestUpdateCLI(t *testing.T) {
	for _, tc := range []struct {
		name    string
		failed  []bool
		threads string
		want    int
	}{
		{"success", []bool{false, false}, "2", 0},
		{"mixed", []bool{true, false, true}, "2", 1},
		{"all-fail", []bool{true, true}, "2", 1},
		{"empty", nil, "2", 0},
		{"zero", []bool{false}, "0", 1},
		{"negative", []bool{false}, "-1", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, charts := helmFixture(t, tc.failed)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestUpdateCLIProcess$", "--", "--threads", tc.threads, "update", root)
			cmd.Env = append(os.Environ(), "K8S_TOOLS_CLI_FIXTURE=1")
			output, err := cmd.CombinedOutput()
			if ctx.Err() != nil {
				t.Fatalf("CLI timed out: %v\n%s", ctx.Err(), output)
			}
			got := 0
			if err != nil {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatal(err)
				}
				got = exitErr.ExitCode()
			}
			if got != tc.want {
				t.Fatalf("exit = %d, want %d\n%s", got, tc.want, output)
			}
			valid := tc.threads == "2"
			assertAttempted(t, charts, valid)
			if !valid && !strings.Contains(string(output), "--threads must be greater than zero") {
				t.Fatalf("missing validation error: %s", output)
			}
			for i, fail := range tc.failed {
				if valid && fail && !strings.Contains(string(output), "chart "+charts[i]+": error fetching:") {
					t.Errorf("missing failed path %s in final error: %s", charts[i], output)
				}
			}
		})
	}
}
