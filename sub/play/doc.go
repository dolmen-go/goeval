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

// Command play is the sub command launched by "goeval -play".
//
// play sends the program code to the Go Playground at https://play.golang.org/compile
// and replays the received events respecting event delays.
//
//	$ curl -s -X POST --data-urlencode body@- https://play.golang.org/compile <<EOF
//	package main
//	import "fmt"
//	func main() {
//	  fmt.Println("Hello, world!")
//	}
//	EOF
//	{"Errors":"","Events":[{"Message":"Hello, world!\n","Kind":"stdout","Delay":0}],"Status":0,"IsTest":false,"TestsFailed":0}
package main
