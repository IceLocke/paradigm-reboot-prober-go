// Package main is the entry point of the fitting-calculator microservice.
//
// The fitting calculator is intentionally a SEPARATE binary from cmd/server
// so that the probe service ("查分器") remains single-purpose. See AGENTS.md
// → "保持查分器本体的单纯性" for the design principle. This binary:
//
//   - Reads best_play_records and chart metadata directly from the shared
//     database using the shared config + model layers, but has NO
//     dependency on internal/service or internal/repository (no HTTP
//     handlers, no caches, no auth logic).
//   - Runs on a configurable ticker interval (config.fitting.interval,
//     typically hours) or once with the `run --once` flag.
//   - Persists results into charts.fitting_level and a dedicated
//     chart_statistics table for offline analysis.
//
// # Subcommands
//
// The binary dispatches on the first positional argument:
//
//	fitting run [flags]       continuous or one-shot calculation
//	fitting analyze [flags]   read-only diagnostic for one chart
//
// When no subcommand is given, `run` is assumed so that existing
// invocations such as `./fitting`, `./fitting --once`, or
// `go run cmd/fitting/main.go --once` continue to behave exactly as
// before the subcommand split.
package main

import (
	"fmt"
	"os"
)

func main() {
	// Subcommand dispatch. The first positional argument selects a
	// subcommand; everything else is forwarded to that subcommand's own
	// flag parser. Unknown first arguments (including flags like "--once")
	// fall through to the default `run` subcommand — this preserves
	// backward compatibility with docker-compose commands and existing
	// shell scripts that predate the split.
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "run":
			cmdRun(os.Args[2:])
			return
		case "analyze":
			cmdAnalyze(os.Args[2:])
			return
		case "-h", "--help", "help":
			printUsage()
			return
		}
	}
	cmdRun(os.Args[1:])
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: fitting [subcommand] [flags]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Subcommands:")
	fmt.Fprintln(os.Stderr, "  run      (default) run the fitting calculator in continuous or --once mode")
	fmt.Fprintln(os.Stderr, "  analyze  read-only diagnostic for one chart (prints bucket breakdown + config sweep)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Run `fitting <subcommand> --help` to see flags for each subcommand.")
}
