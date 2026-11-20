package funcorder

import (
	"fmt"
	"go/ast"
	"go/token"
	"maps"
	"slices"
)

type use struct{ user, at int }

func (m *model) plainTypes(decls []ast.Decl) map[int][]int {
	plain := map[int][]int{}
	for t, d := range decls {
		ts := soloTypeSpec(d)
		if ts == nil || m.hasMethods(ts.Name.Name) {
			continue
		}
		if users := usersOf(ts, decls); len(users) > 0 {
			plain[t] = users
		}
	}
	return plain
}

func (m *model) hasMethods(name string) bool {
	return slices.ContainsFunc(m.funcs(), func(x *fn) bool { return x.recv == name })
}

func (m *model) typePlaced(t int, uses map[int]use) bool {
	at := uses[t].at
	if t > at {
		return false
	}
	for i := t + 1; i < at; i++ {
		if _, ok := uses[i]; !ok || uses[i].at != at {
			return false
		}
	}
	return true
}

func (m *model) typeMessage(t int, u use) string {
	name := soloTypeSpec(m.decls[t]).Name.Name
	user := declName(m.decls[u.user])
	if x := m.fns[u.user]; x != nil {
		user = x.name
	}
	return fmt.Sprintf("type %q should be directly above its first user %q", name, user)
}

func (m *model) placeTypes(units [][]int, out []int) []int {
	pos := make(map[int]int, len(out))
	for p, i := range out {
		pos[i] = p
	}
	uses := m.firstUses(units, func(i int) int { return pos[i] })
	above := map[int][]int{}
	for _, t := range slices.Sorted(maps.Keys(uses)) {
		above[uses[t].at] = append(above[uses[t].at], t)
	}
	var placed []int
	for p, i := range out {
		placed = append(append(placed, above[p]...), i)
	}
	return placed
}

func (m *model) firstUses(units [][]int, pos func(int) int) map[int]use {
	start := map[int]int{}
	for _, u := range units {
		p := pos(slices.MinFunc(u, func(a, b int) int { return pos(a) - pos(b) }))
		for _, i := range u {
			start[i] = p
		}
	}
	uses := map[int]use{}
	for t, users := range m.plain {
		u := slices.MinFunc(users, func(a, b int) int { return start[a] - start[b] })
		uses[t] = use{user: u, at: start[u]}
	}
	return uses
}

func usersOf(ts *ast.TypeSpec, decls []ast.Decl) []int {
	var users []int
	for i, d := range decls {
		if gd, ok := d.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
			continue
		}
		if refersTo(d, ts) {
			users = append(users, i)
		}
	}
	return users
}

func refersTo(n ast.Node, ts *ast.TypeSpec) bool {
	found := false
	ast.Inspect(n, func(n ast.Node) bool {
		id, ok := n.(*ast.Ident)
		found = found || ok && id.Obj != nil && id.Obj.Decl == ts
		return !found
	})
	return found
}

func soloTypeSpec(d ast.Decl) *ast.TypeSpec {
	gd, ok := d.(*ast.GenDecl)
	if !ok || gd.Tok != token.TYPE || len(gd.Specs) != 1 {
		return nil
	}
	ts, _ := gd.Specs[0].(*ast.TypeSpec)
	return ts
}

func declName(d ast.Decl) string {
	if gd, ok := d.(*ast.GenDecl); ok {
		if vs, ok := gd.Specs[0].(*ast.ValueSpec); ok {
			return vs.Names[0].Name
		}
	}
	return ""
}
