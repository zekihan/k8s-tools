package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// runCommand executes a command in a given directory and returns its output.
// It suppresses stdout and stderr of the command.
func runCommand(dir string, name string, arg ...string) (string, error) {
	cmd := exec.Command(name, arg...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

// findHelmCharts recursively walks through directories to find Helm charts.
func findHelmCharts(searchDir string) ([]string, error) {
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
		if entry.Name() == "Chart.yaml" {
			isChart, err := isHelmDir(searchDir)
			if err != nil {
				return nil, err
			}
			if isChart {
				charts = append(charts, searchDir)
			}
			continue
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

		subCharts, err := findHelmCharts(fullPath)
		if err != nil {
			return nil, err
		}
		charts = append(charts, subCharts...)
	}

	return charts, nil
}
