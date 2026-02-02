package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/davidsenack/agentbox/internal/config"
)

// Proxy is an HTTP/HTTPS forward proxy with auth injection
type Proxy struct {
	authInjector *AuthInjector
	logger       *Logger
	server       *http.Server
	port         int
}

// New creates a new proxy server
func New(port int, authConfigs []config.AuthConfig, logger *Logger) *Proxy {
	p := &Proxy{
		authInjector: NewAuthInjector(authConfigs),
		logger:       logger,
		port:         port,
	}

	p.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      p,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return p
}

// Start starts the proxy server
func (p *Proxy) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", p.server.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	go func() {
		<-ctx.Done()
		p.server.Close()
	}()

	if err := p.server.Serve(ln); err != http.ErrServerClosed {
		return err
	}
	return nil
}

// ServeHTTP handles HTTP requests
func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		p.handleConnect(w, r)
	} else {
		p.handleHTTP(w, r)
	}
}

// handleConnect handles HTTPS CONNECT tunneling with optional MITM for auth injection
func (p *Proxy) handleConnect(w http.ResponseWriter, r *http.Request) {
	host := r.Host

	hostname, _, err := net.SplitHostPort(host)
	if err != nil {
		hostname = host
	}

	// Check if we need to inject auth for this host
	if p.authInjector.NeedsInjection(hostname) {
		// MITM: terminate TLS and inject auth
		p.handleConnectMITM(w, r, hostname)
		return
	}

	// Standard CONNECT tunneling - no inspection
	targetConn, err := net.DialTimeout("tcp", host, 10*time.Second)
	if err != nil {
		p.logger.LogError(host, err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		targetConn.Close()
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		targetConn.Close()
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	p.logger.LogPass(host, r.RemoteAddr)

	// Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(targetConn, clientConn)
		targetConn.Close()
	}()
	go func() {
		defer wg.Done()
		io.Copy(clientConn, targetConn)
		clientConn.Close()
	}()
	wg.Wait()
}

// handleConnectMITM handles HTTPS for hosts that need auth injection
// Currently falls back to passthrough - HTTPS auth injection requires CA setup
func (p *Proxy) handleConnectMITM(w http.ResponseWriter, r *http.Request, hostname string) {
	// Log that we can't inject auth for HTTPS without CA
	p.logger.LogAuthSkipped(hostname, "HTTPS requires CA for auth injection - passing through")

	// Fall back to standard tunnel (no auth injection possible)
	targetConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		p.logger.LogError(r.Host, err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		targetConn.Close()
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		targetConn.Close()
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Bidirectional tunnel
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(targetConn, clientConn)
		targetConn.Close()
	}()
	go func() {
		defer wg.Done()
		io.Copy(clientConn, targetConn)
		clientConn.Close()
	}()
	wg.Wait()
}

// handleHTTP handles plain HTTP proxy requests with auth injection
func (p *Proxy) handleHTTP(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Host
	if host == "" {
		host = r.Host
	}

	hostname, _, err := net.SplitHostPort(host)
	if err != nil {
		hostname = host
	}

	// Create outgoing request
	outReq := r.Clone(r.Context())
	outReq.RequestURI = ""

	// Remove hop-by-hop headers
	removeHopHeaders(outReq.Header)

	// Inject authentication if configured
	if injected := p.authInjector.Inject(hostname, outReq.Header); injected {
		p.logger.LogAuthInjected(host, r.RemoteAddr)
	}

	// Forward the request
	resp, err := http.DefaultTransport.RoundTrip(outReq)
	if err != nil {
		p.logger.LogError(host, err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	removeHopHeaders(resp.Header)
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	// Copy status code and body
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)

	p.logger.LogPass(host, r.RemoteAddr)
}

var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te",
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

func removeHopHeaders(h http.Header) {
	for _, hh := range hopHeaders {
		h.Del(hh)
	}
}
