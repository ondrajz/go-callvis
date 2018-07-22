// go-callvis: a tool to help visualize the call graph of a Go program.
//
package main

import (
	"flag"
	"fmt"
	"go/build"
	"log"
	"net/http"
	"os"
)

var (
	Version = "v0.4-dev"
)

var (
	focusFlag   = flag.String("focus", "main", "Focus specific package using name or import path.")
	groupFlag   = flag.String("group", "", "Grouping functions by packages and/or types [pkg, type] (separated by comma)")
	limitFlag   = flag.String("limit", "", "Limit package paths to given prefixes (separated by comma)")
	ignoreFlag  = flag.String("ignore", "", "Ignore package paths containing given prefixes (separated by comma)")
	includeFlag = flag.String("include", "", "Include package paths with given prefixes (separated by comma)")
	nostdFlag   = flag.Bool("nostd", false, "Omit calls to/from packages in standard library.")
	nointerFlag = flag.Bool("nointer", false, "Omit calls to unexported functions.")
	testFlag    = flag.Bool("tests", false, "Include test code.")
	debugFlag   = flag.Bool("debug", false, "Enable verbose log.")
	versionFlag = flag.Bool("version", false, "Show version and exit.")
	httpFlag    = flag.String("http", ":7878", "HTTP service address.")
)

func init() {
	// Graphviz options
	flag.UintVar(&minlen, "minlen", 2, "Minimum edge length (for wider output).")
	flag.Float64Var(&nodesep, "nodesep", 0.35, "Minimum space between two adjacent nodes in the same rank (for taller output).")
}

const Usage = `go-callvis: visualize call graph of a Go program.

Usage:

  go-callvis [flags] package

  Package must be main package otherwise -tests flag must be used.

Flags:

`

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Fprintf(os.Stderr, "go-callvis %s\n", Version)
		os.Exit(0)
	}
	if *debugFlag {
		log.SetFlags(log.Lmicroseconds)
	}

	args := flag.Args()
	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, Usage)
		flag.PrintDefaults()
		os.Exit(2)
	}

	tests := *testFlag
	httpAddr := *httpFlag

	a := newAnalysis(&build.Default, tests, args)

	http.HandleFunc("/", a.handler)

	log.Printf("http serving at %s", httpAddr)

	if err := http.ListenAndServe(httpAddr, nil); err != nil {
		log.Fatal(err)
	}
}

func logf(f string, a ...interface{}) {
	if *debugFlag {
		log.Printf(f, a...)
	}
}
