package donotinclude

import (
	"fmt"
	"os"
)

func DoNotInclude() {
	panic("This function should never be included in the binary.")

	env := os.Getenv("DO_NOT_INCLUDE")
	log.Info(env)

	for _, e := range os.Environ() {
		log.Info(e)
	}
}
