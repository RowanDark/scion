package sources

import "context"

// Source is implemented by every enumeration backend.
type Source interface {
	Name() string
	ID() string
	NeedsKey() bool
	IsAvailable() bool
	Run(ctx context.Context, domain string) ([]string, error)
}
