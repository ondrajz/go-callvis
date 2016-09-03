package main

import (
	"flag"
	"fmt"
	"go/build"
	"os"
	"path"
	"sort"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/go/pointer"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

var (
	testFlag = flag.Bool("test", false,
		"Loads test code (*_test.go) for imported packages")

	limitFlag = flag.String("limit", "",
		"limit package path")

	focusFlag = flag.String("focus", "",
		"focus package name")

	subFlag = flag.String("sub", "",
		"subgraph by [type, pkg]")

	minlenFlag = flag.Uint("minlen", 2,
		"minlen of edge")

	ptalogFlag = flag.String("ptalog", "",
		"Location of the points-to analysis log file, or empty to disable logging.")
)

func main() {
	flag.Parse()
	if err := doCallgraph(&build.Default, *focusFlag, *limitFlag, *subFlag, *minlenFlag, *testFlag, flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "go-callmap: %s\n", err)
		os.Exit(1)
	}
}

func doCallgraph(ctxt *build.Context, focusPkg, limitPath, subgraph string, minlen uint, tests bool, args []string) error {
	conf := loader.Config{Build: &build.Default}

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "missing arguments")
		return nil
	}
	_, err := conf.FromArgs(args, tests)
	if err != nil {
		return err
	}

	iprog, err := conf.Load()
	if err != nil {
		fmt.Print(err) // type error in some package
		return nil
	}

	prog := ssautil.CreateProgram(iprog, 0)
	prog.Build()

	main, err := mainPackage(prog, tests)
	if err != nil {
		return err
	}
	config := &pointer.Config{
		Mains:          []*ssa.Package{main},
		BuildCallGraph: true,
	}
	result, err := pointer.Analyze(config)
	if err != nil {
		return err // internal error in pointer analysis
	}
	result.CallGraph.DeleteSyntheticNodes()

	subType := subgraph == "type"
	subPkg := subgraph == "pkg"

	var edges []string
	edgeMap := make(map[string]struct{})
	callgraph.GraphVisitEdges(result.CallGraph, func(edge *callgraph.Edge) error {
		caller := edge.Caller.Func
		callee := edge.Callee.Func
		if caller.Pkg != nil && callee.Synthetic == "" &&
			(caller.Pkg.Pkg.Name() == focusPkg || callee.Pkg.Pkg.Name() == focusPkg) &&
			!strings.HasPrefix(caller.Pkg.Pkg.Path(), path.Join(main.Pkg.Path(), "vendor")) &&
			!strings.HasPrefix(callee.Pkg.Pkg.Path(), path.Join(main.Pkg.Path(), "vendor")) &&
			strings.HasPrefix(caller.Pkg.Pkg.Path(), limitPath) &&
			strings.HasPrefix(callee.Pkg.Pkg.Path(), limitPath) {

			props := []string{}
			if edge.Site != nil && edge.Site.Common().StaticCallee() == nil {
				props = append(props, "arrowhead=empty", "style=dashed")
			}

			callerProps := []string{}
			callerSign := caller.Signature
			if caller.Parent() != nil {
				callerSign = caller.Parent().Signature
			}
			callerLabel := fmt.Sprintf("%s\n%s", caller.Pkg.Pkg.Name(), caller.RelString(caller.Pkg.Pkg))
			if caller.Pkg.Pkg.Name() == focusPkg {
				callerProps = append(callerProps, "fillcolor=lightblue")
				callerLabel = fmt.Sprintf("%s", caller.RelString(caller.Pkg.Pkg))
				if subType && callerSign.Recv() != nil {
					callerParts := strings.Split(callerLabel, ".")
					callerLabel = callerParts[len(callerParts)-1]
				}
			} else if subPkg {
				callerLabel = fmt.Sprintf("%s", caller.RelString(caller.Pkg.Pkg))
			}
			callerProps = append(callerProps, fmt.Sprintf("label=%q", callerLabel))
			if caller.Parent() != nil {
				callerProps = append(callerProps, "style=\"dotted,rounded,filled\"")
			}
			callerNode := fmt.Sprintf("%q [%s]", caller, strings.Join(callerProps, " "))
			if subType && caller.Pkg.Pkg.Name() == focusPkg && callerSign.Recv() != nil {
				callerNode = fmt.Sprintf("subgraph \"cluster_%s\" { penwidth=0.5; fontsize=18; label=\"%s\"; style=filled; fillcolor=snow; %s; }",
					callerSign.Recv().Type(), strings.Split(fmt.Sprint(callerSign.Recv().Type()), ".")[1], callerNode)
			} else if subPkg && caller.Pkg.Pkg.Name() != focusPkg {
				callerNode = fmt.Sprintf("subgraph \"cluster_%s\" { penwidth=0.5; fontsize=18; label=\"%s\"; style=filled; fillcolor=snow; %s; }",
					caller.Pkg.Pkg.Name(), caller.Pkg.Pkg.Name(), callerNode)
			}

			calleeProps := []string{}
			calleeSign := callee.Signature
			if callee.Parent() != nil {
				calleeSign = callee.Parent().Signature
			}
			calleeLabel := fmt.Sprintf("%s\n%s", callee.Pkg.Pkg.Name(), callee.RelString(callee.Pkg.Pkg))
			if callee.Pkg.Pkg.Name() == focusPkg {
				calleeProps = append(calleeProps, "fillcolor=lightblue")
				calleeLabel = fmt.Sprintf("%s", callee.RelString(callee.Pkg.Pkg))
				if subType && calleeSign.Recv() != nil {
					calleeParts := strings.Split(calleeLabel, ".")
					calleeLabel = calleeParts[len(calleeParts)-1]
				}
			} else if subPkg {
				calleeLabel = fmt.Sprintf("%s", callee.RelString(callee.Pkg.Pkg))
			}
			calleeProps = append(calleeProps, fmt.Sprintf("label=%q", calleeLabel))
			if callee.Parent() != nil {
				calleeProps = append(calleeProps, "style=\"dotted,rounded,filled\"")
			}
			calleeNode := fmt.Sprintf("%q [%s]", callee, strings.Join(calleeProps, " "))
			if subType && callee.Pkg.Pkg.Name() == focusPkg && calleeSign.Recv() != nil {
				calleeNode = fmt.Sprintf("subgraph \"cluster_%s\" { penwidth=0.5; fontsize=18; label=\"%s\"; style=filled; fillcolor=snow; %s; }",
					calleeSign.Recv().Type(), strings.Split(fmt.Sprint(calleeSign.Recv().Type()), ".")[1], calleeNode)
			} else if subPkg && callee.Pkg.Pkg.Name() != focusPkg {
				calleeNode = fmt.Sprintf("subgraph \"cluster_%s\" { penwidth=0.5; fontsize=18; label=\"%s\"; style=filled; fillcolor=snow; %s; }",
					callee.Pkg.Pkg.Name(), callee.Pkg.Pkg.Name(), calleeNode)
			}

			s := fmt.Sprintf("%s;%s; %q -> %q [%s]",
				callerNode, calleeNode,
				caller, callee, strings.Join(props, " "))
			if _, ok := edgeMap[s]; !ok {
				edges = append(edges, s)
				edgeMap[s] = struct{}{}
			}
		}
		return nil
	})

	sort.Strings(edges)

	fmt.Printf(`digraph G {
        label="%s";
        labelloc=t;
        bgcolor=aliceblue;
        rankdir=LR;
        fontsize=22;
        fontname="Ubuntu";
        edge [minlen=%d];
        node [shape=box style="rounded,filled" fillcolor=wheat fontname="Ubuntu"];
`, focusPkg, minlen)
	for _, edge := range edges {
		fmt.Println("\t", edge)
	}
	fmt.Println("}")

	//fmt.Println(len(edges), "edges")
	return nil
}

// mainPackage returns the main package to analyze.
// The resulting package has a main() function.
func mainPackage(prog *ssa.Program, tests bool) (*ssa.Package, error) {
	pkgs := prog.AllPackages()

	// TODO(adonovan): allow independent control over tests, mains and libraries.
	// TODO(adonovan): put this logic in a library; we keep reinventing it.

	if tests {
		// If -test, use all packages' tests.
		if len(pkgs) > 0 {
			if main := prog.CreateTestMainPackage(pkgs...); main != nil {
				return main, nil
			}
		}
		return nil, fmt.Errorf("no tests")
	}

	// Otherwise, use the first package named main.
	for _, pkg := range pkgs {
		if pkg.Pkg.Name() == "main" {
			if pkg.Func("main") == nil {
				return nil, fmt.Errorf("no func main() in main package")
			}
			return pkg, nil
		}
	}

	return nil, fmt.Errorf("no main package")
}
