package load

import (
	"github.com/skudasov/loadgen"
	"log"
)

func CheckFromName(name string) loadgen.RuntimeCheckFunc {
	switch name {
	default:
		log.Printf("unknown check type: %s, skipping", name)
		return nil
	}
}
