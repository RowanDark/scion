package output

import (
	"fmt"
	"io"
	"time"
)

// TextFormatter writes one domain per line.
type TextFormatter struct{}

func (t *TextFormatter) Write(w io.Writer, results []Result, target string, timestamp time.Time, meta Meta) error {
	for _, r := range results {
		prefix := ""
		if r.New != nil && *r.New {
			prefix = "[NEW] "
		}
		if _, err := fmt.Fprintf(w, "%s%s\n", prefix, r.Domain); err != nil {
			return err
		}
	}
	return nil
}
