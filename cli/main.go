package main

import (
	"fmt"
	"os"

	"github.com/germtb/gap/cli/cmd"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "init":
		if err := cmd.RunInit(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "gap: %v\n", err)
			os.Exit(1)
		}
	case "codegen":
		if err := cmd.RunCodegen(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "gap: %v\n", err)
			os.Exit(1)
		}
	case "run":
		if err := cmd.RunRun(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "gap: %v\n", err)
			os.Exit(1)
		}
	case "build":
		if err := cmd.RunBuild(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "gap: %v\n", err)
			os.Exit(1)
		}
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "gap: unknown command %q\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`gap - Full-stack Go + TypeScript framework

Usage:
  gap <command> [arguments]

Commands:
  init <name>    Create a new gap project
  codegen        Run proto codegen (Go + TypeScript)
  run [path]     Start server and client dev server
  build [path]   Build for production (server binary + client bundle)
  help           Show this help message

Init Options:
  --module <path>          Go module path (default: project name)
  --framework react|vanilla  Client framework (default: react)
  -y                       Skip confirmation, use defaults

Codegen Options:
  --proto <file>         Proto file path (default: proto/service.proto)
  --go-out <dir>         Go output directory (default: server/generated)
  --ts-out <dir>         TypeScript output directory (default: client/src/generated)
  --routes-dir <dir>     Routes directory (default: client/src/routes)
  --preload-out <path>   Preload config output (default: server/generated/preload_routes.go)
  --force                Force codegen even if proto hasn't changed

Build Options:
  -o <dir>               Output directory (default: <path>/build)

Examples:
  gap init myapp -y && gap run myapp
  gap run .
  gap run ./examples/with-auth
  gap build . -o dist

Use "gap help" for more information.`)
}
