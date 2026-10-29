package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// isHelmDir checks if a directory contains a Helm Chart.yaml file.
func isHelmDir(dir string) (bool, error) {
	chartPath := filepath.Join(dir, "Chart.yaml")
	info, err := os.Stat(chartPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("error checking Helm chart %s: %w", chartPath, err)
	}
	return !info.IsDir(), nil
}

const defaultCommandTimeout = 5 * time.Minute
const maxCommandOutput = 64 * 1024

// tailBuffer retains bounded diagnostics while continuing to drain both streams.
// os/exec serializes writes when stdout and stderr use the same writer.
type tailBuffer struct {
	data      []byte
	truncated bool
}

func (b *tailBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if len(b.data)+n > maxCommandOutput {
		b.truncated = true
		if n >= maxCommandOutput {
			b.data = append(b.data[:0], p[n-maxCommandOutput:]...)
			return n, nil
		}
		b.data = b.data[len(b.data)+n-maxCommandOutput:]
	}
	b.data = append(b.data, p...)
	return n, nil
}

func runCommand(ctx context.Context, timeout time.Duration, dir, name string, arg ...string) (string, error) {
	if timeout <= 0 {
		return "", fmt.Errorf("--command-timeout must be greater than zero (got %s)", timeout)
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, arg...)
	cmd.Dir = dir
	// Bound waits for output pipes even if a descendant escapes the process group.
	cmd.WaitDelay = time.Second
	cleanup := configureCommand(cmd)
	var output tailBuffer
	cmd.Stdout, cmd.Stderr = &output, &output
	err := cmd.Run()
	if err != nil {
		cleanup()
		// Keep the original exit/start error as well as the cancellation cause.
		err = errors.Join(err, ctx.Err())
	}
	result := strings.TrimSpace(string(output.data))
	if output.truncated {
		result = "[output truncated; last 64 KiB]\n" + result
	}
	return result, err
}

// findHelmCharts recursively walks through directories to find Helm charts.
func findHelmCharts(ctx context.Context, searchDir string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return nil, fmt.Errorf("error reading directory %s: %w", searchDir, err)
	}

	// A chart owns its vendored subcharts. Apply the same stop-at-chart rule
	// whether searchDir was reached recursively or supplied by the caller.
	isChart, err := isHelmDir(searchDir)
	if err != nil {
		return nil, err
	}
	if isChart {
		return []string{searchDir}, nil
	}

	charts := make([]string, 0)

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !entry.IsDir() {
			continue
		}

		fullPath := filepath.Join(searchDir, entry.Name())
		isChart, err := isHelmDir(fullPath)
		if err != nil {
			return nil, err
		}
		if isChart {
			charts = append(charts, fullPath)
			continue
		}

		subCharts, err := findHelmCharts(ctx, fullPath)
		if err != nil {
			return nil, err
		}
		charts = append(charts, subCharts...)
	}

	return charts, nil
}
