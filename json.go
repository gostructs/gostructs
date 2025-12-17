package main

import (
	"encoding/json"
	"go/token"
	"io"
)

// DiagnosticType represents the type of diagnostic
type DiagnosticType string

const (
	DiagUnused        DiagnosticType = "unused"
	DiagDuplicate     DiagnosticType = "duplicate"
	DiagNearDuplicate DiagnosticType = "near_duplicate"
	DiagSubset        DiagnosticType = "subset"
	DiagSameNames     DiagnosticType = "same_names"
	DiagMergeable     DiagnosticType = "mergeable"
	DiagTagIssue      DiagnosticType = "tag_issue"
)

// Location represents a source code location
type Location struct {
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// positionToLocation converts a token.Position to Location
func positionToLocation(pos token.Position) Location {
	return Location{
		File:   pos.Filename,
		Line:   pos.Line,
		Column: pos.Column,
	}
}

// RelatedStruct holds information about a related struct (for duplicates)
type RelatedStruct struct {
	Name     string   `json:"name"`
	Location Location `json:"location"`
}

// Diagnostic represents a single diagnostic message
type Diagnostic struct {
	Type     DiagnosticType `json:"type"`
	Name     string         `json:"name"`
	Location Location       `json:"location"`
	Message  string         `json:"message"`
	Severity string         `json:"severity"`
	// For similarity diagnostics
	Score   float64        `json:"score,omitempty"`
	Related *RelatedStruct `json:"related,omitempty"`
}

// JSONOutput holds all diagnostics
type JSONOutput struct {
	Version     string       `json:"version"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// WriteJSON writes diagnostics as JSON
func WriteJSON(w io.Writer, diagnostics []Diagnostic, version string) error {
	output := JSONOutput{
		Version:     version,
		Diagnostics: diagnostics,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

// UnusedToDiagnostics converts unused structs to diagnostics
func UnusedToDiagnostics(unused []UnusedStruct) []Diagnostic {
	diagnostics := make([]Diagnostic, 0, len(unused))
	for _, u := range unused {
		diagnostics = append(diagnostics, Diagnostic{
			Type:     DiagUnused,
			Name:     u.Name,
			Location: positionToLocation(u.Position),
			Message:  "struct is unused",
			Severity: "warning",
		})
	}
	return diagnostics
}

// SimilarityToDiagnostics converts similarity results to diagnostics
func SimilarityToDiagnostics(results []SimilarityResult) []Diagnostic {
	diagnostics := make([]Diagnostic, 0, len(results)*2)

	for _, r := range results {
		diagType := similarityTypeToDiagType(r.Type)
		message := similarityMessage(r)

		// Create diagnostic for struct A
		diagnostics = append(diagnostics, Diagnostic{
			Type:     diagType,
			Name:     r.StructA.Name,
			Location: positionToLocation(r.StructA.Position),
			Message:  message,
			Severity: "warning",
			Score:    r.Score,
			Related: &RelatedStruct{
				Name:     r.StructB.Name,
				Location: positionToLocation(r.StructB.Position),
			},
		})

		// Create diagnostic for struct B
		diagnostics = append(diagnostics, Diagnostic{
			Type:     diagType,
			Name:     r.StructB.Name,
			Location: positionToLocation(r.StructB.Position),
			Message:  message,
			Severity: "warning",
			Score:    r.Score,
			Related: &RelatedStruct{
				Name:     r.StructA.Name,
				Location: positionToLocation(r.StructA.Position),
			},
		})
	}

	return diagnostics
}

func similarityTypeToDiagType(t SimilarityType) DiagnosticType {
	switch t {
	case Duplicate:
		return DiagDuplicate
	case NearDuplicate:
		return DiagNearDuplicate
	case Subset:
		return DiagSubset
	case SameNames:
		return DiagSameNames
	case Mergeable:
		return DiagMergeable
	default:
		return DiagMergeable
	}
}

func similarityMessage(r SimilarityResult) string {
	switch r.Type {
	case Duplicate:
		return "duplicate struct"
	case NearDuplicate:
		return "near duplicate struct"
	case Subset:
		return "struct is subset of another"
	case SameNames:
		return "struct has same field names with different types"
	case Mergeable:
		return "struct could potentially be merged"
	default:
		return "similar struct detected"
	}
}

// TagsToDiagnostics converts tag issues to diagnostics
func TagsToDiagnostics(issues []TagIssue) []Diagnostic {
	diagnostics := make([]Diagnostic, 0, len(issues))
	for _, issue := range issues {
		diagnostics = append(diagnostics, Diagnostic{
			Type:     DiagTagIssue,
			Name:     issue.StructName + "." + issue.FieldName,
			Location: positionToLocation(issue.Position),
			Message:  issue.Message,
			Severity: "warning",
		})
	}
	return diagnostics
}
