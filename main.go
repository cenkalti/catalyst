package main

import (
	"encoding/json"
	"flag"
	"os"
)

var configFile = flag.String("config", "", "")

type Config struct {
	Actions    []Action `json:"actions"`
	EntryPoint []string `json:"entryPoint"`
}

type Action struct {
	Type        string `json:"type"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

func main() {
	flag.Parse()
	if *configFile == "" {
		os.Stderr.WriteString("config argument is missing\n")
		os.Exit(1)
	}
	f, err := os.Open(*configFile)
	if err != nil {
		os.Stderr.WriteString("cannot open config file: " + err.Error() + "\n")
		os.Exit(2)
	}
	var config Config
	dec := json.NewDecoder(f)
	err = dec.Decode(&config)
	if err != nil {
		os.Stderr.WriteString("cannot decode json in config: " + err.Error() + "\n")
		os.Exit(3)
	}
	f.Close()
}
