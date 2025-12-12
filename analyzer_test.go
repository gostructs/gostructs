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
	// Test with the main gostructs package itself - should have no unused structs
	unused, err := Analyze([]string{"."})
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if len(unused) > 0 {
		names := make([]string, len(unused))
		for i, u := range unused {
			names[i] = u.Name
		}
		t.Errorf("expected no unused structs in gostructs package, got: %v", names)
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

func TestFindSimilarStructs(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get current file path")
	}
	testdata := filepath.Join(filepath.Dir(filename), "testdata", "similar")

	opts := SimilarityOptions{
		MinFields:          2,
		MinSimilarityScore: 0.5,
	}

	results, err := FindSimilarStructs([]string{testdata}, opts)
	if err != nil {
		t.Fatalf("FindSimilarStructs failed: %v", err)
	}

	// Should find: UserA-UserB, BaseEntity-Product, ConfigA-ConfigB, RequestA-RequestB
	if len(results) < 4 {
		t.Errorf("expected at least 4 similar pairs, got %d", len(results))
	}

	// Verify duplicate detection (UserA-UserB)
	foundDuplicate := false
	for _, r := range results {
		names := []string{r.StructA.Name, r.StructB.Name}
		sort.Strings(names)
		if names[0] == "UserA" && names[1] == "UserB" {
			foundDuplicate = true
			if r.Type != Duplicate {
				t.Errorf("UserA-UserB should be Duplicate, got %v", r.Type)
			}
			if r.Score != 1.0 {
				t.Errorf("UserA-UserB should have score 1.0, got %f", r.Score)
			}
		}
	}
	if !foundDuplicate {
		t.Error("did not find UserA-UserB duplicate pair")
	}
}

func TestSimilarityThreshold(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get current file path")
	}
	testdata := filepath.Join(filepath.Dir(filename), "testdata", "similar")

	// High threshold should find fewer results
	opts := SimilarityOptions{
		MinFields:          2,
		MinSimilarityScore: 0.9,
	}

	results, err := FindSimilarStructs([]string{testdata}, opts)
	if err != nil {
		t.Fatalf("FindSimilarStructs failed: %v", err)
	}

	// Only exact duplicates should pass 0.9 threshold
	for _, r := range results {
		if r.Score < 0.9 {
			t.Errorf("result with score %f should not be included with threshold 0.9", r.Score)
		}
	}
}

func TestMinFieldsFilter(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get current file path")
	}
	testdata := filepath.Join(filepath.Dir(filename), "testdata", "similar")

	opts := SimilarityOptions{
		MinFields:          3,
		MinSimilarityScore: 0.5,
	}

	results, err := FindSimilarStructs([]string{testdata}, opts)
	if err != nil {
		t.Fatalf("FindSimilarStructs failed: %v", err)
	}

	// Single-field structs and 2-field structs should not appear
	for _, r := range results {
		if r.StructA.Name == "Single1" || r.StructB.Name == "Single1" ||
			r.StructA.Name == "Single2" || r.StructB.Name == "Single2" {
			t.Error("single-field structs should be filtered out")
		}
	}
}

func TestSubsetDetection(t *testing.T) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to get current file path")
	}
	testdata := filepath.Join(filepath.Dir(filename), "testdata", "similar")

	opts := SimilarityOptions{
		MinFields:          2,
		MinSimilarityScore: 0.5,
	}

	results, err := FindSimilarStructs([]string{testdata}, opts)
	if err != nil {
		t.Fatalf("FindSimilarStructs failed: %v", err)
	}

	foundSubset := false
	for _, r := range results {
		names := []string{r.StructA.Name, r.StructB.Name}
		sort.Strings(names)
		if names[0] == "BaseEntity" && names[1] == "Product" {
			foundSubset = true
			if r.Type != Subset {
				t.Errorf("BaseEntity-Product should be Subset, got %v", r.Type)
			}
		}
	}
	if !foundSubset {
		t.Error("did not find BaseEntity-Product subset relationship")
	}
}
