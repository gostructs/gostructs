# gostructs - Feature Roadmap

## Current Features
- [x] Detect unused struct definitions
- [x] Find duplicate/similar structs
- [x] Validate struct tags
- [x] Skip generated files by default

---

## Planned Features

### Memory Layout Analysis
- [ ] **Field alignment analysis** - detect structs with suboptimal field ordering
- [ ] **Memory waste calculation** - show bytes wasted due to padding
- [ ] **Suggest field reorder** - recommend optimal field arrangement
- [ ] **Show size savings** - display percentage memory savings (like structslop)
- [ ] **Struct visualization** - ASCII/JSON layout output showing byte offsets

### Struct Tags
- [x] **Tag validation** - detect malformed struct tags
- [ ] **Tag consistency** - check for inconsistent tag naming (json vs JSON)
- [ ] **Tag alignment** - detect misaligned tags for readability
- [ ] **Missing tags** - warn when some fields have tags but others don't

### Initialization Analysis
- [ ] **Uninitialized fields** - find structs that are partially initialized
- [ ] **Required fields** - detect fields that should always be set

### Code Quality
- [ ] **Empty structs** - detect structs with no fields
- [ ] **Single-field structs** - flag potentially unnecessary wrappers
- [ ] **Deeply nested structs** - warn about excessive nesting
- [ ] **Large structs** - flag structs exceeding size threshold

### Auto-fix Capabilities
- [ ] **-fix flag** - automatically reorder fields for optimal alignment
- [ ] **Preserve comments** - use DST (Decorated Syntax Tree) to keep comments intact
- [ ] **Dry-run mode** - show changes without applying

### Filtering Options
- [x] **Skip generated files** - ignore `*.pb.go`, `*_gen.go`, `*_generated.go` (enabled by default)
- [ ] **Skip by comment** - honor `// nolint:gostructs` directives
- [ ] **Include/exclude patterns** - filter by package or file path

### Output Formats
- [x] **JSON output** - machine-readable output for CI integration
- [ ] **SARIF output** - for GitHub code scanning
- [ ] **Diff output** - show suggested changes as unified diff

---

## Reference Tools

| Tool | Focus Area | Link |
|------|------------|------|
| fieldalignment | Memory layout (official) | golang.org/x/tools |
| betteralign | Memory layout + comments | github.com/dkorunic/betteralign |
| structslop | Memory optimization | github.com/orijtech/structslop |
| structlayout | Visualization | honnef.co/go/tools |
| tagalign | Tag alignment | github.com/4meepo/tagalign |
| tagliatelle | Tag naming | github.com/ldez/tagliatelle |
| exhaustruct | Initialization | golangci-lint |
