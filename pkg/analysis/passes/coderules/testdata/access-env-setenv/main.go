package donotinclude

import (
	"os"
	"syscall"
)

func DoNotInclude() {
	panic("This function should never be included in the binary.")

	// NOT allowed
	_ = os.Setenv("DO_NOT_INCLUDE_OS", "value")
	_ = syscall.Setenv("DO_NOT_INCLUDE_SYSCALL", "value")

	// GF_PLUGIN are allowed
	_ = os.Setenv("GF_PLUGIN_ALLOWED_ENV", "value")
	_ = syscall.Setenv("GF_PLUGIN_ALLOWED_ENV", "value")
}
