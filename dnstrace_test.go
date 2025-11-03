package main

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/miekg/dns"
)

const serverAddr = "127.0.0.1:4453"

func init() {
	// Start a primitive DNS server for testing
	go startTestDNSServer()
}

func startTestDNSServer() {
	// Listen on UDP port 5353
	udpServer := &dns.Server{Addr: serverAddr, Net: "udp"}
	dns.HandleFunc(".", handleDNSRequest)
	go func() {
		err := udpServer.ListenAndServe()
		if err != nil {
			log.Printf("Failed to start UDP DNS server: %s\n", err.Error())
		}
	}()

	// // Listen on TCP port 5353
	// tcpServer := &dns.Server{Addr: serverAddr, Net: "tcp"}
	// go func() {
	// 	err := tcpServer.ListenAndServe()
	// 	if err != nil {
	// 		log.Printf("Failed to start TCP DNS server: %s\n", err.Error())
	// 	}
	// }()

	log.Println("Test DNS server started on " + serverAddr + " (UDP/TCP)")
}

func handleDNSRequest(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Compress = false

	if r.Opcode == dns.OpcodeQuery {
		for _, q := range r.Question {
			switch q.Qtype {
			case dns.TypeA:
				if q.Name == dns.Fqdn("www.ucla.edu") {
					rr, err := dns.NewRR(fmt.Sprintf("%s A 192.168.1.1", q.Name))
					if err == nil {
						m.Answer = append(m.Answer, rr)
					}
				}
			case dns.TypeAAAA:
				if q.Name == dns.Fqdn("www.ucla.edu") { // Example for AAAA
					rr, err := dns.NewRR(fmt.Sprintf("%s AAAA ::1", q.Name))
					if err == nil {
						m.Answer = append(m.Answer, rr)
					}
				}
			case dns.TypeTXT:
				if q.Name == dns.Fqdn("www.ucla.edu") { // Example for TXT
					rr, err := dns.NewRR(fmt.Sprintf("%s TXT \"hello world\"", q.Name))
					if err == nil {
						m.Answer = append(m.Answer, rr)
					}
				}
			}
		}
	}
	_ = w.WriteMsg(m)
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
			do(context.Background())
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
		*pServer = serverAddr
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
