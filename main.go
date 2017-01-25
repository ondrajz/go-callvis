// go-callvis: a tool to help visualize the call graph of a Go program.
//
package main

import (
	"flag"
	"fmt"
	"go/build"
	"log"
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

	if err := run(&build.Default, *focusFlag, groupBy, limitPaths, ignorePaths, *nostdFlag, *testFlag, flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "go-callvis: %s\n", err)
		os.Exit(1)
	}
}

func run(ctxt *build.Context, focus string, groupBy map[string]bool, limitPaths, ignorePaths []string, nostd, tests bool, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing arguments")
	}

	t0 := time.Now()
	conf := loader.Config{Build: ctxt}
	_, err := conf.FromArgs(args, tests)
	if err != nil {
		return err
	}
	load, err := conf.Load()
	if err != nil {
		return err
	}
	logf("loading took: %v", time.Since(t0))
	logf("%d imported (%d created)", len(load.Imported), len(load.Created))

	t0 = time.Now()
	prog := ssautil.CreateProgram(load, 0)
	prog.Build()
	pkgs := prog.AllPackages()
	logf("building took: %v", time.Since(t0))

	var focusPkg *build.Package
	if focus != "" {
		focusPkg, err = conf.Build.Import(focus, "", 0)
		if err != nil {
			if strings.Contains(focus, "/") {
				return err
			}
			// try to find package by name
			var foundPaths []string
			for _, p := range pkgs {
				if p.Pkg.Name() == focus {
					foundPaths = append(foundPaths, p.Pkg.Path())
				}
			}
			if len(foundPaths) == 0 {
				return err
			} else if len(foundPaths) > 1 {
				for _, p := range foundPaths {
					fmt.Fprintf(os.Stderr, " - %s\n", p)
				}
				return fmt.Errorf("found %d packages with name %q, use import path not name", len(foundPaths), focus)
			}
			if focusPkg, err = conf.Build.Import(foundPaths[0], "", 0); err != nil {
				return err
			}
		}
		logf("focusing: %v", focusPkg.ImportPath)
	}

	var mains []*ssa.Package
	if tests {
		for _, pkg := range pkgs {
			if main := prog.CreateTestMainPackage(pkg); main != nil {
				mains = append(mains, main)
			}
		}
		if mains == nil {
			return fmt.Errorf("no tests")
		}
	} else {
		mains = append(mains, ssautil.MainPackages(pkgs)...)
		if len(mains) == 0 {
			return fmt.Errorf("no main packages")
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
		return err
	}
	logf("analysis took: %v", time.Since(t0))

	return printOutput(mains[0].Pkg, result.CallGraph,
		focusPkg, limitPaths, ignorePaths, groupBy, nostd)
}

func logf(f string, a ...interface{}) {
	if *debugFlag {
		log.Printf(f, a...)
	}
}
