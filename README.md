# gounused

Detect unused struct declarations in Go code.

## Installation

```
go install github.com/afify/gounused@latest
```

## Usage

```
gounused ./...          # analyze entire module
gounused ./pkg/...      # analyze specific package
gounused .              # analyze current directory
```

## Output

```
path/to/file.go:15:6: struct Foo is unused
```

Exit codes: `0` success, `1` unused structs found, `2` error.
