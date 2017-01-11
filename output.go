package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/template"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
)

var stdout io.Writer = os.Stdout

type dotData struct {
	Title  string
	Minlen uint
	Edges  []string
}

var dotTmpl = template.Must(template.New("digraph").Parse(`digraph G {
    label="{{.Title}}";
    labelloc="t";
    bgcolor="aliceblue";
    rankdir="LR";
    fontsize="22";
    fontname="Ubuntu";
    edge [minlen={{.Minlen}}];
    node [shape=box style="rounded,filled" fillcolor=wheat fontname="Ubuntu"];
{{range .Edges}}{{printf "\t%s\n" .}}{{end}}
}`))

func printOutput(cg *callgraph.Graph, focusPkg, limitPath string, ignorePaths []string, groupBy string, minlen uint) error {
	subType := groupBy == "type"
	subPkg := groupBy == "pkg"

	var edges []string
	edgeMap := make(map[string]struct{})

	err := callgraph.GraphVisitEdges(cg, func(edge *callgraph.Edge) error {
		caller := edge.Caller.Func
		callee := edge.Callee.Func

		if caller.Pkg == nil || callee.Synthetic != "" {
			return nil
		}
		if focusPkg != "" &&
			!(caller.Pkg.Pkg.Name() == focusPkg || callee.Pkg.Pkg.Name() == focusPkg) {
			return nil
		}
		if !(strings.HasPrefix(caller.Pkg.Pkg.Path(), limitPath) &&
			strings.HasPrefix(callee.Pkg.Pkg.Path(), limitPath)) {
			return nil
		}
		for _, p := range ignorePaths {
			if strings.HasPrefix(caller.Pkg.Pkg.Path(), p) ||
				strings.HasPrefix(callee.Pkg.Pkg.Path(), p) {
				return nil
			}
		}

		props := make(properties)
		callerSign := caller.Signature
		if caller.Parent() != nil {
			callerSign = caller.Parent().Signature
		}
		callerLabel := fmt.Sprintf("%s\n%s", caller.Pkg.Pkg.Name(), caller.RelString(caller.Pkg.Pkg))
		if caller.Pkg.Pkg.Name() == focusPkg {
			props["fillcolor"] = "lightblue"
			callerLabel = fmt.Sprintf("%s", caller.RelString(caller.Pkg.Pkg))
			if subType && callerSign.Recv() != nil {
				callerParts := strings.Split(callerLabel, ".")
				callerLabel = callerParts[len(callerParts)-1]
			}
		} else if subPkg {
			callerLabel = fmt.Sprintf("%s", caller.RelString(caller.Pkg.Pkg))
		}
		props["label"] = callerLabel
		if caller.Parent() != nil {
			props["style"] = "dotted,rounded,filled"
		} else if caller.Object() != nil && caller.Object().Exported() {
			props["style"] = "bold,rounded,filled"
		}
		callerNode := fmt.Sprintf("%q [%s]", caller, props)

		if subType && caller.Pkg.Pkg.Name() == focusPkg && callerSign.Recv() != nil {
			parts := strings.Split(fmt.Sprint(callerSign.Recv().Type()), ".")
			clusterLabel := parts[len(parts)-1]
			callerNode = fmt.Sprintf("subgraph \"cluster_%s\" { penwidth=0.5; fontsize=18; label=%q; style=filled; fillcolor=snow; %s; }",
				callerSign.Recv().Type(), clusterLabel, callerNode)
		} else if subPkg && caller.Pkg.Pkg.Name() != focusPkg {
			callerNode = fmt.Sprintf("subgraph \"cluster_%s\" { penwidth=0.5; fontsize=18; label=%q; style=filled; fillcolor=snow; %s; }",
				caller.Pkg.Pkg.Name(), caller.Pkg.Pkg.Name(), callerNode)
		}

		props = make(properties)
		calleeSign := callee.Signature
		if callee.Parent() != nil {
			calleeSign = callee.Parent().Signature
		}
		calleeLabel := fmt.Sprintf("%s\n%s", callee.Pkg.Pkg.Name(), callee.RelString(callee.Pkg.Pkg))
		if callee.Pkg.Pkg.Name() == focusPkg {
			props["fillcolor"] = "lightblue"
			calleeLabel = fmt.Sprintf("%s", callee.RelString(callee.Pkg.Pkg))
			if subType && calleeSign.Recv() != nil {
				calleeParts := strings.Split(calleeLabel, ".")
				calleeLabel = calleeParts[len(calleeParts)-1]
			}
		} else if subPkg {
			calleeLabel = fmt.Sprintf("%s", callee.RelString(callee.Pkg.Pkg))
		}
		props["label"] = calleeLabel
		if callee.Parent() != nil {
			props["style"] = "dotted,rounded,filled"
		} else if callee.Object() != nil && callee.Object().Exported() {
			props["style"] = "bold,rounded,filled"
		}
		calleeNode := fmt.Sprintf("%q [%s]", callee, props)

		if subType && callee.Pkg.Pkg.Name() == focusPkg && calleeSign.Recv() != nil {
			parts := strings.Split(fmt.Sprint(calleeSign.Recv().Type()), ".")
			clusterLabel := parts[len(parts)-1]
			calleeNode = fmt.Sprintf("subgraph \"cluster_%s\" { penwidth=0.5; fontsize=18; label=%q; style=filled; fillcolor=snow; %s; }",
				calleeSign.Recv().Type(), clusterLabel, calleeNode)
		} else if subPkg && callee.Pkg.Pkg.Name() != focusPkg {
			calleeNode = fmt.Sprintf("subgraph \"cluster_%s\" { penwidth=0.5; fontsize=18; label=%q; style=filled; fillcolor=snow; %s; }",
				callee.Pkg.Pkg.Name(), callee.Pkg.Pkg.Name(), calleeNode)
		}

		props = make(properties)
		if edge.Site != nil && edge.Site.Common().StaticCallee() == nil {
			props["style"] = "dashed"
		}
		switch edge.Site.(type) {
		case *ssa.Go:
			props["arrowhead"] = "empty"
		case *ssa.Defer:
			props["arrowhead"] = "odot"
		}
		if callee.Pkg.Pkg.Name() != focusPkg || caller.Pkg.Pkg.Name() != focusPkg {
			props["color"] = "saddlebrown"
		}
		s := fmt.Sprintf("\n\t%s;\n\t%s;\n\t%q -> %q [%s]",
			callerNode, calleeNode, caller, callee, props)

		if _, ok := edgeMap[s]; !ok {
			edges = append(edges, s)
			edgeMap[s] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return err
	}

	sort.Strings(edges)
	logf("has %d edges", len(edges))

	data := dotData{
		Title:  focusPkg,
		Minlen: minlen,
		Edges:  edges,
	}
	if err := dotTmpl.Execute(stdout, data); err != nil {
		return err
	}

	return nil
}

type properties map[string]interface{}

func (p properties) String() string {
	l := []string{}
	for k, v := range p {
		l = append(l, fmt.Sprintf("%s=%q", k, v))
	}
	return strings.Join(l, " ")
}
