package main

import (
	"fmt"
)

func runInit() {
	if err := initRepo(); err != nil {
		fmt.Println("Error initializing repository:", err)
		return
	}
	fmt.Println("Initialized empty Codex repository in .codex/")
}
