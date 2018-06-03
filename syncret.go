package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"io"
	"log"
	"os"
	"strings"
)

type secret struct {
	Name        string `json:"name"`
	Value       string `json:"-"`
	Description string `json:"description"`
	Pattern     string `json:"pattern"`
}

var commit = flag.Bool("commit", false, "Sync changes to the parameter store rather than just printing metadata")
var prefix = flag.String("prefix", "", "A prefix present in the FS but not in the parameter store")
var trim = flag.Bool("trim", true, "Trim trailing whitespace from input data")
var rootDir = flag.String("root", "", "Directory relative to which paths are interpreted")

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s [FILE ...]:\n\n", os.Args[0])
		fmt.Fprint(flag.CommandLine.Output(),
			"Synchronizes a directory of encrypted secrets and metadata with AWS's parameter store.\n\n"+
				"By default, just prints metadata to stdout; provide the -commit flag to upload.\n\n"+
				"If files are provided as arguments, they will be used; otherwise, paths will be read from stdin.\n\n")
		flag.PrintDefaults()
	}
}

func run(loader loader, in io.Reader, handler handler) error {
	var paths []string
	if len(flag.Args()) > 0 {
		paths = flag.Args()
	} else {
		log.Println("Reading secret paths from stdin...")
		for scanner := bufio.NewScanner(in); scanner.Scan(); {
			paths = append(paths, scanner.Text())
		}
	}
	log.Printf("Found %d paths", len(paths))

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
		log.Fatalf("Error creating loader: %v", err)
	}

	var handler handler
	if *commit {
		handler = &committer{ssm.New(session.Must(session.NewSession()))}
	} else {
		handler = newPrinter(os.Stdout)
	}

	if err := run(loader, os.Stdin, handler); err != nil {
		log.Fatal(err)
	}
}
