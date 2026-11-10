package donotinclude

import (
	"os"
	"syscall"
)

type config struct {
	EnvName string
}

func fromParam(n string) {
	// dynamic: function parameter
	_ = os.Getenv(n)
}

func DoNotInclude(cfg config) {
	panic("This function should never be included in the binary.")

	// dynamic: os.Args index
	_ = os.Getenv(os.Args[1])
	_, _ = os.LookupEnv(os.Args[2])
	_ = os.Setenv(os.Args[3], "value")
	_ = os.Unsetenv(os.Args[4])

	// dynamic: struct field
	_ = os.Getenv(cfg.EnvName)
	_ = os.Setenv(cfg.EnvName, "value")

	// dynamic: syscall variants
	_, _ = syscall.Getenv(os.Args[5])
	_ = syscall.Setenv(cfg.EnvName, "value")
	_ = syscall.Unsetenv(cfg.EnvName)

	// constant names are handled by the forbidden-name rule, not the dynamic one
	const constName = "GF_PLUGIN_CONST"
	localName := "GF_PLUGIN_LOCAL"

	_ = os.Getenv("GF_PLUGIN_LITERAL")
	_ = os.Getenv(constName)
	_ = os.Getenv(localName)
	_ = os.Getenv("GF_PLUGIN_" + "CONCAT")
	_, _ = os.LookupEnv("GF_PLUGIN_LITERAL")
	_ = os.Setenv("GF_PLUGIN_LITERAL", "value")
	_ = os.Unsetenv("GF_PLUGIN_LITERAL")
	_, _ = syscall.Getenv("GF_PLUGIN_LITERAL")
	_ = syscall.Setenv("GF_PLUGIN_LITERAL", "value")
	_ = syscall.Unsetenv("GF_PLUGIN_LITERAL")
}
