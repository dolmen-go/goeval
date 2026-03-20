package main

import (
	"io"
	"log"
	"net/http"
	"os"
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
	log.SetPrefix("share: ")

	http.DefaultTransport = &uaTransport{rt: http.DefaultTransport, UserAgent: os.Args[1]}

	resp, err := http.Post("https://play.golang.org/share", "text/plain; charset=utf-8", os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Fatalf("%s: %s", resp.Request.URL, resp.Status)
	}

	id, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	io.WriteString(os.Stdout, "https://go.dev/play/p/"+string(id)+"\n")
}
