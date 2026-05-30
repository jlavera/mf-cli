package proxy

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

// reloadInterval is how often the proxy re-reads the routes directory. Polling
// is used instead of signals because `mf up` runs as the user while the proxy
// runs as root, so a cross-privilege SIGHUP would be denied. A short interval
// keeps newly started projects reachable quickly.
const reloadInterval = 2 * time.Second

// Server is a reverse proxy that routes requests by Host header.
type Server struct {
	HTTPAddr  string // e.g. ":80"
	HTTPSAddr string // e.g. ":443"
	CA        *CA

	mu      sync.RWMutex
	routes  map[string]string // hostname → target URL
	lastSig string            // signature of the last loaded route set
}

// NewServer creates a proxy server.
func NewServer(httpAddr, httpsAddr string, ca *CA) *Server {
	return &Server{
		HTTPAddr:  httpAddr,
		HTTPSAddr: httpsAddr,
		CA:        ca,
		routes:    make(map[string]string),
	}
}

// Start runs both HTTP and HTTPS listeners and keeps routes fresh by polling
// the routes directory (also reloads on SIGHUP for manual use).
func (s *Server) Start() error {
	if err := s.reloadRoutes(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load routes: %v\n", err)
	}

	go s.watchRoutes()

	handler := http.HandlerFunc(s.handleRequest)
	errCh := make(chan error, 2)

	go func() {
		fmt.Printf("HTTP proxy listening on %s\n", s.HTTPAddr)
		errCh <- http.ListenAndServe(s.HTTPAddr, handler)
	}()

	if s.CA != nil {
		go func() {
			server := &http.Server{
				Addr:      s.HTTPSAddr,
				Handler:   handler,
				TLSConfig: &tls.Config{GetCertificate: s.CA.GetCertificate},
			}
			fmt.Printf("HTTPS proxy listening on %s\n", s.HTTPSAddr)
			errCh <- server.ListenAndServeTLS("", "")
		}()
	}

	return <-errCh
}

// watchRoutes polls the routes directory and reloads on SIGHUP.
func (s *Server) watchRoutes() {
	sighup := make(chan os.Signal, 1)
	signal.Notify(sighup, syscall.SIGHUP)

	ticker := time.NewTicker(reloadInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
		case <-sighup:
		}
		if err := s.reloadRoutes(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: route reload failed: %v\n", err)
		}
	}
}

func (s *Server) reloadRoutes() error {
	routes, err := LoadAllRoutes()
	if err != nil {
		return err
	}
	sig := routesSignature(routes)

	s.mu.Lock()
	changed := sig != s.lastSig
	s.routes = routes
	s.lastSig = sig
	s.mu.Unlock()

	if changed {
		fmt.Printf("Loaded %d routes\n", len(routes))
	}
	return nil
}

// routesSignature returns a stable string representation of a route set so we
// can detect changes and avoid logging on every poll.
func routesSignature(routes map[string]string) string {
	keys := make([]string, 0, len(routes))
	for k := range routes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(routes[k])
		b.WriteByte(';')
	}
	return b.String()
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	hostname := r.Host
	if h, _, err := net.SplitHostPort(hostname); err == nil {
		hostname = h
	}

	s.mu.RLock()
	target, ok := s.routes[hostname]
	s.mu.RUnlock()

	if !ok {
		http.Error(w, fmt.Sprintf("no route for host: %s", hostname), http.StatusBadGateway)
		return
	}

	targetURL, err := url.Parse(target)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid target URL: %s", target), http.StatusInternalServerError)
		return
	}

	rp := httputil.NewSingleHostReverseProxy(targetURL)
	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		http.Error(w, fmt.Sprintf("proxy error for %s → %s: %v", hostname, target, err), http.StatusBadGateway)
	}
	rp.ServeHTTP(w, r)
}
