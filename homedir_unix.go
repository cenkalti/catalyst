// +build darwin dragonfly freebsd linux netbsd openbsd

// took it from https://github.com/mitchellh/go-homedir

package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func catalystDir() (string, error) {
	home, err := homeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "Application Support", "Catalyst"), nil
}

func homeDir() (string, error) {
	// First prefer the HOME environmental variable
	if home := os.Getenv("HOME"); home != "" {
		return home, nil
	}

	// If that fails, try getent
	var stdout bytes.Buffer
	cmd := exec.Command("getent", "passwd", strconv.Itoa(os.Getuid()))
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		// If "getent" is missing, ignore it
		if err != exec.ErrNotFound {
			return "", err
		}
	} else {
		if passwd := strings.TrimSpace(stdout.String()); passwd != "" {
			// username:password:uid:gid:gecos:home:shell
			passwdParts := strings.SplitN(passwd, ":", 7)
			if len(passwdParts) > 5 {
				return passwdParts[5], nil
			}
		}
	}

	// If all else fails, try the shell
	stdout.Reset()
	cmd = exec.Command("sh", "-c", "cd && pwd")
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}

	result := strings.TrimSpace(stdout.String())
	if result == "" {
		return "", errors.New("blank output when reading home directory")
	}

	return result, nil
}
