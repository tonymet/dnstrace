package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/HdrHistogram/hdrhistogram-go"
	"github.com/miekg/dns"
)

var (
	Tag     = ""
	Commit  = ""
	version = "dev (no info)"
)

var (
	pServer      = flag.String("s", "127.0.0.1", "DNS server IP:port to test.")
	pType        = flag.String("type", "A", "Query type. (TXT, A, AAAA)") //TODO: Rest of them pt 1
	pCount       = flag.Int64("n", 1, "Number of queries to issue. Note that the total number of queries issued = number*concurrency*len(queries).")
	pConcurrency = flag.Uint("c", 1, "Number of concurrent queries to issue.")
	pExpect      = flag.String("expect", "", "Expect a specific response (comma-separated).")
	pRecurse     = flag.Bool("recurse", true, "Allow DNS recursion.")
	pUdpSize     = flag.Uint("edns0", 0, "Enable EDNS0 with specified size.")
	pNetwork     = flag.String("network", "udp", "tcp OR udp")
	pVersion     = flag.Bool("version", false, "Print Version")
	pHistMin     = flag.Duration("min", (time.Microsecond * 400), "Minimum value for timing histogram.")
	pHistMax     = flag.Duration("max", dnsTimeout, "Maximum value for histogram.")
	pHistPre     = flag.Int("precision", 1, "Significant figure for histogram precision. [1-5]")
	pHistDisplay = flag.Bool("distribution", true, "Display distribution histogram of timings to stdout.")
	pCsv         = flag.String("csv", "", "Export distribution to CSV. /path/to/file.csv")

	pQueries []string
)

var (
	count    int64
	cerror   int64
	ecount   int64
	success  int64
	matched  int64
	mismatch int64
)

const dnsTimeout = time.Second * 4

type rstats struct {
	codes map[int]int64
	hist  *hdrhistogram.Histogram
}

func isExpected(a string) bool {
	for _, b := range strings.Split(*pExpect, ",") {
		if b == a {
			return true
		}
	}
	return false
}

func do(ctx context.Context) chan rstats {
	stats := make(chan rstats, *pConcurrency)
	go func() {
		questions := make([]string, len(pQueries))
		for i, q := range pQueries {
			questions[i] = dns.Fqdn(q)
		}
		qType, ok := dns.StringToType[*pType]
		if !ok {
			panic(fmt.Errorf("Unknown type %q", *pType))
		}
		srv := *pServer
		if !strings.Contains(srv, ":") {
			srv += ":53"
		}
		var (
			wg sync.WaitGroup
			w  uint
		)
		wg.Add(int(*pConcurrency))
		defer func() {
			wg.Wait()
			close(stats)
		}()
		for w = 0; w < *pConcurrency; w++ {
			co, err := dns.DialTimeout(*pNetwork, srv, dnsTimeout)
			if err != nil {
				atomic.AddInt64(&cerror, 1)
				fmt.Fprintln(os.Stderr, "i/o error dialing: ", err.Error())
			}
			go func() {
				defer func() {
					co.Close()
					wg.Done()
				}()
				st := rstats{hist: hdrhistogram.New(pHistMin.Nanoseconds(), pHistMax.Nanoseconds(), *pHistPre)}
				st.codes = make(map[int]int64)
				var (
					m dns.Msg
					i int64
				)
				for i = 0; i < *pCount; i++ {
					for _, q := range questions {
						var deadline time.Time
						select {
						case <-ctx.Done():
							return
						default:
							deadline, ok = ctx.Deadline()
							if !ok {
								deadline = time.Time{}
							}
						}
						atomic.AddInt64(&count, 1)
						if udpSize := uint16(*pUdpSize); udpSize > 0 {
							m.SetEdns0(udpSize, true)
							co.UDPSize = udpSize
						}
						m.SetQuestion(q, qType)
						m.RecursionDesired = *pRecurse
						start := time.Now()
						if err := co.SetWriteDeadline(deadline); err != nil {
							panic(err)
						}
						if err = co.WriteMsg(&m); err != nil {
							atomic.AddInt64(&ecount, 1)
							fmt.Fprintln(os.Stderr, "i/o error writing: ", err.Error())
							continue
						}
						_ = co.SetReadDeadline(deadline)
						r, err := co.ReadMsg()
						if err != nil {
							atomic.AddInt64(&ecount, 1)
							fmt.Fprintln(os.Stderr, "i/o error reading: ", err.Error())
							continue
						}
						timing := time.Since(start)
						_ = st.hist.RecordValue(timing.Nanoseconds())
						if r.Rcode == dns.RcodeSuccess {
							if r.Id != m.Id {
								atomic.AddInt64(&mismatch, 1)
								continue // Mismatch ID, skip further processing for this response
							}
							atomic.AddInt64(&success, 1)
							if expect := *pExpect; len(expect) > 0 {
								for _, s := range r.Answer {
									ok := false
									switch s.Header().Rrtype {
									//TODO: Rest of them pt 3
									case dns.TypeA:
										a := s.(*dns.A)
										ok = isExpected(a.A.To4().String())

									case dns.TypeAAAA:
										a := s.(*dns.AAAA)
										ok = isExpected(a.AAAA.String())

									case dns.TypeTXT:
										t := s.(*dns.TXT)
										ok = isExpected(strings.Join(t.Txt, ""))
									}

									if ok {
										atomic.AddInt64(&matched, 1)
										break
									}
								}
							}
						}
						st.codes[r.Rcode]++
					}
				}
				stats <- st
			}()
		}
	}()
	return stats
}

func printProgress() {
	var total = uint64(*pCount) * uint64(len(pQueries)) * uint64(*pConcurrency)
	acount := atomic.LoadInt64(&count)
	acerror := atomic.LoadInt64(&cerror)
	aecount := atomic.LoadInt64(&ecount)
	amismatch := atomic.LoadInt64(&mismatch)
	asuccess := atomic.LoadInt64(&success)
	amatched := atomic.LoadInt64(&matched)

	fmt.Printf("Total requests:\t %d of %d (%0.1f%%)\n", acount, total, 100.0*float64(acount)/float64(total))

	if acerror > 0 || aecount > 0 {
		fmt.Fprint(os.Stdout, "Connection errors:\t", acerror, "\n")
		fmt.Fprint(os.Stdout, "Read/Write errors:\t", aecount, "\n")
	}

	if amismatch > 0 {
		fmt.Fprint(os.Stdout, "ID mismatch errors:\t", amismatch, "\n")
	}

	fmt.Fprint(os.Stdout, "DNS success codes:\t", asuccess, "\n")

	if len(strings.Split(*pExpect, ",")) > 0 {
		fmt.Fprint(os.Stdout, "Expected results:\t", amatched, "\n")
	}

}

func printReport(startTime time.Time, stats chan rstats, csv *os.File) {
	timings := hdrhistogram.New(pHistMin.Nanoseconds(), pHistMax.Nanoseconds(), *pHistPre)
	codeTotals := make(map[int]int64)
	for s := range stats {
		timings.Merge(s.hist)
		for k, v := range s.codes {
			codeTotals[k] = codeTotals[k] + v
		}
	}
	testDuration := time.Since(startTime)
	if csv != nil {
		writeBars(csv, timings.Distribution())
		fmt.Println()
		fmt.Println("DNS distribution written to", csv.Name())
	}
	printProgress()
	if len(codeTotals) > 0 {
		fmt.Println()
		fmt.Println("DNS response codes")
		for i := dns.RcodeSuccess; i <= dns.RcodeBadCookie; i++ {
			if c, ok := codeTotals[i]; ok {
				fmt.Fprint(os.Stdout, "\t", dns.RcodeToString[i]+":\t", c, "\n")
			}
		}
	}
	fmt.Println()
	fmt.Println("Time taken for tests:\t", testDuration.String())
	fmt.Printf("Questions per second:\t %0.1f\n", float64(count)/testDuration.Seconds())
	min := time.Duration(timings.Min())
	mean := time.Duration(timings.Mean())
	sd := time.Duration(timings.StdDev())
	max := time.Duration(timings.Max())

	if tc := timings.TotalCount(); tc > 0 {
		fmt.Println()
		fmt.Println("DNS timings,", tc, "datapoints")
		fmt.Println("\t min:\t\t", min)
		fmt.Println("\t mean:\t\t", mean)
		fmt.Println("\t [+/-sd]:\t", sd)
		fmt.Println("\t max:\t\t", max)

		dist := timings.Distribution()
		if *pHistDisplay && tc > 1 {

			fmt.Println()
			fmt.Println("DNS distribution,", tc, "datapoints")

			printBars(dist)
		}
	}
}

func writeBars(f *os.File, bars []hdrhistogram.Bar) {
	_, _ = f.WriteString("From (ns), To (ns), Count\n")
	for _, b := range bars {
		_, _ = f.WriteString(b.String())
	}
}

func printBars(bars []hdrhistogram.Bar) {
	var max int64
	w := tabwriter.NewWriter(os.Stdout, 1, 1, 3, ' ', 0)
	fmt.Fprintln(w, "Latency\tSize\tCount")
	fmt.Fprintln(w, "---\t-----\t---")
	for _, b := range bars {
		if b.Count == 0 {
			continue
		}
		if b.Count > max {
			max = b.Count
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", time.Duration(b.To/2+b.From/2).Round(time.Microsecond).String(),
			makeBar(b.Count, max),
			strconv.FormatInt(b.Count, 10),
		)
	}
	w.Flush()
}

func makeBar(c int64, max int64) string {
	if c == 0 {
		return ""
	}
	t := int((43 * float64(c) / float64(max)) + 0.5)
	return strings.Repeat("▄", t)
}

const fileNoBuffer = 9 // app itself needs about 9 for libs

func main() {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
	flag.Parse()
	pQueries = flag.Args()
	if *pVersion {
		fmt.Printf("Version: %s\n", version)
		return
	}

	if maxFiles, err := GetMaxOpenFiles(); err == nil {
		var needed = uint64(*pConcurrency) + uint64(fileNoBuffer)
		if maxFiles < needed {
			fmt.Fprintf(os.Stderr, "current process limit for number of files is %d and insufficient for level of requested concurrency.\n", maxFiles)
			os.Exit(2)
		}
	}

	var (
		csvFile *os.File
		err     error
	)
	if *pCsv != "" {
		csvFile, err = os.Create(*pCsv)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(2)
		}
		defer csvFile.Close()
	}

	sigsInt := make(chan os.Signal, 8)
	signal.Notify(sigsInt, syscall.SIGINT)
	sigsHup := make(chan os.Signal, 8)
	signal.Notify(sigsHup, syscall.SIGHUP)
	defer close(sigsInt)
	defer close(sigsHup)
	ctx, cancel := context.WithTimeout(context.Background(), dnsTimeout)
	go func() {
		<-sigsInt
		printProgress()
		fmt.Fprintln(os.Stderr, "Cancelling benchmark ^C, again to terminate now.")
		cancel()
		<-sigsInt
		os.Exit(130)
	}()
	go func() {
		for range sigsHup {
			printProgress()
		}
	}()
	start := time.Now()
	res := do(ctx)
	printReport(start, res, csvFile)
	if cerror > 0 || ecount > 0 || mismatch > 0 {
		os.Exit(1)
	}
}
