package huntdocs

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mirurobotics/gotools/internal/services/lint/linter"
)

// Severity indicates how important a documentation gap is.
type Severity int

const (
	SeverityMedium   Severity = 1
	SeverityHigh     Severity = 2
	SeverityCritical Severity = 3
)

func (s Severity) String() string {
	switch s {
	case SeverityCritical:
		return "critical"
	case SeverityHigh:
		return "high"
	case SeverityMedium:
		return "medium"
	default:
		return "unknown"
	}
}

// ParseSeverity converts a string to a Severity value.
func ParseSeverity(s string) (Severity, error) {
	switch strings.ToLower(s) {
	case "critical":
		return SeverityCritical, nil
	case "high":
		return SeverityHigh, nil
	case "medium":
		return SeverityMedium, nil
	default:
		return 0, fmt.Errorf("unknown severity %q (use critical, high, or medium)", s)
	}
}

// Kind describes what type of symbol is missing documentation.
type Kind string

const (
	KindPackage  Kind = "package"
	KindFunc     Kind = "function"
	KindType     Kind = "type"
	KindMethod   Kind = "method"
	KindConst    Kind = "const"
	KindVar      Kind = "var"
	KindField    Kind = "field"
	KindIface    Kind = "interface"
)

// Gap represents a single documentation gap found in the codebase.
type Gap struct {
	File     string
	Line     int
	Symbol   string
	Kind     Kind
	Severity Severity
	Reason   string
}

func (g Gap) String() string {
	return fmt.Sprintf("%s:%d: [%s] %s %s: %s",
		g.File, g.Line, g.Severity, g.Kind, g.Symbol, g.Reason)
}

// Opts configures a hunt-docs run.
type Opts struct {
	Path        string
	MinSeverity Severity
	Out         io.Writer
	Err         io.Writer
}

// Run scans Go files under the given path for documentation gaps.
// Returns the list of gaps found at or above the minimum severity.
func Run(opts Opts) ([]Gap, error) {
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.Err == nil {
		opts.Err = os.Stderr
	}

	files, err := linter.FindGoFiles(opts.Path)
	if err != nil {
		return nil, err
	}

	var gaps []Gap
	for _, path := range files {
		fileGaps, err := scanFile(path)
		if err != nil {
			_, _ = fmt.Fprintf(opts.Err, "warning: %s: %v\n", path, err)
			continue
		}
		gaps = append(gaps, fileGaps...)
	}

	// Filter by minimum severity.
	filtered := gaps[:0]
	for _, g := range gaps {
		if g.Severity >= opts.MinSeverity {
			filtered = append(filtered, g)
		}
	}
	gaps = filtered

	// Sort by severity (highest first), then file and line.
	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].Severity != gaps[j].Severity {
			return gaps[i].Severity > gaps[j].Severity
		}
		if gaps[i].File != gaps[j].File {
			return gaps[i].File < gaps[j].File
		}
		return gaps[i].Line < gaps[j].Line
	})

	return gaps, nil
}

func scanFile(path string) ([]Gap, error) {
	// #nosec G304 -- path comes from FindGoFiles
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if isGenerated(src) {
		return nil, nil
	}

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	isTest := strings.HasSuffix(path, "_test.go")

	var gaps []Gap

	// Check package doc (only for non-test, non-internal packages).
	if !isTest && !isInternalPackage(path) {
		gaps = append(gaps, checkPackageDoc(fset, path, f)...)
	}

	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if isTest {
				continue
			}
			gaps = append(gaps, checkFuncDecl(fset, path, d)...)
		case *ast.GenDecl:
			if isTest {
				continue
			}
			gaps = append(gaps, checkGenDecl(fset, path, d)...)
		}
	}

	return gaps, nil
}

func checkPackageDoc(fset *token.FileSet, path string, f *ast.File) []Gap {
	if f.Doc != nil && strings.TrimSpace(f.Doc.Text()) != "" {
		return nil
	}

	// Only flag if this is the first file alphabetically in the package
	// to avoid duplicate reports. We flag it as medium since internal
	// packages typically don't need package docs.
	dir := filepath.Dir(path)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			if filepath.Join(dir, name) == path {
				return []Gap{{
					File:     path,
					Line:     fset.Position(f.Package).Line,
					Symbol:   f.Name.Name,
					Kind:     KindPackage,
					Severity: SeverityMedium,
					Reason:   "missing package documentation comment",
				}}
			}
			// Another file comes first alphabetically; skip.
			return nil
		}
	}

	return nil
}

func checkFuncDecl(fset *token.FileSet, path string, d *ast.FuncDecl) []Gap {
	if !d.Name.IsExported() {
		return nil
	}

	kind := KindFunc
	symbol := d.Name.Name
	if d.Recv != nil && len(d.Recv.List) > 0 {
		kind = KindMethod
		symbol = receiverTypeName(d.Recv.List[0].Type) + "." + d.Name.Name
	}

	if d.Doc != nil && strings.TrimSpace(d.Doc.Text()) != "" {
		return checkDocQuality(fset, path, d.Doc, d.Name.Name, kind, symbol)
	}

	return []Gap{{
		File:     path,
		Line:     fset.Position(d.Name.Pos()).Line,
		Symbol:   symbol,
		Kind:     kind,
		Severity: SeverityHigh,
		Reason:   "exported " + string(kind) + " missing documentation comment",
	}}
}

func checkGenDecl(fset *token.FileSet, path string, d *ast.GenDecl) []Gap {
	var gaps []Gap
	for _, spec := range d.Specs {
		switch s := spec.(type) {
		case *ast.TypeSpec:
			gaps = append(gaps, checkTypeSpec(fset, path, d, s)...)
		case *ast.ValueSpec:
			gaps = append(gaps, checkValueSpec(fset, path, d, s)...)
		}
	}
	return gaps
}

func checkTypeSpec(fset *token.FileSet, path string, d *ast.GenDecl, s *ast.TypeSpec) []Gap {
	if !s.Name.IsExported() {
		return nil
	}

	// Use the GenDecl doc if it's a single-spec declaration,
	// otherwise use the TypeSpec doc.
	doc := s.Doc
	if doc == nil && len(d.Specs) == 1 {
		doc = d.Doc
	}

	var gaps []Gap

	if doc == nil || strings.TrimSpace(doc.Text()) == "" {
		sev := SeverityHigh
		kind := KindType
		if _, ok := s.Type.(*ast.InterfaceType); ok {
			kind = KindIface
			sev = SeverityHigh
		}
		gaps = append(gaps, Gap{
			File:     path,
			Line:     fset.Position(s.Name.Pos()).Line,
			Symbol:   s.Name.Name,
			Kind:     kind,
			Severity: sev,
			Reason:   "exported " + string(kind) + " missing documentation comment",
		})
	}

	// Check exported fields in structs.
	st, ok := s.Type.(*ast.StructType)
	if !ok || st.Fields == nil {
		return gaps
	}
	for _, field := range st.Fields.List {
		for _, name := range field.Names {
			if !name.IsExported() {
				continue
			}
			if field.Doc == nil || strings.TrimSpace(field.Doc.Text()) == "" {
				gaps = append(gaps, Gap{
					File:     path,
					Line:     fset.Position(name.Pos()).Line,
					Symbol:   s.Name.Name + "." + name.Name,
					Kind:     KindField,
					Severity: SeverityMedium,
					Reason:   "exported struct field missing documentation comment",
				})
			}
		}
	}

	return gaps
}

func checkValueSpec(fset *token.FileSet, path string, d *ast.GenDecl, s *ast.ValueSpec) []Gap {
	// Use GenDecl doc for single-spec declarations.
	doc := s.Doc
	if doc == nil && len(d.Specs) == 1 {
		doc = d.Doc
	}

	var gaps []Gap
	for _, name := range s.Names {
		if !name.IsExported() {
			continue
		}
		if doc != nil && strings.TrimSpace(doc.Text()) != "" {
			continue
		}

		kind := KindVar
		sev := SeverityMedium
		if d.Tok == token.CONST {
			kind = KindConst
			sev = SeverityMedium
		}

		gaps = append(gaps, Gap{
			File:     path,
			Line:     fset.Position(name.Pos()).Line,
			Symbol:   name.Name,
			Kind:     kind,
			Severity: sev,
			Reason:   "exported " + string(kind) + " missing documentation comment",
		})
	}
	return gaps
}

// checkDocQuality validates that a doc comment follows Go conventions.
func checkDocQuality(
	fset *token.FileSet,
	path string,
	doc *ast.CommentGroup,
	name string,
	kind Kind,
	symbol string,
) []Gap {
	text := strings.TrimSpace(doc.Text())
	if text == "" {
		return nil
	}

	// Go convention: doc comment should start with the symbol name.
	if !strings.HasPrefix(text, name+" ") && !strings.HasPrefix(text, name+"\n") {
		return []Gap{{
			File:     path,
			Line:     fset.Position(doc.Pos()).Line,
			Symbol:   symbol,
			Kind:     kind,
			Severity: SeverityMedium,
			Reason: fmt.Sprintf(
				"doc comment should start with %q", name),
		}}
	}

	return nil
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverTypeName(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return receiverTypeName(t.X)
	case *ast.IndexListExpr:
		return receiverTypeName(t.X)
	}
	return ""
}

func isInternalPackage(path string) bool {
	return strings.Contains(filepath.ToSlash(path), "/internal/")
}

func isGenerated(src []byte) bool {
	// Check first 2KB for generated marker.
	limit := 2048
	if len(src) < limit {
		limit = len(src)
	}
	header := string(src[:limit])
	return strings.Contains(header, "Code generated") ||
		strings.Contains(header, "DO NOT EDIT")
}
