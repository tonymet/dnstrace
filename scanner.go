package main

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strings"

	"codeberg.org/miekg/dns"
)

// ExpectedAnswer represents a single type:address entry.
type ExpectedAnswer struct {
	Type    string
	Address string
	NetAddr net.IP
}

type AnswerScanner struct {
	*bufio.Scanner
}

func (as AnswerScanner) ReadAnswer() (rec ExpectedAnswer, err error) {
	token := as.Scanner.Bytes()
	if err := rec.UnmarshalText(token); err != nil {
		return ExpectedAnswer{}, fmt.Errorf("scan error on token '%s': %w", token, err)
	}
	return
}

// Unmarshaller that conversts type:Answer into Record
func (r *ExpectedAnswer) UnmarshalText(text []byte) error {
	parts := strings.SplitN(string(text), ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid format '%s'. Expected TYPE:ADDRESS", text)
	}
	r.Type = strings.TrimSpace(parts[0])
	r.Address = strings.TrimSpace(parts[1])
	if ip := net.ParseIP(r.Address); ip != nil {
		r.NetAddr = ip
	} else {
		r.NetAddr = net.IP{}
	}
	return nil
}

// AnswerList is a list of parsed records.
type AnswerList []ExpectedAnswer

// ScanRecords uses bufio.Scanner with a custom Split function
// to read the records from the input string.
func ScanRecords(input string) (AnswerList, error) {
	var list AnswerList = make(AnswerList, 0)
	scanner := bufio.NewScanner(bytes.NewBufferString(input))
	split := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.IndexByte(data, ','); i >= 0 {
			return i + 1, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}
	scanner.Split(split)
	as := AnswerScanner{scanner}
	for as.Scan() {
		if answer, err := as.ReadAnswer(); err == nil {
			list = append(list, answer)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scanner error: %w", err)
	}
	return list, nil
}

func (want AnswerList) Compare(have []dns.RR) (ok bool) {
	if len(want) != len(have) {
		return false
	}
	for j, s := range have {
		expect := want[j]
		switch dns.RRToType(s) {
		case dns.TypeA:
			ok = s.(*dns.A).A.Equal(expect.NetAddr)
			if !ok {
				return
			}
		case dns.TypeAAAA:
			ok = s.(*dns.AAAA).AAAA.Equal(expect.NetAddr)
			if !ok {
				return
			}
		case dns.TypeCNAME:
			ok = s.(*dns.CNAME).Target == expect.Address
			if !ok {
				return
			}
		case dns.TypeTXT:
			ok = strings.Join(s.(*dns.TXT).Txt, "") == expect.Address
			if !ok {
				return
			}
		}
	}
	return true
}
