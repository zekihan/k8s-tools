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

func TestCLIVersion(t *testing.T) {
	code, stdout, stderr := runInputCLIStreams(t, "--version")
	if code != 0 || stdout != version+"\n" || stderr != "" {
		t.Fatalf("exit = %d, stdout = %q, stderr = %q; want version on stdout", code, stdout, stderr)
	}
}

func TestCLIVersionWriteFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stdout")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	stdout, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer stdout.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestUpdateCLIProcess$", "--", "--version")
	cmd.Env = append(os.Environ(), "K8S_TOOLS_CLI_FIXTURE=1")
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = stdout, &stderr
	err = cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 || !strings.Contains(stderr.String(), "Error writing version") {
		t.Fatalf("exit error = %v, stderr = %q; want a version write error and exit status 1", err, &stderr)
	}
}
