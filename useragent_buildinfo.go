package main

import (
	"runtime/debug"
	"strings"
)

var version = "v1.2.0" // FIXME set at compile time with -ldflags="-X main.version="

// getUserAgent returns the HTTP User-Agent header value to use for -play and -share.
//
// Reference: https://www.rfc-editor.org/rfc/rfc9110#name-user-agent
func getUserAgent() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		// The HTTP specification allows comments in header values: they are enclosed by parenthesis.
		return "goeval/" + version + " (github.com/dolmen-go/goeval)"
	}
	if !ok || bi.Main.Path == "" {
		// The HTTP specification allows comments in header values: they are enclosed by parenthesis.
		return "goeval/" + version + " (" + bi.Path + ")"
	}
	// "go run" reports "(devel)" as version but in header value parenthesis are reserved chars (for comments).
	return "goeval/" + strings.Trim(bi.Main.Version, "()") + " (" + bi.Main.Path + ")"
}
