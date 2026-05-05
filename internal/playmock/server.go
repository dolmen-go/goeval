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
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

// Run launches an HTTP server on a local TCP port.
//
// The returned cleanup function must be called to shutdown the server.
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

// TestRun wraps [Run] for a [testing.T] context.
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

// RunProxy runs an HTTPS server, exposed through a local, ephemeral HTTP proxy.
//
// The proxy URL (localhost, dynamically allocated port, credentials) is returned,
// with a root CA certificate that authenticates the server.
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
	proxyLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, nil, err
	}

	// Serve until shutdown closes the listeners
	go innerServer.Serve(tlsLn)
	go proxySrv.Serve(proxyLn)

	proxyURL = "http://" + proxyAuth + "@" + proxyLn.Addr().String()
	cleanup = func() {
		vLn.Close() // Stop accepting new connections
		innerServer.Shutdown(ctx)
		proxySrv.Shutdown(ctx)
	}

	return proxyURL, caPEM, cleanup, nil
}

func randSerialNumber() *big.Int {
	var b [16]byte // 128 bits
	var n big.Int
	for {
		// Since the API guarantees no error and a full buffer,
		// we can safely ignore the return values.
		_, _ = rand.Read(b[:])
		n.SetBytes(b[:])
		// X.509 serial numbers must be positive.
		// Probability of n being 0 is 1 in 2^128,
		// but we check for formal correctness.
		if n.Sign() == 1 {
			return &n
		}
	}
}

// newCert returns a TLS certificate from an ephemeral CA.
func newCerts(host string, expiresAt time.Time) (certPEM, keyPEM, caPEM []byte, err error) {
	const keyBits = 2048

	// 1. Generate a CA Key and Certificate
	caPrivKey, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return nil, nil, nil, err
	}

	caTemplate := &x509.Certificate{
		SerialNumber: randSerialNumber(),
		Subject: pkix.Name{
			Organization: []string{"Ephemeral Auth Authority"},
			CommonName:   "Ephemeral CA",
		},
		NotBefore:             time.Now().Add(-1 * time.Minute),
		NotAfter:              expiresAt.Add(5 * time.Minute),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, nil, err
	}

	// 2. Generate Server Key and Certificate signed by the CA
	serverPrivKey, err := rsa.GenerateKey(rand.Reader, keyBits)
	if err != nil {
		return nil, nil, nil, err
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: randSerialNumber(),
		Subject: pkix.Name{
			Organization: []string{"Ephemeral Server"},
			CommonName:   host,
		},
		NotBefore:   caTemplate.NotBefore,
		NotAfter:    expiresAt,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}

	// Add the hostname to SANs (Subject Alternative Names)
	if ip := net.ParseIP(host); ip != nil {
		serverTemplate.IPAddresses = []net.IP{ip}
	} else {
		serverTemplate.DNSNames = []string{host}
	}

	serverBytes, err := x509.CreateCertificate(rand.Reader, serverTemplate, caTemplate, &serverPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, nil, err
	}

	// 3. Encode to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: serverBytes,
	})
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(serverPrivKey),
	})
	caPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	return certPEM, keyPEM, caPEM, nil
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

// TestRunProxy wraps [RunProxy] for a [testing.T] context.
func TestRunProxy(t interface {
	Context() context.Context
	Fatalf(string, ...interface{})
	Cleanup(func())
}, serverURL string, h http.Handler) (proxyURL string, caPEM []byte) {
	proxyURL, caPEM, shutdown, err := RunProxy(t.Context(), serverURL, h)
	if err != nil {
		t.Fatalf("failed to run proxy: %v", err)
		return "", nil // unreachable
	}
	t.Cleanup(shutdown)
	return proxyURL, caPEM
}

// ProxyEnv builds environment variables to use to connect to [RunProxy]:
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
