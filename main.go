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
	"time"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/pointer"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
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

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: go-callvis [flags] package\n")
	fmt.Fprintf(os.Stderr, "\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func init() {
	// Graphviz options
	flag.UintVar(&minlen, "minlen", 2, "Minimum edge length (for wider output).")
	flag.Float64Var(&nodesep, "nodesep", 0.35, "Minimum space between two adjacent nodes in the same rank (for taller output).")
	flag.Usage = usage
}

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Fprintf(os.Stderr, "go-callvis %s\n", Version)
		os.Exit(0)
	}
	if *debugFlag {
		log.SetFlags(log.Lmicroseconds)
	}

	withTests := *testFlag
	httpAddr := *httpFlag

	args := flag.Args()
	if flag.NArg() != 1 {
		usage()
	}

	t0 := time.Now()
	conf := loader.Config{Build: &build.Default}
	_, err := conf.FromArgs(args, withTests)
	if err != nil {
		log.Fatalln("invalid args:", err)
	}
	load, err := conf.Load()
	if err != nil {
		log.Fatalln("failed conf load:", err)
	}
	logf("loading.. %d imported (%d created) took: %v",
		len(load.Imported), len(load.Created), time.Since(t0))

	t0 = time.Now()
	prog := ssautil.CreateProgram(load, 0)
	prog.Build()
	pkgs := prog.AllPackages()

	var mains []*ssa.Package
	if withTests {
		for _, pkg := range pkgs {
			if main := prog.CreateTestMainPackage(pkg); main != nil {
				mains = append(mains, main)
			}
		}
		if mains == nil {
			log.Fatalln("no tests")
		}
	} else {
		mains = append(mains, ssautil.MainPackages(pkgs)...)
		if len(mains) == 0 {
			log.Fatalln("no main packages")
		}
	}
	logf("building.. %d packages (%d main) took: %v",
		len(pkgs), len(mains), time.Since(t0))

	t0 = time.Now()
	ptrcfg := &pointer.Config{
		Mains:          mains,
		BuildCallGraph: true,
	}
	result, err := pointer.Analyze(ptrcfg)
	if err != nil {
		log.Fatalln("analyze failed:", err)
	}
	logf("analysis took: %v", time.Since(t0))

	a := analysis{
		conf:   conf,
		pkgs:   pkgs,
		mains:  mains,
		result: result,
	}

	http.HandleFunc("/", a.handler)

	log.Printf("http serving at %s", httpAddr)
	if err := http.ListenAndServe(httpAddr, nil); err != nil {
		log.Fatal(err)
	}
}

type analysis struct {
	conf   loader.Config
	pkgs   []*ssa.Package
	mains  []*ssa.Package
	result *pointer.Result
}

func (a *analysis) handler(w http.ResponseWriter, r *http.Request) {
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

	groupBy := make(map[string]bool)
	for _, g := range strings.Split(group, ",") {
		g := strings.TrimSpace(g)
		if g == "" {
			continue
		}
		if g != "pkg" && g != "type" {
			http.Error(w, "invalid group option", http.StatusInternalServerError)
			return
		}
		groupBy[g] = true
	}

	var limitPaths []string
	for _, p := range strings.Split(limit, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			limitPaths = append(limitPaths, p)
		}
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

	var err error
	var focusPkg *build.Package
	if focus != "" {
		focusPkg, err = a.conf.Build.Import(focus, "", 0)
		if err != nil {
			if strings.Contains(focus, "/") {
				http.Error(w, "focus failed", http.StatusInternalServerError)
				return
			}
			// try to find package by name
			var foundPaths []string
			for _, p := range a.pkgs {
				if p.Pkg.Name() == focus {
					foundPaths = append(foundPaths, p.Pkg.Path())
				}
			}
			if len(foundPaths) == 0 {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			} else if len(foundPaths) > 1 {
				for _, p := range foundPaths {
					fmt.Fprintf(os.Stderr, " - %s\n", p)
				}
				err := fmt.Errorf("found %d packages with name %q, use import path not name", len(foundPaths), focus)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if focusPkg, err = a.conf.Build.Import(foundPaths[0], "", 0); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		logf("focusing: %v", focusPkg.ImportPath)
	}

	dot, err := printOutput(a.mains[0].Pkg, a.result.CallGraph,
		focusPkg, limitPaths, ignorePaths, includePaths, groupBy, nostd, nointer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if r.Form.Get("format") == "dot" {
		log.Println("writing dot output..")
		fmt.Fprint(w, dot)
		return
	}

	log.Println("converting dot to svg..")
	img, err := dotToImage(dot)
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
