package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
)

var version = "1.0.0"

var (
	defaultThreadCount = runtime.NumCPU()
	defaultLogLevel    = "info"
)

var (
	showVersion     = flag.Bool("version", false, "Print the application version")
	verbose         = flag.Bool("v", false, "Enable verbose (DEBUG level) logging (global)")
	logLevel        = flag.String("log-level", defaultLogLevel, "Set log level: debug, info, warn, error (global)")
	threadCount     = flag.Int("threads", defaultThreadCount, "Number of threads to use for Helm chart updates (global)")
	kubeconfig      = flag.String("kubeconfig", "", "Path to the kubeconfig file (global)")
	namespace       = flag.String("namespace", "default", "Kubernetes namespace to use (global)")
	explicitContext = flag.String("context", "", "Kubernetes context to use (global)")
)

func main() {
	flag.Usage = printGlobalHelp // Set global usage
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	if *verbose {
		initLogger("debug")
	} else {
		initLogger(*logLevel)
	}

	args := flag.Args()
	if len(args) < 1 {
		printGlobalHelp()
		os.Exit(1)
	}

	switch args[0] {
	case "update":
		handleUpdateCmd(args[1:])
	case "help":
		// Allow "help <command>" for specific help
		if len(args) <= 1 {
			printGlobalHelp()
			os.Exit(0)
		}

		switch args[1] {
		case "update":
			printUpdateHelp(flag.NewFlagSet("update", flag.ExitOnError)) // Pass a dummy flagset for consistency
		default:
			fmt.Printf("Unknown command: %s\n\n", args[1])
			printGlobalHelp()
		}
	default:
		fmt.Printf("Unknown command: %s\n\n", args[0])
		printGlobalHelp()
		os.Exit(1)
	}
}
