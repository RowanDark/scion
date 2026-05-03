package output

import (
	"fmt"
	"io"
	"time"
)

// Result holds a single discovered domain with optional annotations.
type Result struct {
	Domain   string
	Source   string
	Resolves *bool
	New      *bool
	Wildcard bool
}

// Formatter writes results to an io.Writer.
type Formatter interface {
	Write(w io.Writer, results []Result, target string, timestamp time.Time, meta Meta) error
}

// Meta carries run-level metadata for structured formatters.
type Meta struct {
	SourcesUsed      []string
	WildcardDetected bool
}

var registry = map[string]Formatter{
	"text": &TextFormatter{},
	"json": &JSONFormatter{},
	"csv":  &CSVFormatter{},
	"md":   &MDFormatter{},
}

// Get returns the Formatter for the given format ID.
func Get(format string) (Formatter, error) {
	f, ok := registry[format]
	if !ok {
		return nil, fmt.Errorf("unknown output format %q (available: text, json, csv, md)", format)
	}
	return f, nil
}

func boolStr(b *bool) string {
	if b == nil {
		return ""
	}
	if *b {
		return "true"
	}
	return "false"
}

func boolPtr(b bool) *bool { return &b }
