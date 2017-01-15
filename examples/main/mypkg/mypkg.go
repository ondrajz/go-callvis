package mypkg

import (
	"log"
	"net/http"
	"os"
)

var l = log.New(os.Stderr, "mypkg", 0)

func init() {
	go func() {
		l.Fatal(http.ListenAndServe(":8080", nil))
	}()
}

func Exported() {
	unexported()
}
func unexported() {}

var T = new(myType)

type Iface interface {
	Dynamic()
}

type myType struct{}

func NewMyType() *myType {
	return &myType{}
}

func (t *myType) Static()  {}
func (t *myType) Dynamic() {}

func Regular() {
	defer deferred()
	go concurrent()
}
func deferred()   {}
func concurrent() {}
