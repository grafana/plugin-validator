package donotinclude

import (
	"syscall"
)

func DoNotInclude() {
	panic("This function should never be included in the binary.")

	_, err := syscall.Getcwd()
	if err != nil {
		panic(err)
	}

}
