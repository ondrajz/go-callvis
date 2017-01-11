package main

import "github.com/TrueFurby/go-callvis/example/mypkg"

func main() {
	accessType()
	callExecution()
	invokeMode()
}

func accessType() {
	mypkg.Exported()
}

func callExecution() {
	mypkg.Regular()
}

type myIface interface {
	Dynamic()
}

func invokeMode() {
	mypkg.T.Static()
	var i myIface = mypkg.T
	i.Dynamic()
}
