package sources

import "context"

// Source is implemented by every enumeration backend.
type Source interface {
	Name() string
	ID() string
	NeedsKey() bool
	IsAvailable() bool
	DefaultTimeout() int // 0 = use global --timeout value
	Run(ctx context.Context, domain string) ([]string, error)
}

// PartialResultError indicates results were collected but the source stopped
// early due to a recoverable condition (secondary rate limit, timeout, etc.)
type PartialResultError struct {
	Reason string
}

func (e *PartialResultError) Error() string {
	return e.Reason
}
