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
	"flag"
)

const (
	decryptEnvVar     = "SYNCRET_DECRYPT"
	secretEnvVar      = "SYNCRET_SUFFIX"
	descriptionEnvVar = "SYNCRET_DESCRIPTION_SUFFIX"
	patternEnvVar     = "SYNCRET_PATTERN_SUFFIX"
)

var (
	defaults = map[string]string{
		decryptEnvVar:     "cat",
		secretEnvVar:      ".gpg",
		descriptionEnvVar: ".description",
		patternEnvVar:     ".pattern",
	}
	prefix  = flag.String("prefix", "", "A prefix present in the FS but not in the parameter store")
	rootDir = flag.String("root", "", "Directory relative to which paths are interpreted")
	trim    = flag.Bool("trim", true, "Trim trailing whitespace from input data")
)

// instantiates a new loader from CLI flags and the OS environ
func newLoader() (loader) {
	return doNewLoader(envMap(os.Environ()), *prefix, *rootDir, *trim)
}

// the basic implementation of a loader which loads stuff from the FS (the only real impl)
type fsLoader struct {
	secretSuffix      string
	descriptionSuffix string
	patternSuffix     string
	decryptCmd        string
	fsPrefix          string
	rootDir           string
	trim              bool
}

func (l fsLoader) LoadAll(paths []string) ([]secret, error) {
	var secrets []secret

	// unique by 'unextended'
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

// loads the secret for a given name, if possible
func (l fsLoader) load(s string) (secret, error) {
	if !strings.HasPrefix(s, l.fsPrefix) {
		return secret{}, fmt.Errorf("path doesn't have expected prefix %v: %v", l.fsPrefix, s)
	}

	secPath := resolve(l.rootDir, s+l.secretSuffix)
	secVal, err := decrypt(l.decryptCmd, secPath)
	if err != nil {
		return secret{}, fmt.Errorf("error loading %v: %v", secPath, err)
	}

	description, err := readVal(resolve(l.rootDir, s+l.descriptionSuffix))
	if err != nil {
		return secret{}, err
	}

	pattern, err := readVal(resolve(l.rootDir, s+l.patternSuffix))
	if err != nil {
		return secret{}, err
	}

	name := s[len(l.fsPrefix):]
	if !strings.HasPrefix(name, "/") {
		name = "/" + name
	}

	return secret{
		Name:        name,
		Value:       sanitize(secVal, l.trim),
		Description: sanitize(description, l.trim),
		Pattern:     sanitize(pattern, l.trim),
	}, nil
}

// responsible for establishing defaults etc.
func doNewLoader(env map[string]string, prefix, rootDir string, trim bool) loader {
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

	if rootDir != "" {
		root, err := filepath.Abs(rootDir)
		if err != nil {
			// nothing reasonable to do if no cwd
			panic(err)
		}
		rootDir = root
	}

	return fsLoader{
		secretSuffix:      envSuffix(secretEnvVar, defaults[secretEnvVar]),
		descriptionSuffix: envSuffix(descriptionEnvVar, defaults[descriptionEnvVar]),
		patternSuffix:     envSuffix(patternEnvVar, defaults[patternEnvVar]),
		decryptCmd:        decryptMethod,
		fsPrefix:          prefix,
		rootDir:           rootDir,
		trim:              trim,
	}
}

// reads a filename, but suppresses os not exist, so nonexistent file is
// returned as an empty string
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

// convert the os provided env list to a map
func envMap(environ []string) map[string]string {
	env := make(map[string]string)
	for _, val := range environ {
		parts := strings.SplitN(val, "=", 2)
		env[parts[0]] = parts[1]
	}
	return env
}

// (/foo/bar/baz.gpg, (.gpg, .beep, .boop)) -> /foo/bar/baz
func unextended(path string, extensions ...string) string {
	for _, extension := range extensions {
		if strings.HasSuffix(path, extension) {
			return path[:len(path)-len(extension)]
		}
	}
	return ""
}

func sanitize(p []byte, trim bool) string {
	if trim {
		return strings.TrimFunc(string(p), unicode.IsSpace)
	}
	return string(p)
}

func resolve(rootDir, p string) string {
	if rootDir != "" {
		return path.Join(rootDir, p)
	}
	return p
}
