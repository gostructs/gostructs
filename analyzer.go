// Package main provides a tool to detect unused struct declarations in Go code.
// It uses graph-based reachability analysis starting from main() and init()
// entry points to find structs that are truly dead code.
package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

type UnusedStruct struct {
	Name     string
	Position token.Position
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
				Name:     n.name,
				Position: n.pos,
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
