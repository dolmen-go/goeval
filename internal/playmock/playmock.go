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
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

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

func (s *Server) mux() *http.ServeMux {
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

// Run launches an HTTP server mocking the Go Playground.
//
// The returned shutdown function must be called to shutdown the server.
func (s *Server) Run(ctx context.Context) (u string, cleanup func(), _ error) {
	mux := s.mux()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}

	srv := &http.Server{Handler: mux}

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

func (s *Server) TestRun(t interface {
	Context() context.Context
	Fatalf(string, ...interface{})
	Cleanup(func())
}) string {
	u, shutdown, err := s.Run(t.Context())
	if err != nil {
		t.Fatalf("failed to run server: %v", err)
		return "" // unreachable
	}
	t.Cleanup(shutdown)
	return u
}

func (s *Server) RunProxy(ctx context.Context, serverURL string) (proxyURL string, caPEM []byte, cleanup func(), _ error) {
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

	certPEM, keyPEM, caPEM, err := newCerts(host, 5*time.Minute)
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
	}

	var mux http.Handler = s.mux()
	if u.Path != "" {
		u.Path = strings.TrimSuffix(u.Path, "/")
		mux = http.StripPrefix(u.Path, mux)
	}

	// 2. Define the main proxy logic
	proxyFunc := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			http.Error(w, "Proxy only supports CONNECT", http.StatusMethodNotAllowed)
			return
		}

		// Check if the client is trying to reach our specific target URL
		if r.Host != targetHost {
			http.Error(w, "Proxy only authorized for "+targetHost, http.StatusForbidden)
			return
		}

		// 3. Hijack the connection to establish the TLS tunnel
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
			return
		}

		clientConn, _, err := hijacker.Hijack()
		if err != nil {
			return
		}
		defer clientConn.Close()

		// Inform the client that the tunnel is established
		_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

		// 4. Wrap the raw connection in TLS (acting as the target server)
		tlsConn := tls.Server(clientConn, tlsConfig)
		handshakeCtx, cancelHandshake := context.WithTimeout(ctx, 5*time.Second)
		defer cancelHandshake()
		if err := tlsConn.HandshakeContext(handshakeCtx); err != nil {
			return
		}
		defer tlsConn.Close()

		// 5. Use httputil to read the decrypted request and pass it to our handler
		// We use a buffered reader for the TLS connection
		tlsReader := bufio.NewReader(tlsConn)
		for {
			// Set a deadline for ReadRequest so it doesn't block forever
			// if the context is cancelled
			_ = tlsConn.SetReadDeadline(time.Now().Add(1 * time.Second))

			req, err := http.ReadRequest(tlsReader)

			// Check if we should stop because the server is shutting down
			select {
			case <-ctx.Done():
				return
			default:
			}

			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // Just a read timeout, loop back and check context
				}
				break // Actual error or EOF
			}

			// Reset deadline for the actual handler processing
			_ = tlsConn.SetReadDeadline(time.Time{})

			// Capture the response using a dummy ResponseWriter that writes to the TLS conn
			respWriter := &tlsResponseWriter{conn: tlsConn, header: make(http.Header)}
			mux.ServeHTTP(respWriter, req)

			if req.Close {
				break
			}
		}
	})

	// 6. Start the listener (plain HTTP for the proxy control channel)
	proxySrv := &http.Server{Handler: proxyFunc}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, nil, err
	}

	// Serve until shutdown closes the listener
	go proxySrv.Serve(ln)

	proxyAddr := "http://" + ln.Addr().String()
	shutdown := func() {
		proxySrv.Shutdown(ctx)
	}

	return proxyAddr, caPEM, shutdown, nil
}

// tlsResponseWriter pipes the handler's output back into the hijacked TLS connection.
type tlsResponseWriter struct {
	conn   net.Conn
	header http.Header
	status int
}

func (w *tlsResponseWriter) Header() http.Header { return w.header }
func (w *tlsResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.conn.Write(b)
}
func (w *tlsResponseWriter) WriteHeader(code int) {
	if w.status != 0 {
		return
	}
	w.status = code
	resp := http.Response{
		StatusCode: code,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     w.header,
	}
	_ = resp.Write(w.conn)
}

func (s *Server) TestRunProxy(t interface {
	Context() context.Context
	Fatalf(string, ...interface{})
	Cleanup(func())
}, serverURL string) (string, []byte) {
	proxyURL, caPEM, shutdown, err := s.RunProxy(t.Context(), serverURL)
	if err != nil {
		t.Fatalf("failed to run proxy: %v", err)
		return "", nil // unreachable
	}
	t.Cleanup(shutdown)
	return proxyURL, caPEM
}
