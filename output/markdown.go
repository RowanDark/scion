package output

import (
	"fmt"
	"io"
	"time"
)

// MDFormatter writes a GitHub-flavored Markdown table.
type MDFormatter struct{}

func (m *MDFormatter) Write(w io.Writer, results []Result, target string, timestamp time.Time, meta Meta) error {
	if _, err := fmt.Fprintln(w, "| Domain | Source | Resolves | New |"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "|--------|--------|----------|-----|"); err != nil {
		return err
	}
	for _, r := range results {
		resolves := ""
		if r.Resolves != nil {
			if *r.Resolves {
				resolves = "✓"
			} else {
				resolves = "✗"
			}
		}
		newMark := "—"
		if r.New != nil && *r.New {
			newMark = "✓"
		}
		if _, err := fmt.Fprintf(w, "| %s | %s | %s | %s |\n", r.Domain, r.Source, resolves, newMark); err != nil {
			return err
		}
	}
	return nil
}
