package main

import (
	"fmt"
	"errors"
)

func verboseLog(format string, a ...interface{}) {
	if !verbose {
		return
	}

	fmt.Printf("info: " + format + "\n", a...)
}

func errorLog(format string, a ...interface{}) {
	fmt.Printf("ERROR! " + format + "\n", a...)
}

func warnLog(format string, a ...interface{}) {
	fmt.Printf("WARNING! " + format + "\n", a...)
}

func fatalLog(format string, a ...interface{}) {
	fmt.Printf("FATAL! " + format + "\n", a...)
	panic(errors.New("unhandled-error"))
}
