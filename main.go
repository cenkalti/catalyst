package main

import (
	"archive/zip"
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
	parts := filepath.SplitList(execFolder)
	bundleDir := "${BUNDLE_DIR}"
	if len(parts) >= 2 {
		bundleDir = filepath.Join(parts[0 : len(parts)-2]...)
	}
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
	for _, file := range config.Files {
		src := file.Source
		dst := filepath.Join(root, file.Destination)
		if _, err := os.Stat(dst); err == nil {
			log.Println("File exists: ", dst)
			continue
		}
		log.Println("Downloading", src)
		tmp, err := ioutil.TempFile("", "catalyst-")
		if err != nil {
			log.Panic(err)
		}
		resp, err := http.Get(src)
		if err != nil {
			log.Panic(err)
		}
		_, err = io.Copy(tmp, resp.Body)
		if err != nil {
			log.Panic(err)
		}
		resp.Body.Close()
		if resp.Header.Get("Content-Type") == "application/zip" || strings.HasSuffix(file.Source, ".zip") {
			log.Println("Extracting zip file")
			_, err = tmp.Seek(0, os.SEEK_SET)
			if err != nil {
				log.Panic(err)
			}
			fi, err := tmp.Stat()
			if err != nil {
				log.Panic(err)
			}
			err = os.Mkdir(dst, 0755)
			if err != nil && !os.IsExist(err) {
				log.Panic(err)
			}
			r, err := zip.NewReader(tmp, fi.Size())
			if err != nil {
				log.Panic(err)
			}
			for _, f := range r.File {
				log.Println(f.Name)
				name := filepath.Join(dst, f.Name)
				fi := f.FileInfo()
				if fi.IsDir() {
					err = os.Mkdir(name, 0755)
					if err != nil {
						log.Panic(err)
					}
				} else {
					srcfile, err := f.Open()
					if err != nil {
						log.Panic(err)
					}
					dstfile, err := os.Create(name)
					if err != nil {
						log.Panic(err)
					}
					_, err = io.CopyN(dstfile, srcfile, fi.Size())
					if err != nil {
						log.Panic(err)
					}
					err = dstfile.Close()
					if err != nil {
						log.Panic(err)
					}
					srcfile.Close()
				}
			}
			tmp.Close()
			os.Remove(tmp.Name())
		} else {
			tmp.Close()
			err = os.Rename(tmp.Name(), dst)
			if err != nil {
				log.Panic(err)
			}
		}
	}
	config.Command = strings.Replace(config.Command, "${CATALYST_DIR}", root, -1)
	config.Command = strings.Replace(config.Command, "${BUNDLE_DIR}", bundleDir, -1)
	for i := range config.Args {
		config.Args[i] = strings.Replace(config.Args[i], "${CATALYST_DIR}", root, -1)
		config.Args[i] = strings.Replace(config.Args[i], "${BUNDLE_DIR}", bundleDir, -1)
	}
	cmd := exec.Command(config.Command, config.Args...)
	err = cmd.Run()
	if err != nil {
		log.Panic(err)
	}
}
