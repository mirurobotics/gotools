package funcorder

import (
	"fmt"
	"go/ast"
	"go/token"
	"maps"
	"slices"
)

type use struct{ user, at int }

func (m *model) plainTypes() map[int][]int {
	plain := map[int][]int{}
	for typeIdx, decl := range m.decls {
		spec := soloTypeSpec(decl)
		isMethod := func(fnc *fn) bool {
			return fnc != nil && fnc.recv == spec.Name.Name
		}
		if spec == nil || slices.ContainsFunc(m.fns, isMethod) {
			continue
		}
		if users := usersOf(spec, m.decls); len(users) > 0 {
			plain[typeIdx] = users
		}
	}
	return plain
}

func (m *model) typePlaced(typeIdx int, uses map[int]use) bool {
	usePos, head := uses[typeIdx].at, m.plainHead(typeIdx)
	if typeIdx > usePos {
		return false
	}
	for i := head[len(head)-1] + 1; i < usePos; i++ {
		if owner, ok := m.plainOf(i); !ok || uses[owner].at != usePos {
			return false
		}
	}
	return true
}

func (m *model) inPlain(i int) bool {
	_, ok := m.plainOf(i)
	return ok
}

func (m *model) plainOf(i int) (int, bool) {
	if ownerIdx, ok := m.typeOwner[i]; ok {
		i = ownerIdx
	}
	_, ok := m.plain[i]
	return i, ok
}

func (m *model) typeMessage(typeIdx int, usage use) string {
	name := soloTypeSpec(m.decls[typeIdx]).Name.Name
	user := m.declName(usage.user)
	if usage.at == usage.user {
		const msg = "type %q should be directly above its first user %q"
		return fmt.Sprintf(msg, name, user)
	}
	if fnc := m.fns[usage.at]; fnc != nil {
		const msg = "type %q should be directly above the methods of %q, " +
			"which hold its first user %q"
		return fmt.Sprintf(msg, name, fnc.recv, user)
	}
	if soloTypeSpec(m.decls[usage.at]) == nil {
		const msg = "type %q should be directly above %q, " +
			"which sits over its first user %q"
		return fmt.Sprintf(msg, name, m.declName(usage.at), user)
	}
	return fmt.Sprintf(
		"type %q should be directly above %q, which holds its first user %q",
		name, m.declName(usage.at), user,
	)
}

func (m *model) declName(i int) string {
	if fnc := m.fns[i]; fnc != nil {
		return fnc.name
	}
	var name string
	ast.Inspect(m.decls[i], func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.TypeSpec:
			name = node.Name.Name
		case *ast.ValueSpec:
			name = node.Names[0].Name
		}
		return name == ""
	})
	return name
}

func (m *model) placeTypes(units [][]int, out []int) []int {
	pos := make(map[int]int, len(out))
	for outPos, i := range out {
		pos[i] = outPos
	}
	uses := m.firstUses(units, func(i int) int { return pos[i] })
	above := map[int][]int{}
	for _, idx := range slices.Sorted(maps.Keys(uses)) {
		above[uses[idx].at] = append(above[uses[idx].at], m.plainHead(idx)...)
	}
	var placed []int
	for outPos, i := range out {
		placed = append(append(placed, above[outPos]...), i)
	}
	return placed
}

func (m *model) firstUses(units [][]int, pos func(int) int) map[int]use {
	start := map[int]int{}
	byPos := func(a, b int) int { return pos(a) - pos(b) }
	for _, unit := range units {
		outPos := pos(slices.MinFunc(unit, byPos))
		for _, i := range unit {
			start[i] = outPos
		}
	}
	uses := map[int]use{}
	byStart := func(a, b int) int { return start[a] - start[b] }
	for typeIdx, users := range m.plain {
		first := slices.MinFunc(users, byStart)
		uses[typeIdx] = use{user: first, at: start[first]}
	}
	return uses
}

func (m *model) plainHead(typeIdx int) []int {
	head := append([]int{typeIdx}, m.typeBlocks[typeIdx]...)
	slices.Sort(head)
	return head
}

func soloTypeSpec(decl ast.Decl) *ast.TypeSpec {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok || genDecl.Tok != token.TYPE || len(genDecl.Specs) != 1 {
		return nil
	}
	spec, _ := genDecl.Specs[0].(*ast.TypeSpec)
	return spec
}

func usersOf(spec *ast.TypeSpec, decls []ast.Decl) []int {
	var users []int
	for i, decl := range decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok != token.VAR {
			continue
		}
		if refersTo(decl, spec) {
			users = append(users, i)
		}
	}
	return users
}

func refersTo(node ast.Node, spec *ast.TypeSpec) bool {
	found := false
	refIdents(node, func(*ast.SelectorExpr) {}, func(id *ast.Ident) bool {
		found = id.Obj != nil && id.Obj.Decl == spec
		return !found
	})
	return found
}
