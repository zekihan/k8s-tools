package main

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestFindHelmCharts(t *testing.T) {
	root := t.TempDir()
	want := []string{filepath.Join(root, "first"), filepath.Join(root, "nested", "second")}
	for _, dir := range want {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte("apiVersion: v2\nname: fixture\nversion: 1.0.0\n"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := findHelmCharts(root)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("charts = %v, want %v", got, want)
	}
	if _, err := findHelmCharts(filepath.Join(root, "missing")); err == nil {
		t.Fatal("expected missing directory error")
	}
}

func TestUpdateRunsHelmInChartDirectory(t *testing.T) {
	bin := t.TempDir()
	chart := t.TempDir()
	script := "#!/bin/sh\nprintf '%s\n' \"$PWD\" \"$*\" > \"$HELM_CALLS\"\nexit \"${HELM_EXIT:-0}\"\n"
	if err := os.WriteFile(filepath.Join(bin, "helm"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	calls := filepath.Join(t.TempDir(), "calls")
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("HELM_CALLS", calls)
	t.Setenv("HELM_EXIT", "0")
	ok, err := update(chart)
	if !ok || err != nil {
		t.Fatalf("update: ok=%v err=%v", ok, err)
	}
	got, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != chart+"\ndependency update\n" {
		t.Fatalf("helm invocation = %q", got)
	}
	t.Setenv("HELM_EXIT", "1")
	if ok, err := update(chart); ok || err == nil {
		t.Fatalf("failed helm: ok=%v err=%v", ok, err)
	}
}
