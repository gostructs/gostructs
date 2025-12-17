// Package main provides tools to analyze struct declarations in Go code.
// It includes detection of unused structs via graph-based reachability analysis,
// and detection of similar/duplicate structs via field comparison.
package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// isGeneratedFile returns true if the filename matches generated file patterns
func isGeneratedFile(filename string) bool {
	base := filepath.Base(filename)
	return strings.HasSuffix(base, ".pb.go") ||
		strings.HasSuffix(base, "_gen.go") ||
		strings.HasSuffix(base, "_generated.go") ||
		strings.HasSuffix(base, "_string.go") ||
		strings.HasSuffix(base, "_enumer.go")
}

type StructLocation struct {
	Name     string
	Position token.Position
}

type UnusedStruct struct {
	StructLocation
}

type node struct {
	name     string
	pos      token.Position
	obj      types.Object
	isStruct bool
}

func Analyze(patterns []string) ([]UnusedStruct, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedFiles,
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}

	// Check for package loading errors
	var errs []string
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			errs = append(errs, e.Error())
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(errs, "\n"))
	}

	g := newGraph()

	for _, pkg := range pkgs {
		g.collectDeclarations(pkg)
	}

	for _, pkg := range pkgs {
		g.collectUsages(pkg)
	}

	for _, pkg := range pkgs {
		g.findEntryPoints(pkg)
	}

	g.propagateReachability()

	var unused []UnusedStruct
	for obj, n := range g.nodes {
		if n.isStruct && !g.reachable[obj] {
			unused = append(unused, UnusedStruct{
				StructLocation: StructLocation{
					Name:     n.name,
					Position: n.pos,
				},
			})
		}
	}

	return unused, nil
}

type graph struct {
	nodes       map[types.Object]*node
	edges       map[types.Object][]types.Object // obj -> objects it uses
	reachable   map[types.Object]bool
	entryPoints []types.Object
}

func newGraph() *graph {
	return &graph{
		nodes:     make(map[types.Object]*node),
		edges:     make(map[types.Object][]types.Object),
		reachable: make(map[types.Object]bool),
	}
}

func (g *graph) collectDeclarations(pkg *packages.Package) {
	for _, file := range pkg.Syntax {
		filename := pkg.Fset.Position(file.Pos()).Filename
		if isGeneratedFile(filename) {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch decl := n.(type) {
			case *ast.TypeSpec:
				obj := pkg.TypesInfo.Defs[decl.Name]
				if obj == nil {
					return true
				}
				_, isStruct := decl.Type.(*ast.StructType)
				g.nodes[obj] = &node{
					name:     decl.Name.Name,
					pos:      pkg.Fset.Position(decl.Name.Pos()),
					obj:      obj,
					isStruct: isStruct,
				}

			case *ast.FuncDecl:
				if decl.Name == nil {
					return true
				}
				obj := pkg.TypesInfo.Defs[decl.Name]
				if obj == nil {
					return true
				}
				g.nodes[obj] = &node{
					name: decl.Name.Name,
					pos:  pkg.Fset.Position(decl.Name.Pos()),
					obj:  obj,
				}

			case *ast.ValueSpec:
				for _, name := range decl.Names {
					obj := pkg.TypesInfo.Defs[name]
					if obj == nil {
						continue
					}
					g.nodes[obj] = &node{
						name: name.Name,
						pos:  pkg.Fset.Position(name.Pos()),
						obj:  obj,
					}
				}
			}
			return true
		})
	}
}

func (g *graph) collectUsages(pkg *packages.Package) {
	for _, file := range pkg.Syntax {
		filename := pkg.Fset.Position(file.Pos()).Filename
		if isGeneratedFile(filename) {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch decl := n.(type) {
			case *ast.FuncDecl:
				if decl.Name == nil {
					return true
				}
				funcObj := pkg.TypesInfo.Defs[decl.Name]
				if funcObj == nil {
					return true
				}

				// If method, add edge from receiver type to this method
				if decl.Recv != nil && len(decl.Recv.List) > 0 {
					recvType := getReceiverType(decl.Recv.List[0].Type)
					if recvType != nil {
						if recvObj := pkg.TypesInfo.Uses[recvType]; recvObj != nil {
							g.edges[recvObj] = append(g.edges[recvObj], funcObj)
						}
					}
				}

				// Collect all usages within the function body
				g.collectUsagesInNode(pkg, funcObj, decl)

			case *ast.TypeSpec:
				typeObj := pkg.TypesInfo.Defs[decl.Name]
				if typeObj == nil {
					return true
				}
				// Collect usages in type definition (embedded types, field types)
				g.collectUsagesInNode(pkg, typeObj, decl.Type)

			case *ast.ValueSpec:
				for _, name := range decl.Names {
					varObj := pkg.TypesInfo.Defs[name]
					if varObj == nil {
						continue
					}
					// Collect usages in type and value
					if decl.Type != nil {
						g.collectUsagesInNode(pkg, varObj, decl.Type)
					}
					for _, val := range decl.Values {
						g.collectUsagesInNode(pkg, varObj, val)
					}
				}
			}
			return true
		})
	}
}

func (g *graph) collectUsagesInNode(pkg *packages.Package, from types.Object, n ast.Node) {
	ast.Inspect(n, func(node ast.Node) bool {
		switch x := node.(type) {
		case *ast.Ident:
			if usedObj := pkg.TypesInfo.Uses[x]; usedObj != nil {
				g.edges[from] = append(g.edges[from], usedObj)
			}
		case *ast.SelectorExpr:
			if usedObj := pkg.TypesInfo.Uses[x.Sel]; usedObj != nil {
				g.edges[from] = append(g.edges[from], usedObj)
			}
		}
		return true
	})
}

func getReceiverType(expr ast.Expr) *ast.Ident {
	switch t := expr.(type) {
	case *ast.Ident:
		return t
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident
		}
	case *ast.IndexExpr: // generic receiver T[X]
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident
		}
	case *ast.IndexListExpr: // generic receiver T[X, Y]
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident
		}
	}
	return nil
}

func (g *graph) findEntryPoints(pkg *packages.Package) {
	for _, file := range pkg.Syntax {
		filename := pkg.Fset.Position(file.Pos()).Filename
		if isGeneratedFile(filename) {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			funcDecl, ok := n.(*ast.FuncDecl)
			if !ok || funcDecl.Name == nil {
				return true
			}

			name := funcDecl.Name.Name
			// Entry points: main, init, and any exported function
			if name == "main" || name == "init" {
				if obj := pkg.TypesInfo.Defs[funcDecl.Name]; obj != nil {
					g.entryPoints = append(g.entryPoints, obj)
				}
			}
			return true
		})
	}

	for _, file := range pkg.Syntax {
		filename := pkg.Fset.Position(file.Pos()).Filename
		if isGeneratedFile(filename) {
			continue
		}
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, spec := range genDecl.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range valueSpec.Names {
					if obj := pkg.TypesInfo.Defs[name]; obj != nil {
						// Package-level vars are initialized at startup
						g.entryPoints = append(g.entryPoints, obj)
					}
				}
			}
		}
	}
}

func (g *graph) propagateReachability() {
	queue := make([]types.Object, 0, len(g.entryPoints))
	queue = append(queue, g.entryPoints...)

	for _, ep := range g.entryPoints {
		g.reachable[ep] = true
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, used := range g.edges[current] {
			if !g.reachable[used] {
				g.reachable[used] = true
				queue = append(queue, used)
			}
		}
	}
}

// ============================================================================
// Similarity Detection
// ============================================================================

// StructField represents a single field in a struct
type StructField struct {
	Name     string
	TypeName string
	Tag      string
	Embedded bool
}

// StructInfo holds all information about a struct for similarity analysis
type StructInfo struct {
	StructLocation
	Fields   []StructField
	FieldSet map[string]string // fieldName -> typeName
}

// SimilarityType categorizes the kind of similarity found
type SimilarityType int

const (
	Duplicate SimilarityType = iota
	NearDuplicate
	Subset
	SameNames
	Mergeable
)

// FieldTypeMismatch captures type differences for same-named fields
type FieldTypeMismatch struct {
	FieldName string
	TypeInA   string
	TypeInB   string
}

// SimilarityResult represents a pair of similar structs
type SimilarityResult struct {
	StructA        StructInfo
	StructB        StructInfo
	Type           SimilarityType
	Score          float64
	CommonFields   []string
	OnlyInA        []string
	OnlyInB        []string
	TypeMismatches []FieldTypeMismatch
}

// SimilarityOptions configures the similarity analysis
type SimilarityOptions struct {
	MinFields          int
	MinSimilarityScore float64
}

// FindSimilarStructs analyzes all structs and returns similarity results
func FindSimilarStructs(patterns []string, opts SimilarityOptions) ([]SimilarityResult, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedFiles,
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, err
	}

	var errs []string
	for _, pkg := range pkgs {
		for _, e := range pkg.Errors {
			errs = append(errs, e.Error())
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(errs, "\n"))
	}

	// Collect all structs
	var structs []StructInfo
	for _, pkg := range pkgs {
		structs = append(structs, collectStructInfos(pkg)...)
	}

	// Filter structs with minimum fields
	var filtered []StructInfo
	for _, s := range structs {
		if len(s.Fields) >= opts.MinFields {
			filtered = append(filtered, s)
		}
	}

	// Compare all pairs
	var results []SimilarityResult
	for i := 0; i < len(filtered); i++ {
		for j := i + 1; j < len(filtered); j++ {
			result := compareStructs(&filtered[i], &filtered[j])
			if result != nil && result.Score >= opts.MinSimilarityScore {
				results = append(results, *result)
			}
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

func collectStructInfos(pkg *packages.Package) []StructInfo {
	var structs []StructInfo

	for _, file := range pkg.Syntax {
		filename := pkg.Fset.Position(file.Pos()).Filename
		if isGeneratedFile(filename) {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			typeSpec, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				return true
			}

			info := StructInfo{
				StructLocation: StructLocation{
					Name:     typeSpec.Name.Name,
					Position: pkg.Fset.Position(typeSpec.Name.Pos()),
				},
				FieldSet: make(map[string]string),
			}

			if structType.Fields != nil {
				for _, field := range structType.Fields.List {
					typeName := typeToString(pkg, field.Type)
					tag := ""
					if field.Tag != nil {
						tag = field.Tag.Value
					}

					if len(field.Names) == 0 {
						// Embedded field
						sf := StructField{
							Name:     typeName,
							TypeName: typeName,
							Tag:      tag,
							Embedded: true,
						}
						info.Fields = append(info.Fields, sf)
						info.FieldSet[typeName] = typeName
					} else {
						for _, ident := range field.Names {
							sf := StructField{
								Name:     ident.Name,
								TypeName: typeName,
								Tag:      tag,
								Embedded: false,
							}
							info.Fields = append(info.Fields, sf)
							info.FieldSet[ident.Name] = typeName
						}
					}
				}
			}

			structs = append(structs, info)
			return true
		})
	}

	return structs
}

func typeToString(pkg *packages.Package, expr ast.Expr) string {
	if tv, ok := pkg.TypesInfo.Types[expr]; ok {
		return tv.Type.String()
	}
	return exprToString(expr)
}

func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + exprToString(t.Elt)
		}
		return "[...]" + exprToString(t.Elt)
	case *ast.MapType:
		return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.ChanType:
		return "chan " + exprToString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func(...)"
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func compareStructs(a, b *StructInfo) *SimilarityResult {
	result := &SimilarityResult{
		StructA: *a,
		StructB: *b,
	}

	// Find common fields and differences
	for fieldName, typeA := range a.FieldSet {
		if typeB, exists := b.FieldSet[fieldName]; exists {
			result.CommonFields = append(result.CommonFields, fieldName)
			if typeA != typeB {
				result.TypeMismatches = append(result.TypeMismatches, FieldTypeMismatch{
					FieldName: fieldName,
					TypeInA:   typeA,
					TypeInB:   typeB,
				})
			}
		} else {
			result.OnlyInA = append(result.OnlyInA, fieldName)
		}
	}

	for fieldName := range b.FieldSet {
		if _, exists := a.FieldSet[fieldName]; !exists {
			result.OnlyInB = append(result.OnlyInB, fieldName)
		}
	}

	// Sort for consistent output
	sort.Strings(result.CommonFields)
	sort.Strings(result.OnlyInA)
	sort.Strings(result.OnlyInB)

	// Calculate similarity score using Jaccard index
	totalFields := len(a.FieldSet) + len(b.FieldSet) - len(result.CommonFields)
	if totalFields == 0 {
		return nil
	}

	// Base score from field name overlap
	nameScore := float64(len(result.CommonFields)) / float64(totalFields)

	// Penalty for type mismatches
	typePenalty := float64(len(result.TypeMismatches)) * 0.1
	result.Score = nameScore - typePenalty
	if result.Score < 0 {
		result.Score = 0
	}

	// Determine similarity type
	result.Type = categorizeSimilarity(result, a, b)

	return result
}

func categorizeSimilarity(r *SimilarityResult, a, b *StructInfo) SimilarityType {
	// Perfect match (same fields, same types)
	if len(r.OnlyInA) == 0 && len(r.OnlyInB) == 0 && len(r.TypeMismatches) == 0 {
		return Duplicate
	}

	// Near duplicate (>80% overlap, no type mismatches)
	if r.Score >= 0.8 && len(r.TypeMismatches) == 0 {
		return NearDuplicate
	}

	// Subset: A contains all of B's fields (or vice versa)
	if len(r.OnlyInB) == 0 && len(r.TypeMismatches) == 0 && len(r.CommonFields) > 0 {
		return Subset // B is subset of A
	}
	if len(r.OnlyInA) == 0 && len(r.TypeMismatches) == 0 && len(r.CommonFields) > 0 {
		return Subset // A is subset of B
	}

	// Same names but different types
	if len(r.CommonFields) > 0 && len(r.TypeMismatches) == len(r.CommonFields) {
		return SameNames
	}

	return Mergeable
}
