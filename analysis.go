package main

import (
	"fmt"
	"go/types"
	"os"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/pointer"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

var Analysis *analysis

type analysis struct {
	prog   *ssa.Program
	pkgs   []*ssa.Package
	mains  []*ssa.Package
	result *pointer.Result
}

func doAnalysis(dir string, tests bool, args []string) error {
	cfg := &packages.Config{
		Mode:  packages.LoadAllSyntax,
		Tests: tests,
		Dir:   dir,
	}
	initial, err := packages.Load(cfg, args...)
	if err != nil {
		return err
	}
	if packages.PrintErrors(initial) > 0 {
		return fmt.Errorf("packages contain errors")
	}

	// Create and build SSA-form program representation.
	prog, pkgs := ssautil.AllPackages(initial, 0)
	prog.Build()

	mains, err := mainPackages(pkgs)
	if err != nil {
		return err
	}
	config := &pointer.Config{
		Mains:          mains,
		BuildCallGraph: true,
	}
	result, err := pointer.Analyze(config)
	if err != nil {
		return err // internal error in pointer analysis
	}
	//cg.DeleteSyntheticNodes()

	Analysis = &analysis{
		prog:   prog,
		pkgs:   pkgs,
		mains:  mains,
		result: result,
	}
	return nil
}

// mainPackages returns the main packages to analyze.
// Each resulting package is named "main" and has a main function.
func mainPackages(pkgs []*ssa.Package) ([]*ssa.Package, error) {
	var mains []*ssa.Package
	for _, p := range pkgs {
		if p != nil && p.Pkg.Name() == "main" && p.Func("main") != nil {
			mains = append(mains, p)
		}
	}
	if len(mains) == 0 {
		return nil, fmt.Errorf("no main packages")
	}
	return mains, nil
}

type renderOpts struct {
	cacheDir string
	focus    string
	group    []string
	ignore   []string
	include  []string
	limit    []string
	nointer  bool
	refresh  bool
	nostd    bool
}

func (a *analysis) render(opts *renderOpts) ([]byte, error) {
	var (
		err      error
		ssaPkg   *ssa.Package
		focusPkg *types.Package
	)

	if opts.focus != "" {
		ssaPkg = a.prog.ImportedPackage(opts.focus)
		if ssaPkg == nil {
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
			// found single package
			if ssaPkg = a.prog.ImportedPackage(foundPaths[0]); ssaPkg == nil {
				return nil, fmt.Errorf("focus failed: %v", err)
			}
		}
		focusPkg = ssaPkg.Pkg
		logf("focusing: %v", focusPkg.Path())
	}

	dot, err := printOutput(a.prog, a.mains[0].Pkg, a.result.CallGraph,
		focusPkg, opts.limit, opts.ignore, opts.include, opts.group, opts.nostd, opts.nointer)
	if err != nil {
		return nil, fmt.Errorf("processing failed: %v", err)
	}

	return dot, nil
}
