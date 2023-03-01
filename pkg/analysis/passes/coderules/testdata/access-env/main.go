package donotinclude

import (
	"fmt"
	"os"
)

func DoNotInclude() {
	panic("This function should never be included in the binary.")

	env := os.Getenv("DO_NOT_INCLUDE")
	fmt.Println(env)

	for _, e := range os.Environ() {
		fmt.Println(e)
	}
}
