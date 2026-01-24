package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(os.Args); err != nil {
		if !isSilent(err) && err.Error() != "" {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(exitCode(err))
	}
}

func run(args []string) error {
	if len(args) < 2 {
		usage()
		return usageError()
	}

	switch args[1] {
	case "capture":
		return cmdCapture(args[2:])
	case "normalize":
		return cmdNormalize(args[2:])
	case "compare":
		return cmdCompare(args[2:])
	case "list":
		return cmdList(args[2:])
	case "init":
		return cmdInit(args[2:])
	case "regression":
		return cmdRegression(args[2:])
	case "golden":
		return cmdGolden(args[2:])
	default:
		usage()
		return usageError()
	}
}

func usage() {
	fmt.Println("usage:")
	fmt.Println("  nvim-snap <command> [options]")
	fmt.Println("")
	fmt.Println("commands:")
	fmt.Println("  capture     capture a snapshot from a scenario")
	fmt.Println("  normalize   normalize a snapshot JSON")
	fmt.Println("  compare     compare two snapshot JSON files")
	fmt.Println("  list        list test cases")
	fmt.Println("  init        scaffold CI workflow")
	fmt.Println("  regression  regression commands (new/save/test)")
	fmt.Println("  golden      golden commands (new/test)")
}
