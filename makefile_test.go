package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMakeLintRejectsGofmtErrors(t *testing.T) {
	makePath, err := exec.LookPath("make")
	if err != nil {
		t.Skip("make is unavailable")
	}
	makefile, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for name, contents := range map[string]string{
		"Makefile":   string(makefile),
		"VERSION":    "1.0.0\n",
		"go.mod":     "module fixture\n",
		"main.go":    "package main\n\nfunc main() {}\n",
		"ignored.go": "//go:build ignore\n\npackage main\n\nfunc broken(\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0644); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, makePath, "lint")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=off")
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil || err == nil || !strings.Contains(string(output), "ignored.go:") {
		t.Fatalf("make lint = %v, want a gofmt failure for ignored.go\n%s", err, output)
	}
}
