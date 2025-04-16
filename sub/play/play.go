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
	resp, err := http.PostForm("https://go.dev/_/compile", url.Values{"version": {"2"}, "body": {string(code)}})
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	// resp.Body = io.NopCloser(io.TeeReader(resp.Body, os.Stdout)); // Enable for debugging
	var r struct {
		Events []struct {
			Delay   time.Duration
			Message string
			Kind    string
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		log.Fatal(err)
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
}
