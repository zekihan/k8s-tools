package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

func update(ctx context.Context, timeout time.Duration, dir string) (bool, error) {
	slog.Info("Updating chart", slog.String("dir", dir))
	fetchOutput, err := runCommand(ctx, timeout, dir, "helm", "dependency", "update")
	if err != nil {
		slog.Debug("Error updating chart", slog.String("dir", dir), slog.Any("error", err), slog.String("output", fetchOutput))
		if fetchOutput != "" {
			return false, fmt.Errorf("error fetching: %w; output: %s", err, fetchOutput)
		}
		return false, fmt.Errorf("error fetching: %w", err)
	}
	return true, nil
}

func updateAll(ctx context.Context, absSearchDir string, threadCount int, timeout time.Duration) error {
	if threadCount <= 0 {
		return fmt.Errorf("--threads must be greater than zero (got %d)", threadCount)
	}

	if timeout <= 0 {
		return fmt.Errorf("--command-timeout must be greater than zero (got %s)", timeout)
	}
	charts, err := findHelmCharts(ctx, absSearchDir)
	if err != nil {
		return fmt.Errorf("error walking directory %s: %w", absSearchDir, err)
	}

	var wg sync.WaitGroup
	failures := make([]error, len(charts))

	// A fixed worker pool bounds both subprocesses and goroutines. Unbuffered
	// dispatch and a second context check stop queued charts on cancellation.
	jobs := make(chan int)
	for worker := 0; worker < min(threadCount, len(charts)); worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				if ctx.Err() != nil {
					return
				}
				chart := charts[i]
				_, err := update(ctx, timeout, chart)
				if err != nil {
					failures[i] = fmt.Errorf("chart %s: %w", chart, err)
					slog.Error("Error updating chart", slog.String("chart", chart), slog.Any("error", err))
				} else {
					slog.Info("Updated chart", slog.String("chart", chart))
				}
			}
		}()
	}
dispatch:
	for i := range charts {
		if ctx.Err() != nil {
			break
		}
		select {
		case <-ctx.Done():
			break dispatch
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()
	// Cancellation must fail even with no charts, without discarding earlier errors.
	return errors.Join(append(failures, ctx.Err())...)
}
