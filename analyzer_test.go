package main

import (
	"path/filepath"
	"runtime"
	"sort"
	"testing"
)

func TestAnalyze(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get current file path")
	}
	testdata := filepath.Join(filepath.Dir(filename), "testdata")

	unused, err := Analyze([]string{testdata})
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	got := make([]string, len(unused))
	for i, u := range unused {
		got[i] = u.Name
	}
	sort.Strings(got)

	want := []string{
		"Embedded",
		"Unused",
		"Unused2",
		"UnusedEmbedder",
		"UsedInUnused",
		"WithMethod",
	}

	if len(got) != len(want) {
		t.Errorf("got %d unused structs, want %d\ngot: %v\nwant: %v", len(got), len(want), got, want)
		return
	}

	for i := range got {
		if got[i] != want[i] {
			t.Errorf("mismatch at index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestAnalyzeNoUnused(t *testing.T) {
	// Test with the main gounused package itself - should have no unused structs
	unused, err := Analyze([]string{"."})
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if len(unused) > 0 {
		names := make([]string, len(unused))
		for i, u := range unused {
			names[i] = u.Name
		}
		t.Errorf("expected no unused structs in gounused package, got: %v", names)
	}
}

func TestNoFalsePositives(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get current file path")
	}
	testdata := filepath.Join(filepath.Dir(filename), "testdata", "used")

	unused, err := Analyze([]string{testdata})
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if len(unused) > 0 {
		names := make([]string, len(unused))
		for i, u := range unused {
			names[i] = u.Name
		}
		t.Errorf("false positives detected - these structs are used but reported as unused: %v", names)
	}
}
