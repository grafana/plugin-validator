package logme

import (
	"fmt"
	"os"
)

var isDebugMode bool = os.Getenv("DEBUG") == "1"

func DebugF(msg string, args ...interface{}) {
	// check if ENV DEBUG is 1
	if isDebugMode {
		fmt.Print("[DEBUG] ")
		fmt.Fprintf(os.Stdout, msg, args...)
	}
}

func Debugln(args ...interface{}) {
	// check if ENV DEBUG is 1
	if isDebugMode {
		fmt.Print("[DEBUG] ")
		fmt.Fprintln(os.Stdout, args...)
	}
}

func InfoF(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, msg, args...)
}

func Infoln(arg ...interface{}) {
	fmt.Fprintln(os.Stdout, arg...)
}

func ErrorF(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg, args...)
}

func Errorln(arg ...interface{}) {
	fmt.Fprintln(os.Stderr, arg...)
}
