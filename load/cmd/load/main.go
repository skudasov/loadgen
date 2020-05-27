package main

import (
	"github.com/skudasov/loadgen"
	"github.com/skudasov/loadgen/load"
)

func main() {
	loadgen.Run(load.AttackerFromName, load.CheckFromName)
}
