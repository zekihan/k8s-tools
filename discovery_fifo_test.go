//go:build linux || darwin

package main

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestUpdateCLIIgnoresChartNamedPipe(t *testing.T) {
	for _, kind := range []string{"root-pipe", "child-symlink-to-pipe"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			container := root
			if kind == "child-symlink-to-pipe" {
				container = filepath.Join(root, "container")
			}
			chart := filepath.Join(container, "chart")
			if err := os.MkdirAll(chart, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(chart, "Chart.yaml"), []byte("apiVersion: v2\nname: fixture\nversion: 1.0.0\n"), 0644); err != nil {
				t.Fatal(err)
			}

			chartPath := filepath.Join(container, "Chart.yaml")
			pipePath := chartPath
			if kind == "child-symlink-to-pipe" {
				pipePath = filepath.Join(container, "pipe")
				if err := os.Symlink("pipe", chartPath); err != nil {
					t.Fatal(err)
				}
			}
			if err := syscall.Mkfifo(pipePath, 0600); err != nil {
				t.Fatal(err)
			}

			installHelm(t, `[ "$*" = "dependency update" ] && [ -f Chart.yaml ] || exit 99
printf 'attempted\n' > .attempted
`)
			code, output := runInputCLI(t, "update", root)
			if code != 0 {
				t.Fatalf("exit = %d, want 0 after skipping non-regular Chart.yaml\n%s", code, output)
			}
			assertAttempted(t, []string{chart}, true)
		})
	}
}
