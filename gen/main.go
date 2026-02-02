//go:build ignore

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"go/format"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/iancoleman/strcase"
)

type constDef struct {
	GoName  string
	CName   string
	Message string
}

type templateData struct {
	Errors   []constDef
	Policies []constDef
}

type namesMap map[string]struct{}

func newNamesMap(names ...string) namesMap {
	m := make(namesMap, len(names))
	for _, name := range names {
		m[name] = struct{}{}
	}
	return m
}

func (m namesMap) has(name string) bool {
	_, ok := m[name]
	return ok
}

type astNode struct {
	Kind  string    `json:"kind"`
	Name  string    `json:"name"`
	Inner []astNode `json:"inner"`
}

func main() {
	tmplPath := flag.String("template", "", "path to template file")
	outPath := flag.String("output", "", "path to output file")
	flag.Parse()

	if *tmplPath == "" || *outPath == "" {
		log.Fatalf("usage: %s -template=<path> -output=<path>", os.Args[0])
	}

	// Parse enums from clang AST
	enums, err := parseEnumsFromAST("LocalAuthentication/LocalAuthentication.h", "LAError", "LAPolicy")
	if err != nil {
		log.Fatalf("failed to parse enums: %v", err)
	}

	// We ignore some values which is not available on macOS
	// clang does not expose availability information in JSON format
	data := templateData{
		Errors:   convertEnumConstants(enums["LAError"], "LAError", "Err", "LAErrorTouchIDNotAvailable", "LAErrorTouchIDNotEnrolled", "LAErrorTouchIDLockout", "LAErrorWatchNotAvailable"),
		Policies: convertEnumConstants(enums["LAPolicy"], "LAPolicy", "Policy", "LAPolicyDeviceOwnerAuthenticationWithWristDetection"),
	}

	t, err := template.ParseFiles(*tmplPath)
	if err != nil {
		log.Fatalf("failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, filepath.Base(*tmplPath), data); err != nil {
		log.Fatalf("failed to execute template: %v", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("failed to format generated code: %v", err)
	}

	if err := os.WriteFile(*outPath, formatted, 0644); err != nil {
		log.Fatalf("failed to write file: %v", err)
	}

	fmt.Printf("Generated %s\n", *outPath)
	fmt.Printf("  LAError: %d constants\n", len(data.Errors))
	fmt.Printf("  LAPolicy: %d constants\n", len(data.Policies))
}

func convertEnumConstants(cNames []string, prefix, goPrefix string, ignored ...string) []constDef {
	ignoreMap := newNamesMap(ignored...)
	var defs []constDef
	for _, cName := range cNames {
		if ignoreMap.has(cName) {
			continue
		}
		name := strings.TrimPrefix(cName, prefix)
		defs = append(defs, constDef{
			GoName:  goPrefix + name,
			CName:   cName,
			Message: strcase.ToDelimited(name, ' '),
		})
	}
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].CName < defs[j].CName
	})
	return defs
}

// parseEnumsFromAST parses the header and extracts multiple enums
func parseEnumsFromAST(header string, enumNames ...string) (map[string][]string, error) {
	cmd := exec.Command("clang", "-Xclang", "-ast-dump=json", "-xobjective-c", "-fsyntax-only", "-")
	cmd.Stdin = strings.NewReader("#import <" + header + ">\n")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("clang failed: %w", err)
	}

	var root astNode
	if err := json.Unmarshal(out, &root); err != nil {
		return nil, fmt.Errorf("failed to parse AST JSON: %w", err)
	}

	allDecls := findEnumDecls(root, newNamesMap(enumNames...))
	grouped := make(map[string][]astNode, len(allDecls))
	for _, decl := range allDecls {
		grouped[decl.Name] = append(grouped[decl.Name], decl)
	}

	result := make(map[string][]string, len(enumNames))
	for _, enumName := range enumNames {
		decls := grouped[enumName]
		if len(decls) != 1 {
			return nil, fmt.Errorf("expected 1 enum %s, found %d", enumName, len(decls))
		}

		var names []string
		for _, child := range decls[0].Inner {
			if child.Kind == "EnumConstantDecl" && child.Name != "" {
				names = append(names, child.Name)
			}
		}
		result[enumName] = names
	}

	return result, nil
}

func findEnumDecls(node astNode, wantNames namesMap) []astNode {
	var found []astNode

	if node.Kind == "EnumDecl" && wantNames.has(node.Name) {
		// Check if it has EnumConstantDecl children (not just a forward declaration)
		for _, child := range node.Inner {
			if child.Kind == "EnumConstantDecl" {
				found = append(found, node)
				break
			}
		}
	}

	for _, child := range node.Inner {
		found = append(found, findEnumDecls(child, wantNames)...)
	}

	return found
}
