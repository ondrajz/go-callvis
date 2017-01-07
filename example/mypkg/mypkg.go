package mypkg

type myType struct{}

// Exported represents exported func
func Exported() *myType {
	return &myType{}
}

func (t *myType) Normal() {}

func (t *myType) Dynamic() {}
