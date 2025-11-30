package main

import (
	"errors"
	"fmt"
	"go/build"
	"go/types"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/callgraph/cha"
	"golang.org/x/tools/go/callgraph/rta"
	"golang.org/x/tools/go/callgraph/vta"
	"golang.org/x/tools/go/callgraph/static"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

type CallGraphType string

const (
	CallGraphTypeStatic CallGraphType = "static"
	CallGraphTypeCha    CallGraphType = "cha"
	CallGraphTypeRta    CallGraphType = "rta"
	CallGraphTypeVta    CallGraphType = "vta"
)

// ==[ type def/func: analysis   ]===============================================
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
	algo     CallGraphType
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

// initFuncs returns all package init functions
func initFuncs(pkgs []*ssa.Package) ([]*ssa.Function, error) {
	var inits []*ssa.Function
	for _, p := range pkgs {
		if p == nil {
			continue
		}
		for name, member := range p.Members {
			fun, ok := member.(*ssa.Function)
			if !ok {
				continue
			}
			if name == "init" || strings.HasPrefix(name, "init#") {
				inits = append(inits, fun)
			}
		}
	}
	return inits, nil
}

// ==[ type def/func: analysis   ]===============================================
type analysis struct {
	opts      *renderOpts
	prog      *ssa.Program
	pkgs      []*ssa.Package
	mainPkg   *ssa.Package
	callgraph *callgraph.Graph
	imports   []*packages.Package
}

var Analysis *analysis

func (a *analysis) DoAnalysis(
	algo CallGraphType,
	dir string,
	tests bool,
	args []string,
) error {
	logf("begin analysis")
	defer logf("analysis done")

	cfg := &packages.Config{
		Mode:       packages.LoadAllSyntax,
		Tests:      tests,
		Dir:        dir,
		BuildFlags: getBuildFlags(),
	}

	logf("loading packages")

	initial, err := packages.Load(cfg, args...)
	if err != nil {
		return err
	}
	if packages.PrintErrors(initial) > 0 {
		return fmt.Errorf("packages contain errors")
	}

	if *importFlag != "" {
		for _, p := range initial {
			for _, i := range p.Imports {
				a.imports = append(a.imports, i)
			}
		}
	}

	logf("loaded %d initial packages, building program", len(initial))

	// Create and build SSA-form program representation.
	mode := ssa.InstantiateGenerics
	prog, pkgs := ssautil.AllPackages(initial, mode)
	prog.Build()

	logf("build done, computing callgraph (algo: %v)", algo)

	var graph *callgraph.Graph
	var mainPkg *ssa.Package

	switch algo {
	case CallGraphTypeStatic:
		graph = static.CallGraph(prog)
	case CallGraphTypeCha:
		graph = cha.CallGraph(prog)
	case CallGraphTypeVta:
		fallthrough
	case CallGraphTypeRta:
		mains, err := mainPackages(prog.AllPackages())
		if err != nil {
			return err
		}
		var roots []*ssa.Function
		mainPkg = mains[0]
		for _, main := range mains {
			roots = append(roots, main.Func("main"))
		}

		inits, err := initFuncs(prog.AllPackages())
		if err != nil {
			return err
		}
		for _, init := range inits {
			roots = append(roots, init)
		}
		graph = rta.Analyze(roots, true).CallGraph

		if algo == CallGraphTypeVta {
			funcs := make(map[*ssa.Function]bool)
			for fun := range graph.Nodes {
				funcs[fun] = true
			}
			graph = vta.CallGraph(funcs, graph)
		}
	default:
		return fmt.Errorf("invalid call graph type: %s", a.opts.algo)
	}

	logf("callgraph resolved with %d nodes", len(graph.Nodes))

	a.prog = prog
	a.pkgs = pkgs
	a.mainPkg = mainPkg
	a.callgraph = graph
	return nil
}

func (a *analysis) OptsSetup() {
	a.opts = &renderOpts{
		cacheDir: *cacheDir,
		focus:    *focusFlag,
		group:    []string{*groupFlag},
		ignore:   []string{*ignoreFlag},
		include:  []string{*includeFlag},
		limit:    []string{*limitFlag},
		nointer:  *nointerFlag,
		nostd:    *nostdFlag,
	}
}

func (a *analysis) ProcessListArgs() (e error) {
	var groupBy []string
	var ignorePaths []string
	var includePaths []string
	var limitPaths []string

	for _, g := range strings.Split(a.opts.group[0], ",") {
		g := strings.TrimSpace(g)
		if g == "" {
			continue
		}
		if g != "pkg" && g != "type" {
			e = errors.New("invalid group option")
			return
		}
		groupBy = append(groupBy, g)
	}

	for _, p := range strings.Split(a.opts.ignore[0], ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			ignorePaths = append(ignorePaths, p)
		}
	}

	for _, p := range strings.Split(a.opts.include[0], ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			includePaths = append(includePaths, p)
		}
	}

	for _, p := range strings.Split(a.opts.limit[0], ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			limitPaths = append(limitPaths, p)
		}
	}

	a.opts.group = groupBy
	a.opts.ignore = ignorePaths
	a.opts.include = includePaths
	a.opts.limit = limitPaths

	return
}

func (a *analysis) OverrideByHTTP(r *http.Request) {
	if f := r.FormValue("f"); f == "all" {
		a.opts.focus = ""
	} else if f != "" {
		a.opts.focus = f
	}
	if std := r.FormValue("std"); std != "" {
		a.opts.nostd = false
	}
	if inter := r.FormValue("nointer"); inter != "" {
		a.opts.nointer = true
	}
	if refresh := r.FormValue("refresh"); refresh != "" {
		a.opts.refresh = true
	}
	if g := r.FormValue("group"); g != "" {
		a.opts.group[0] = g
	}
	if l := r.FormValue("limit"); l != "" {
		a.opts.limit[0] = l
	}
	if ign := r.FormValue("ignore"); ign != "" {
		a.opts.ignore[0] = ign
	}
	if inc := r.FormValue("include"); inc != "" {
		a.opts.include[0] = inc
	}
	return
}

// basically do printOutput() with previously checking
// focus option and respective package
func (a *analysis) Render() ([]byte, error) {
	var (
		err      error
		ssaPkg   *ssa.Package
		focusPkg *types.Package
	)

	start := time.Now()
	logf("begin rendering")

	if a.opts.focus != "" {
		if ssaPkg = a.prog.ImportedPackage(a.opts.focus); ssaPkg == nil {
			if strings.Contains(a.opts.focus, "/") {
				return nil, fmt.Errorf("focus failed: %v", err)
			}
			// try to find package by name
			var foundPaths []string
			for _, p := range a.pkgs {
				if p.Pkg.Name() == a.opts.focus {
					foundPaths = append(foundPaths, p.Pkg.Path())
				}
			}
			if len(foundPaths) == 0 {
				return nil, fmt.Errorf("focus failed, could not find package: %v", a.opts.focus)
			} else if len(foundPaths) > 1 {
				for _, p := range foundPaths {
					fmt.Fprintf(os.Stderr, " - %s\n", p)
				}
				return nil, fmt.Errorf("focus failed, found multiple packages with name: %v", a.opts.focus)
			}
			// found single package
			if ssaPkg = a.prog.ImportedPackage(foundPaths[0]); ssaPkg == nil {
				return nil, fmt.Errorf("focus failed: %v", err)
			}
		}
		focusPkg = ssaPkg.Pkg
		logf("focusing package: %v (path: %v)", focusPkg.Name(), focusPkg.Path())
	}

	dot, err := printOutput(
		a.prog,
		a.mainPkg,
		a.callgraph,
		focusPkg,
		a.opts.limit,
		a.opts.ignore,
		a.opts.include,
		a.opts.group,
		a.opts.nostd,
		a.opts.nointer,
	)
	if err != nil {
		return nil, fmt.Errorf("processing failed: %v", err)
	}

	logf("rendering done (took %v sec)", time.Since(start).Round(time.Millisecond).Seconds())

	return dot, nil
}

func (a *analysis) FindCachedImg() string {
	if a.opts.cacheDir == "" || a.opts.refresh {
		return ""
	}

	focus := a.opts.focus
	if focus == "" {
		focus = "all"
	}
	focusFilePath := focus + "." + *outputFormat
	absFilePath := filepath.Join(a.opts.cacheDir, focusFilePath)

	if exists, err := pathExists(absFilePath); err != nil || !exists {
		log.Println("not cached img:", absFilePath)
		return ""
	}

	log.Println("hit cached img")
	return absFilePath
}

func (a *analysis) CacheImg(img string) error {
	if a.opts.cacheDir == "" || img == "" {
		return nil
	}

	focus := a.opts.focus
	if focus == "" {
		focus = "all"
	}
	absCacheDirPrefix := filepath.Join(a.opts.cacheDir, focus)
	absCacheDirPath := strings.TrimRightFunc(absCacheDirPrefix, func(r rune) bool {
		return r != '\\' && r != '/'
	})
	err := os.MkdirAll(absCacheDirPath, os.ModePerm)
	if err != nil {
		return err
	}

	absFilePath := absCacheDirPrefix + "." + *outputFormat
	_, err = copyFile(img, absFilePath)
	if err != nil {
		return err
	}

	return nil
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func copyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)

	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func getBuildFlags() []string {
	buildFlagTags := getBuildFlagTags(build.Default.BuildTags)
	if len(buildFlagTags) == 0 {
		return nil
	}

	return []string{buildFlagTags}
}

func getBuildFlagTags(buildTags []string) string {
	if len(buildTags) > 0 {
		return "-tags=" + strings.Join(buildTags, ",")
	}

	return ""
}
