package load

import (
	"github.com/skudasov/loadgen"
	"log"
)

func AttackerFromName(name string) loadgen.Attack {
	switch name {
	case "first_test":
		return loadgen.WithMonitor(new(FirstTestAttack))
	case "second_test":
		return loadgen.WithMonitor(new(SecondTestAttack))
	case "third_test":
		return loadgen.WithMonitor(new(ThirdTestAttack))
	case "fourth_test":
		return loadgen.WithMonitor(new(FourthTestAttack))
	default:
		log.Fatalf("unknown attacker type: %s", name)
		return nil
	}
}
