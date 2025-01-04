//go:build !cgo

package main

func runDotToImage(outfname string, format string, dot []byte) (string, error) {
    return runDotToImageCallSystemGraphviz(outfname, format, dot)
}
