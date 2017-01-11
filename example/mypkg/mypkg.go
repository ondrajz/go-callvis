package mypkg

import (
	"log"
	"net/http"
)

func init() {
	go func() {
		log.Fatal(http.ListenAndServe(":8000", nil))
	}()
}

func Exported() {
	unexported()
}
func unexported() {}

var T = new(myType)

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
