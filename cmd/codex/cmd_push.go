package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
)

func runPush(args []string) {
	flags := flag.NewFlagSet("push", flag.ExitOnError)
	flags.Parse(args)
	if flags.NArg() < 1 {
		fmt.Println("Usage: codex push <remote-url>")
		return
	}
	url := flags.Arg(0)
	if err := ensureRepo(); err != nil {
		fmt.Println(err)
		return
	}
	head, err := getHEAD()
	if err != nil {
		fmt.Println("Error reading HEAD:", err)
		return
	}
	if head == "" {
		fmt.Println("Nothing to push: no commits")
		return
	}
	// read commit object
	commitData, err := ioutil.ReadFile(".codex/objects/" + head + ".json")
	if err != nil {
		fmt.Println("Error reading commit object:", err)
		return
	}
	// POST commit
	resp, err := http.Post(url+"/push", "application/json", bytes.NewReader(commitData))
	if err != nil {
		fmt.Println("Error pushing to remote:", err)
		return
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	var out map[string]interface{}
	_ = json.Unmarshal(body, &out)
	fmt.Println("Push response:", out)
}
