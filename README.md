# tonymet/DNStrace

⭐Credit to [redsift/dnstrace](https://github.com/redsift/dnstrace) .  This work is a refactor of his design by reducing deps and resources

[![Go Report Card](https://goreportcard.com/badge/github.com/tonymet/dnstrace)](https://goreportcard.com/report/github.com/tonymet/dnstrace)
[![Release](https://img.shields.io/github/release/tonymet/dnstrace/all.svg)](https://github.com/tonymet/dnstrace/releases)
[![Build Status](https://github.com/tonymet/dnstrace/workflows/Release/badge.svg)](https://github.com/tonymet/dnstrace/actions)

DNStrace bypasses OS resolvers and is provided as a Docker packaged prebuilt static binary.
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

### Install

## With Go
`go install github.com/tonymet/dnstrace@latest` will install the binary in your `$GOPATH/bin`.

## From Binary Release
```
curl -LO https://github.com/tonymet/dnstrace/releases/download/v0.2.9/dnstrace_0.2.9_windows_amd64.tar.gz
tar -zxf *tar.gz
sudo install dnstrace /usr/local/bin
```


### Docker

[![Latest](https://images.microbadger.com/badges/version/tonymet/dnstrace.svg)](https://microbadger.com/images/tonymet/dnstrace)

This tool is available in a prebuilt image.

`docker run tonymet/dnstrace --help`

If your local test setup lets you reach 50k QPS and above, you can expect the docker networking to add ~2% overhead to throughput and ~8% to mean latency (tested on Linux Docker 1.12.3).
If this is significant for your purposes you may wish to run with `--net=host`

## C10K and the like

As you approach thousands of concurrent connections on OS-X, you may run into connection errors due to insufficient file handles or threads. This is likely due to process limits so remember to adjust these limits if you intent to increase concurrency levels beyond 1000.

Note that using `sudo ulimit` will create a root shell, adjusts its limits, and then exit causing no real effect. Instead use `launchctl` first on OS-X.

```
$ sudo launchctl limit maxfiles 1000000 1000000
$ ulimit -n 12288
```

## Progress

For long runs, the user can send a SIGHUP via `kill -1 pid` to get the current progress counts.

## Example

```
$ docker run tonymet/dnstrace -n 10 -c 10 --server 8.8.8.8 --recurse tonymet.io

Benchmarking 8.8.8.8:53 via udp with 10 conncurrent requests


Total requests:	 100 of 100 (100.0%)
DNS success codes:     	100

DNS response codes
       	NOERROR:       	100

Time taken for tests:  	 107.091332ms
Questions per second:  	 933.8

DNS timings, 100 datapoints
       	 min:  		 3.145728ms
       	 mean: 		 9.484369ms
       	 [+/-sd]:    5.527339ms
       	 max:  		 27.262975ms

DNS distribution, 100 datapoints
    LATENCY   |                                             | COUNT
+-------------+---------------------------------------------+-------+
  3.276799ms  | ▄▄▄▄▄▄▄▄                                    |     2
  3.538943ms  | ▄▄▄▄▄▄▄▄▄▄▄▄                                |     3
  3.801087ms  | ▄▄▄▄▄▄▄▄▄▄▄▄                                |     3
  4.063231ms  | ▄▄▄▄▄▄▄▄▄▄▄▄                                |     3
  4.325375ms  | ▄▄▄▄▄▄▄▄                                    |     2
  4.587519ms  |                                             |     0
  4.849663ms  |                                             |     0
  5.111807ms  | ▄▄▄▄                                        |     1
  5.373951ms  | ▄▄▄▄                                        |     1
  5.636095ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                            |     4
  5.898239ms  | ▄▄▄▄▄▄▄▄▄▄▄▄                                |     3
  6.160383ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                        |     5
  6.422527ms  | ▄▄▄▄▄▄▄▄▄▄▄▄                                |     3
  6.684671ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                        |     5
  6.946815ms  | ▄▄▄▄▄▄▄▄                                    |     2
  7.208959ms  | ▄▄▄▄                                        |     1
  7.471103ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄         |     9
  7.733247ms  | ▄▄▄▄▄▄▄▄                                    |     2
  7.995391ms  | ▄▄▄▄▄▄▄▄                                    |     2
  8.257535ms  | ▄▄▄▄▄▄▄▄▄▄▄▄                                |     3
  8.650751ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                        |     5
  9.175039ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄ |    11
  9.699327ms  | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                     |     6
  10.223615ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                            |     4
  10.747903ms | ▄▄▄▄                                        |     1
  11.272191ms | ▄▄▄▄                                        |     1
  11.796479ms |                                             |     0
  12.320767ms |                                             |     0
  12.845055ms |                                             |     0
  13.369343ms |                                             |     0
  13.893631ms | ▄▄▄▄                                        |     1
  14.417919ms | ▄▄▄▄                                        |     1
  14.942207ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                        |     5
  15.466495ms |                                             |     0
  15.990783ms | ▄▄▄▄                                        |     1
  16.515071ms |                                             |     0
  17.301503ms |                                             |     0
  18.350079ms |                                             |     0
  19.398655ms | ▄▄▄▄                                        |     1
  20.447231ms | ▄▄▄▄▄▄▄▄                                    |     2
  21.495807ms | ▄▄▄▄                                        |     1
  22.544383ms |                                             |     0
  23.592959ms |                                             |     0
  24.641535ms | ▄▄▄▄                                        |     1
  25.690111ms | ▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄▄                            |     4
  26.738687ms | ▄▄▄▄                                        |     1
```

## Test Server
```
 docker run -p 5353:53/udp `
   -p 5335:53/tcp `
   -v ${PWD}\unbound.conf:/etc/unbound/unbound.conf  `
   alpinelinux/unbound

```