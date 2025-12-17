package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func runAdd(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: codex add <path>")
		return
	}
	if err := ensureRepo(); err != nil {
		fmt.Println(err)
		return
	}
	path := args[0]
	f, err := os.Open(path)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer f.Close()
	key, data, err := hashObject(f)
	if err != nil {
		fmt.Println("Error hashing file:", err)
		return
	}
	// write raw data as object
	if err := writeObject(key, data); err != nil {
		fmt.Println("Error writing object:", err)
		return
	}
	if err := stageObject(key); err != nil {
		fmt.Println("Error staging object:", err)
		return
	}
	fmt.Printf("Added %s as object %s\n", filepath.Base(path), key)
}
