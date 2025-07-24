/*
   Copyright 2025 Olivier Mengué.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"runtime/debug"
	"strings"
)

var version = "v1.4.0" // FIXME set at compile time with -ldflags="-X main.version="

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
