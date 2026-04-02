package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

func printUsage() {
	fmt.Println("mini-runc - a tiny container runtime")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  mini-runc run [flags] <command> [args...]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  run     create a new container and run a command inside it")
}

func main() {
	_ = godotenv.Load()

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	switch os.Args[1] {
	case "run":
		run(os.Args[2:])
	case "child":
		child()
	default:
		fmt.Printf("Unknown subcommand: %s\n\n", os.Args[1])
		printUsage()
	}
}
