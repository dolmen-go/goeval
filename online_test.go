//go:build !goeval.offline

package main_test

// Show "goeval -play": running code remotely on https://play.golang.org/
func Example_play() {
	goeval(`-play`, `fmt.Println(time.Now())`)

	// Output:
	// 2009-11-10 23:00:00 +0000 UTC m=+0.000000001
}
