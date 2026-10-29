package main

import (
	"flag"
	"fmt"
)

func printGlobalHelp() {
	fmt.Println("Usage: k8s-tools [global options] <command> [command options] [arguments]")
	fmt.Println("\nGlobal options:")
	flag.PrintDefaults() // Prints global flags
	fmt.Println("\nAvailable commands:")
	fmt.Println("  update  Update Helm chart dependencies in the specified directory.")
	fmt.Println("  help   Show this help message or help for a specific command (e.g., k8s-tools help update).")
	fmt.Println("\nUse \"k8s-tools <command> --help\" for more information about a command.")
}

func printUpdateHelp(cmd *flag.FlagSet) {
	fmt.Println("Usage: k8s-tools [--threads N] [--command-timeout 5m] update [directory]")
	fmt.Println("\nOptions:")
	cmd.PrintDefaults() // Prints command-specific flags
	fmt.Println("\nArguments:")
	fmt.Println("  directory  The directory to search for root Helm charts (default: $HOME/repos).")
	fmt.Println("\nRuns helm dependency update for each root chart. Global flags must precede update.\nEach Helm command has a positive deadline (default: 5m). SIGINT/SIGTERM cancel pending and running work.")
}
