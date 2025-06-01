package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"
)

func handleUpdateCmd(args []string) {
	pullCmd := flag.NewFlagSet("update", flag.ExitOnError)
	pullCmd.Usage = func() { printUpdateHelp(pullCmd) }
	err := pullCmd.Parse(args)
	if err != nil {
		slog.Error("Error parsing pull command", slog.Any("error", err))
		os.Exit(1)
	}

	searchDir := "$HOME/repos"
	if pullCmd.NArg() > 0 {
		searchDir = pullCmd.Arg(0)
	}

	searchDir = os.ExpandEnv(searchDir)
	absSearchDir, err := filepath.Abs(searchDir)
	if err != nil {
		slog.Error("Error getting absolute path", slog.String("dir", searchDir), slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Starting git pull script", slog.String("dir", absSearchDir))

	if err := updateAll(absSearchDir, *threadCount); err != nil {
		slog.Error("Error updating Helm charts", slog.Any("error", err))
		os.Exit(1)
	}

	slog.Info("Helm chart dependencies update completed", slog.String("dir", absSearchDir))
}
