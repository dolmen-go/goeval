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
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
)

// Mock allows to mock the [Go Playground] backend server that runs (/compile)
// or stores (/save) Go programs.
//
// [Go Playground]: https://play.golang.org
type Mock struct {
	Compile Compile
	Share   Share
}

// Handler returns an [http.Handler] that calls [m.Compile] or [m.Share].
//
//	POST /compile
//	POST /share
func (m *Mock) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle(m.Compile.MuxPattern(), m.Compile)
	mux.Handle(m.Share.MuxPattern(), m.Share)
	return mux
}

// Compile wraps an custom implementation of the compile-and-run service like https://play.golang.org/compile.
type Compile func(*CompileRequest) (*CompileResponse, error)

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

// MuxPattern returns a pattern for registering [Compile] in a [*net/http.ServeMux].
func (Compile) MuxPattern() string {
	return "POST /compile"
}

// ServeHTTP implements the [net/http.Handler] interface.
func (compile Compile) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	if compile == nil {
		http.Error(w, "Not Implemented", http.StatusNotImplemented)
		return
	}

	resp, err := compile(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// Share wraps an custom implementation of the share service like https://play.golang.org/share.
type Share func(*ShareRequest) (*ShareResponse, error)

type ShareRequest struct {
	Body string
}

type ShareResponse struct {
	ID string
}

// MuxPattern returns a pattern for registering Share in a [*net/http.ServeMux].
func (Share) MuxPattern() string {
	return "POST /share"
}

// ServeHTTP implements the [net/http.Handler] interface.
func (share Share) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	if share == nil {
		http.Error(w, "Not Implemented", http.StatusNotImplemented)
		return
	}

	resp, err := share(&ShareRequest{Body: string(body)})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(resp.ID))
}
