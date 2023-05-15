package main

import (
	"fmt"
)

type LogLevel string

const (
	LogLevelDefault LogLevel = "default"
	LogLevelSilent  LogLevel = "silent"
	LogLevelDebug   LogLevel = "debug"
)

func LogLevelFromString(s string) LogLevel {
	switch s {
	case "debug":
		return LogLevelDebug
	default:
		return LogLevelDefault
	}
}

func logln(a ...any) {
	if LogLevelFromString(*logLevel) != LogLevelSilent {
		fmt.Println(a...)
	}
}

func logf(format string, a ...any) {
	if LogLevelFromString(*logLevel) != LogLevelSilent {
		fmt.Printf(format, a...)
		fmt.Println()
	}
}

func debugln(a ...any) {
	if LogLevelFromString(*logLevel) == LogLevelDebug {
		fmt.Print("DEBUG: ")
		fmt.Println(a...)
	}
}

func debugf(format string, a ...any) {
	if LogLevelFromString(*logLevel) == LogLevelDebug {
		fmt.Print("DEBUG: ")
		fmt.Printf(format, a...)
		fmt.Println()
	}
}
