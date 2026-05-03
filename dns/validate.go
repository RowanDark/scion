package dns

import (
	"context"
	"net"
	"sync"
	"time"
)

// ValidateDomains resolves each domain and returns a map of domain->resolves.
func ValidateDomains(domains []string, concurrency int, timeout time.Duration) map[string]bool {
	results := make(map[string]bool, len(domains))
	var mu sync.Mutex

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for _, d := range domains {
		d := d
		sem <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()

			addrs, err := net.DefaultResolver.LookupHost(ctx, d)
			resolves := err == nil && len(addrs) > 0

			mu.Lock()
			results[d] = resolves
			mu.Unlock()
		}()
	}
	wg.Wait()
	return results
}
