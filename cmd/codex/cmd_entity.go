package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
)

type Entity struct {
	URN        string                 `json:"urn"`
	Type       string                 `json:"type"`
	Labels     map[string]string      `json:"labels"`
	Properties map[string]interface{} `json:"properties"`
}

func runEntity(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: codex entity add --id <id> --type <type> --label en=Name")
		return
	}
	switch args[0] {
	case "add":
		flags := flag.NewFlagSet("entity add", flag.ExitOnError)
		id := flags.String("id", "", "Entity id (e.g., achilles)")
		typ := flags.String("type", "", "Entity type (Character, Place)")
		labels := flags.String("label", "", "Label as lang=Label (e.g., en=Achilles)")
		flags.Parse(args[1:])
		if *id == "" || *typ == "" {
			fmt.Println("--id and --type required")
			return
		}
		e := Entity{URN: "urn:codex:entity/" + *id, Type: *typ, Labels: map[string]string{}, Properties: map[string]interface{}{}}
		if *labels != "" {
			// simple parsing for one label
			var lang, label string
			n, _ := fmt.Sscanf(*labels, "%2s=%s", &lang, &label)
			if n == 2 {
				e.Labels[lang] = label
			}
		}
		b, _ := json.MarshalIndent(e, "", "  ")
		key, _, _ := hashObject(bytesReader(b))
		_ = writeObject(key, b)
		_ = stageObject(key)
		fmt.Printf("Added entity %s (object %s)\n", e.URN, key)
	default:
		fmt.Println("Unknown entity subcommand")
	}
}

// helper to create io.Reader from bytes
func bytesReader(b []byte) *readerWrapper { return &readerWrapper{data: b} }

type readerWrapper struct {
	data []byte
	pos  int
}

func (r *readerWrapper) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
