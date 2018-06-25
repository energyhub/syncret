package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

const doc = `Usage of %s [FILE ...]:

Synchronizes a directory of encrypted secrets and metadata with AWS's parameter store.

By default, just prints metadata to stdout; provide the -commit flag to upload.
				
If files are provided as arguments, they will be used; otherwise, paths will be read from stdin.

`

var commit = flag.Bool("commit", false, "Sync changes to the parameter store rather than just printing metadata")

// the core struct; json serializable but drops value when so serialized.
type secret struct {
	Name        string `json:"name"`
	Value       string `json:"-"`
	Description string `json:"description,omitempty"`
	Pattern     string `json:"pattern,omitempty"`
}

// "syncs" a secret, which either succeeds or fails with an error
type syncer interface {
	Sync(secret secret) error
}

// given a list of paths, return the secrets found within or an error
// basically pulled out for testing
type loader interface {
	LoadAll(paths []string) ([]secret, error)
}

func init() {
	// overwrite default usage text
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), doc, os.Args[0])
		flag.PrintDefaults()
	}
}

// get a list of paths, either from stdin or from CLI arguments
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

// given the paths, a loader, and a syncer, load the secret in each path and sync it
func run(loader loader, syncer syncer, paths []string) error {
	secrets, err := loader.LoadAll(paths)
	if err != nil {
		return err
	}

	if len(secrets) == 0 {
		return nil // no op
	}

	for _, secret := range secrets {
		if err := syncer.Sync(secret); err != nil {
			return err
		}
		log.Printf("Successfully synced: %s", secret.Name)
	}

	return nil
}

func main() {
	flag.Parse()

	loader, err := newLoader()
	if err != nil {
		log.Fatalf("Error creating fsLoader: %v", err)
	}

	var handler syncer
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
