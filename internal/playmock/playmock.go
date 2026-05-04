/*
   Copyright 2026 Olivier Mengué.

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

package playmock

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

// Server allows to mock the Go Playground server.
type Server struct {
	Compile func(*CompileRequest) (*CompileResponse, error)
	Share   func(*ShareRequest) (*ShareResponse, error)
}

type CompileRequest struct {
	Body    string `json:"Body"`
	WithVet bool   `json:"WithVet"`
}

type CompileResponse struct {
	Errors      string         `json:"Errors,omitempty"`
	Events      []CompileEvent `json:"Events,omitempty"`
	Status      int            `json:"Status,omitempty"`
	IsTest      bool           `json:"IsTest,omitempty"`
	TestsFailed int            `json:"TestsFailed,omitempty"`
	VetErrors   string         `json:"VetErrors,omitempty"`
	VetOK       bool           `json:"VetOK,omitempty"`
}

type CompileEvent struct {
	Message string        `json:"Message,omitempty"`
	Kind    string        `json:"Kind,omitempty"`
	Delay   time.Duration `json:"Delay,omitempty"`
}

type ShareRequest struct {
	Body string
}

type ShareResponse struct {
	ID string
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /compile", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		contentType := r.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotAcceptable)
			return
		}
		var req CompileRequest
		switch mediaType {
		case "application/json":
			if len(params) > 0 {
				if charset, ok := params["charset"]; ok && charset != "utf-8" {
					http.Error(w, "Invalid charset", http.StatusNotAcceptable)
					return
				}
			}
			dec := json.NewDecoder(&io.LimitedReader{R: r.Body, N: 4096})
			if err := dec.Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		case "application/x-www-form-urlencoded":
			if err := r.ParseForm(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			req.Body = r.FormValue("body")
			req.WithVet = r.FormValue("withVet") == "true"
		default:
			http.Error(w, "Unexpected media type", http.StatusNotAcceptable)
			return
		}

		if s.Compile == nil {
			http.Error(w, "Not Implemented", http.StatusNotImplemented)
			return
		}

		resp, err := s.Compile(&req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("POST /share", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		contentType := r.Header.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil || (!strings.HasPrefix(mediaType, "application/") && !strings.HasPrefix(mediaType, "text/")) {
			http.Error(w, err.Error(), http.StatusNotAcceptable)
			return
		}
		if len(params) > 0 {
			if charset, ok := params["charset"]; ok && charset != "utf-8" {
				http.Error(w, "Invalid charset", http.StatusNotAcceptable)
				return
			}
		}

		body, err := io.ReadAll(&io.LimitedReader{R: r.Body, N: 4096})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if s.Share == nil {
			http.Error(w, "Not Implemented", http.StatusNotImplemented)
			return
		}

		resp, err := s.Share(&ShareRequest{Body: string(body)})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(resp.ID))
	})

	return mux
}

// Run launches an HTTP server.
//
// The returned shutdown function must be called to shutdown the server.
func Run(ctx context.Context, h http.Handler) (u string, cleanup func(), _ error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}

	srv := &http.Server{Handler: h}

	shutdown := func() {
		srv.Shutdown(context.Background())
	}

	go func() {
		<-ctx.Done()
		shutdown()
	}()

	go srv.Serve(l)

	return fmt.Sprintf("http://%s", l.Addr().String()), shutdown, nil
}

func TestRun(t interface {
	Context() context.Context
	Fatalf(string, ...interface{})
	Cleanup(func())
}, h http.Handler) string {
	u, shutdown, err := Run(t.Context(), h)
	if err != nil {
		t.Fatalf("failed to run server: %v", err)
		return "" // unreachable
	}
	t.Cleanup(shutdown)
	return u
}

func RunProxy(ctx context.Context, serverURL string, h http.Handler) (proxyURL string, caPEM []byte, cleanup func(), _ error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return "", nil, nil, err
	}
	if u.Scheme != "https" {
		return "", nil, nil, errors.New("RunProxy handles only https URLs")
	}

	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		host = u.Host // Use host as-is if no port is present
		port = "443"
	}

	randData := "_" // just to be sure that we start with a character that isn't a digit
	for len(randData) < 16 {
		randData += strings.ToLower(rand.Text())
	}
	proxyAuth := randData[:8] + ":" + randData[8:16]
	proxyAuthHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(proxyAuth))

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Minute)
		ctx, _ = context.WithDeadlineCause(ctx, deadline, fmt.Errorf("certificate for %v will expire", host))
	}

	certPEM, keyPEM, caPEM, err := newCerts(host, deadline.Add(1*time.Minute))
	if err != nil {
		return "", nil, nil, fmt.Errorf("can't create certificates: %w", err)
	}

	// 1. Prepare TLS configuration for the hijacked connection
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return "", nil, nil, err
	}

	// Ensure targetHost includes port for comparison if necessary
	targetHost := host + ":" + port

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		NextProtos:   []string{"h2", "http/1.1"}, // Enable HTTP/2 support!
	}

	u.Path = strings.TrimSuffix(u.Path, "/")
	h = http.StripPrefix(u.Path, h)

	// 1. Initialize the Virtual Listener and the Inner Server
	vLn := newVirtualListener()
	tlsLn := tls.NewListener(vLn, tlsConfig)
	innerServer := &http.Server{
		Handler:     h,
		BaseContext: func(net.Listener) context.Context { return ctx },
	}

	// 2. Define the main proxy logic
	proxyFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Proxy-Authorization"); auth != proxyAuthHeader {
			w.Header().Add("Proxy-Authenticate", `Basic realm="Proxy Server"`)
			http.Error(w, "Proxy Authentication Required", http.StatusProxyAuthRequired)
			return
		}
		if r.Method != http.MethodConnect {
			http.Error(w, "Proxy only supports CONNECT", http.StatusMethodNotAllowed)
			return
		}

		// Check if the client is trying to reach our specific target URL
		if r.Host != targetHost {
			http.Error(w, "Proxy only authorized for "+targetHost, http.StatusForbidden)
			return
		}

		// Hijack the connection to establish the TLS tunnel
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
			return
		}

		clientConn, _, err := hijacker.Hijack()
		if err != nil {
			return
		}

		// Inform the client that the tunnel is established
		_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

		// Push the RAW connection. The innerServer's tls.Listener will
		// pick it up and perform the TLS handshake.
		select {
		case vLn.conns <- clientConn:
		case <-vLn.done:
			clientConn.Close()
		case <-ctx.Done():
			clientConn.Close()
		}
	})

	// 3. Start the proxy listener (plain HTTP for the proxy control channel)
	proxySrv := &http.Server{Handler: proxyFunc}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, nil, err
	}

	// Serve until shutdown closes the listeners
	go innerServer.Serve(tlsLn)
	go proxySrv.Serve(ln)

	proxyAddr := "http://" + proxyAuth + "@" + ln.Addr().String()
	shutdown := func() {
		vLn.Close() // Stop accepting new connections
		innerServer.Shutdown(ctx)
		proxySrv.Shutdown(ctx)
	}

	return proxyAddr, caPEM, shutdown, nil
}

type virtualListener struct {
	conns  chan net.Conn
	done   chan struct{}
	closed atomic.Bool
}

func newVirtualListener() *virtualListener {
	return &virtualListener{
		conns: make(chan net.Conn),
		done:  make(chan struct{}),
	}
}

func (l *virtualListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *virtualListener) Close() error {
	if l.closed.CompareAndSwap(false, true) {
		close(l.done)
	}
	return nil
}

var virtualListenerIP = net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}

func (*virtualListener) Addr() net.Addr {
	return &virtualListenerIP
}

func TestRunProxy(t interface {
	Context() context.Context
	Fatalf(string, ...interface{})
	Cleanup(func())
}, serverURL string, h http.Handler) (string, []byte) {
	proxyURL, caPEM, shutdown, err := RunProxy(t.Context(), serverURL, h)
	if err != nil {
		t.Fatalf("failed to run proxy: %v", err)
		return "", nil // unreachable
	}
	t.Cleanup(shutdown)
	return proxyURL, caPEM
}

// ProxyEnv builds environment variables to use to connect to [Server.RunProxy]:
//
//   - HTTPS_PROXY
//   - SSL_CERT_FILE
//
// Note: [crypto/x509.SystemCertPool()] has builtin support for SSL_CERT_FILE only on some platforms.
// So you might want to add explicit support for that variable in a Go program that connects to the proxy
// (see [net/http.Transport], [tls.Config]) for Windows, macOS support:
//
//	caPEM, _ := os.ReadFile(os.Getenv("SSL_CERT_FILE"))
//	pool := x509.NewCertPool()
//	_ = pool.AppendCertsFromPEM(caPEM)
//	http.DefaultTransport.(*http.Transport).TLSClientConfig.RootCAs = pool
func ProxyEnv(t interface {
	TempDir() string
	Fatalf(string, ...any)
}, proxyURL string, caPEM []byte) []string {

	caCertFile := filepath.Join(t.TempDir(), "cacert.pem")
	if err := os.WriteFile(caCertFile, caPEM, 0400); err != nil {
		t.Fatalf("can't write SSL_CERT_FILE: %v", err)
	}

	return []string{
		"HTTPS_PROXY=" + proxyURL,
		"SSL_CERT_FILE=" + caCertFile,
	}
}
