package main

import (
	"log"
	"net/http"

	"github.com/TrueFurby/go-callvis/example/mypkg"
)

var (
	t = mypkg.New()
)

type myIface interface {
	Dynamic()
}

func init() {
	go func() {
		log.Fatal(http.ListenAndServe(":8000", nil))
	}()
}

func Direct() {
	var i myIface = t
	i.Dynamic()
	t.Normal()
}

func deferred() {
	defer t.Normal()
}

func concurrent() {
	go t.Normal()
}

func main() {
	defer deferred()
	Direct()
	go concurrent()
}
