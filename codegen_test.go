package loadgen

import (
	"fmt"
	"testing"

	. "github.com/dave/jennifer/jen"
)

func TestCodegen(t *testing.T) {
	f := NewFile("main")
	f.Func().Id("AttackerFromName").Params(
		Id("name").Id("string"),
	).Block(
		Switch(Id("name")).Block(
			Case(Lit("yandex")).Block(
				Return(Qual("loadgen", "WithMonitor").Call(Id("new").Call(Id("test")))),
			),
			Default().Block(
				Return(Lit("none")),
			),
		),
	)
	fmt.Printf("%#v", f)
}
