package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"golang.org/x/term"
)

var version = "0.0.3"

// ANSI color codes
const (
	colorReset   = "\033[0m"
	colorYellow  = "\033[33m"
	colorBold    = "\033[1m"
	colorGreen   = "\033[32m"
	colorMagenta = "\033[35m"
	colorDim     = "\033[2m"
)

func useColors() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func main() {
	showVersion := flag.Bool("v", false, "print version")
	flag.BoolVar(showVersion, "version", false, "print version")

	findUnused := flag.Bool("u", false, "find unused structs")
	flag.BoolVar(findUnused, "unused", false, "find unused structs")

	findDuplicate := flag.Bool("d", false, "find duplicate/similar structs")
	flag.BoolVar(findDuplicate, "duplicate", false, "find duplicate/similar structs")

	jsonOutput := flag.Bool("j", false, "output in JSON format")
	flag.BoolVar(jsonOutput, "json", false, "output in JSON format")

	findTags := flag.Bool("t", false, "validate struct tags")
	flag.BoolVar(findTags, "tags", false, "validate struct tags")

	warnOnly := flag.Bool("w", false, "warn only, don't exit 1 on findings")
	flag.BoolVar(warnOnly, "warn", false, "warn only, don't exit 1 on findings")

	minScore := flag.Float64("min-score", 0.5, "minimum similarity score (0.0-1.0) for -d")
	minFields := flag.Int("min-fields", 2, "minimum fields to consider for -d")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gostructs [flags] [packages]\n\n")
		fmt.Fprintf(os.Stderr, "Analyze struct declarations in Go code.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  gostructs ./...           # run all analyses\n")
		fmt.Fprintf(os.Stderr, "  gostructs -u ./...        # find unused structs\n")
		fmt.Fprintf(os.Stderr, "  gostructs -d ./...        # find duplicate structs\n")
		fmt.Fprintf(os.Stderr, "  gostructs -d -min-score 0.8 ./...\n")
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("gostructs %s\n", version)
		os.Exit(0)
	}

	patterns := flag.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	// Default: run all if no flags specified
	runAll := !*findUnused && !*findDuplicate && !*findTags
	if runAll {
		*findUnused = true
		*findDuplicate = true
		*findTags = true
	}

	var allDiagnostics []Diagnostic
	exitCode := 0

	if *findUnused {
		unused, code := runUnusedAnalysis(patterns, *jsonOutput)
		if code > exitCode {
			exitCode = code
		}
		if *jsonOutput {
			allDiagnostics = append(allDiagnostics, UnusedToDiagnostics(unused)...)
		}
	}

	if *findDuplicate {
		opts := SimilarityOptions{
			MinFields:          *minFields,
			MinSimilarityScore: *minScore,
		}
		results, code := runDuplicateAnalysis(patterns, opts, *jsonOutput)
		if code > exitCode {
			exitCode = code
		}
		if *jsonOutput {
			allDiagnostics = append(allDiagnostics, SimilarityToDiagnostics(results)...)
		}
	}

	if *findTags {
		issues, code := runTagAnalysis(patterns, *jsonOutput)
		if code > exitCode {
			exitCode = code
		}
		if *jsonOutput {
			allDiagnostics = append(allDiagnostics, TagsToDiagnostics(issues)...)
		}
	}

	if *jsonOutput {
		if err := WriteJSON(os.Stdout, allDiagnostics, version); err != nil {
			fmt.Fprintf(os.Stderr, "error writing JSON: %v\n", err)
			os.Exit(2)
		}
	}

	if *warnOnly && exitCode == 1 {
		exitCode = 0
	}

	os.Exit(exitCode)
}

func runUnusedAnalysis(patterns []string, jsonOutput bool) ([]UnusedStruct, int) {
	unused, err := Analyze(patterns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return nil, 2
	}

	if len(unused) == 0 {
		return unused, 0
	}

	sort.Slice(unused, func(i, j int) bool {
		if unused[i].Position.Filename != unused[j].Position.Filename {
			return unused[i].Position.Filename < unused[j].Position.Filename
		}
		return unused[i].Position.Line < unused[j].Position.Line
	})

	if !jsonOutput {
		colored := useColors()
		for _, u := range unused {
			if colored {
				fmt.Printf("%s: struct %s%s%s is unused\n",
					u.Position,
					colorBold+colorYellow, u.Name, colorReset)
			} else {
				fmt.Printf("%s: struct %s is unused\n", u.Position, u.Name)
			}
		}
	}

	return unused, 1
}

func runDuplicateAnalysis(patterns []string, opts SimilarityOptions, jsonOutput bool) ([]SimilarityResult, int) {
	results, err := FindSimilarStructs(patterns, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return nil, 2
	}

	if len(results) == 0 {
		return results, 0
	}

	if !jsonOutput {
		printSimilarityResults(results)
	}

	return results, 1
}

func runTagAnalysis(patterns []string, jsonOutput bool) ([]TagIssue, int) {
	issues, err := ValidateTags(patterns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return nil, 2
	}

	if len(issues) == 0 {
		return issues, 0
	}

	if !jsonOutput {
		printTagIssues(issues)
	}

	return issues, 1
}

func printTagIssues(issues []TagIssue) {
	colored := useColors()

	for _, issue := range issues {
		if colored {
			fmt.Printf("%s: %s.%s: %s%s%s\n",
				issue.Position,
				issue.StructName,
				issue.FieldName,
				colorBold+colorYellow, issue.Message, colorReset)
		} else {
			fmt.Printf("%s: %s.%s: %s\n",
				issue.Position,
				issue.StructName,
				issue.FieldName,
				issue.Message)
		}
	}
}

func printSimilarityResults(results []SimilarityResult) {
	colored := useColors()

	for i, r := range results {
		if i > 0 {
			fmt.Println()
		}

		typeStr := similarityTypeString(r.Type)
		scoreStr := fmt.Sprintf("%.0f%%", r.Score*100)

		if colored {
			fmt.Printf("%s%s%s %s(%s)%s\n",
				colorBold, typeStr, colorReset,
				colorDim, scoreStr, colorReset)
		} else {
			fmt.Printf("%s (%s)\n", typeStr, scoreStr)
		}

		printStructLocation(r.StructA, colored, "  ")
		printStructLocation(r.StructB, colored, "  ")

		if len(r.CommonFields) > 0 {
			if colored {
				fmt.Printf("  %scommon:%s %s\n",
					colorGreen, colorReset,
					strings.Join(r.CommonFields, ", "))
			} else {
				fmt.Printf("  common: %s\n", strings.Join(r.CommonFields, ", "))
			}
		}

		if len(r.TypeMismatches) > 0 {
			if colored {
				fmt.Printf("  %stype mismatches:%s\n", colorYellow, colorReset)
			} else {
				fmt.Println("  type mismatches:")
			}
			for _, tm := range r.TypeMismatches {
				fmt.Printf("    %s: %s vs %s\n", tm.FieldName, tm.TypeInA, tm.TypeInB)
			}
		}

		if r.Type == Subset {
			printSubsetSuggestion(r, colored)
		}
	}

	fmt.Printf("\nFound %d similar struct pair(s)\n", len(results))
}

func printStructLocation(s StructInfo, colored bool, indent string) {
	if colored {
		fmt.Printf("%s%s%s%s %s(%d fields)%s\n",
			indent,
			colorBold+colorYellow, s.Name, colorReset,
			colorDim, len(s.Fields), colorReset)
		fmt.Printf("%s  %s%s%s\n", indent, colorDim, s.Position, colorReset)
	} else {
		fmt.Printf("%s%s (%d fields)\n", indent, s.Name, len(s.Fields))
		fmt.Printf("%s  %s\n", indent, s.Position)
	}
}

func printSubsetSuggestion(r SimilarityResult, colored bool) {
	smaller, larger := r.StructA, r.StructB
	if len(r.OnlyInA) > 0 {
		smaller, larger = r.StructB, r.StructA
	}

	if colored {
		fmt.Printf("  %ssuggestion:%s embed %s in %s\n",
			colorMagenta, colorReset, smaller.Name, larger.Name)
	} else {
		fmt.Printf("  suggestion: embed %s in %s\n", smaller.Name, larger.Name)
	}
}

func similarityTypeString(t SimilarityType) string {
	switch t {
	case Duplicate:
		return "DUPLICATE"
	case NearDuplicate:
		return "NEAR DUPLICATE"
	case Subset:
		return "SUBSET"
	case SameNames:
		return "SAME NAMES"
	case Mergeable:
		return "MERGEABLE"
	default:
		return "SIMILAR"
	}
}
