package mypkg

type myType struct{}

// Exported func
func Exported() *myType {
	return &myType{}
}

func (t *myType) Static() {}

func (t *myType) Dynamic() {}
