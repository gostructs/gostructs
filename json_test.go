package main

import (
	"bytes"
	"encoding/json"
	"go/token"
	"testing"
)

func TestPositionToLocation(t *testing.T) {
	pos := token.Position{
		Filename: "/path/to/file.go",
		Line:     42,
		Column:   10,
	}

	loc := positionToLocation(pos)

	if loc.File != pos.Filename {
		t.Errorf("File: got %q, want %q", loc.File, pos.Filename)
	}
	if loc.Line != pos.Line {
		t.Errorf("Line: got %d, want %d", loc.Line, pos.Line)
	}
	if loc.Column != pos.Column {
		t.Errorf("Column: got %d, want %d", loc.Column, pos.Column)
	}
}

func TestWriteJSON(t *testing.T) {
	diagnostics := []Diagnostic{
		{
			Type:     DiagUnused,
			Name:     "TestStruct",
			Location: Location{File: "test.go", Line: 10, Column: 1},
			Message:  "struct is unused",
			Severity: "warning",
		},
	}

	var buf bytes.Buffer
	err := WriteJSON(&buf, diagnostics, "1.2.3")
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	var output JSONOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if output.Version != "1.2.3" {
		t.Errorf("Version: got %q, want %q", output.Version, "1.2.3")
	}

	if len(output.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(output.Diagnostics))
	}

	d := output.Diagnostics[0]
	if d.Type != DiagUnused {
		t.Errorf("Type: got %q, want %q", d.Type, DiagUnused)
	}
	if d.Name != "TestStruct" {
		t.Errorf("Name: got %q, want %q", d.Name, "TestStruct")
	}
}

func TestWriteJSONEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := WriteJSON(&buf, []Diagnostic{}, "0.0.1")
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	var output JSONOutput
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if output.Version != "0.0.1" {
		t.Errorf("Version: got %q, want %q", output.Version, "0.0.1")
	}

	if len(output.Diagnostics) != 0 {
		t.Errorf("expected 0 diagnostics, got %d", len(output.Diagnostics))
	}
}

func TestUnusedToDiagnostics(t *testing.T) {
	unused := []UnusedStruct{
		{
			StructLocation: StructLocation{
				Name: "Foo",
				Position: token.Position{
					Filename: "foo.go",
					Line:     15,
					Column:   6,
				},
			},
		},
		{
			StructLocation: StructLocation{
				Name: "Bar",
				Position: token.Position{
					Filename: "bar.go",
					Line:     20,
					Column:   1,
				},
			},
		},
	}

	diagnostics := UnusedToDiagnostics(unused)

	if len(diagnostics) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d", len(diagnostics))
	}

	// Check first diagnostic
	d := diagnostics[0]
	if d.Type != DiagUnused {
		t.Errorf("Type: got %q, want %q", d.Type, DiagUnused)
	}
	if d.Name != "Foo" {
		t.Errorf("Name: got %q, want %q", d.Name, "Foo")
	}
	if d.Location.File != "foo.go" {
		t.Errorf("Location.File: got %q, want %q", d.Location.File, "foo.go")
	}
	if d.Location.Line != 15 {
		t.Errorf("Location.Line: got %d, want %d", d.Location.Line, 15)
	}
	if d.Severity != "warning" {
		t.Errorf("Severity: got %q, want %q", d.Severity, "warning")
	}
}

func TestSimilarityToDiagnostics(t *testing.T) {
	results := []SimilarityResult{
		{
			StructA: StructInfo{
				StructLocation: StructLocation{
					Name: "UserA",
					Position: token.Position{
						Filename: "a.go",
						Line:     10,
						Column:   1,
					},
				},
			},
			StructB: StructInfo{
				StructLocation: StructLocation{
					Name: "UserB",
					Position: token.Position{
						Filename: "b.go",
						Line:     20,
						Column:   1,
					},
				},
			},
			Type:  Duplicate,
			Score: 1.0,
		},
	}

	diagnostics := SimilarityToDiagnostics(results)

	// Each result produces 2 diagnostics (one for each struct)
	if len(diagnostics) != 2 {
		t.Fatalf("expected 2 diagnostics, got %d", len(diagnostics))
	}

	// Check first diagnostic (struct A)
	d := diagnostics[0]
	if d.Type != DiagDuplicate {
		t.Errorf("Type: got %q, want %q", d.Type, DiagDuplicate)
	}
	if d.Name != "UserA" {
		t.Errorf("Name: got %q, want %q", d.Name, "UserA")
	}
	if d.Score != 1.0 {
		t.Errorf("Score: got %f, want %f", d.Score, 1.0)
	}
	if d.Related == nil {
		t.Fatal("Related should not be nil")
	}
	if d.Related.Name != "UserB" {
		t.Errorf("Related.Name: got %q, want %q", d.Related.Name, "UserB")
	}

	// Check second diagnostic (struct B)
	d2 := diagnostics[1]
	if d2.Name != "UserB" {
		t.Errorf("Name: got %q, want %q", d2.Name, "UserB")
	}
	if d2.Related.Name != "UserA" {
		t.Errorf("Related.Name: got %q, want %q", d2.Related.Name, "UserA")
	}
}

func TestSimilarityTypeToDiagType(t *testing.T) {
	tests := []struct {
		input SimilarityType
		want  DiagnosticType
	}{
		{Duplicate, DiagDuplicate},
		{NearDuplicate, DiagNearDuplicate},
		{Subset, DiagSubset},
		{SameNames, DiagSameNames},
		{Mergeable, DiagMergeable},
	}

	for _, tc := range tests {
		got := similarityTypeToDiagType(tc.input)
		if got != tc.want {
			t.Errorf("similarityTypeToDiagType(%v): got %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestSimilarityMessage(t *testing.T) {
	tests := []struct {
		simType SimilarityType
		want    string
	}{
		{Duplicate, "duplicate struct"},
		{NearDuplicate, "near duplicate struct"},
		{Subset, "struct is subset of another"},
		{SameNames, "struct has same field names with different types"},
		{Mergeable, "struct could potentially be merged"},
	}

	for _, tc := range tests {
		r := SimilarityResult{Type: tc.simType}
		got := similarityMessage(r)
		if got != tc.want {
			t.Errorf("similarityMessage(%v): got %q, want %q", tc.simType, got, tc.want)
		}
	}
}
