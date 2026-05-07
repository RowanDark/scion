package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"

	scionColor "github.com/RowanDark/scion/color"
	scionDNS "github.com/RowanDark/scion/dns"
	"github.com/RowanDark/scion/diff"
	"github.com/RowanDark/scion/filter"
	"github.com/RowanDark/scion/output"
	"github.com/RowanDark/scion/sources"
)

var version = "dev" // overridden by -ldflags at build time

const bannerText = ` ___  ___ _  ___  _ __
/ __|/ __| |/ _ \| '_ \
\__ \ (__| | (_) | | | |
|___/\___|_|\___/|_| |_|  v1.0

`

var allSources = []sources.Source{
	&sources.CrtSh{},
	&sources.CertSpotter{},
	&sources.HackerTarget{},
	&sources.Wayback{},
	&sources.RapidDNS{},
	&sources.AlienVault{},
	&sources.LeakIX{},
	&sources.VirusTotal{},
	&sources.SecurityTrails{},
	&sources.Shodan{},
	&sources.URLScanSource{},
	&sources.CensysSource{},
	&sources.GitHubSource{},
	&sources.BufferOverSource{},
	&sources.DNSRepoSource{},
	&sources.FullHuntSource{},
}

func main() {
	os.Exit(run())
}

func run() int {
	var (
		subsOnly       bool
		outputFmt      string
		outFile        string
		timeoutSecs    int
		concurrency    int
		dnsConcurrency int
		silent         bool
		noColor        bool
		verify         bool
		scopeFile      string
		compare        string
		sourcesFlag    string
		listSources    bool
		printVersion   bool
	)

	flag.BoolVar(&subsOnly, "subs-only", false, "Only return subdomains of the target domain")
	flag.StringVar(&outputFmt, "output", "text", "Output format: text, json, csv, md")
	flag.StringVar(&outputFmt, "o", "text", "Output format (shorthand)")
	flag.StringVar(&outFile, "out-file", "", "Write output to file (default: stdout)")
	flag.StringVar(&outFile, "f", "", "Write output to file (shorthand)")
	flag.IntVar(&timeoutSecs, "timeout", 30, "Per-source HTTP timeout in seconds")
	flag.IntVar(&concurrency, "concurrency", 5, "Max concurrent source goroutines")
	flag.IntVar(&dnsConcurrency, "dns-concurrency", 30, "Max concurrent DNS validation goroutines")
	flag.BoolVar(&silent, "silent", false, "Suppress banner, warnings, and status messages")
	flag.BoolVar(&noColor, "no-color", false, "Disable color output (auto-disabled when not a terminal)")
	flag.BoolVar(&verify, "verify", false, "DNS-validate each result")
	flag.StringVar(&scopeFile, "scope-file", "", "Path to scope file for filtering")
	flag.StringVar(&compare, "compare", "", "Path to previous Scion output for diff")
	flag.StringVar(&sourcesFlag, "sources", "", "Comma-separated source IDs to use (default: all available)")
	flag.BoolVar(&listSources, "list-sources", false, "Print all sources and exit")
	flag.BoolVar(&printVersion, "version", false, "Print version and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: scion [flags] <domain>\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if noColor {
		scionColor.Disable()
	}

	if printVersion {
		fmt.Println(version)
		return 0
	}

	if !silent {
		fmt.Fprint(os.Stderr, scionColor.Cyan(bannerText))
	}

	if listSources {
		printSourceTable()
		return 0
	}

	// Collect target domains from positional args or stdin
	var domains []string
	args := flag.Args()
	if len(args) > 0 {
		domains = args
	} else if !term.IsTerminal(int(os.Stdin.Fd())) {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				domains = append(domains, line)
			}
		}
	} else {
		fmt.Fprintln(os.Stderr, "error: no domain provided")
		flag.Usage()
		return 1
	}

	if len(domains) == 0 {
		fmt.Fprintln(os.Stderr, "error: no domain provided")
		flag.Usage()
		return 1
	}

	formatter, err := output.Get(outputFmt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s error: %v\n", scionColor.Red("[scion]"), err)
		return 1
	}

	var w io.Writer = os.Stdout
	if outFile != "" {
		f, err := os.Create(outFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s error: cannot open output file: %v\n", scionColor.Red("[scion]"), err)
			return 1
		}
		defer f.Close()
		w = f
	}

	exitCode := 0
	for _, domain := range domains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain == "" {
			continue
		}
		code := runDomain(domain, w, formatter, sourcesFlag, silent, subsOnly, verify,
			scopeFile, compare, concurrency, dnsConcurrency, timeoutSecs)
		if code > exitCode {
			exitCode = code
		}
	}
	return exitCode
}

type sourceRunResult struct {
	sourceID   string
	sourceName string
	domains    []string
	err        error
	duration   time.Duration
}

func runDomain(
	domain string,
	w io.Writer,
	formatter output.Formatter,
	sourcesFlag string,
	silent, subsOnly, verify bool,
	scopeFile, compare string,
	concurrency, dnsConcurrency, timeoutSecs int,
) int {
	start := time.Now()

	// Build source list
	activeSources := buildSourceList(sourcesFlag, silent)
	if len(activeSources) == 0 {
		fmt.Fprintln(os.Stderr, scionColor.Red("[scion]")+" error: no sources available")
		return 1
	}

	timeout := time.Duration(timeoutSecs) * time.Second

	// Wildcard detection
	wildcardDetected := false
	if verify {
		wc, err := scionDNS.DetectWildcard(domain)
		if err == nil && wc {
			wildcardDetected = true
			if !silent {
				fmt.Fprintf(os.Stderr, "%s Wildcard DNS detected for %s — validation results may be noisy\n",
					scionColor.Yellow("[WARN]"), domain)
			}
		}
	}

	// Channels
	type rawResult struct {
		source string
		domain string
	}
	resultCh := make(chan rawResult, 1000)
	statusCh := make(chan sourceRunResult, len(activeSources))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	// Drain domain results concurrently to prevent deadlock.
	type domainEntry struct {
		source string
	}
	domainMap := make(map[string]domainEntry)
	var domainMapMu sync.Mutex
	var drainWg sync.WaitGroup
	drainWg.Add(1)
	go func() {
		defer drainWg.Done()
		for r := range resultCh {
			d := strings.ToLower(r.domain)
			if d == "" {
				continue
			}
			if subsOnly && !strings.HasSuffix(d, "."+domain) && d != domain {
				continue
			}
			domainMapMu.Lock()
			if _, exists := domainMap[d]; !exists {
				domainMap[d] = domainEntry{source: r.source}
			}
			domainMapMu.Unlock()
		}
	}()

	// Rolling status printer — reads from statusCh and prints as sources finish.
	var allSourceResults []sourceRunResult
	var statusWg sync.WaitGroup
	statusWg.Add(1)
	go func() {
		defer statusWg.Done()
		for sr := range statusCh {
			allSourceResults = append(allSourceResults, sr)
			if silent {
				continue
			}
			var partialErr *sources.PartialResultError
			if sr.err != nil && errors.As(sr.err, &partialErr) {
				count := len(sr.domains)
				fmt.Fprintf(os.Stderr, "%s %-16s %s\n",
					scionColor.Yellow("["+sr.sourceID+"]"),
					scionColor.Yellow("⚠ partial"),
					fmt.Sprintf("— %d result%s (%s)", count, plural(count), partialErr.Reason))
			} else if sr.err != nil {
				fmt.Fprintf(os.Stderr, "%s %-16s %s\n",
					scionColor.Red("["+sr.sourceID+"]"),
					scionColor.Red("✗ error"),
					scionColor.Red("— "+sr.err.Error()))
			} else {
				count := len(sr.domains)
				fmt.Fprintf(os.Stderr, "%s %-16s %s\n",
					scionColor.Green("["+sr.sourceID+"]"),
					scionColor.Green("✓ done"),
					fmt.Sprintf("— %d result%s", count, plural(count)))
			}
		}
	}()

	for _, src := range activeSources {
		src := src
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			srcTimeout := timeout
			if src.DefaultTimeout() > 0 {
				srcTimeout = time.Duration(src.DefaultTimeout()) * time.Second
			}
			ctx, cancel := context.WithTimeout(context.Background(), srcTimeout)
			defer cancel()

			srcStart := time.Now()
			domains, err := src.Run(ctx, domain)
			elapsed := time.Since(srcStart)

			statusCh <- sourceRunResult{
				sourceID:   src.ID(),
				sourceName: src.Name(),
				domains:    domains,
				err:        err,
				duration:   elapsed,
			}

			var partialErr *sources.PartialResultError
			if err != nil && !errors.As(err, &partialErr) {
				return
			}

			for _, d := range domains {
				resultCh <- rawResult{source: src.ID(), domain: d}
			}
		}()
	}

	wg.Wait()
	close(resultCh)
	close(statusCh)
	drainWg.Wait()
	statusWg.Wait()

	// Sort for deterministic output
	allDomains := make([]string, 0, len(domainMap))
	for d := range domainMap {
		allDomains = append(allDomains, d)
	}
	sort.Strings(allDomains)

	// DNS validation
	var resolveMap map[string]bool
	if verify {
		resolveMap = scionDNS.ValidateDomains(allDomains, dnsConcurrency, timeout)
	}

	// Scope filtering
	var scope []string
	var err error
	if scopeFile != "" {
		scope, err = filter.LoadScope(scopeFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s error loading scope file: %v\n", scionColor.Red("[scion]"), err)
			return 1
		}
	}

	// Diff
	var previousResults map[string]bool
	if compare != "" {
		previousResults, err = diff.LoadPreviousResults(compare)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s error loading compare file: %v\n", scionColor.Red("[scion]"), err)
			return 1
		}
	}

	// Build final result list
	sourcesUsed := make([]string, 0)
	sourcesUsedSet := make(map[string]bool)
	var results []output.Result
	for _, d := range allDomains {
		entry := domainMap[d]

		if scopeFile != "" && !filter.MatchesScope(d, scope) {
			continue
		}

		r := output.Result{
			Domain:   d,
			Source:   entry.source,
			Wildcard: wildcardDetected,
		}

		if verify && resolveMap != nil {
			resolves := resolveMap[d]
			r.Resolves = boolPtr(resolves)
		}

		if compare != "" && previousResults != nil {
			isNew := !previousResults[d]
			r.New = boolPtr(isNew)
		}

		results = append(results, r)
		if !sourcesUsedSet[entry.source] {
			sourcesUsedSet[entry.source] = true
			sourcesUsed = append(sourcesUsed, entry.source)
		}
	}

	if len(results) == 0 {
		if !silent {
			fmt.Fprintf(os.Stderr, "%s No results found for %s\n", scionColor.Yellow("[scion]"), domain)
		}
		return 2
	}

	sort.Strings(sourcesUsed)
	meta := output.Meta{
		SourcesUsed:      sourcesUsed,
		WildcardDetected: wildcardDetected,
	}

	if err := formatter.Write(w, results, domain, time.Now().UTC(), meta); err != nil {
		fmt.Fprintf(os.Stderr, "%s error writing output: %v\n", scionColor.Red("[scion]"), err)
		return 1
	}

	if !silent {
		printSummary(results, allSourceResults, time.Since(start))
	}

	return 0
}

func printSummary(results []output.Result, sourceResults []sourceRunResult, elapsed time.Duration) {
	ok := 0
	failed := 0
	partial := 0
	for _, sr := range sourceResults {
		var partialErr *sources.PartialResultError
		if sr.err == nil {
			ok++
		} else if errors.As(sr.err, &partialErr) {
			partial++
		} else {
			failed++
		}
	}

	sep := scionColor.Cyan(strings.Repeat("─", 47))
	fmt.Fprintf(os.Stderr, "\n%s\n", sep)
	sourceLine := fmt.Sprintf("%d queried · %d ok · %d failed", len(sourceResults), ok, failed)
	if partial > 0 {
		sourceLine += fmt.Sprintf(" · %d partial", partial)
	}
	fmt.Fprintf(os.Stderr, "  %-10s %s\n", scionColor.Cyan("Sources"), sourceLine)
	fmt.Fprintf(os.Stderr, "  %-10s %d unique\n",
		scionColor.Cyan("Results"), len(results))
	fmt.Fprintf(os.Stderr, "  %-10s %s\n",
		scionColor.Cyan("Time"), elapsed.Round(time.Millisecond).String())
	fmt.Fprintf(os.Stderr, "%s\n\n", sep)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func buildSourceList(sourcesFlag string, silent bool) []sources.Source {
	if sourcesFlag == "" {
		var active []sources.Source
		skipped := 0
		for _, s := range allSources {
			if s.IsAvailable() {
				active = append(active, s)
			} else {
				if !silent {
					fmt.Fprintf(os.Stderr, "%s skipping %s: key not set\n",
						scionColor.Yellow("[scion]"), scionColor.Yellow(s.Name()))
				}
				skipped++
			}
		}
		if !silent && skipped > 0 {
			fmt.Fprintln(os.Stderr) // spacing after skip messages, before rolling log
		}
		return active
	}

	ids := make(map[string]bool)
	for _, id := range strings.Split(sourcesFlag, ",") {
		ids[strings.TrimSpace(id)] = true
	}

	var active []sources.Source
	skipped := 0
	for _, s := range allSources {
		if ids[s.ID()] {
			if !s.IsAvailable() {
				if !silent {
					fmt.Fprintf(os.Stderr, "%s skipping %s: key not set\n",
						scionColor.Yellow("[scion]"), scionColor.Yellow(s.Name()))
				}
				skipped++
				continue
			}
			active = append(active, s)
		}
	}
	if !silent && skipped > 0 {
		fmt.Fprintln(os.Stderr) // spacing after skip messages, before rolling log
	}
	return active
}

func printSourceTable() {
	fmt.Fprintln(os.Stderr, "Scion — available sources")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintf(os.Stderr, "%-17s%-17s%-17s%s\n", "Source", "ID", "Key Required", "Status")
	fmt.Fprintln(os.Stderr, strings.Repeat("─", 56))
	for _, s := range allSources {
		keyCol := "No"
		if s.NeedsKey() {
			keyCol = keyEnvVar(s.ID())
		}
		status := "✓ ready"
		if !s.IsAvailable() {
			status = "✗ key not set"
		}
		fmt.Fprintf(os.Stderr, "%-17s%-17s%-17s%s\n", s.Name(), s.ID(), keyCol, status)
	}
}

func keyEnvVar(id string) string {
	switch id {
	case "virustotal":
		return "VT_API_KEY"
	case "securitytrails":
		return "ST_API_KEY"
	case "shodan":
		return "SHODAN_API_KEY"
	case "urlscan":
		return "URLSCAN_API_KEY"
	case "leakix":
		return "LEAKIX_API_KEY"
	case "censys":
		return "CENSYS_API_ID/SECRET"
	case "github":
		return "GITHUB_TOKEN"
	case "bufferover":
		return "BUFFEROVER_KEY"
	case "fullhunt":
		return "FULLHUNT_KEY"
	}
	return ""
}

func boolPtr(b bool) *bool { return &b }
