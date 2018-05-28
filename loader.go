package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"unicode"
)

type loader struct {
	secretSuffix      string
	descriptionSuffix string
	patternSuffix     string
	decryptCmd        string
	rootDir           string
	fsPrefix          string
	trim              bool
}

func (l loader) TrimExt(fname string) string {
	return unextended(fname, l.secretSuffix, l.patternSuffix, l.descriptionSuffix)
}

func (l loader) Load(s string) (secret, error) {
	if !strings.HasPrefix(s, l.fsPrefix) {
		return secret{}, fmt.Errorf("path doesn't have expected prefix %v: %v", l.fsPrefix, s)
	}

	secPath := l.resolve(s + l.secretSuffix)
	secVal, err := decrypt(l.decryptCmd, secPath)
	if err != nil {
		return secret{}, fmt.Errorf("error loading %v: %v", secPath, err)
	}

	description, err := readVal(l.resolve(s + l.descriptionSuffix))
	if err != nil {
		return secret{}, err
	}

	pattern, err := readVal(l.resolve(s + l.patternSuffix))
	if err != nil {
		return secret{}, err
	}

	return secret{
		Name:        s[len(l.fsPrefix):],
		Value:       l.sanitize(secVal),
		Description: l.sanitize(description),
		Pattern:     l.sanitize(pattern),
	}, nil
}

func (l loader) resolve(p string) string {
	if l.rootDir != "" {
		return path.Join(l.rootDir, p)
	}
	return p
}

func (l loader) sanitize(b []byte) string {
	if l.trim {
		return strings.TrimRightFunc(string(b), unicode.IsSpace)
	}
	return string(b)
}

func readVal(fname string) ([]byte, error) {
	val, err := ioutil.ReadFile(fname)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("error reading %v: %v", fname, err)
	}
	return val, nil
}

func decrypt(decryptCmd, path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	cmd := exec.Command(decryptCmd, path)
	cmd.Stderr = os.Stderr

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func unextended(path string, extensions ...string) string {
	for _, extension := range extensions {
		if strings.HasSuffix(path, extension) {
			return path[:len(path)-len(extension)]
		}
	}
	return ""
}
