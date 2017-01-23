package main

import (
	"fmt"
	"go/build"
	"go/types"
	"io"
	"os"
	"strings"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
)

var output io.Writer = os.Stdout

func printOutput(mainPkg *types.Package, cg *callgraph.Graph, focusPkg *build.Package, limitPaths, ignorePaths []string, groupBy map[string]bool) error {
	groupType := groupBy["type"]
	groupPkg := groupBy["pkg"]

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
		cluster.Attrs["label"] = focusPkg.Name
	}

	nodes := []*dotNode{}
	edges := []*dotEdge{}

	nodeMap := make(map[string]*dotNode)
	edgeMap := make(map[string]*dotEdge)

	cg.DeleteSyntheticNodes()

	logf("%d limit prefixes: %v", len(limitPaths), limitPaths)
	logf("%d ignore prefixes: %v", len(ignorePaths), ignorePaths)

	var inLimit = func(from, to *callgraph.Node) bool {
		if len(limitPaths) == 0 {
			return true
		}
		var fromOk, toOk bool
		fromPath := from.Func.Pkg.Pkg.Path()
		toPath := to.Func.Pkg.Pkg.Path()
		for _, p := range limitPaths {
			if strings.HasPrefix(fromPath, p) {
				fromOk = true
			}
			if strings.HasPrefix(toPath, p) {
				toOk = true
			}
			if fromOk && toOk {
				logf("in limit: %s -> %s", from, to)
				return true
			}
		}
		logf("NOT in limit: %s -> %s", from, to)
		return false
	}

	err := callgraph.GraphVisitEdges(cg, func(edge *callgraph.Edge) error {
		caller := edge.Caller
		callee := edge.Callee

		// omit synthetic calls
		if caller.Func.Pkg == nil || callee.Func.Synthetic != "" {
			return nil
		}

		callerPkg := caller.Func.Pkg.Pkg
		calleePkg := callee.Func.Pkg.Pkg

		// focus specific pkg
		if focusPkg != nil &&
			!(callerPkg.Path() == focusPkg.ImportPath || calleePkg.Path() == focusPkg.ImportPath) {
			return nil
		}

		// limit path prefixes
		if !inLimit(caller, callee) {
			return nil
		}

		// ignore path prefixes
		for _, p := range ignorePaths {
			if strings.HasPrefix(callerPkg.Path(), p) ||
				strings.HasPrefix(calleePkg.Path(), p) {
				return nil
			}
		}

		var sprintNode = func(node *callgraph.Node) *dotNode {
			// only once
			key := node.Func.String()
			if n, ok := nodeMap[key]; ok {
				return n
			}

			isFocused := focusPkg != nil &&
				node.Func.Pkg.Pkg.Path() == focusPkg.ImportPath
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
							"label":     label,
							"style":     "filled",
							"fillcolor": "lightyellow",
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
			(calleePkg.Path() != focusPkg.ImportPath || callerPkg.Path() != focusPkg.ImportPath) {
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
		return err
	}

	logf("%d edges", len(edges))

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

	return WriteDot(output, dot)
}
