package main

import (
	"reflect"
	"sort"
	"testing"
)

func TestFilterUnresolved(t *testing.T) {
	domains := []string{
		"resolves-a.example.com",
		"dead-a.example.com",
		"resolves-b.example.com",
		"dead-b.example.com",
		"resolves-c.example.com",
	}
	resolveMap := map[string]bool{
		"resolves-a.example.com": true,
		"dead-a.example.com":     false,
		"resolves-b.example.com": true,
		"dead-b.example.com":     false,
		"resolves-c.example.com": true,
	}

	got, unresolvedCount := filterUnresolved(domains, resolveMap)

	want := []string{
		"resolves-a.example.com",
		"resolves-b.example.com",
		"resolves-c.example.com",
	}
	sort.Strings(got)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("filterUnresolved() domains = %v, want %v", got, want)
	}
	if unresolvedCount != 2 {
		t.Errorf("filterUnresolved() unresolvedCount = %d, want 2", unresolvedCount)
	}
}

func TestFilterUnresolvedAllResolve(t *testing.T) {
	domains := []string{"a.example.com", "b.example.com"}
	resolveMap := map[string]bool{"a.example.com": true, "b.example.com": true}

	got, unresolvedCount := filterUnresolved(domains, resolveMap)
	if !reflect.DeepEqual(got, domains) {
		t.Errorf("filterUnresolved() domains = %v, want %v", got, domains)
	}
	if unresolvedCount != 0 {
		t.Errorf("filterUnresolved() unresolvedCount = %d, want 0", unresolvedCount)
	}
}

func TestFilterUnresolvedNoneResolve(t *testing.T) {
	domains := []string{"a.example.com", "b.example.com"}
	resolveMap := map[string]bool{"a.example.com": false, "b.example.com": false}

	got, unresolvedCount := filterUnresolved(domains, resolveMap)
	if len(got) != 0 {
		t.Errorf("filterUnresolved() domains = %v, want empty", got)
	}
	if unresolvedCount != 2 {
		t.Errorf("filterUnresolved() unresolvedCount = %d, want 2", unresolvedCount)
	}
}

func TestBuildSourceListUnknownSourceIsSkipped(t *testing.T) {
	// A source ID no longer in the registry (e.g. named in an old user
	// config after a source was removed) must never panic or fail — it
	// should just be warned about and skipped.
	active := buildSourceList("removed-source-1,removed-source-2,not-a-real-source", true)
	if len(active) != 0 {
		t.Errorf("buildSourceList() with only unknown source IDs = %d sources, want 0", len(active))
	}
}

func TestBuildSourceListKnownSourceStillWorks(t *testing.T) {
	active := buildSourceList("crtsh", true)
	if len(active) != 1 {
		t.Fatalf("buildSourceList(%q) = %d sources, want 1", "crtsh", len(active))
	}
	if active[0].ID() != "crtsh" {
		t.Errorf("buildSourceList(%q) source ID = %q, want %q", "crtsh", active[0].ID(), "crtsh")
	}
}
