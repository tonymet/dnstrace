module github.com/redsift/dnstrace

go 1.12

require (
	github.com/HdrHistogram/hdrhistogram-go v1.1.0 // indirect
	github.com/alecthomas/kingpin v2.2.5+incompatible
	github.com/alecthomas/template v0.0.0-20160405071501-a0175ee3bccc // indirect
	github.com/alecthomas/units v0.0.0-20151022065526-2efee857e7cf // indirect
	github.com/fatih/color v1.5.0
	github.com/mattn/go-colorable v0.0.8-0.20170210172801-5411d3eea597 // indirect
	github.com/mattn/go-isatty v0.0.2-0.20170307163044-57fdcb988a5c // indirect
	github.com/mattn/go-runewidth v0.0.2 // indirect
	github.com/miekg/dns v0.0.0-20170812192144-0598bd43cf51
	github.com/olekukonko/tablewriter v0.0.0-20170719101040-be5337e7b39e
	go.uber.org/ratelimit v0.0.0-20161026005643-d15fa2e2a63d
)

replace github.com/codahale/hdrhistogram v0.0.0 => github.com/HdrHistogram/hdrhistogram-go v1.1.0
