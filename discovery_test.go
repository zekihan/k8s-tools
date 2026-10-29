package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestIsHelmDir(t *testing.T) {
	for _, kind := range []string{"missing", "file", "directory", "dangling-symlink", "valid-symlink", "loop"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "Chart.yaml")
			var err error
			switch kind {
			case "file":
				err = os.WriteFile(path, nil, 0644)
			case "directory":
				err = os.Mkdir(path, 0755)
			case "dangling-symlink":
				err = os.Symlink("absent", path)
			case "valid-symlink":
				if err = os.WriteFile(filepath.Join(root, "target"), nil, 0644); err != nil {
					t.Fatal(err)
				}
				err = os.Symlink("target", path)
			case "loop":
				err = os.Symlink("Chart.yaml", path)
			}
			if err != nil {
				t.Fatal(err)
			}
			got, err := isHelmDir(root)
			if kind == "loop" {
				if got || !errors.Is(err, syscall.ELOOP) || !strings.Contains(err.Error(), path) {
					t.Fatalf("isHelmDir = %v, %v", got, err)
				}
			} else if want := kind == "file" || kind == "valid-symlink"; err != nil || got != want {
				t.Fatalf("isHelmDir = %v, %v; want %v, nil", got, err, want)
			}
		})
	}
}

func TestDiscoverySymlinkLoop(t *testing.T) {
	for _, relative := range []string{".", "child", "nested/child"} {
		t.Run(relative, func(t *testing.T) {
			root, charts := helmFixture(t, []bool{false})
			dir := filepath.Join(root, relative)
			if err := os.MkdirAll(dir, 0755); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "Chart.yaml")
			if err := os.Symlink("Chart.yaml", path); err != nil {
				t.Fatal(err)
			}
			got, err := findHelmCharts(root)
			if got != nil || !errors.Is(err, syscall.ELOOP) || !strings.Contains(err.Error(), path) {
				t.Fatalf("discovery = %v, %v; want no partial results and path-qualified ELOOP", got, err)
			}
			if err := updateAll(root, 2); !errors.Is(err, syscall.ELOOP) {
				t.Fatalf("updateAll = %v", err)
			}
			assertAttempted(t, charts, false)

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestUpdateCLIProcess$", "--", "update", root)
			cmd.Env = append(os.Environ(), "K8S_TOOLS_CLI_FIXTURE=1")
			output, err := cmd.CombinedOutput()
			var exitErr *exec.ExitError
			if ctx.Err() != nil || !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 || !strings.Contains(string(output), path) || strings.Contains(string(output), "panic:") {
				t.Fatalf("CLI error = %v, context = %v, output = %s", err, ctx.Err(), output)
			}
			assertAttempted(t, charts, false)
		})
	}
}

func TestDiscoveryChartEntry(t *testing.T) {
	for _, kind := range []string{"missing", "file", "directory", "dangling-symlink"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "Chart.yaml")
			var err error
			switch kind {
			case "file":
				err = os.WriteFile(path, nil, 0644)
			case "directory":
				err = os.Mkdir(path, 0755)
			case "dangling-symlink":
				err = os.Symlink("absent", path)
			}
			if err != nil {
				t.Fatal(err)
			}
			got, err := findHelmCharts(root)
			want := 0
			if kind == "file" {
				want = 1
			}
			if err != nil || len(got) != want || (want == 1 && got[0] != root) {
				t.Fatalf("discovery = %v, %v", got, err)
			}
		})
	}
}

func TestDiscoveryStopsAtRootChart(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "root")
	dependency := filepath.Join(root, "charts", "dependency")
	for _, chart := range []string{root, dependency} {
		if err := os.MkdirAll(chart, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(chart, "Chart.yaml"), []byte("apiVersion: v2\nname: fixture\nversion: 1.0.0\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	for _, searchDir := range []string{root, parent} {
		got, err := findHelmCharts(searchDir)
		if err != nil {
			t.Fatalf("findHelmCharts(%s): %v", searchDir, err)
		}
		if !reflect.DeepEqual(got, []string{root}) {
			t.Errorf("findHelmCharts(%s) = %v, want [%s]", searchDir, got, root)
		}
	}
}

func TestUpdateAllDoesNotUpdateVendoredSubchart(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "root")
	dependency := filepath.Join(root, "charts", "dependency")
	siblings := []string{filepath.Join(parent, "sibling-a"), filepath.Join(parent, "sibling-b")}
	charts := append([]string{root, dependency}, siblings...)
	for _, chart := range charts {
		if err := os.MkdirAll(chart, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(chart, "Chart.yaml"), []byte("apiVersion: v2\nname: fixture\nversion: 1.0.0\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	bin := t.TempDir()
	calls := filepath.Join(t.TempDir(), "calls")
	script := "#!/bin/sh\nprintf '%s\\n' \"$PWD\" >> \"$HELM_CALLS\"\n[ \"$*\" = \"dependency update\" ]\n"
	if err := os.WriteFile(filepath.Join(bin, "helm"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HELM_CALLS", calls)

	if err := updateAll(parent, 4); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Fields(string(contents))
	sort.Strings(got)
	want := []string{root, siblings[0], siblings[1]}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Helm ran in %v, want %v (dependency %s must be owned by %s)", got, want, dependency, root)
	}
}

func TestDiscoveryPermissionError(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.Mkdir(child, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(child, "Chart.yaml")
	if err := os.WriteFile(path, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(child, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chmod(child, 0755); err != nil {
			t.Error(err)
		}
	})
	if _, err := os.Stat(path); !errors.Is(err, os.ErrPermission) {
		t.Skipf("filesystem/user does not enforce directory permissions: %v", err)
	}
	if ok, err := isHelmDir(child); ok || !errors.Is(err, os.ErrPermission) {
		t.Fatalf("isHelmDir = %v, %v", ok, err)
	}
	got, err := findHelmCharts(root)
	if got != nil || !errors.Is(err, os.ErrPermission) || !strings.Contains(err.Error(), path) {
		t.Fatalf("discovery = %v, %v", got, err)
	}
}
