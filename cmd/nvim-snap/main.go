package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "capture":
		cmdCapture(os.Args[2:])
	case "normalize":
		cmdNormalize(os.Args[2:])
	case "compare":
		cmdCompare(os.Args[2:])
	case "list":
		cmdList(os.Args[2:])
	case "init":
		cmdInit(os.Args[2:])
	case "regression":
		cmdRegression(os.Args[2:])
	case "golden":
		cmdGolden(os.Args[2:])
	default:
		usage()
		os.Exit(2)
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
