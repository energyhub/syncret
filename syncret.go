package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

type secret struct {
	Name        string `json:"name"`
	Value       string `json:"-"`
	Description string `json:"description,omitempty"`
	Pattern     string `json:"pattern,omitempty"`
}

const doc = `Usage of %s [FILE ...]:

Synchronizes a directory of encrypted secrets and metadata with AWS's parameter store.

By default, just prints metadata to stdout; provide the -commit flag to upload.
				
If files are provided as arguments, they will be used; otherwise, paths will be read from stdin.

`

var commit = flag.Bool("commit", false, "Sync changes to the parameter store rather than just printing metadata")
var prefix = flag.String("prefix", "", "A prefix present in the FS but not in the parameter store")
var trim = flag.Bool("trim", true, "Trim trailing whitespace from input data")
var rootDir = flag.String("root", "", "Directory relative to which paths are interpreted")

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), doc, os.Args[0])
		flag.PrintDefaults()
	}
}

func getPaths(in io.Reader, args []string) []string {
	var paths []string
	if len(args) > 0 {
		paths = args
	} else {
		log.Println("Reading secret paths from stdin...")
		for scanner := bufio.NewScanner(in); scanner.Scan(); {
			paths = append(paths, scanner.Text())
		}
	}
	return paths
}

func run(loader loader, handler handler, paths []string) error {
	secrets, err := loader.LoadAll(paths)
	if err != nil {
		return err
	}

	if len(secrets) == 0 {
		return nil
	}

	for _, secret := range secrets {
		if err := handler.Handle(secret); err != nil {
			return err
		}
		log.Printf("Successfully synced: %s", secret.Name)
	}

	return nil
}

func envMap(environ []string) map[string]string {
	env := make(map[string]string)
	for _, val := range environ {
		parts := strings.SplitN(val, "=", 2)
		env[parts[0]] = parts[1]
	}
	return env
}

func main() {
	flag.Parse()

	loader, err := newLoader(envMap(os.Environ()), *rootDir, *prefix, *trim)
	if err != nil {
		log.Fatalf("Error creating fsLoader: %v", err)
	}

	var handler handler
	if *commit {
		handler = newCommitter()
	} else {
		handler = newPrinter(os.Stdout)
	}

	paths := getPaths(os.Stdin, flag.Args())
	log.Printf("Found %d paths", len(paths))

	if err := run(loader, handler, paths); err != nil {
		log.Fatal(err)
	}
}
