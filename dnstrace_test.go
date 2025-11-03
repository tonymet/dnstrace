package main

import (
	"context"
	"io"
	"log"
	"net"
	"testing"

	"codeberg.org/miekg/dns"
)

const serverAddr = "127.0.0.1:4454"

func init() {
	startTestDNSServer()
}

func startTestDNSServer() {
	dns.HandleFunc(".", handleDNSRequest)
	udpServer := &dns.Server{Addr: serverAddr, Net: "udp", ReusePort: true, MaxTCPQueries: -1}
	go func() {
		if err := udpServer.ListenAndServe(); err != nil {
			log.Printf("Failed to setup the server: %s", err.Error())
		}
	}()
	log.Println("Test DNS server started on " + udpServer.Addr + " (UDP)")
}

func handleDNSRequest(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	r.Response = true
	if r.Opcode == dns.OpcodeQuery {
		for _, q := range r.Question {
			switch dns.RRToType(q) {
			case dns.TypeA:
				if q.Header().Name == "www.ucla.edu." {
					rr := &dns.A{Hdr: dns.Header{Name: "www.ucla.edu.", Class: dns.ClassINET},
						A: net.IPv4zero}
					m.Answer = append(m.Answer, rr)
				}
			case dns.TypeAAAA:
				if q.Header().Name == "www.ucla.edu." { // Example for AAAA
					rr := &dns.A{Hdr: dns.Header{Name: "www.ucla.edu.", Class: dns.ClassINET},
						A: net.IPv6zero}
					m.Answer = append(m.Answer, rr)
				}
			}
		}
	}
	_, _ = io.Copy(w, m)
}

func Test_do(t *testing.T) {
	tests := []struct {
		name         string // description of this test case
		wantCount    int64
		pCount       int64
		pConcurrency uint
		query        string
	}{
		{"www.ucla.edu", 20, 5, 4, "www.ucla.edu"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pCount = &tt.pCount
			pConcurrency = &tt.pConcurrency
			*pServer = serverAddr
			pQueries = []string{tt.query}
			//do(context.Background())
			do(t.Context())
			got := countResults()
			if tt.wantCount != got {
				t.Errorf("got = %v, want %v", got, tt.wantCount)
			}
		})
	}
}

func Benchmark_do(b *testing.B) {
	tests := []struct {
		name         string // description of this test case
		pCount       int64
		pConcurrency uint
		query        string
	}{
		{"www.ucla.edu", 5, 4, "www.ucla.edu"},
	}
	tt := tests[0]
	for range b.N {
		pCount = &tt.pCount
		pConcurrency = &tt.pConcurrency
		//*pServer = serverAddr
		*pServer = "127.0.0.1:5353"
		pQueries = []string{tt.query}
		do(context.Background())
	}
}

func countResults() (c int64) {
	for _, v := range allStats.codes {
		c += v
	}
	return
}
