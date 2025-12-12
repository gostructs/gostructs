package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
)

var version = "0.0.1"

func main() {
	showVersion := flag.Bool("version", false, "print version")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: gounused [flags] [packages]\n\n")
		fmt.Fprintf(os.Stderr, "Detect unused struct declarations in Go code.\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  gounused ./...\n")
		fmt.Fprintf(os.Stderr, "  gounused ./pkg/...\n")
		fmt.Fprintf(os.Stderr, "  gounused .\n")
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("gounused %s\n", version)
		os.Exit(0)
	}

	patterns := flag.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	unused, err := Analyze(patterns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	if len(unused) == 0 {
		os.Exit(0)
	}

	sort.Slice(unused, func(i, j int) bool {
		if unused[i].Position.Filename != unused[j].Position.Filename {
			return unused[i].Position.Filename < unused[j].Position.Filename
		}
		return unused[i].Position.Line < unused[j].Position.Line
	})

	for _, u := range unused {
		fmt.Printf("%s: struct %s is unused\n", u.Position, u.Name)
	}

	os.Exit(1)
}
