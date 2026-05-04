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
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
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
