package output

import (
	"encoding/json"
	"io"
	"time"
)

// JSONFormatter writes structured JSON output.
type JSONFormatter struct{}

type jsonResult struct {
	Domain   string  `json:"domain"`
	Source   string  `json:"source"`
	Resolves *bool   `json:"resolves,omitempty"`
	New      *bool   `json:"new,omitempty"`
	Wildcard bool    `json:"wildcard"`
}

type jsonOutput struct {
	Target           string       `json:"target"`
	Timestamp        time.Time    `json:"timestamp"`
	Total            int          `json:"total"`
	SourcesUsed      []string     `json:"sources_used"`
	WildcardDetected bool         `json:"wildcard_detected"`
	Results          []jsonResult `json:"results"`
}

func (j *JSONFormatter) Write(w io.Writer, results []Result, target string, timestamp time.Time, meta Meta) error {
	out := jsonOutput{
		Target:           target,
		Timestamp:        timestamp,
		Total:            len(results),
		SourcesUsed:      meta.SourcesUsed,
		WildcardDetected: meta.WildcardDetected,
		Results:          make([]jsonResult, len(results)),
	}
	if out.SourcesUsed == nil {
		out.SourcesUsed = []string{}
	}
	for i, r := range results {
		out.Results[i] = jsonResult{
			Domain:   r.Domain,
			Source:   r.Source,
			Resolves: r.Resolves,
			New:      r.New,
			Wildcard: r.Wildcard,
		}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
