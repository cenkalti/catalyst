package main

import (
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

var configFile = flag.String("config", "", "")

type Config struct {
	Files   []Action `json:"files"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
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
	basedir, err := catalystDir()
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(4)
	}
	err = os.Mkdir(basedir, 0755)
	if err != nil && !os.IsExist(err) {
		os.Stderr.WriteString("cannot create directory: " + basedir + ": " + err.Error() + "\n")
		os.Exit(5)
	}
	for _, file := range config.Files {
		src := file.Source
		dst := filepath.Join(basedir, file.Destination)
		os.Stdout.WriteString("Downloading " + src + "\n")
		out, err := os.Create(dst)
		if err != nil {
			os.Stderr.WriteString("cannot create file: " + dst + ": " + err.Error() + "\n")
			os.Exit(6)
		}
		resp, err := http.Get(src)
		if err != nil {
			os.Stderr.WriteString("cannot get file: " + src + ": " + err.Error() + "\n")
			os.Exit(7)
		}
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			os.Stderr.WriteString("cannot download file: " + src + ": " + err.Error() + "\n")
			os.Exit(8)
		}
		out.Close()
		resp.Body.Close()
	}
	cmd := exec.Command(config.Command, config.Args...)
	err = cmd.Run()
	if err != nil {
		os.Stderr.WriteString("cannot run entry point: " + err.Error() + "\n")
		os.Exit(9)
	}
}
