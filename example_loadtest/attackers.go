package example_loadtest

import (
	"github.com/skudasov/loadgen"
	"log"
)

func AttackerFromName(name string) loadgen.Attack {
	switch name {
	case "google":
		return loadgen.WithMonitor(new(GoogleAttack))
	default:
		log.Fatalf("unknown attacker type: %s", name)
		return nil
	}
}

func CheckFromName(name string) loadgen.RuntimeCheckFunc {
	switch name {
	case "google":
		return func(r *loadgen.Runner, d map[string]string) bool {
			return false
		}
	default:
		log.Fatalf("unknown attacker check func type: %s", name)
		return nil
	}
}
