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
	"strings"
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

	doAnalysis(&build.Default, tests, args)

	http.HandleFunc("/", handler)

	log.Printf("http serving at %s", httpAddr)

	if err := http.ListenAndServe(httpAddr, nil); err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" && !strings.HasSuffix(r.URL.Path, ".svg") {
		http.NotFound(w, r)
		return
	}

	logf("----------------------")
	logf(" => handling request:  %v", r.URL)
	logf("----------------------")

	focus := *focusFlag
	nostd := *nostdFlag
	nointer := *nointerFlag
	group := *groupFlag
	limit := *limitFlag
	ignore := *ignoreFlag
	include := *includeFlag

	if f := r.FormValue("f"); f == "all" {
		focus = ""
	} else if f != "" {
		focus = f
	}
	if std := r.FormValue("std"); std != "" {
		nostd = false
	}
	if inter := r.FormValue("nointer"); inter != "" {
		nointer = true
	}
	if g := r.FormValue("group"); g != "" {
		group = g
	}
	if l := r.FormValue("limit"); l != "" {
		limit = l
	}
	if ign := r.FormValue("ignore"); ign != "" {
		ignore = ign
	}
	if inc := r.FormValue("include"); inc != "" {
		include = inc
	}

	var groupBy []string
	for _, g := range strings.Split(group, ",") {
		g := strings.TrimSpace(g)
		if g == "" {
			continue
		}
		if g != "pkg" && g != "type" {
			http.Error(w, "invalid group option", http.StatusInternalServerError)
			return
		}
		groupBy = append(groupBy, g)
	}

	var ignorePaths []string
	for _, p := range strings.Split(ignore, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			ignorePaths = append(ignorePaths, p)
		}
	}

	var includePaths []string
	for _, p := range strings.Split(include, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			includePaths = append(includePaths, p)
		}
	}

	var limitPaths []string
	for _, p := range strings.Split(limit, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			limitPaths = append(limitPaths, p)
		}
	}

	opts := renderOpts{
		focus:   focus,
		group:   groupBy,
		ignore:  ignorePaths,
		include: includePaths,
		limit:   limitPaths,
		nointer: nointer,
		nostd:   nostd,
	}

	output, err := Analysis.render(opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Form.Get("format") == "dot" {
		log.Println("writing dot output..")
		fmt.Fprint(w, output)
		return
	}

	log.Println("converting dot to svg..")
	img, err := dotToImage(output)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("serving file:", img)
	http.ServeFile(w, r, img)

}

func logf(f string, a ...interface{}) {
	if *debugFlag {
		log.Printf(f, a...)
	}
}
