package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
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
		log.Fatal("config argument is missing")
	}
	f, err := os.Open(*configFile)
	if err != nil {
		log.Fatal(err)
	}
	var config Config
	dec := json.NewDecoder(f)
	err = dec.Decode(&config)
	if err != nil {
		log.Fatal(err)
	}
	f.Close()
	basedir, err := catalystDir()
	if err != nil {
		log.Fatal(err)
	}
	err = os.Mkdir(basedir, 0755)
	if err != nil && !os.IsExist(err) {
		log.Fatal(err)
	}
	for _, file := range config.Files {
		src := file.Source
		dst := filepath.Join(basedir, file.Destination)
		if _, err := os.Stat(dst); err == nil {
			log.Println("File exists: ", dst)
			continue
		}
		log.Println("Downloading", src)
		tmp, err := ioutil.TempFile("", "catalyst-")
		if err != nil {
			log.Fatal(err)
		}
		resp, err := http.Get(src)
		if err != nil {
			log.Fatal(err)
		}
		_, err = io.Copy(tmp, resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		tmp.Close()
		resp.Body.Close()
		err = os.Rename(tmp.Name(), dst)
		if err != nil {
			log.Fatal(err)
		}
	}
	cmd := exec.Command(config.Command, config.Args...)
	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
