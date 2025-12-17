package main

import (
	"encoding/json"
	"flag"
	"fmt"
)

type Annotation struct {
	TextURN   string  `json:"text_urn"`
	EntityURN string  `json:"entity_urn"`
	Start     int     `json:"start"`
	End       int     `json:"end"`
	Certainty float64 `json:"certainty"`
}

func runAnnotate(args []string) {
	flags := flag.NewFlagSet("annotate", flag.ExitOnError)
	text := flags.String("text", "", "Text URN")
	entity := flags.String("entity", "", "Entity URN")
	start := flags.Int("start", 0, "Start char index")
	end := flags.Int("end", 0, "End char index")
	cert := flags.Float64("certainty", 1.0, "Certainty 0.0-1.0")
	flags.Parse(args)
	if *text == "" || *entity == "" {
		fmt.Println("--text and --entity required")
		return
	}
	a := Annotation{TextURN: *text, EntityURN: *entity, Start: *start, End: *end, Certainty: *cert}
	b, _ := json.MarshalIndent(a, "", "  ")
	key, _, _ := hashObject(bytesReader(b))
	_ = writeObject(key, b)
	_ = stageObject(key)
	fmt.Printf("Annotated %s -> %s (object %s)\n", *text, *entity, key)
}
