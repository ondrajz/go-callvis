package mypkg

type myType struct{}

func New() *myType {
	t := &myType{}
	return t
}

func (t *myType) Normal() {}

func (t *myType) Dynamic() {}
