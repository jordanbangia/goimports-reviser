package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/jordanbangia/goimports-reviser/v2/reviser"
)

// Project build specific vars
var (
	Tag       string
	Commit    string
	SourceURL string
	GoVersion string
)

const (
	projectNameArg         = "project-name"
	filePathArg            = "file-path"
	versionArg             = "version"
	removeUnusedImportsArg = "rm-unused"
	setAlias               = "set-alias"
	localPkgPrefixesArg    = "local"
)

var (
	projectName               = flag.String(projectNameArg, "", "project name (ex. : github.com/jordanbangia/goimports-reviser")
	localPackagePrefixes      = flag.String(localPkgPrefixesArg, "", "local package prefixes, comma separated")
	shouldRemoveUnusedImports = flag.Bool(removeUnusedImportsArg, false, "remove unused imports")
	shouldSetAlias            = flag.Bool(setAlias, false, "set alias for versioned package names")
	showVersion               = flag.Bool(versionArg, false, "show version")
)

func printUsage() {
	fmt.Fprintf(os.Stderr, "usage: goimports-reviser [flags] [path ...]\n")
	flag.PrintDefaults()
}

func printVersion() {
	fmt.Printf(
		"version: %s\nbuild with: %s\ntag: %s\ncommit: %s\nsource: %s\n",
		strings.TrimPrefix(Tag, "v"),
		GoVersion,
		Tag,
		Commit,
		SourceURL,
	)
}

func main() {
	flag.Usage = printUsage
	flag.Parse()

	if *showVersion {
		printVersion()
		return
	}

	if *projectName == "" {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("-%s should be set", projectNameArg))
		os.Exit(1)
	}

	if flag.NArg() == 0 {
		fmt.Fprint(os.Stderr, "no files passed")
		os.Exit(1)
	}

	for i := 0; i < flag.NArg(); i++ {
		formattedOutput, hasChange, err := reviser.Execute(
			*projectName,
			filePath,
			*localPackagePrefixes,
			reviser.WithRemoveUnusedImports(*shouldRemoveUnusedImports),
			reviser.WithUseAlias(*shouldSetAlias))
		if err != nil {
			log.Fatalf("%+v", errors.WithStack(err))
		}
	}

	if !hasChange {
		return
	}

	if err := ioutil.WriteFile(filePath, formattedOutput, 0644); err != nil {
		log.Fatalf("failed to write fixed result to file(%s): %+v", filePath, errors.WithStack(err))
	}
}
