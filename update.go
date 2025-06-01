package main

import (
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
	charts, err := findHelmCharts(absSearchDir)
	if err != nil {
		return fmt.Errorf("error walking directory %s: %v", absSearchDir, err)
	}

	var wg sync.WaitGroup

	semaphore := make(chan struct{}, threadCount) // Limit to threadCount concurrent operations
	defer close(semaphore)

	wg.Add(len(charts))
	for _, chart := range charts {
		semaphore <- struct{}{} // Acquire a slot

		go func(chart string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release the slot

			ok, err := update(chart)
			if !ok && err != nil {
				slog.Error("Error updating chart", slog.String("chart", chart), slog.Any("error", err))
			} else {
				slog.Info("Updated chart", slog.String("chart", chart))
			}
		}(chart)
	}

	wg.Wait() // Wait for all goroutines to finish

	return nil
}
