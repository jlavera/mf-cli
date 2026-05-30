package dns

import (
	"fmt"
	"net"

	"github.com/miekg/dns"
)

// Server is a local DNS server that resolves all queries to a fixed IP address.
type Server struct {
	Addr    string // listen address (e.g. "127.0.0.1:5354")
	ReplyIP net.IP // IP to return for all A queries
	server  *dns.Server
}

// NewServer creates a DNS server that responds to all A queries with replyIP.
func NewServer(addr string, replyIP net.IP) *Server {
	return &Server{Addr: addr, ReplyIP: replyIP}
}

// Start runs the DNS server (blocks until stopped or error).
func (s *Server) Start() error {
	mux := dns.NewServeMux()
	mux.HandleFunc(".", s.handleQuery)

	s.server = &dns.Server{
		Addr:    s.Addr,
		Net:     "udp",
		Handler: mux,
	}

	fmt.Printf("DNS server listening on %s (resolving to %s)\n", s.Addr, s.ReplyIP)
	return s.server.ListenAndServe()
}

// Stop gracefully shuts down the DNS server.
func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Shutdown()
	}
	return nil
}

func (s *Server) handleQuery(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	for _, q := range r.Question {
		if q.Qtype == dns.TypeA {
			m.Answer = append(m.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   q.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    60,
				},
				A: s.ReplyIP,
			})
		}
	}

	w.WriteMsg(m)
}
