package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"log"
	"os"
	"path/filepath"
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

func newLoader() (loader, error) {
	envSuffix := func(name string, defaultVal string) string {
		suffix := strings.TrimLeft(os.Getenv(name), ".")
		if suffix == "" {
			return defaultVal
		}
		return "." + suffix
	}

	decryptMethod := "cat"
	if method, ok := os.LookupEnv("SYNCRET_DECRYPT"); ok {
		decryptMethod = method
	}

	absRoot := ""
	if *rootDir != "" {
		root, err := filepath.Abs(*rootDir)
		if err != nil {
			return loader{}, fmt.Errorf("error finding absolute path for %v: %v", *rootDir, err)
		}
		absRoot = root
	}

	return loader{
		secretSuffix:      envSuffix("SYNCRET_SUFFIX", ".gpg"),
		descriptionSuffix: envSuffix("SYNCRET_DESCRIPTION_SUFFIX", ".description"),
		patternSuffix:     envSuffix("SYNCRET_PATTERN_SUFFIX", ".pattern"),
		decryptCmd:        decryptMethod,
		rootDir:           absRoot,
		fsPrefix:          *prefix,
		trim:              *trim,
	}, nil
}

func sync(loader loader, paths []string, handler Handler) error {
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
		log.Printf("Succesfully synced: %s", secret.Name)
	}

	return nil
}

func main() {
	flag.Parse()

	var handler Handler
	if *commit {
		handler = &Committer{ssm.New(session.Must(session.NewSession()))}
	} else {
		handler = NewPrinter(os.Stdout)
	}

	var paths []string
	if len(flag.Args()) > 0 {
		paths = flag.Args()
	} else {
		log.Println("Reading secret paths from stdin...")
		for scanner := bufio.NewScanner(os.Stdin); scanner.Scan(); {
			paths = append(paths, scanner.Text())
		}
	}
	log.Printf("Found %d paths", len(paths))

	loader, err := newLoader()
	if err != nil {
		log.Fatalf("Error creating loader: %v", err)
	}

	if err := sync(loader, paths, handler); err != nil {
		log.Fatalf("Failed syncing: %v", err)
	}
}
