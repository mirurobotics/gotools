package funcorder

import (
	"fmt"
	"go/ast"
	"go/token"
	"maps"
	"slices"
)

type use struct{ user, at int }

type plainType struct{ head, users []int }

func (m *model) plainTypes() map[int]plainType {
	plain := map[int]plainType{}
	for t, d := range m.decls {
		ts := soloTypeSpec(d)
		if _, ok := m.heads[typeName(ts)]; ts == nil || ok {
			continue
		}
		if users := usersOf(ts, m.decls); len(users) > 0 {
			plain[t] = plainType{head: constHead(m.decls, t, ts), users: users}
		}
	}
	return plain
}

func (m *model) typePlaced(t int, uses map[int]use) bool {
	at, head := uses[t].at, m.plain[t].head
	if t > at {
		return false
	}
	for i := head[len(head)-1] + 1; i < at; i++ {
		if !m.inPlain(i) || uses[m.plainOf[i]].at != at {
			return false
		}
	}
	return true
}

func (m *model) inPlain(i int) bool {
	_, ok := m.plainOf[i]
	return ok
}

func (m *model) typeMessage(t int, u use) string {
	name := typeName(soloTypeSpec(m.decls[t]))
	user := m.declName(u.user)
	if u.at == u.user {
		const msg = "type %q should be directly above its first user %q"
		return fmt.Sprintf(msg, name, user)
	}
	if x := m.fns[u.at]; x != nil {
		const msg = "type %q should be directly above the methods of %q, " +
			"which hold its first user %q"
		return fmt.Sprintf(msg, name, x.recv, user)
	}
	return fmt.Sprintf(
		"type %q should be directly above %q, which holds its first user %q",
		name, m.declName(u.at), user,
	)
}

func (m *model) placeTypes(units [][]int, out []int) []int {
	pos := make(map[int]int, len(out))
	for p, i := range out {
		pos[i] = p
	}
	uses := m.firstUses(units, func(i int) int { return pos[i] })
	above := map[int][]int{}
	for _, t := range slices.Sorted(maps.Keys(uses)) {
		above[uses[t].at] = append(above[uses[t].at], m.plain[t].head...)
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
	for t, p := range m.plain {
		u := slices.MinFunc(p.users, func(a, b int) int { return start[a] - start[b] })
		uses[t] = use{user: u, at: start[u]}
	}
	return uses
}

func (m *model) declName(i int) string {
	if x := m.fns[i]; x != nil {
		return x.name
	}
	var name string
	ast.Inspect(m.decls[i], func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.TypeSpec:
			name = s.Name.Name
		case *ast.ValueSpec:
			name = s.Names[0].Name
		}
		return name == ""
	})
	return name
}

func constHead(decls []ast.Decl, t int, ts *ast.TypeSpec) []int {
	head := []int{t}
	for i := t + 1; i < len(decls); i++ {
		gd, ok := decls[i].(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST || !refersTo(gd, ts) {
			break
		}
		head = append(head, i)
	}
	return head
}

func usersOf(ts *ast.TypeSpec, decls []ast.Decl) []int {
	var users []int
	for i, d := range decls {
		if gd, ok := d.(*ast.GenDecl); ok && gd.Tok != token.VAR {
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
	refIdents(n, func(*ast.SelectorExpr) {}, func(id *ast.Ident) bool {
		found = id.Obj != nil && id.Obj.Decl == ts
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

func typeName(ts *ast.TypeSpec) string {
	if ts == nil {
		return ""
	}
	return ts.Name.Name
}
