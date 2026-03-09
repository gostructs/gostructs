package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"strings"
	"unicode"

	"golang.org/x/tools/go/packages"
)

// TagIssueType represents the type of tag issue
type TagIssueType string

const (
	TagMalformed     TagIssueType = "malformed"
	TagDuplicateKey  TagIssueType = "duplicate_key"
	TagInvalidKey    TagIssueType = "invalid_key"
	TagEmptyValue    TagIssueType = "empty_value"
	TagBadQuotes     TagIssueType = "bad_quotes"
	TagUnknownOption TagIssueType = "unknown_option"
)

// TagIssue represents a problem found in a struct tag
type TagIssue struct {
	StructName string
	FieldName  string
	Position   token.Position
	Type       TagIssueType
	Tag        string
	Message    string
}

// ValidateTags analyzes struct tags in the given packages
func ValidateTags(patterns []string) ([]TagIssue, error) {
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

	var issues []TagIssue
	for _, pkg := range pkgs {
		issues = append(issues, validatePackageTags(pkg)...)
	}

	return issues, nil
}

func validatePackageTags(pkg *packages.Package) []TagIssue {
	var issues []TagIssue

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

			structName := typeSpec.Name.Name

			if structType.Fields != nil {
				for _, field := range structType.Fields.List {
					if field.Tag == nil {
						continue
					}

					fieldName := ""
					if len(field.Names) > 0 {
						fieldName = field.Names[0].Name
					} else {
						// Embedded field
						fieldName = typeToString(pkg, field.Type)
					}

					pos := pkg.Fset.Position(field.Tag.Pos())
					tagValue := field.Tag.Value

					// Remove backticks
					if len(tagValue) >= 2 && tagValue[0] == '`' && tagValue[len(tagValue)-1] == '`' {
						tagValue = tagValue[1 : len(tagValue)-1]
					}

					fieldIssues := validateTag(structName, fieldName, pos, tagValue)
					issues = append(issues, fieldIssues...)
				}
			}

			return true
		})
	}

	return issues
}

func validateTag(structName, fieldName string, pos token.Position, tag string) []TagIssue {
	var issues []TagIssue

	// Check for basic parsing errors using reflect.StructTag
	st := reflect.StructTag(tag)

	// Track seen keys for duplicate detection
	seenKeys := make(map[string]bool)

	// Parse tag manually to detect issues
	for tag != "" {
		// Skip leading spaces
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		// Scan to colon
		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
			i++
		}
		if i == 0 {
			issues = append(issues, TagIssue{
				StructName: structName,
				FieldName:  fieldName,
				Position:   pos,
				Type:       TagMalformed,
				Tag:        tag,
				Message:    "tag key cannot be empty",
			})
			break
		}
		if i+1 >= len(tag) || tag[i] != ':' {
			issues = append(issues, TagIssue{
				StructName: structName,
				FieldName:  fieldName,
				Position:   pos,
				Type:       TagMalformed,
				Tag:        tag,
				Message:    fmt.Sprintf("tag key %q missing colon", tag[:i]),
			})
			break
		}
		if tag[i+1] != '"' {
			issues = append(issues, TagIssue{
				StructName: structName,
				FieldName:  fieldName,
				Position:   pos,
				Type:       TagBadQuotes,
				Tag:        tag,
				Message:    fmt.Sprintf("tag value for key %q not quoted", tag[:i]),
			})
			break
		}

		key := tag[:i]
		tag = tag[i+1:]

		// Validate key format
		if !isValidTagKey(key) {
			issues = append(issues, TagIssue{
				StructName: structName,
				FieldName:  fieldName,
				Position:   pos,
				Type:       TagInvalidKey,
				Tag:        key,
				Message:    fmt.Sprintf("invalid tag key %q", key),
			})
		}

		// Check for duplicate keys
		if seenKeys[key] {
			issues = append(issues, TagIssue{
				StructName: structName,
				FieldName:  fieldName,
				Position:   pos,
				Type:       TagDuplicateKey,
				Tag:        key,
				Message:    fmt.Sprintf("duplicate tag key %q", key),
			})
		}
		seenKeys[key] = true

		// Scan quoted value
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			issues = append(issues, TagIssue{
				StructName: structName,
				FieldName:  fieldName,
				Position:   pos,
				Type:       TagBadQuotes,
				Tag:        tag,
				Message:    "unclosed quote in tag value",
			})
			break
		}

		value := tag[1:i]
		tag = tag[i+1:]

		// Check for empty value (might be intentional, so just info)
		if value == "" && key != "-" {
			// Get the actual value using reflect to handle escapes
			actualValue, _ := st.Lookup(key)
			if actualValue == "" {
				issues = append(issues, TagIssue{
					StructName: structName,
					FieldName:  fieldName,
					Position:   pos,
					Type:       TagEmptyValue,
					Tag:        key,
					Message:    fmt.Sprintf("empty value for tag key %q", key),
				})
			}
		}

		// Validate common tag options
		issues = append(issues, validateTagOptions(structName, fieldName, pos, key, value)...)
	}

	return issues
}

func isValidTagKey(key string) bool {
	if key == "" {
		return false
	}
	for i, r := range key {
		if i == 0 && !unicode.IsLetter(r) && r != '_' {
			return false
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
			return false
		}
	}
	return true
}

var tagOpts = map[string]map[string]bool{
	"json": {
		"omitempty": true,
		"string":    true,
	},
	"xml": {
		"omitempty": true,
		"attr":      true,
		"chardata":  true,
		"innerxml":  true,
		"comment":   true,
		"any":       true,
		"cdata":     true,
	},
	"yaml": {
		"omitempty": true,
		"inline":    true,
		"flow":      true,
	},
	"toml": {
		"omitempty": true,
		"inline":    true,
	},
}

func validateTagOptions(structName, fieldName string, pos token.Position, key, value string) []TagIssue {
	var issues []TagIssue

	opts, ok := tagOpts[key]
	if !ok {
		return issues
	}

	parts := strings.Split(value, ",")
	for i, part := range parts {
		if i == 0 {
			continue
		}
		part = strings.TrimSpace(part)
		if part != "" && !opts[part] && part != "-" {
			issues = append(issues, TagIssue{
				StructName: structName,
				FieldName:  fieldName,
				Position:   pos,
				Type:       TagUnknownOption,
				Tag:        key,
				Message:    fmt.Sprintf("unknown option %q for %s tag", part, key),
			})
		}
	}

	return issues
}
