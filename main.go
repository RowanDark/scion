package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"

	scionDNS "github.com/RowanDark/scion/dns"
	"github.com/RowanDark/scion/diff"
	"github.com/RowanDark/scion/filter"
	"github.com/RowanDark/scion/output"
	"github.com/RowanDark/scion/sources"
)

var version = "dev" // overridden by -ldflags at build time

const banner = `
 ___  ___ _  ___  _ __
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
	&sources.Facebook{},
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

	if printVersion {
		fmt.Println(version)
		return 0
	}

	if !silent {
		fmt.Fprint(os.Stderr, banner)
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
		fmt.Fprintf(os.Stderr, "[scion] error: %v\n", err)
		return 1
	}

	var w io.Writer = os.Stdout
	if outFile != "" {
		f, err := os.Create(outFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[scion] error: cannot open output file: %v\n", err)
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

func runDomain(
	domain string,
	w io.Writer,
	formatter output.Formatter,
	sourcesFlag string,
	silent, subsOnly, verify bool,
	scopeFile, compare string,
	concurrency, dnsConcurrency, timeoutSecs int,
) int {
	// Build source list
	activeSources := buildSourceList(sourcesFlag, silent)
	if len(activeSources) == 0 {
		fmt.Fprintln(os.Stderr, "[scion] error: no sources available")
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
				fmt.Fprintf(os.Stderr, "[WARN] Wildcard DNS detected for %s — validation results may be noisy\n", domain)
			}
		}
	}

	// Run sources concurrently
	type sourceResult struct {
		source string
		domain string
	}
	resultCh := make(chan sourceResult, 1000)
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	sourcesUsed := make([]string, 0, len(activeSources))
	var sourcesMu sync.Mutex

	// Drain resultCh in a dedicated goroutine that starts before any source is
	// launched. This prevents deadlock: without concurrent draining, the channel
	// buffer can fill while the main goroutine is still blocked on the semaphore
	// launching sources, causing source goroutines to block on send and never
	// release the semaphore — a classic channel/semaphore deadlock.
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

			domains, err := src.Run(ctx, domain)
			if err != nil {
				if !silent {
					fmt.Fprintf(os.Stderr, "[scion] %s: %v\n", src.ID(), err)
				}
				return
			}

			sourcesMu.Lock()
			sourcesUsed = append(sourcesUsed, src.ID())
			sourcesMu.Unlock()

			for _, d := range domains {
				resultCh <- sourceResult{source: src.ID(), domain: d}
			}
		}()
	}

	// Phase 1 complete: all sources launched. Wait for every source goroutine to
	// finish, close the channel, then wait for the drain goroutine to flush all
	// remaining results. Post-processing (validation, filtering, diff) only begins
	// after drainWg.Wait() returns, guaranteeing a complete result set.
	wg.Wait()
	close(resultCh)
	drainWg.Wait()

	// Sort for deterministic output
	allDomains := make([]string, 0, len(domainMap))
	for d := range domainMap {
		allDomains = append(allDomains, d)
	}
	sort.Strings(allDomains)

	// DNS validation uses dedicated concurrency setting
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
			fmt.Fprintf(os.Stderr, "[scion] error loading scope file: %v\n", err)
			return 1
		}
	}

	// Diff
	var previousResults map[string]bool
	if compare != "" {
		previousResults, err = diff.LoadPreviousResults(compare)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[scion] error loading compare file: %v\n", err)
			return 1
		}
	}

	// Build final result list
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
	}

	if len(results) == 0 {
		if !silent {
			fmt.Fprintf(os.Stderr, "[scion] No results found for %s\n", domain)
		}
		return 2
	}

	sort.Strings(sourcesUsed)
	meta := output.Meta{
		SourcesUsed:      sourcesUsed,
		WildcardDetected: wildcardDetected,
	}

	if err := formatter.Write(w, results, domain, time.Now().UTC(), meta); err != nil {
		fmt.Fprintf(os.Stderr, "[scion] error writing output: %v\n", err)
		return 1
	}

	return 0
}

func buildSourceList(sourcesFlag string, silent bool) []sources.Source {
	if sourcesFlag == "" {
		var active []sources.Source
		for _, s := range allSources {
			if s.IsAvailable() {
				active = append(active, s)
			} else if !silent {
				fmt.Fprintf(os.Stderr, "[scion] skipping %s: key not set\n", s.Name())
			}
		}
		return active
	}

	ids := make(map[string]bool)
	for _, id := range strings.Split(sourcesFlag, ",") {
		ids[strings.TrimSpace(id)] = true
	}

	var active []sources.Source
	for _, s := range allSources {
		if ids[s.ID()] {
			if !s.IsAvailable() {
				if !silent {
					fmt.Fprintf(os.Stderr, "[scion] skipping %s: key not set\n", s.Name())
				}
				continue
			}
			active = append(active, s)
		}
	}
	return active
}

func printSourceTable() {
	fmt.Println("Scion — available sources")
	fmt.Println()
	fmt.Printf("%-17s%-17s%-17s%s\n", "Source", "ID", "Key Required", "Status")
	fmt.Println(strings.Repeat("─", 56))
	for _, s := range allSources {
		keyCol := "No"
		if s.NeedsKey() {
			keyCol = keyEnvVar(s.ID())
		}
		status := "✓ ready"
		if !s.IsAvailable() {
			status = "✗ key not set"
		}
		fmt.Printf("%-17s%-17s%-17s%s\n", s.Name(), s.ID(), keyCol, status)
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
	case "facebook":
		return "FB_APP_ID/SECRET"
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
