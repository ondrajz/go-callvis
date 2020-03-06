package main

import (
	"bytes"
	"fmt"
	"go/build"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
)

func isSynthetic(edge *callgraph.Edge) bool {
	return edge.Caller.Func.Pkg == nil || edge.Callee.Func.Synthetic != ""
}

func inStd(node *callgraph.Node) bool {
	pkg, _ := build.Import(node.Func.Pkg.Pkg.Path(), "", 0)
	return pkg.Goroot
}

func printOutput(prog *ssa.Program, mainPkg *types.Package, cg *callgraph.Graph, focusPkg *types.Package,
	limitPaths, ignorePaths, includePaths []string, groupBy []string, nostd, nointer bool) ([]byte, error) {
	var groupType, groupPkg bool
	for _, g := range groupBy {
		if g == "pkg" {
			groupPkg = true
		} else if g == "type" {
			groupType = true
		}
	}

	cluster := NewDotCluster("focus")
	cluster.Attrs = dotAttrs{
		"bgcolor":   "white",
		"label":     "",
		"labelloc":  "t",
		"labeljust": "c",
		"fontsize":  "18",
	}
	if focusPkg != nil {
		cluster.Attrs["bgcolor"] = "#e6ecfa"
		cluster.Attrs["label"] = focusPkg.Name()
	}

	var (
		nodes []*dotNode
		edges []*dotEdge
	)

	nodeMap := make(map[string]*dotNode)
	edgeMap := make(map[string]*dotEdge)

	cg.DeleteSyntheticNodes()

	logf("%d limit prefixes: %v", len(limitPaths), limitPaths)
	logf("%d ignore prefixes: %v", len(ignorePaths), ignorePaths)
	logf("%d include prefixes: %v", len(includePaths), includePaths)
	logf("no std packages: %v", nostd)

	var isFocused = func(edge *callgraph.Edge) bool {
		caller := edge.Caller
		callee := edge.Callee
		if focusPkg != nil && (caller.Func.Pkg.Pkg.Path() == focusPkg.Path() || callee.Func.Pkg.Pkg.Path() == focusPkg.Path()) {
			return true
		}
		fromFocused := false
		toFocused := false
		for _, e := range caller.In {
			if !isSynthetic(e) && focusPkg != nil &&
				e.Caller.Func.Pkg.Pkg.Path() == focusPkg.Path() {
				fromFocused = true
				break
			}
		}
		for _, e := range callee.Out {
			if !isSynthetic(e) && focusPkg != nil &&
				e.Callee.Func.Pkg.Pkg.Path() == focusPkg.Path() {
				toFocused = true
				break
			}
		}
		if fromFocused && toFocused {
			logf("edge semi-focus: %s", edge)
			return true
		}
		return false
	}

	var inIncludes = func(node *callgraph.Node) bool {
		pkgPath := node.Func.Pkg.Pkg.Path()
		for _, p := range includePaths {
			if strings.HasPrefix(pkgPath, p) {
				return true
			}
		}
		return false
	}

	var inLimits = func(node *callgraph.Node) bool {
		pkgPath := node.Func.Pkg.Pkg.Path()
		for _, p := range limitPaths {
			if strings.HasPrefix(pkgPath, p) {
				return true
			}
		}
		return false
	}

	var inIgnores = func(node *callgraph.Node) bool {
		pkgPath := node.Func.Pkg.Pkg.Path()
		for _, p := range ignorePaths {
			if strings.HasPrefix(pkgPath, p) {
				return true
			}
		}
		return false
	}

	var isInter = func(edge *callgraph.Edge) bool {
		//caller := edge.Caller
		callee := edge.Callee
		if callee.Func.Object() != nil && !callee.Func.Object().Exported() {
			return true
		}
		return false
	}

	count := 0
	err := callgraph.GraphVisitEdges(cg, func(edge *callgraph.Edge) error {
		count++

		caller := edge.Caller
		callee := edge.Callee

		pos := prog.Fset.Position(caller.Func.Pos())
		//file := fmt.Sprintf("%s:%d", pos.Filename, pos.Line)
		filename := filepath.Base(pos.Filename)

		// omit synthetic calls
		if isSynthetic(edge) {
			return nil
		}

		callerPkg := caller.Func.Pkg.Pkg
		calleePkg := callee.Func.Pkg.Pkg

		// focus specific pkg
		if focusPkg != nil &&
			!isFocused(edge) {
			return nil
		}

		// omit std
		if nostd &&
			(inStd(caller) || inStd(callee)) {
			return nil
		}

		// omit inter
		if nointer && isInter(edge) {
			return nil
		}

		include := false
		// include path prefixes
		if len(includePaths) > 0 &&
			(inIncludes(caller) || inIncludes(callee)) {
			logf("include: %s -> %s", caller, callee)
			include = true
		}

		if !include {
			// limit path prefixes
			if len(limitPaths) > 0 &&
				(!inLimits(caller) || !inLimits(callee)) {
				logf("NOT in limit: %s -> %s", caller, callee)
				return nil
			}

			// ignore path prefixes
			if len(ignorePaths) > 0 &&
				(inIgnores(caller) || inIgnores(callee)) {
				logf("IS ignored: %s -> %s", caller, callee)
				return nil
			}
		}

		//var buf bytes.Buffer
		//data, _ := json.MarshalIndent(caller.Func, "", " ")
		//logf("call node: %s -> %s\n %v", caller, callee, string(data))
		logf("call node: %s -> %s (%s -> %s) %v\n", caller.Func.Pkg, callee.Func.Pkg, caller, callee, filename)

		var sprintNode = func(node *callgraph.Node) *dotNode {
			// only once
			key := node.Func.String()
			if n, ok := nodeMap[key]; ok {
				return n
			}

			// is focused
			isFocused := focusPkg != nil &&
				node.Func.Pkg.Pkg.Path() == focusPkg.Path()
			attrs := make(dotAttrs)

			// node label
			label := node.Func.RelString(node.Func.Pkg.Pkg)

			// func signature
			sign := node.Func.Signature
			if node.Func.Parent() != nil {
				sign = node.Func.Parent().Signature
			}

			// omit type from label
			if groupType && sign.Recv() != nil {
				parts := strings.Split(label, ".")
				label = parts[len(parts)-1]
			}

			pkg, _ := build.Import(node.Func.Pkg.Pkg.Path(), "", 0)
			// set node color
			if isFocused {
				attrs["fillcolor"] = "lightblue"
			} else if pkg.Goroot {
				attrs["fillcolor"] = "#adedad"
			} else {
				attrs["fillcolor"] = "moccasin"
			}

			// include pkg name
			if !groupPkg && !isFocused {
				label = fmt.Sprintf("%s\n%s", node.Func.Pkg.Pkg.Name(), label)
			}

			attrs["label"] = label

			// func styles
			if node.Func.Parent() != nil {
				attrs["style"] = "dotted,filled"
			} else if node.Func.Object() != nil && node.Func.Object().Exported() {
				attrs["penwidth"] = "1.5"
			} else {
				attrs["penwidth"] = "0.5"
			}

			c := cluster

			// group by pkg
			if groupPkg && !isFocused {
				label := node.Func.Pkg.Pkg.Name()
				if pkg.Goroot {
					label = node.Func.Pkg.Pkg.Path()
				}
				key := node.Func.Pkg.Pkg.Path()
				if _, ok := c.Clusters[key]; !ok {
					c.Clusters[key] = &dotCluster{
						ID:       key,
						Clusters: make(map[string]*dotCluster),
						Attrs: dotAttrs{
							"penwidth":  "0.8",
							"fontsize":  "16",
							"label":     fmt.Sprintf("%s", label),
							"style":     "filled",
							"fillcolor": "lightyellow",
							"URL":       fmt.Sprintf("/?f=%s", key),
							"fontname":  "Tahoma bold",
							"tooltip":   fmt.Sprintf("package: %s", key),
							"rank":      "sink",
						},
					}
					if pkg.Goroot {
						c.Clusters[key].Attrs["fillcolor"] = "#E0FFE1"
					}
				}
				c = c.Clusters[key]
			}

			// group by type
			if groupType && sign.Recv() != nil {
				label := strings.Split(node.Func.RelString(node.Func.Pkg.Pkg), ".")[0]
				key := sign.Recv().Type().String()
				if _, ok := c.Clusters[key]; !ok {
					c.Clusters[key] = &dotCluster{
						ID:       key,
						Clusters: make(map[string]*dotCluster),
						Attrs: dotAttrs{
							"penwidth":  "0.5",
							"fontsize":  "15",
							"fontcolor": "#222222",
							"label":     label,
							"labelloc":  "b",
							"style":     "rounded,filled",
							"fillcolor": "wheat2",
							"tooltip":   fmt.Sprintf("type: %s", key),
						},
					}
					if isFocused {
						c.Clusters[key].Attrs["fillcolor"] = "lightsteelblue"
					} else if pkg.Goroot {
						c.Clusters[key].Attrs["fillcolor"] = "#c2e3c2"
					}
				}
				c = c.Clusters[key]
			}

			n := &dotNode{
				ID:    node.Func.String(),
				Attrs: attrs,
			}

			if c != nil {
				c.Nodes = append(c.Nodes, n)
			} else {
				nodes = append(nodes, n)
			}

			nodeMap[key] = n
			return n
		}
		callerNode := sprintNode(edge.Caller)
		calleeNode := sprintNode(edge.Callee)

		// edges
		attrs := make(dotAttrs)

		// dynamic call
		if edge.Site != nil && edge.Site.Common().StaticCallee() == nil {
			attrs["style"] = "dashed"
		}

		// go & defer calls
		switch edge.Site.(type) {
		case *ssa.Go:
			attrs["arrowhead"] = "normalnoneodot"
		case *ssa.Defer:
			attrs["arrowhead"] = "normalnoneodiamond"
		}

		// colorize calls outside focused pkg
		if focusPkg != nil &&
			(calleePkg.Path() != focusPkg.Path() || callerPkg.Path() != focusPkg.Path()) {
			attrs["color"] = "saddlebrown"
		}

		e := &dotEdge{
			From:  callerNode,
			To:    calleeNode,
			Attrs: attrs,
		}

		// omit duplicate calls
		key := fmt.Sprintf("%s = %s => %s", caller.Func, edge.Description(), callee.Func)
		if _, ok := edgeMap[key]; !ok {
			edges = append(edges, e)
			edgeMap[key] = e
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	logf("%d/%d edges", len(edges), count)

	dot := &dotGraph{
		Title:   mainPkg.Path(),
		Minlen:  minlen,
		Cluster: cluster,
		Nodes:   nodes,
		Edges:   edges,
		Options: map[string]string{
			"minlen":  fmt.Sprint(minlen),
			"nodesep": fmt.Sprint(nodesep),
		},
	}

	var buf bytes.Buffer
	if err := WriteDot(&buf, dot); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
