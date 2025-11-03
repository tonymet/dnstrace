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

	pQueries             []string
	expectedAnswerValues AnswerList
	allStats             rstats
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
	sync.Mutex
}

func init() {
	allStats.codes = make(map[int]int64)
	allStats.hist = hdrhistogram.New(pHistMin.Nanoseconds(), pHistMax.Microseconds(), *pHistPre)
}

func do(ctx context.Context) {
	deadline, _ := ctx.Deadline()
	questions := make([]dns.Question, len(pQueries))
	var (
		wg sync.WaitGroup
		id uint16
	)
	wg.Add(int(*pConcurrency))
	defer func() {
		wg.Wait()
	}()
	qType, ok := dns.StringToType[*pType]
	if !ok {
		panic(fmt.Errorf("Unknown type %q", *pType))
	}
	for i, q := range pQueries {
		questions[i] = dns.Question{Name: dns.Fqdn(q), Qtype: qType, Qclass: dns.ClassINET}
	}
	srv := *pServer
	if !strings.Contains(srv, ":") {
		srv += ":53"
	}
	var m dns.Msg
	m.Question = make([]dns.Question, 1)
	m.RecursionDesired = *pRecurse
	for range *pConcurrency {
		co, err := dns.DialTimeout(*pNetwork, srv, dnsTimeout)
		_ = co.SetWriteDeadline(deadline)
		_ = co.SetReadDeadline(deadline)
		if err != nil {
			atomic.AddInt64(&cerror, 1)
			fmt.Fprintln(os.Stderr, "i/o error dialing: ", err.Error())
		}
		go func() {
			defer func() {
				co.Close()
				wg.Done()
			}()
			select {
			case <-ctx.Done():
				return
			default:
			}
			if udpSize := uint16(*pUdpSize); udpSize > 0 {
				m.SetEdns0(udpSize, true)
				co.UDPSize = udpSize
			}
			for _, q := range questions {
				m.Question[0] = q
				for range *pCount {
					atomic.AddInt64(&count, 1)
					m.Id = id
					id++
					start := time.Now()
					if err = co.WriteMsg(&m); err != nil {
						atomic.AddInt64(&ecount, 1)
						fmt.Fprintln(os.Stderr, "i/o error writing: ", err.Error())
						continue
					}
					r, err := co.ReadMsg()
					if err != nil {
						atomic.AddInt64(&ecount, 1)
						fmt.Fprintln(os.Stderr, "i/o error reading: ", err.Error())
						continue
					}
					timing := time.Since(start)
					allStats.Lock()
					_ = allStats.hist.RecordValue(timing.Nanoseconds())
					allStats.Unlock()
					if r.Rcode == dns.RcodeSuccess {
						if r.Id != m.Id {
							atomic.AddInt64(&mismatch, 1)
							continue
						}
						atomic.AddInt64(&success, 1)
						if *pExpect != "" {
							if len(expectedAnswerValues) != len(r.Answer) {
								fmt.Fprintf(os.Stdout, "expected answer count %d does not equal actual %d\n", len(expectedAnswerValues), len(r.Answer))
								continue
							}
							ok := expectedAnswerValues.Compare(r.Answer)
							if ok {
								atomic.AddInt64(&matched, 1)
								break
							}
						}
					}
					allStats.Lock()
					allStats.codes[r.Rcode]++
					allStats.Unlock()
				}
			}
		}()
	}
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

	if len(expectedAnswerValues) > 0 {
		fmt.Fprint(os.Stdout, "Expected results:\t", amatched, "\n")
	}
}

func printReport(testDuration time.Duration, csv *os.File) {
	if csv != nil {
		writeBars(csv, allStats.hist.Distribution())
		fmt.Println()
		fmt.Println("DNS distribution written to", csv.Name())
	}
	printProgress()
	if len(allStats.codes) > 0 {
		fmt.Println()
		fmt.Println("DNS response codes")
		for i := dns.RcodeSuccess; i <= dns.RcodeBadCookie; i++ {
			if c, ok := allStats.codes[i]; ok {
				fmt.Fprint(os.Stdout, "\t", dns.RcodeToString[i]+":\t", c, "\n")
			}
		}
	}
	fmt.Println()
	fmt.Println("Time taken for tests:\t", testDuration.String())
	fmt.Printf("Questions per second:\t %0.1f\n", float64(count)/testDuration.Seconds())
	min := time.Duration(allStats.hist.Min())
	mean := time.Duration(allStats.hist.Mean())
	sd := time.Duration(allStats.hist.StdDev())
	max := time.Duration(allStats.hist.Max())

	if tc := allStats.hist.TotalCount(); tc > 0 {
		fmt.Println()
		fmt.Println("DNS timings,", tc, "datapoints")
		fmt.Println("\t min:\t\t", min)
		fmt.Println("\t mean:\t\t", mean)
		fmt.Println("\t [+/-sd]:\t", sd)
		fmt.Println("\t max:\t\t", max)

		dist := allStats.hist.Distribution()
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

	if *pExpect != "" {
		var err error
		expectedAnswerValues, err = ScanRecords(*pExpect)
		if err != nil {
			panic(err)
		}
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
	do(ctx)
	testDuration := time.Since(start)
	printReport(testDuration, csvFile)
	if cerror > 0 || ecount > 0 || mismatch > 0 {
		os.Exit(1)
	}
}
