package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

func main() {
	code, _ := io.ReadAll(os.Stdin)
	// TODO User-Agent
	resp, err := http.PostForm("https://play.golang.org/compile", url.Values{"body": {string(code)}})
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
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
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
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
