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
	fmt.Println("  help   Show this help message or help for a specific command (e.g., k8s-tools help pull).")
	fmt.Println("\nUse \"k8s-tools <command> --help\" for more information about a command.")
}

func printUpdateHelp(cmd *flag.FlagSet) {
	fmt.Println("Usage: k8s-tools update [options] [directory]")
	fmt.Println("\nOptions:")
	cmd.PrintDefaults() // Prints command-specific flags
	fmt.Println("\nArguments:")
	fmt.Println("  directory  The directory to search for git repositories (default: $HOME/repos).")
	fmt.Println("\nThis command pulls updates from all git repositories in the specified directory.")
}
