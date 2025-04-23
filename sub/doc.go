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

// This directory contains sub commands of [github.com/dolmen-go/goeval].
//
// Sub commands have the following constraints:
//   - only stdlib dependencies
//   - compiled in GOPATH mode (GO111MODULE=off)
//
// The source code of each command is embedded (see [embed]) in the goeval binary and commands are launched with "go run".
package sub
