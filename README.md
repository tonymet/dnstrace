# tonymet/DNStrace

⭐Credit to [redsift/dnstrace](https://github.com/redsift/dnstrace) .  This work is a refactor of his design by reducing deps and resources

[![Go Report Card](https://goreportcard.com/badge/github.com/tonymet/dnstrace)](https://goreportcard.com/report/github.com/tonymet/dnstrace)
[![Release](https://img.shields.io/github/release/tonymet/dnstrace/all.svg)](https://github.com/tonymet/dnstrace/releases)
[![Build Status](https://github.com/tonymet/dnstrace/workflows/Release/badge.svg)](https://github.com/tonymet/dnstrace/actions)

Concurrent DNS resolver to benchmark and load-test DNS servers.
Basic latency measurement, result checking and histograms are supported.
Currently, only `A`, `AAAA` and `TXT` questions are supported.
Do not use on public DNS servers

## Usage

```
$ dnstrace -h
usage: dnstrace [<flags>] <queries>...
A DNS benchmark.
 -c uint
        Number of concurrent queries to issue. (default 1)
  -csv string
        Export distribution to CSV. /path/to/file.csv
  -distribution
        Display distribution histogram of timings to stdout. (default true)
  -edns0 uint
        Enable EDNS0 with specified size.
  -expect string
        Expect a specific response (comma-separated).
  -max duration
        Maximum value for histogram. (default 4s)
  -min duration
        Minimum value for timing histogram. (default 400µs)
  -n int
        Number of queries to issue. Note that the total number of queries issued = number*concurrency*len(queries). (default 1)
  -precision int
        Significant figure for histogram precision. [1-5] (default 1)
  -read duration
        DNS read timeout. (default 4s)
  -recurse
        Allow DNS recursion. (default true)
  -s string
        DNS server IP:port to test. (default "127.0.0.1")
  -tcp
        Use TCP fot DNS requests.
  -type string
        Query type. (TXT, A, AAAA) (default "A")
  -version
        Print Version
  -write duration
        DNS write timeout. (default 1s)

Args:
  <queries>  Queries to issue.
```

## Installation

## With Go
`go install github.com/tonymet/dnstrace@latest` will install the binary in your `$GOPATH/bin`.

## From Binary Release
```
curl -LO https://github.com/tonymet/dnstrace/releases/download/v0.2.9/dnstrace_0.2.9_windows_amd64.tar.gz
tar -zxf *tar.gz
sudo install dnstrace /usr/local/bin
```

## Progress

For long runs, the user can send a SIGHUP via `kill -1 pid` to get the current progress counts.

## Example
```
 dnstrace -n 10 -c 10 -s 127.0.0.1:5353 www.ucla.edu
Total requests:  100 of 100 (100.0%)
DNS success codes:      100
Expected results:       0

DNS response codes
        NOERROR:        100

Time taken for tests:    6.4319ms
Questions per second:    15547.5

DNS timings, 100 datapoints
         min:            0s
         mean:           600.309µs
         [+/-sd]:        373.12µs
         max:            1.835007ms

DNS distribution, 100 datapoints
Latency                                                 Count
131µs     ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                           22
393µs     ▄▄▄▄▄▄▄▄▄▄▄▄                                  14
655µs     ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄   49
918µs     ▄                                             1
1.18ms    ▄▄▄▄▄▄▄▄▄                                     10
```

## Test Server
```
 docker run -p 5353:53/udp `
   -p 5335:53/tcp `
   -v ${PWD}\unbound.conf:/etc/unbound/unbound.conf  `
   alpinelinux/unbound

```