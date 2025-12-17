# gostructs

Analyze struct declarations in Go code.

## Installation

```
go install github.com/gostructs/gostructs@latest
```

## Usage

```
gostructs ./...          # run all analyses
gostructs -u ./...       # find unused structs
gostructs -d ./...       # find duplicate structs
gostructs ./pkg/...      # analyze specific package
```

## Flags

| Flag | Description |
|------|-------------|
| `-u`, `-unused` | Find unused structs |
| `-d`, `-duplicate` | Find duplicate/similar structs |
| `-t`, `-tags` | Validate struct tags |
| `-j`, `-json` | Output in JSON format |
| `-min-score` | Minimum similarity score (0.0-1.0) for `-d` (default: 0.5) |
| `-min-fields` | Minimum fields to consider for `-d` (default: 2) |
| `-v`, `-version` | Print version |

Note: Generated files (`*.pb.go`, `*_gen.go`, `*_generated.go`) are automatically skipped.

## Output

```
path/to/file.go:15:6: struct Foo is unused
```

Exit codes: `0` success, `1` issues found, `2` error.
