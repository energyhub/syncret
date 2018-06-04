package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

const (
	decryptEnvVar     = "SYNCRET_DECRYPT"
	secretEnvVar      = "SYNCRET_SUFFIX"
	descriptionEnvVar = "SYNCRET_DESCRIPTION_SUFFX"
	patternEnvVar     = "SYNCRET_PATTERN_SUFFX"
)

var defaults = map[string]string{
	decryptEnvVar:     "cat",
	secretEnvVar:      ".gpg",
	descriptionEnvVar: ".description",
	patternEnvVar:     ".pattern",
}

type loader interface {
	LoadAll(paths []string) ([]secret, error)
}

type fsLoader struct {
	secretSuffix      string
	descriptionSuffix string
	patternSuffix     string
	decryptCmd        string
	rootDir           string
	fsPrefix          string
	trim              bool
}

func newLoader(env map[string]string, rootDir string, prefix string, trim bool) (loader, error) {
	envSuffix := func(name string, defaultVal string) string {
		suffix := strings.TrimLeft(env[name], ".")
		if suffix == "" {
			return defaultVal
		}
		return "." + suffix
	}

	decryptMethod := defaults[decryptEnvVar]
	if method, ok := env[decryptEnvVar]; ok {
		decryptMethod = method
	}

	absRoot := ""
	if rootDir != "" {
		root, err := filepath.Abs(rootDir)
		if err != nil {
			return fsLoader{}, fmt.Errorf("error finding absolute path for %v: %v", rootDir, err)
		}
		absRoot = root
	}

	return fsLoader{
		secretSuffix:      envSuffix(secretEnvVar, defaults[secretEnvVar]),
		descriptionSuffix: envSuffix(descriptionEnvVar, defaults[descriptionEnvVar]),
		patternSuffix:     envSuffix(patternEnvVar, defaults[patternEnvVar]),
		decryptCmd:        decryptMethod,
		rootDir:           absRoot,
		fsPrefix:          prefix,
		trim:              trim,
	}, nil
}

func (l fsLoader) LoadAll(paths []string) ([]secret, error) {
	var secrets []secret

	seen := make(map[string]bool)
	for _, p := range paths {
		name := unextended(p, l.secretSuffix, l.patternSuffix, l.descriptionSuffix)
		if name == "" {
			return nil, fmt.Errorf("unrecognized path: %v", p)
		}

		if !seen[name] {
			seen[name] = true
			secret, err := l.load(name)
			if err != nil {
				return nil, err
			}
			secrets = append(secrets, secret)
		}
	}

	return secrets, nil
}

func (l fsLoader) load(s string) (secret, error) {
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

	name := s[len(l.fsPrefix):]
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}

	return secret{
		Name:        name,
		Value:       l.sanitize(secVal),
		Description: l.sanitize(description),
		Pattern:     l.sanitize(pattern),
	}, nil
}

func (l fsLoader) resolve(p string) string {
	if l.rootDir != "" {
		return path.Join(l.rootDir, p)
	}
	return p
}

func (l fsLoader) sanitize(b []byte) string {
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
