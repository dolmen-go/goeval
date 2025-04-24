/*
   Copyright 2019-2025 Olivier Mengué.

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

// Command goeval allows to run Go snippets given on the command line.
//
// A Go toolchain must be available in $PATH as goeval relies on "go run".
//
// The code, given either as the first argument or on stdin, is wrapped as
// the body of a main() function in a main package, and executed with "go run".
//
// Imports are implicit (they are usually resolved automatically thanks to
// [goimports]) but they can be explicitely specified using -i.
// If at least one package import is given with a version (import-path@version),
// a full Go module is assembled, and imports without version are resolved
// as the latest version available in the local Go module cache (GOMODCACHE).
//
// In GOPATH mode (the default), the local Go context is involved only if the current
// directory happens to be in GOPATH and the package is imported.
// In Go module mode, the local Go context (go.mod, .go source files) is completely
// ignored for resolving imports and compiling the snippet.
//
// -play runs the code in the sandbox of [the Go Playground] instead of the local
// machine and replays the output.
//
// -share posts the code for storage on [the Go Playground] and displays the URL.
//
// 🚀 Quick Start
//
//	go install github.com/dolmen-go/goeval@latest
//	goeval 'fmt.Println("Hello, world")'
//
// [goimports]: https://pkg.go.dev/golang.org/x/tools/imports
// [the Go Playground]: https://go.dev/play
package main
