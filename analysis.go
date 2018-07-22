package main

import (
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

var Analysis *analysis

type analysis struct {
	conf   loader.Config
	pkgs   []*ssa.Package
	mains  []*ssa.Package
	result *pointer.Result
}

func doAnalysis(buildCtx *build.Context, tests bool, args []string) {
	t0 := time.Now()
	conf := loader.Config{Build: buildCtx}
	_, err := conf.FromArgs(args, tests)
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

	Analysis = &analysis{
		conf:   conf,
		pkgs:   pkgs,
		mains:  mains,
		result: result,
	}
}

type renderOpts struct {
	focus   string
	group   []string
	ignore  []string
	include []string
	limit   []string
	nointer bool
	nostd   bool
}

func (a *analysis) render(opts renderOpts) ([]byte, error) {
	var err error
	var focusPkg *build.Package
	if opts.focus != "" {
		focusPkg, err = a.conf.Build.Import(opts.focus, "", 0)
		if err != nil {
			if strings.Contains(opts.focus, "/") {
				return nil, fmt.Errorf("focus failed: %v", err)
			}
			// try to find package by name
			var foundPaths []string
			for _, p := range a.pkgs {
				if p.Pkg.Name() == opts.focus {
					foundPaths = append(foundPaths, p.Pkg.Path())
				}
			}
			if len(foundPaths) == 0 {
				return nil, fmt.Errorf("focus failed, could not find package: %v", opts.focus)
			} else if len(foundPaths) > 1 {
				for _, p := range foundPaths {
					fmt.Fprintf(os.Stderr, " - %s\n", p)
				}
				return nil, fmt.Errorf("focus failed, found multiple packages with name: %v", opts.focus)
			}
			if focusPkg, err = a.conf.Build.Import(foundPaths[0], "", 0); err != nil {
				return nil, fmt.Errorf("focus failed: %v", err)
			}
		}
		logf("focusing: %v", focusPkg.ImportPath)
	}

	dot, err := printOutput(a.mains[0].Pkg, a.result.CallGraph,
		focusPkg, opts.limit, opts.ignore, opts.include, opts.group, opts.nostd, opts.nointer)
	if err != nil {
		return nil, fmt.Errorf("processing failed: %v", err)
	}

	return dot, nil
}
