package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
)

func handleUpdateCmd(args []string) {
	pullCmd := flag.NewFlagSet("update", flag.ContinueOnError)
	parseCLIFlags(pullCmd, args, printUpdateHelp)

	if pullCmd.NArg() > 1 {
		fmt.Fprintln(os.Stderr, "expected at most one directory argument; global flags must precede update")
		os.Exit(1)
	}

	searchDir := pullCmd.Arg(0)
	if pullCmd.NArg() == 0 {
		home := os.Getenv("HOME")
		if home == "" {
			fmt.Fprintln(os.Stderr, "HOME is unset or empty. Specify a directory for update.")
			os.Exit(1)
		}
		searchDir = home + "/repos"
	}

	absSearchDir, err := absoluteUpdateDir(searchDir)
	if err != nil {
		slog.Error("Error resolving update directory", slog.String("dir", searchDir), slog.Any("error", err))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("Starting Helm dependency updates", slog.String("dir", absSearchDir))

	// Helm uses separate process groups, so updateAll must finish before SIGPIPE
	// can terminate the CLI.
	ctx, stopBrokenPipe := signal.NotifyContext(ctx, syscall.SIGPIPE)
	err = updateAll(ctx, absSearchDir, *threadCount, *commandTimeout)
	stopBrokenPipe()
	if err != nil {
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
