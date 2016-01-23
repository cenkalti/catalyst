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
	"strings"

	"github.com/cenkalti/catalyst/Godeps/_workspace/src/github.com/kardianos/osext"
	"github.com/cenkalti/catalyst/Godeps/_workspace/src/github.com/mitchellh/go-homedir"
	"github.com/mitchellh/ioprogress"
	"github.com/termie/go-shutil"
)

var configFile = flag.String("config", "", "")

type Config struct {
	Actions []Action `json:"actions"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type Action struct {
	Type        string `json:"type"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

func catalystDir() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support", "Catalyst"), nil
}

func main() {
	flag.Parse()
	execFolder, err := osext.ExecutableFolder()
	if err != nil {
		log.Panic(err)
	}
	bundleDir := execFolder[0 : len(execFolder)-15]
	log.Println("bundle dir", bundleDir)
	if *configFile == "" {
		log.Println("-config flag is not set, trying catalyst.json near binary")
		*configFile = filepath.Join(execFolder, "catalyst.json")
	}
	f, err := os.Open(*configFile)
	if err != nil {
		log.Panic(err)
	}
	var config Config
	dec := json.NewDecoder(f)
	err = dec.Decode(&config)
	if err != nil {
		log.Panic(err)
	}
	f.Close()
	root, err := catalystDir()
	if err != nil {
		log.Panic(err)
	}
	err = os.Mkdir(root, 0755)
	if err != nil && !os.IsExist(err) {
		log.Panic(err)
	}
	config.Command = strings.Replace(config.Command, "${CATALYST_DIR}", root, -1)
	config.Command = strings.Replace(config.Command, "${BUNDLE_DIR}", bundleDir, -1)
	for i := range config.Args {
		config.Args[i] = strings.Replace(config.Args[i], "${CATALYST_DIR}", root, -1)
		config.Args[i] = strings.Replace(config.Args[i], "${BUNDLE_DIR}", bundleDir, -1)
	}
	for _, action := range config.Actions {
		action.Source = strings.Replace(action.Source, "${CATALYST_DIR}", root, -1)
		action.Source = strings.Replace(action.Source, "${BUNDLE_DIR}", bundleDir, -1)
		action.Destination = strings.Replace(action.Destination, "${CATALYST_DIR}", root, -1)
		action.Destination = strings.Replace(action.Destination, "${BUNDLE_DIR}", bundleDir, -1)
		log.Println("action", action.Type, action.Source, action.Destination)
		switch action.Type {
		case "get":
			if _, err := os.Stat(action.Destination); err == nil {
				log.Println("File exists: ", action.Destination)
				continue
			}
			log.Println("Downloading", action.Source)
			tmp, err := ioutil.TempFile("", "catalyst-")
			if err != nil {
				log.Panic(err)
			}
			resp, err := http.Get(action.Source)
			if err != nil {
				log.Panic(err)
			}
			if resp.ContentLength > 0 {
				progressR := &ioprogress.Reader{
					Reader: resp.Body,
					Size:   resp.ContentLength,
				}
				_, err = io.Copy(tmp, progressR)
			}
			_, err = io.Copy(tmp, resp.Body)
			if err != nil {
				log.Panic(err)
			}
			resp.Body.Close()
			err = tmp.Close()
			if err != nil {
				log.Panic(err)
			}
			if resp.Header.Get("Content-Type") == "application/zip" || strings.HasSuffix(action.Source, ".zip") {
				log.Println("Extracting zip file")
				unzip := exec.Command("unzip", tmp.Name(), "-d", action.Destination)
				err = unzip.Run()
				if err != nil {
					log.Panic(err)
				}
				os.Remove(tmp.Name())
			} else {
				err = os.Rename(tmp.Name(), action.Destination)
				if err != nil {
					log.Panic(err)
				}
			}
		case "copyFile":
			err = shutil.CopyFile(action.Source, action.Destination, false)
			if err != nil {
				log.Println(err)
			}
		case "copyTree":
			log.Println("copy tree", action.Source, action.Destination)
			err = shutil.CopyTree(action.Source, action.Destination, nil)
			if err != nil {
				log.Println(err)
			}
		}
	}
	cmd := exec.Command(config.Command, config.Args...)
	err = cmd.Run()
	if err != nil {
		log.Panic(err)
	}
}
