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

var Version = "0.0.0-src"

var (
	focusFlag   = flag.String("focus", "main", "Focus package with name or import path.")
	limitFlag   = flag.String("limit", "", "Limit package paths to prefix. (separate multiple by comma)")
	groupFlag   = flag.String("group", "", "Grouping functions by [pkg, type] (separate multiple by comma).")
	ignoreFlag  = flag.String("ignore", "", "Ignore package paths with prefix (separate multiple by comma).")
	nostdFlag   = flag.Bool("nostd", false, "Omit calls to/from std packages.")
	testFlag    = flag.Bool("tests", false, "Include test code.")
	debugFlag   = flag.Bool("debug", false, "Enable verbose log.")
	versionFlag = flag.Bool("version", false, "Show version and exit.")
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
	focus := *focusFlag
	nostd := *nostdFlag
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

	handler := func(w http.ResponseWriter, r *http.Request) {
		if f := r.FormValue("f"); f != "" {
			focus = f
		}

		groupBy := make(map[string]bool)
		for _, g := range strings.Split(*groupFlag, ",") {
			g := strings.TrimSpace(g)
			if g == "" {
				continue
			}
			if g != "pkg" && g != "type" {
				fmt.Fprintf(os.Stderr, "go-callvis: %s\n", "invalid group option")
				os.Exit(1)
			}
			groupBy[g] = true
		}

		limitPaths := []string{}
		for _, p := range strings.Split(*limitFlag, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				limitPaths = append(limitPaths, p)
			}
		}

		ignorePaths := []string{}
		for _, p := range strings.Split(*ignoreFlag, ",") {
			p = strings.TrimSpace(p)
			if p != "" {
				ignorePaths = append(ignorePaths, p)
			}
		}

		var focusPkg *build.Package
		if focus != "" {
			focusPkg, err = conf.Build.Import(focus, "", 0)
			if err != nil {
				if strings.Contains(focus, "/") {
					log.Fatalln("failed:", err)
				}
				// try to find package by name
				var foundPaths []string
				for _, p := range pkgs {
					if p.Pkg.Name() == focus {
						foundPaths = append(foundPaths, p.Pkg.Path())
					}
				}
				if len(foundPaths) == 0 {
					log.Fatalln("failed:", err)
				} else if len(foundPaths) > 1 {
					for _, p := range foundPaths {
						fmt.Fprintf(os.Stderr, " - %s\n", p)
					}
					log.Fatalf("found %d packages with name %q, use import path not name", len(foundPaths), focus)
				}
				if focusPkg, err = conf.Build.Import(foundPaths[0], "", 0); err != nil {
					log.Fatalln("failed:", err)
				}
			}
			logf("focusing: %v", focusPkg.ImportPath)
		}

		out, err := printOutput(mains[0].Pkg, result.CallGraph,
			focusPkg, limitPaths, ignorePaths, groupBy, nostd)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, out)
	}

	//handler()
	/*	fmt.Println("\n\n//-----")
		nostd = false
		handler()*/

	http.HandleFunc("/", handler)

	log.Println("serving..")

	log.Fatal(http.ListenAndServe(":7878", nil))
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
