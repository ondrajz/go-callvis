// go-callvis: a tool to help visualize the call graph of a Go program.
//
package main

import (
	"flag"
	"fmt"
	"go/build"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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

func main() {
	// Graphviz options
	flag.UintVar(&minlen, "minlen", 2, "Minimum edge length (for wider output).")
	flag.Float64Var(&nodesep, "nodesep", 0.35, "Minimum space between two adjacent nodes in the same rank (for taller output).")

	flag.Parse()

	if *versionFlag {
		fmt.Fprintf(os.Stderr, "go-callvis %s\n", Version)
		os.Exit(0)
	}
	if *debugFlag {
		log.SetFlags(log.Lmicroseconds)
	}

	args := flag.Args()
	tests := *testFlag
	/*if mains, err := getMains(conf focusFlag, groupBy, limitPaths, ignorePaths, *nostdFlag, *testFlag,, flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "go-callvis: %s\n", err)
		os.Exit(1)
	}*/
	if len(args) == 0 {
		log.Fatalln("missing arguments")
	}

	t0 := time.Now()
	conf := loader.Config{Build: &build.Default}
	_, err := conf.FromArgs(args, tests)
	if err != nil {
		log.Fatalln("invalid args:", err)
	}
	load, err := conf.Load()
	if err != nil {
		log.Fatalln("failed conf load:", err)
	}
	logf("loading took: %v", time.Since(t0))
	logf("%d imported (%d created)", len(load.Imported), len(load.Created))

	t0 = time.Now()
	prog := ssautil.CreateProgram(load, 0)
	prog.Build()
	pkgs := prog.AllPackages()
	logf("building took: %v", time.Since(t0))

	var mains []*ssa.Package
	if tests {
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
	logf("%d packages (%d main)", len(pkgs), len(mains))

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

	//handler()
	/*	fmt.Println("\n\n//-----")
		nostd = false
		handler()*/

	a := analysis{
		conf:   conf,
		pkgs:   pkgs,
		mains:  mains,
		result: result,
	}

	http.HandleFunc("/", a.handler)

	web := &url.URL{
		Scheme: "http",
		Host:   "localhost" + *httpFlag,
	}
	log.Printf("serving at %s", web)

	log.Fatal(http.ListenAndServe(*httpFlag, nil))
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

	focus := *focusFlag
	nostd := *nostdFlag
	nointer := *nointerFlag
	group := *groupFlag
	limit := *limitFlag
	ignore := *ignoreFlag
	include := *includeFlag

	f := r.FormValue("f")
	if f == "all" {
		focus = ""
	} else if f != "" {
		focus = f
	}
	std := r.FormValue("std")
	if std != "" {
		nostd = false
	}
	inter := r.FormValue("nointer")
	if inter != "" {
		nointer = true
	}
	g := r.FormValue("group")
	if g != "" {
		group = g
	}
	l := r.FormValue("limit")
	if l != "" {
		limit = l
	}
	ign := r.FormValue("ignore")
	if ign != "" {
		ignore = ign
	}
	inc := r.FormValue("include")
	if inc != "" {
		include = inc
	}

	groupBy := make(map[string]bool)
	for _, g := range strings.Split(group, ",") {
		g := strings.TrimSpace(g)
		if g == "" {
			continue
		}
		if g != "pkg" && g != "type" {
			//fmt.Fprintf(os.Stderr, "go-callvis: %s\n", "invalid group option")
			//os.Exit(1)
			http.Error(w, "invalid group option", http.StatusInternalServerError)
			return
		}
		groupBy[g] = true
	}

	limitPaths := []string{}
	for _, p := range strings.Split(limit, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			limitPaths = append(limitPaths, p)
		}
	}

	ignorePaths := []string{}
	for _, p := range strings.Split(ignore, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			ignorePaths = append(ignorePaths, p)
		}
	}

	includePaths := []string{}
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
		fmt.Fprint(w, dot)
		return
	}

	log.Println("dot to image..")
	//fmt.Fprintln(w, dot)
	img, err := dotToImage(dot)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Println("serving file:", img)
	http.ServeFile(w, r, img)
}

func dotToImage(dot string) (string, error) {
	img := "/tmp/test000.svg"
	cmd := exec.Command("/usr/bin/dot", "-Tsvg", "-o", img)
	cmd.Stdin = strings.NewReader(dot)
	/*cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", err
	}*/
	/*stdout, err := cmd.StdoutPipe()
	if nil != err {
		return "", err
	}
	reader := bufio.NewReader(stdout)
	go func(reader io.Reader) {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			log.Printf("Reading from subprocess: %s", scanner.Text())
			stdin.Write([]byte(dot))
		}
	}(reader)*/
	/*go func() {
		_, err := stdin.Write([]byte(dot))
		if err != nil {
			log.Println("stdin.Write error", err)
		}
	}()*/
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return img, nil
}

/*
func getMains( ctxt *build.Context, focus string, groupBy map[string]bool, limitPaths, ignorePaths []string, nostd, tests bool, args []string) ([]*ssa.Package, error) {


    return mains, result, nil
}*/
/*
func serve(mainPkg *types.Package, ) {
}*/

func logf(f string, a ...interface{}) {
	if *debugFlag {
		log.Printf(f, a...)
	}
}
