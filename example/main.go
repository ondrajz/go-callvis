package main

import (
	"log"
	"net/http"

	"github.com/TrueFurby/go-callvis/example/mypkg"
)

var t = mypkg.Exported()

type myIface interface {
	Dynamic()
}

func init() {
	go func() {
		log.Fatal(http.ListenAndServe(":8000", nil))
	}()
}

func main() {
	defer deferred()
	direct()
	go concurrent()
}

func direct() {
	var i myIface = t
	i.Dynamic()
	t.Static()
}

func deferred() {
	defer t.Static()
}

func concurrent() {
	go t.Static()
}
