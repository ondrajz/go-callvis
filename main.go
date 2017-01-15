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
	focusFlag   = flag.String("focus", "main", "Focus package with name.")
	limitFlag   = flag.String("limit", "", "Limit package path to prefix.")
	groupFlag   = flag.String("group", "", "Grouping by [type, pkg].")
	ignoreFlag  = flag.String("ignore", "", "Ignore package paths with prefix (separated by comma).")
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

	ignorePaths := []string{}
	for _, p := range strings.Split(*ignoreFlag, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			ignorePaths = append(ignorePaths, p)
		}
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

	if err := run(&build.Default, *focusFlag, *limitFlag, groupBy, ignorePaths, *testFlag, flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "go-callvis: %s\n", err)
		os.Exit(1)
	}
}

func run(ctxt *build.Context, focusPkg, limitPath string, groupBy map[string]bool, ignorePaths []string, tests bool, args []string) error {
	conf := loader.Config{Build: ctxt}

	if len(args) == 0 {
		return fmt.Errorf("missing arguments")
	}

	t0 := time.Now()
	_, err := conf.FromArgs(args, tests)
	if err != nil {
		return err
	}
	iprog, err := conf.Load()
	if err != nil {
		return err
	}
	logf("load took: %v", time.Since(t0))

	t0 = time.Now()
	prog := ssautil.CreateProgram(iprog, 0)
	prog.Build()
	logf("build took: %v", time.Since(t0))

	t0 = time.Now()
	mains, err := mainPackages(prog, tests)
	if err != nil {
		return err
	}
	logf("%d mains", len(mains))
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
		focusPkg, limitPath, ignorePaths, groupBy)
}

func mainPackages(prog *ssa.Program, tests bool) ([]*ssa.Package, error) {
	pkgs := prog.AllPackages()
	logf("%d packages", len(pkgs))

	var mains []*ssa.Package
	if tests {
		for _, pkg := range pkgs {
			if main := prog.CreateTestMainPackage(pkg); main != nil {
				mains = append(mains, main)
			}
		}
		if mains == nil {
			return nil, fmt.Errorf("no tests")
		}
		return mains, nil
	}

	mains = append(mains, ssautil.MainPackages(pkgs)...)
	if len(mains) == 0 {
		return nil, fmt.Errorf("no main packages")
	}

	return mains, nil
}

func logf(f string, a ...interface{}) {
	if *debugFlag {
		log.Printf(f, a...)
	}
}
