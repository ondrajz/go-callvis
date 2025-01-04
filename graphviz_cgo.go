//go:build cgo

package main

import (
    "fmt"
    "log"
    "os"
    "path/filepath"

    "github.com/goccy/go-graphviz"
)

func runDotToImage(outfname string, format string, dot []byte) (string, error) {
    g := graphviz.New()
    graph, err := graphviz.ParseBytes(dot)
    if err != nil {
        return "", err
    }
    defer func() {
        if err := graph.Close(); err != nil {
            log.Printf("error closing graph: %v", err)
        }
        if err := g.Close(); err != nil {
            log.Printf("error closing graphviz: %v", err)
        }
    }()
    var path string
    if outfname == "" {
        path = filepath.Join(os.TempDir(), fmt.Sprintf("go-callvis_export.%s", format))
    } else {
        path = fmt.Sprintf("%s.%s", outfname, format)
    }
    if err := g.RenderFilename(graph, graphviz.Format(format), path); err != nil {
        return "", err
    }
    return path, nil
}
