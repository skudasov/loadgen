package main

import (
	"github.com/skudasov/loadgen"
	"github.com/skudasov/loadgen/example_loadtest"
)

func main() {
	loadgen.Run(example_loadtest.AttackerFromName, example_loadtest.CheckFromName)
}
