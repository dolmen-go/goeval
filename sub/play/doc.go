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
