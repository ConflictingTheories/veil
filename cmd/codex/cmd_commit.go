package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"time"
)

func runCommit(args []string) {
	flags := flag.NewFlagSet("commit", flag.ExitOnError)
	msg := flags.String("m", "", "Commit message")
	flags.Parse(args)
	if *msg == "" {
		fmt.Println("Usage: codex commit -m \"message\"")
		return
	}
	if err := ensureRepo(); err != nil {
		fmt.Println(err)
		return
	}
	idx, err := readIndex()
	if err != nil {
		fmt.Println("Error reading index:", err)
		return
	}
	parent, _ := getHEAD()
	c := Commit{
		Message:   *msg,
		Parent:    parent,
		Timestamp: time.Now().Unix(),
		Objects:   idx,
	}
	b, _ := json.MarshalIndent(c, "", "  ")
	hash, err := writeCommit(c)
	if err != nil {
		fmt.Println("Error writing commit:", err)
		return
	}
	if err := updateHeadCommit(hash); err != nil {
		fmt.Println("Error updating HEAD:", err)
		return
	}
	if err := clearIndex(); err != nil {
		fmt.Println("Error clearing index:", err)
		return
	}
	fmt.Printf("Committed %s\n", hash)
	_ = b
}
