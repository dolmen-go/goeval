// This directory contains sub commands of [github.com/dolmen-go/goeval].
//
// Sub commands have the following constraints:
//   - only stdlib dependencies
//   - compiled in GOPATH mode (GO111MODULE=off)
//
// The source code of each command is embedded (see [embed]) in the goeval binary and commands are launched with "go run".
package sub
