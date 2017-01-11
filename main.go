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
	testFlag    = flag.Bool("test", false, "Include test code.")
	minlenFlag  = flag.Uint("minlen", 2, "Min length of an edge (for wider output).")
	debugFlag   = flag.Bool("debug", false, "Enable debug mode.")
	versionFlag = flag.Bool("version", false, "Show version and exit.")
	// deprecated
	subFlag = flag.String("sub", "", "Deprecated!!! Use 'group' instead!")
)

func main() {
	flag.Parse()
	if *versionFlag {
		fmt.Fprintf(os.Stderr, "go-callvis %s\n", Version)
		os.Exit(0)
	}
	if *debugFlag {
		log.SetFlags(log.Lmicroseconds)
	}

	// migrate deprecated
	if *subFlag != "" {
		fmt.Fprintln(os.Stderr, "Warning! Using 'sub' flag is deprecated, use 'group' instead!")
		groupFlag = subFlag
	}

	var ignorePaths []string
	for _, p := range strings.Split(*ignoreFlag, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			ignorePaths = append(ignorePaths, p)
		}
	}

	if err := run(&build.Default, *focusFlag, *limitFlag, *groupFlag, ignorePaths, *minlenFlag, *testFlag, flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "go-callvis: %s\n", err)
		os.Exit(1)
	}
}

func run(ctxt *build.Context, focusPkg, limitPath, groupBy string, ignorePaths []string, minlen uint, tests bool, args []string) error {
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
	prog := ssautil.CreateProgram(iprog, 0)
	prog.Build()
	logf("program build took: %v", time.Since(t0))

	t0 = time.Now()
	mains, err := mainPackages(prog, tests)
	if err != nil {
		return err
	}
	config := &pointer.Config{
		Mains:          mains,
		BuildCallGraph: true,
	}
	result, err := pointer.Analyze(config)
	if err != nil {
		return err
	}
	result.CallGraph.DeleteSyntheticNodes()
	logf("callgraph analysis took: %v", time.Since(t0))

	if err := printOutput(result.CallGraph,
		focusPkg, limitPath, ignorePaths, groupBy, minlen); err != nil {
		return err
	}

	return nil
}

func mainPackages(prog *ssa.Program, tests bool) ([]*ssa.Package, error) {
	pkgs := prog.AllPackages()

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
