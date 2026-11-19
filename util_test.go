package main

import (
	"bytes"
	"context"
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
	got, err := findHelmCharts(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("charts = %v, want %v", got, want)
	}
	if _, err := findHelmCharts(context.Background(), filepath.Join(root, "missing")); err == nil {
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
	ok, err := update(context.Background(), defaultCommandTimeout, chart)
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
	if ok, err := update(context.Background(), defaultCommandTimeout, chart); ok || err == nil {
		t.Fatalf("failed helm: ok=%v err=%v", ok, err)
	}
}

func TestTailBufferReusesStorage(t *testing.T) {
	chunk := make([]byte, 8*1024)
	var output tailBuffer
	if allocs := testing.AllocsPerRun(20, func() {
		for range 64 {
			output.Write(chunk)
		}
	}); allocs != 0 {
		t.Fatalf("capturing 512 KiB after filling the tail allocated %.0f times, want zero", allocs)
	}
}

func TestTailBufferKeepsLastBytes(t *testing.T) {
	var output tailBuffer
	var sent []byte
	for i, chunk := range [][]byte{
		bytes.Repeat([]byte("a"), maxCommandOutput),
		[]byte("bc"),
		bytes.Repeat([]byte("d"), maxCommandOutput-1),
		nil,
		bytes.Repeat([]byte("ef"), maxCommandOutput/2+1),
		[]byte("gh"),
	} {
		if n, err := output.Write(chunk); n != len(chunk) || err != nil {
			t.Fatalf("write %d returned %d, %v; want %d, nil", i, n, err, len(chunk))
		}
		sent = append(sent, chunk...)
		want := sent[max(0, len(sent)-maxCommandOutput):]
		if got := output.String(); got != string(want) {
			t.Errorf("after write %d, captured %d bytes that differ from the expected %d-byte tail", i, len(got), len(want))
		}
		if output.truncated != (len(sent) > maxCommandOutput) {
			t.Errorf("after write %d, truncated = %v with %d bytes written", i, output.truncated, len(sent))
		}
	}
}
