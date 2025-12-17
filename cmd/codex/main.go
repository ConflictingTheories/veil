package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: codex <command> [args]")
		fmt.Println("Commands: init, add, commit, status, entity, annotate, push, server")
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "init":
		runInit()
	case "add":
		runAdd(os.Args[2:])
	case "commit":
		runCommit(os.Args[2:])
	case "status":
		runStatus()
	case "entity":
		runEntity(os.Args[2:])
	case "annotate":
		runAnnotate(os.Args[2:])
	case "push":
		runPush(os.Args[2:])
	case "server":
		runServer(os.Args[2:])
	case "gui":
		runGUI(os.Args[2:])
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		os.Exit(2)
	}
}
