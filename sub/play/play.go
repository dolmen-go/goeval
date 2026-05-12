package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
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

	if caCertsFile := os.Getenv("SSL_CERT_FILE"); caCertsFile != "" {
		if tr, ok := http.DefaultTransport.(*http.Transport); ok {
			tlsConfig := tr.TLSClientConfig
			if tlsConfig == nil {
				tlsConfig = new(tls.Config)
			}
			caPEM, err := os.ReadFile(caCertsFile)
			if err == nil {
				pool := x509.NewCertPool()
				if pool.AppendCertsFromPEM(caPEM) {
					tlsConfig.RootCAs = pool

					if tr.TLSClientConfig == nil {
						tr.TLSClientConfig = tlsConfig
					}
				}
			}
		}
	}

	http.DefaultTransport = &uaTransport{rt: http.DefaultTransport, UserAgent: os.Args[1]}

	code, _ := io.ReadAll(os.Stdin)
	resp, err := http.PostForm("https://play.golang.org/compile", url.Values{"body": {string(code)}})
	if err != nil {
		if cverr := new(tls.CertificateVerificationError); errors.As(err, &cverr) {
			for _, crt := range cverr.UnverifiedCertificates {
				log.Printf("Subject: %q, Issuer: %q", crt.Subject, crt.Issuer)
			}
		}
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
		fmt.Fprint(os.Stderr, r.Errors)
		// The Playground doesn't set an exit code in case of error, but we want that.
		if r.Status == 0 {
			r.Status = 1
		}
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
