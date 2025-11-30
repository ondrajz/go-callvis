package main

import (
	"github.com/OscarBohlin/go-callvis/examples/main/mypkg"
)

func main() {
	funcs()
	var c calls
	c.execution()
	c.invocation()
}

func funcs() {
	mypkg.Exported()
}

type calls struct{}

func (calls) execution() {
	mypkg.Regular()
}

func (calls) invocation() {
	mypkg.T.Static()
	var i mypkg.Iface = mypkg.T
	i.Dynamic()
}
