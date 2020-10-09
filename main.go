package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/pkg/errors"

	"github.com/incu6us/goimports-reviser/reviser"
)

const (
	projectNameArg         = "project-name"
	filePathArg            = "file-path"
	versionArg             = "version"
	removeUnusedImportsArg = "rm-unused"
	setAlias               = "set-alias"
	extraGroupsArg         = "extra-groups"
)

// Project build specific vars
var (
	Tag       string
	Commit    string
	SourceURL string
	GoVersion string

	shouldShowVersion         *bool
	shouldRemoveUnusedImports *bool
	shouldSetAlias            *bool
)

var projectName, filePath, extraGroups string

func init() {
	flag.StringVar(
		&projectName,
		projectNameArg,
		"",
		"Your project name(ex.: github.com/incu6us/goimports-reviser). Required parameter.",
	)

	shouldRemoveUnusedImports = flag.Bool(
		removeUnusedImportsArg,
		false,
		"Remove unused imports. Optional parameter.",
	)

	shouldSetAlias = flag.Bool(
		setAlias,
		false,
		"Set alias for versioned package names, like 'github.com/go-pg/pg/v9'. "+
			"In this case import will be set as 'pg \"github.com/go-pg/pg/v9\"'. Optional parameter.",
	)

	flag.StringVar(&extraGroups, extraGroupsArg, "", "Extra groupings to include, comma separated list.  Ordering is same as provided. Optional parameter.")

	if Tag != "" {
		shouldShowVersion = flag.Bool(
			versionArg,
			false,
			"Show version.",
		)
	}
}

func printUsage() {
	if _, err := fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0]); err != nil {
		log.Fatalf("failed to print usage: %s", err)
	}

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
	flag.Parse()
	paths := flag.Args()
	fmt.Println(paths)
	f, err := os.Stat(paths[0])
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(f.IsDir())

	if shouldShowVersion != nil && *shouldShowVersion {
		printVersion()
		return
	}

	if err := validateInputs(projectName, filePath); err != nil {
		fmt.Printf("%s\n\n", err)
		printUsage()
		os.Exit(1)
	}

	options := []reviser.Option{}
	if shouldRemoveUnusedImports != nil && *shouldRemoveUnusedImports {
		options = append(options, reviser.WithRemoveUnusedImports(true))
	}

	if shouldSetAlias != nil && *shouldSetAlias {
		options = append(options, reviser.WithAliasForVersionSuffix(true))
	}

	if extraGroups != "" {
		options = append(options, reviser.WithExtraImportGroups(cleanExtraGroups(extraGroups)))
	}

	formattedOutput, hasChange, err := reviser.Execute(projectName, filePath, options...)
	if err != nil {
		log.Fatalf("%+v", errors.WithStack(err))
	}

	if !hasChange {
		return
	}

	if err := ioutil.WriteFile(filePath, formattedOutput, 0644); err != nil {
		log.Fatalf("failed to write fixed result to file(%s): %+v", filePath, errors.WithStack(err))
	}
}

func validateInputs(projectName, filePath string) error {
	var errMessages []string

	if projectName == "" {
		errMessages = append(errMessages, fmt.Sprintf("-%s should be set", projectNameArg))
	}

	if filePath == "" {
		errMessages = append(errMessages, fmt.Sprintf("-%s should be set", filePathArg))
	}

	if len(errMessages) > 0 {
		return errors.New(strings.Join(errMessages, "\n"))
	}

	return nil
}

func cleanExtraGroups(extraGroupsString string) []string {
	groups := strings.Split(extraGroupsString, ",")
	for i, _ := range groups {
		groups[i] = strings.TrimSpace(groups[i])
	}
	return groups
}
