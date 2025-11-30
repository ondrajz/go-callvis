// go-callvis: a tool to help visualize the call graph of a Go program.
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"go/build"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/pkg/browser"
	"golang.org/x/tools/go/buildutil"
	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/ssa"
)

const Usage = `go-callvis: visualize call graph of a Go program.

Usage:

  go-callvis [flags] package

  Package should be main package, otherwise -tests flag must be used.

Flags:

`

var (
	focusFlag     = flag.String("focus", "main", "Focus specific package using name or import path.")
	groupFlag     = flag.String("group", "pkg", "Grouping functions by packages and/or types [pkg, type] (separated by comma)")
	limitFlag     = flag.String("limit", "", "Limit package paths to given prefixes (separated by comma)")
	ignoreFlag    = flag.String("ignore", "", "Ignore package paths containing given prefixes (separated by comma)")
	includeFlag   = flag.String("include", "", "Include package paths with given prefixes (separated by comma)")
	nostdFlag     = flag.Bool("nostd", false, "Omit calls to/from packages in standard library.")
	nointerFlag   = flag.Bool("nointer", false, "Omit calls to unexported functions.")
	testFlag      = flag.Bool("tests", false, "Include test code.")
	graphvizFlag  = flag.Bool("graphviz", false, "Use Graphviz's dot program to render images.")
	httpFlag      = flag.String("http", ":7878", "HTTP service address.")
	skipBrowser   = flag.Bool("skipbrowser", false, "Skip opening browser.")
	outputFile    = flag.String("file", "", "output filename - omit to use server mode")
	outputFormat  = flag.String("format", "svg", "output file format [svg | png | jpg | ...]")
	importFlag    = flag.String("exportImport", "", "Writes the imports found in provided name or import path to specified file.")
	cacheDir      = flag.String("cacheDir", "", "Enable caching to avoid unnecessary re-rendering, you can force rendering by adding 'refresh=true' to the URL query or emptying the cache directory")
	callgraphAlgo = flag.String("algo", string(CallGraphTypeStatic), fmt.Sprintf("The algorithm used to construct the call graph. Possible values inlcude: %q, %q, %q, %q",
		CallGraphTypeStatic, CallGraphTypeCha, CallGraphTypeRta, CallGraphTypeVta))

	debugFlag   = flag.Bool("debug", false, "Enable verbose log.")
	versionFlag = flag.Bool("version", false, "Show version and exit.")
)

func init() {
	flag.Var((*buildutil.TagsFlag)(&build.Default.BuildTags), "tags", buildutil.TagsFlagDoc)
	// Graphviz options
	flag.UintVar(&minlen, "minlen", 2, "Minimum edge length (for wider output).")
	flag.Float64Var(&nodesep, "nodesep", 0.35, "Minimum space between two adjacent nodes in the same rank (for taller output).")
	flag.StringVar(&nodeshape, "nodeshape", "box", "graph node shape (see graphvis manpage for valid values)")
	flag.StringVar(&nodestyle, "nodestyle", "filled,rounded", "graph node style (see graphvis manpage for valid values)")
	flag.StringVar(&rankdir, "rankdir", "LR", "Direction of graph layout [LR | RL | TB | BT]")
}

func logf(f string, a ...interface{}) {
	if *debugFlag {
		log.Printf(f, a...)
	}
}

func parseHTTPAddr(addr string) string {
	host, port, _ := net.SplitHostPort(addr)
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "80"
	}
	u := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%s", host, port),
	}
	return u.String()
}

func openBrowser(url string) {
	time.Sleep(time.Millisecond * 100)
	if err := browser.OpenURL(url); err != nil {
		log.Printf("OpenURL error: %v", err)
	}
}

func outputDot(fname string, outputFormat string) {
	// get cmdline default for analysis
	Analysis.OptsSetup()

	if e := Analysis.ProcessListArgs(); e != nil {
		log.Fatalf("%v\n", e)
	}

	output, err := Analysis.Render()
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	log.Println("writing dot output")

	writeErr := os.WriteFile(fmt.Sprintf("%s.gv", fname), output, 0755)
	if writeErr != nil {
		log.Fatalf("%v\n", writeErr)
	}

	log.Printf("converting dot to %s\n", outputFormat)

	if outputFormat == "csv" {
		dotToCsv(fmt.Sprintf("%v.csv", fname))
		return
	}

	_, err = dotToImage(fname, outputFormat, output)
	if err != nil {
		log.Fatalf("%v\n", err)
	}
}

func writeCsv(writer *csv.Writer, nodesMap map[*ssa.Function]*callgraph.Node) {
	for fun, node := range nodesMap {
		if node.Out == nil {
			continue
		}

		line := []string{fun.String()}

		for _, edge := range node.Out {
			line = append(line, edge.Callee.Func.String())
		}
		err := writer.Write(line)
		if err != nil {
			log.Fatalf("error writing csv: %v\n", err)
		}
	}
}

//Creates a adjacency list in csv format for all nodes and edges
func dotToCsv(fname string) (error) {
	f, err := os.Create(fname)
	if err != nil {
		log.Fatalf("could not create file %v: %v\n", fname, err)
	}

	writer := csv.NewWriter(f)
	writeCsv(writer, Analysis.callgraph.Nodes)
	writer.Flush()
	f.Close()
	return nil
}

func outputImports(fname string) {
	output := ""
	for _, p := range Analysis.imports {
		output += fmt.Sprintf("%v\n", p.String())
	}
	bytes := []byte(output)
	err := os.WriteFile(fname, bytes, 0755)
	if err != nil {
		 log.Fatalf("%v\n", err)
	}
}

//noinspection GoUnhandledErrorResult
func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Fprintln(os.Stderr, Version())
		os.Exit(0)
	}
	if *debugFlag {
		log.SetFlags(log.Lmicroseconds)
	}

	if flag.NArg() != 1 {
		fmt.Fprint(os.Stderr, Usage)
		flag.PrintDefaults()
		os.Exit(2)
	}

	args := flag.Args()
	tests := *testFlag
	httpAddr := *httpFlag
	urlAddr := parseHTTPAddr(httpAddr)

	Analysis = new(analysis)
	if err := Analysis.DoAnalysis(CallGraphType(*callgraphAlgo), "", tests, args); err != nil {
		log.Fatal(err)
	}

	if *importFlag != "" {
		outputImports(*importFlag)
	}

	http.HandleFunc("/", handler)

	if *outputFile == "" {
		*outputFile = "output"
		if !*skipBrowser {
			go openBrowser(urlAddr)
		}

		log.Printf("http serving at %s", urlAddr)

		if err := http.ListenAndServe(httpAddr, nil); err != nil {
			log.Fatal(err)
		}
	} else {
		outputDot(*outputFile, *outputFormat)
	}
}
