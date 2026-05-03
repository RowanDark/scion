package output

import (
	"encoding/csv"
	"io"
	"time"
)

// CSVFormatter writes comma-separated values with a header row.
type CSVFormatter struct{}

func (c *CSVFormatter) Write(w io.Writer, results []Result, target string, timestamp time.Time, meta Meta) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"domain", "source", "resolves", "new", "wildcard"}); err != nil {
		return err
	}
	for _, r := range results {
		wildcard := "false"
		if r.Wildcard {
			wildcard = "true"
		}
		if err := cw.Write([]string{
			r.Domain,
			r.Source,
			boolStr(r.Resolves),
			boolStr(r.New),
			wildcard,
		}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}
