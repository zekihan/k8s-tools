package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
)

func handleUpdateCmd(args []string) {
	pullCmd := flag.NewFlagSet("update", flag.ExitOnError)
	pullCmd.Usage = func() { printUpdateHelp(pullCmd) }
	err := pullCmd.Parse(args)
	if err != nil {
		slog.Error("Error parsing pull command", slog.Any("error", err))
		os.Exit(1)
	}

	if pullCmd.NArg() > 1 {
		slog.Error("expected at most one directory argument; global flags must precede update")
		os.Exit(1)
	}

	searchDir := os.ExpandEnv("$HOME/repos")
	if pullCmd.NArg() > 0 {
		searchDir = pullCmd.Arg(0)
	}

	absSearchDir, err := absoluteUpdateDir(searchDir)
	if err != nil {
		slog.Error("Error resolving update directory", slog.String("dir", searchDir), slog.Any("error", err))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("Starting Helm dependency updates", slog.String("dir", absSearchDir))

	if err := updateAll(ctx, absSearchDir, *threadCount, *commandTimeout); err != nil {
		slog.Error("Error updating Helm charts", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Helm chart dependencies update completed", slog.String("dir", absSearchDir))
}

func absoluteUpdateDir(dir string) (string, error) {
	separator := string(filepath.Separator)
	if !slices.Contains(strings.Split(dir, separator), "..") {
		return filepath.Abs(dir)
	}

	// Resolve symlinks before a parent component can remove them from the path.
	if !filepath.IsAbs(dir) {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		dir = wd + separator + dir
	}
	return filepath.EvalSymlinks(dir)
}
