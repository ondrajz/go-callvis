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
	"os/exec"
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
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		f := r.FormValue("f")
		if f != "" {
			focus = f
		}

		groupBy := make(map[string]bool)
		for _, g := range strings.Split(*groupFlag, ",") {
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
					http.Error(w, "focus failed", http.StatusInternalServerError)
					return
				}
				// try to find package by name
				var foundPaths []string
				for _, p := range pkgs {
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
				if focusPkg, err = conf.Build.Import(foundPaths[0], "", 0); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			logf("focusing: %v", focusPkg.ImportPath)
		}

		dot, err := printOutput(mains[0].Pkg, result.CallGraph,
			focusPkg, limitPaths, ignorePaths, groupBy, nostd)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
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

	//handler()
	/*	fmt.Println("\n\n//-----")
		nostd = false
		handler()*/

	http.HandleFunc("/", handler)

	log.Println("serving..")

	log.Fatal(http.ListenAndServe(":7878", nil))
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
