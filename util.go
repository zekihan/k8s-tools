package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// isHelmDir checks if a directory contains a Helm Chart.yaml file.
func isHelmDir(dir string) bool {
	gitPath := filepath.Join(dir, "Chart.yaml")
	info, err := os.Stat(gitPath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
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
		return nil, fmt.Errorf("error reading directory %s: %v", searchDir, err)
	}

	charts := make([]string, 0)

	for _, entry := range entries {
		if entry.Name() == "Chart.yaml" {
			charts = append(charts, searchDir)
			continue
		}

		if !entry.IsDir() {
			continue
		}

		fullPath := filepath.Join(searchDir, entry.Name())
		if isHelmDir(fullPath) {
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
