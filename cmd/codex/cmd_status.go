package main

import (
	"fmt"
)

func runStatus() {
	if err := ensureRepo(); err != nil {
		fmt.Println(err)
		return
	}
	idx, err := readIndex()
	if err != nil {
		fmt.Println("Error reading index:", err)
		return
	}
	head, _ := getHEAD()
	fmt.Println("On branch main")
	if head == "" {
		fmt.Println("No commits yet")
	} else {
		fmt.Println("HEAD:", head)
	}
	fmt.Printf("Staged objects: %d\n", len(idx))
}
