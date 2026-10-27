package main

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

func update(dir string) (bool, error) {
	slog.Info("Updating chart", slog.String("dir", dir))
	fetchOutput, err := runCommand(dir, "helm", "dependency", "update")
	if err != nil {
		slog.Debug("Error updating chart", slog.String("dir", dir), slog.Any("error", err), slog.String("output", fetchOutput))
		return false, fmt.Errorf("error fetching: %w", err)
	}
	return true, nil
}

func updateAll(absSearchDir string, threadCount int) error {
	if threadCount <= 0 {
		return fmt.Errorf("--threads must be greater than zero (got %d)", threadCount)
	}

	charts, err := findHelmCharts(absSearchDir)
	if err != nil {
		return fmt.Errorf("error walking directory %s: %w", absSearchDir, err)
	}

	var wg sync.WaitGroup
	failures := make([]error, len(charts))

	semaphore := make(chan struct{}, threadCount) // Limit to threadCount concurrent operations
	defer close(semaphore)

	wg.Add(len(charts))
	for i, chart := range charts {
		semaphore <- struct{}{} // Acquire a slot

		go func(i int, chart string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release the slot

			_, err := update(chart)
			if err != nil {
				failures[i] = fmt.Errorf("chart %s: %w", chart, err)
				slog.Error("Error updating chart", slog.String("chart", chart), slog.Any("error", err))
			} else {
				slog.Info("Updated chart", slog.String("chart", chart))
			}
		}(i, chart)
	}

	wg.Wait() // Wait for all goroutines to finish

	return errors.Join(failures...)
}
