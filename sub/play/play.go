package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

type uaTransport struct {
	rt        http.RoundTripper
	UserAgent string
}

func (t *uaTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.UserAgent)
	return t.rt.RoundTrip(req)
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("play: ")

	http.DefaultTransport = &uaTransport{rt: http.DefaultTransport, UserAgent: os.Args[1]}

	code, _ := io.ReadAll(os.Stdin)
	resp, err := http.PostForm("https://play.golang.org/compile", url.Values{"body": {string(code)}})
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("%s: %s", resp.Request.URL, resp.Status)
		// Dump the full HTTP response
		// fmt.Fprintln(os.Stderr, resp.Status)
		// resp.Header.Write(os.Stderr)
		// io.Copy(os.Stderr, resp.Body)
		// os.Exit(1)
	}

	// resp.Body = io.NopCloser(io.TeeReader(resp.Body, os.Stdout)); // Enable for debugging
	var r struct {
		Errors string
		Events []struct {
			Delay   time.Duration
			Message string
			Kind    string
		}
		Status int
		// IsTest      bool // unused
		// TestsFailed int  // unused
	}

	var buf bytes.Buffer
	rBody := io.TeeReader(resp.Body, &buf)

	if err := json.NewDecoder(rBody).Decode(&r); err != nil {
		// Dump the full HTTP response
		fmt.Fprintln(os.Stderr, resp.Status)
		resp.Header.Write(os.Stderr)
		io.Copy(os.Stderr, io.MultiReader(&buf, resp.Body))

		log.Fatal(err)
	}
	if r.Errors != "" {
		log.Print(r.Errors)
	}
	// Replay events
	for _, ev := range r.Events {
		time.Sleep(ev.Delay)
		if ev.Kind == "stdout" {
			io.WriteString(os.Stdout, ev.Message)
		} else {
			io.WriteString(os.Stderr, ev.Message)
		}
	}
	os.Exit(r.Status)
}
