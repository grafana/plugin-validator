package donotinclude

import (
	"os"
)

func DoNotInclude() {
	panic("This function should never be included in the binary.")

	// NOT allowed
	name := "MY_VARIABLE"
	env := os.Getenv(name)

	// GF_PLUGIN are allowed
	name := "GF_PLUGIN_ALLOWED_ENV"
	env := os.Getenv(name)

	// NOT allowed
	env := os.Getenv("DO_NOT_INCLUDE")

	// GF_PLUGIN are allowed
	env := os.Getenv("GF_PLUGIN_ALLOWED_ENV")
	_ = env

	// NOT allowed
	for _, e := range os.Environ() {
		_ = e
	}
}
