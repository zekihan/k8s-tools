//go:build unix

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func installHelm(t *testing.T, script string) {
	t.Helper()
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "helm"), []byte("#!/bin/sh\n"+script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func waitFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", path)
}

func awaitUpdate(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("update did not stop promptly")
		return nil
	}
}

// Killed orphan descendants can remain zombies until the host init reaps them.
func assertStopped(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) {
			return
		}
		if runtime.GOOS == "linux" {
			stat, _ := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
			if end := strings.LastIndex(string(stat), ")"); end >= 0 && strings.HasPrefix(string(stat[end+1:]), " Z ") {
				return
			}
		}
		time.Sleep(5 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL) // Do not leak even if the assertion fails.
	t.Errorf("process %d is still running", pid)
}

const blockedHelm = `echo $$ > .parent
sleep 60 &
echo $! > .child
wait
`

func TestCommandDeadlineKillsDescendants(t *testing.T) {
	installHelm(t, blockedHelm)
	chart := t.TempDir()
	start := time.Now()
	_, err := runCommand(context.Background(), 200*time.Millisecond, chart, "helm", "dependency", "update")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v", err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("deadline did not bound command")
	}
	assertStopped(t, filepath.Join(chart, ".parent"))
	assertStopped(t, filepath.Join(chart, ".child"))
}

func TestCancelStopsQueueAndPreservesFailure(t *testing.T) {
	root, charts := helmFixture(t, []bool{true, false, false, false})
	installHelm(t, `echo attempted > .attempted
if [ -f .fail ]; then echo 'repository rejected fixture' >&2; exit 7; fi
`+blockedHelm)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- updateAll(ctx, root, 1, time.Minute) }()
	waitFile(t, filepath.Join(charts[1], ".child"))
	cancel()
	err := awaitUpdate(t, done)
	var exitErr *exec.ExitError
	if !errors.Is(err, context.Canceled) || !errors.As(err, &exitErr) || !strings.Contains(err.Error(), charts[0]) || !strings.Contains(err.Error(), "repository rejected fixture") {
		t.Fatalf("cancellation masked earlier failure/diagnostic: %v", err)
	}
	assertAttempted(t, charts[2:], false)
	assertStopped(t, filepath.Join(charts[1], ".parent"))
	assertStopped(t, filepath.Join(charts[1], ".child"))
}

func TestDeadlineContinuesIndependentCharts(t *testing.T) {
	root, charts := helmFixture(t, []bool{true, false})
	installHelm(t, `echo attempted > .attempted
if [ -f .fail ]; then
`+blockedHelm+`
fi
`)
	err := updateAll(context.Background(), root, 1, 200*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), charts[0]) || strings.Contains(err.Error(), "chart "+charts[1]+":") {
		t.Fatalf("error = %v", err)
	}
	assertAttempted(t, charts, true)
	assertStopped(t, filepath.Join(charts[0], ".child"))
}

func TestUpdateConcurrencyBound(t *testing.T) {
	root, charts := helmFixture(t, []bool{false, false, false, false})
	installHelm(t, `echo attempted > .attempted
while [ ! -f "$HELM_RELEASE" ]; do sleep 0.01; done
`)
	release := filepath.Join(t.TempDir(), "release")
	t.Setenv("HELM_RELEASE", release)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- updateAll(ctx, root, 2, 5*time.Second) }()
	waitFile(t, filepath.Join(charts[0], ".attempted"))
	waitFile(t, filepath.Join(charts[1], ".attempted"))
	assertAttempted(t, charts[2:], false)
	if err := os.WriteFile(release, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := awaitUpdate(t, done); err != nil {
		t.Fatal(err)
	}
	assertAttempted(t, charts, true)
}

func TestPreCanceledDiscoveryAndUpdate(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	root, charts := helmFixture(t, []bool{false})
	if got, err := findHelmCharts(ctx, root); got != nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("discovery = %v, %v", got, err)
	}
	if err := updateAll(ctx, root, 1, time.Minute); !errors.Is(err, context.Canceled) {
		t.Fatalf("update = %v", err)
	}
	assertAttempted(t, charts, false)
}

func TestCommandOutputBound(t *testing.T) {
	installHelm(t, `i=0
while [ "$i" -lt 3000 ]; do
 echo 'xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx' >&2
 i=$((i+1))
done
echo final-diagnostic >&2
exit 9
`)
	output, err := runCommand(context.Background(), 5*time.Second, t.TempDir(), "helm")
	if err == nil || len(output) > maxCommandOutput+100 || !strings.HasPrefix(output, "[output truncated;") || !strings.HasSuffix(output, "final-diagnostic") {
		t.Fatalf("error = %v, output length = %d", err, len(output))
	}
}

func TestUpdateCLISignals(t *testing.T) {
	for _, sig := range []os.Signal{os.Interrupt, syscall.SIGTERM} {
		t.Run(sig.String(), func(t *testing.T) {
			root, charts := helmFixture(t, []bool{false, false})
			installHelm(t, blockedHelm)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestUpdateCLIProcess$", "--", "--threads", "1", "update", root)
			cmd.Env = append(os.Environ(), "K8S_TOOLS_CLI_FIXTURE=1")
			// File output avoids a test pipe deadlock if the CLI regresses.
			log, err := os.CreateTemp(t.TempDir(), "cli")
			if err != nil {
				t.Fatal(err)
			}
			defer log.Close()
			cmd.Stdout, cmd.Stderr = log, log
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() { _ = cmd.Process.Kill() }()
			waitFile(t, filepath.Join(charts[0], ".child"))
			if err := cmd.Process.Signal(sig); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			err = awaitUpdate(t, done)
			var exitErr *exec.ExitError
			output, _ := os.ReadFile(log.Name())
			if ctx.Err() != nil || !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 || !strings.Contains(string(output), "context canceled") {
				t.Fatalf("CLI error = %v, output = %s", err, output)
			}
			assertAttempted(t, charts[1:], false)
			assertStopped(t, filepath.Join(charts[0], ".parent"))
			assertStopped(t, filepath.Join(charts[0], ".child"))
		})
	}
}

func TestUpdateCLICommandTimeout(t *testing.T) {
	for _, timeout := range []string{"200ms", "0", "-1s"} {
		t.Run(timeout, func(t *testing.T) {
			root, charts := helmFixture(t, []bool{false})
			installHelm(t, blockedHelm)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestUpdateCLIProcess$", "--", "--command-timeout", timeout, "update", root)
			cmd.Env = append(os.Environ(), "K8S_TOOLS_CLI_FIXTURE=1")
			output, err := cmd.CombinedOutput()
			var exitErr *exec.ExitError
			if ctx.Err() != nil || !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
				t.Fatalf("CLI error = %v, output = %s", err, output)
			}
			want := "--command-timeout must be greater than zero"
			if timeout == "200ms" {
				want = "context deadline exceeded"
				assertStopped(t, filepath.Join(charts[0], ".child"))
			} else {
				assertAttempted(t, charts, false)
			}
			if !strings.Contains(string(output), want) {
				t.Fatalf("missing %q: %s", want, output)
			}
		})
	}
}

func TestExitedParentCannotLeavePipeWaitHung(t *testing.T) {
	installHelm(t, `sleep 60 &
echo $! > .child
exit 0
`)
	chart := t.TempDir()
	start := time.Now()
	_, err := runCommand(context.Background(), time.Minute, chart, "helm")
	if !errors.Is(err, exec.ErrWaitDelay) {
		t.Fatalf("error = %v, want bounded output-pipe error", err)
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("output-pipe wait was not bounded")
	}
	assertStopped(t, filepath.Join(chart, ".child"))
}
