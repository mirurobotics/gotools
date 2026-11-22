package analysis

import (
	"go/ast"
	"go/token"
)

// DeclaredName is an identifier a file declares, with what it refers to.
type DeclaredName struct {
	Ident *ast.Ident
	Kind  NameKind
	Type  ast.Expr // type of a param, result, field or interface method, else nil
}

// WalkDeclaredNames calls visit for each name of a NameKind that file declares.
func WalkDeclaredNames(file *ast.File, visit func(DeclaredName)) {
	walker := &nameWalker{visit: visit, topDecl: nil}
	for _, decl := range file.Decls {
		walker.topDecl = decl
		ast.Inspect(decl, walker.inspect)
	}
}

// NameKind classifies what a declared name refers to.
type NameKind int

const (
	KindLocalVar   NameKind = iota // :=, var or range variable in a function
	KindPackageVar                 // var outside any function
	KindParam                      // param of a func declaration or literal
	KindResult                     // named result of a func declaration or literal
	KindField                      // struct field
	KindFunc                       // func without a receiver, other than KindTestFunc
	KindTestFunc                   // go test Test, Benchmark, Fuzz or Example func
	KindMethod                     // func with a receiver
	KindIfaceMethod                // interface method
	KindConst                      // const at any scope
	KindType                       // type at any scope
)

type nameWalker struct {
	visit   func(DeclaredName)
	topDecl ast.Decl
}

func (w *nameWalker) inspect(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.FuncDecl:
		w.emit(node.Name, funcKind(node), nil)
		w.walkSignature(node.Type)
	case *ast.FuncLit:
		w.walkSignature(node.Type)
	case *ast.GenDecl:
		w.walkGenDecl(node)
		return false
	case *ast.StructType:
		w.walkFields(node.Fields, KindField)
	case *ast.InterfaceType:
		w.walkFields(node.Methods, KindIfaceMethod)
	case *ast.AssignStmt:
		if node.Tok == token.DEFINE {
			w.walkDefined(node)
		}
	case *ast.RangeStmt:
		if node.Tok == token.DEFINE {
			w.walkRangeVars(node)
		}
	}
	return true
}

func (w *nameWalker) walkSignature(funcType *ast.FuncType) {
	w.walkFields(funcType.Params, KindParam)
	w.walkFields(funcType.Results, KindResult)
}

func (w *nameWalker) walkFields(fields *ast.FieldList, kind NameKind) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		w.emitAll(field.Names, kind, field.Type)
	}
}

func (w *nameWalker) walkGenDecl(decl *ast.GenDecl) {
	kind := KindLocalVar
	switch {
	case decl.Tok == token.CONST:
		kind = KindConst
	case decl == w.topDecl:
		kind = KindPackageVar
	}
	for _, spec := range decl.Specs {
		switch spec := spec.(type) {
		case *ast.ValueSpec:
			w.emitAll(spec.Names, kind, nil)
		case *ast.TypeSpec:
			w.emit(spec.Name, KindType, nil)
		}
		if decl.Tok != token.CONST {
			ast.Inspect(spec, w.inspect)
		}
	}
}

func (w *nameWalker) emitAll(idents []*ast.Ident, kind NameKind, typ ast.Expr) {
	for _, ident := range idents {
		w.emit(ident, kind, typ)
	}
}

func (w *nameWalker) walkDefined(stmt *ast.AssignStmt) {
	for _, expr := range stmt.Lhs {
		ident, ok := expr.(*ast.Ident)
		if ok && !isRedeclared(ident, stmt) {
			w.emit(ident, KindLocalVar, nil)
		}
	}
}

func (w *nameWalker) walkRangeVars(stmt *ast.RangeStmt) {
	for _, expr := range []ast.Expr{stmt.Key, stmt.Value} {
		if ident, ok := expr.(*ast.Ident); ok {
			w.emit(ident, KindLocalVar, nil)
		}
	}
}

func (w *nameWalker) emit(ident *ast.Ident, kind NameKind, typ ast.Expr) {
	w.visit(DeclaredName{Ident: ident, Kind: kind, Type: typ})
}

func funcKind(decl *ast.FuncDecl) NameKind {
	switch {
	case decl.Recv != nil:
		return KindMethod
	case IsTestFunc(decl):
		return KindTestFunc
	}
	return KindFunc
}

func isRedeclared(ident *ast.Ident, stmt ast.Node) bool {
	return ident.Obj != nil && ident.Obj.Decl != stmt
}
