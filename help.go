package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

func parseCLIFlags(flags *flag.FlagSet, args []string, printHelp func(*flag.FlagSet)) {
	// Parse writes flag errors before returning. Choose where usage goes afterward.
	flags.Usage = func() {}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printRequestedHelp(flags, printHelp)
			os.Exit(0)
		}
		printHelp(flags)
		os.Exit(2)
	}
}

func printRequestedHelp(flags *flag.FlagSet, printHelp func(*flag.FlagSet)) {
	flags.SetOutput(os.Stdout)
	printHelp(flags)
}

func printGlobalHelp(flags *flag.FlagSet) {
	out := flags.Output()
	fmt.Fprintln(out, "Usage: k8s-tools [global options] <command> [command options] [arguments]")
	fmt.Fprintln(out, "\nGlobal options:")
	flags.PrintDefaults()
	fmt.Fprintln(out, "\nAvailable commands:")
	fmt.Fprintln(out, "  update  Update Helm chart dependencies in the specified directory.")
	fmt.Fprintln(out, "  help   Show this help message or help for a specific command (e.g., k8s-tools help update).")
	fmt.Fprintln(out, "\nUse \"k8s-tools <command> --help\" for more information about a command.")
}

func printUpdateHelp(cmd *flag.FlagSet) {
	out := cmd.Output()
	fmt.Fprintln(out, "Usage: k8s-tools [--threads N] [--command-timeout 5m] update [directory]")
	fmt.Fprintln(out, "\nOptions:")
	cmd.PrintDefaults()
	fmt.Fprintln(out, "\nArguments:")
	fmt.Fprintln(out, "  directory  The directory to search for root Helm charts (default: $HOME/repos).")
	fmt.Fprintln(out, "\nRuns helm dependency update for each root chart. Global flags must precede update.\nEach Helm command has a positive deadline (default: 5m). SIGINT/SIGTERM cancel pending and running work.")
}
