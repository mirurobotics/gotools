package funcorder

import (
	"fmt"
	"go/ast"
	"go/token"
	"maps"
	"slices"
)

type use struct{ user, at int }

func (m *model) typeUsers(withMethods bool) map[int][]int {
	byType := map[int][]int{}
	for typeIdx, decl := range m.decls {
		spec := soloTypeSpec(decl)
		isMethod := func(fnc *fn) bool {
			return fnc != nil && fnc.recv == spec.Name.Name
		}
		if spec == nil || slices.ContainsFunc(m.fns, isMethod) != withMethods {
			continue
		}
		users := usersOf(spec, m.decls, !withMethods)
		others := slices.DeleteFunc(slices.Clone(users), m.isConst)
		if len(others) > 0 {
			users = others
		}
		if len(users) > 0 {
			byType[typeIdx] = users
		}
	}
	return byType
}

func (m *model) isConst(i int) bool { return m.declTok(i) == token.CONST }

func (m *model) typePlaced(typeIdx int, uses map[int]use) bool {
	_, nested := m.nesting(uses)
	usage, head := uses[typeIdx], m.heads([]int{typeIdx}, nested)
	usePos := usage.at
	if m.isNested(usage) {
		usePos = usage.user
	}
	if typeIdx > usePos {
		return false
	}
	for offset, i := range head {
		if i != head[0]+offset {
			return false
		}
	}
	for i := head[len(head)-1] + 1; i < usePos; i++ {
		if owner, ok := m.plainOf(i); !ok || uses[owner].at != usage.at {
			return false
		}
	}
	return true
}

func (m *model) typeMessage(typeIdx int, usage use) string {
	name := soloTypeSpec(m.decls[typeIdx]).Name.Name
	user := m.declName(usage.user)
	if usage.at == usage.user || m.isNested(usage) {
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
		if name != "" {
			return false
		}
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
	above, nested := m.nesting(m.firstUses(units, func(i int) int { return pos[i] }))
	var placed []int
	for outPos, i := range out {
		placed = append(append(placed, m.heads(above[outPos], nested)...), i)
	}
	return placed
}

func (m *model) nesting(uses map[int]use) (map[int][]int, map[int][]int) {
	above, nested := map[int][]int{}, map[int][]int{}
	for _, idx := range slices.Sorted(maps.Keys(uses)) {
		if usage := uses[idx]; m.isNested(usage) {
			nested[usage.user] = append(nested[usage.user], idx)
		} else {
			above[usage.at] = append(above[usage.at], idx)
		}
	}
	return above, nested
}

func (m *model) isNested(usage use) bool { return m.inPlain(usage.user) }

func (m *model) inPlain(i int) bool {
	_, ok := m.plainOf(i)
	return ok
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
	var useOf func(typeIdx int) use
	startOf := func(i int) int {
		if owner, ok := m.plainOf(i); ok {
			return useOf(owner).at
		}
		return start[i]
	}
	useOf = func(typeIdx int) use {
		if usage, ok := uses[typeIdx]; ok {
			return usage
		}
		uses[typeIdx] = use{user: typeIdx, at: 0}
		byStart := func(a, b int) int { return startOf(a) - startOf(b) }
		first := slices.MinFunc(m.plain[typeIdx], byStart)
		uses[typeIdx] = use{user: first, at: startOf(first)}
		return uses[typeIdx]
	}
	for typeIdx := range m.plain {
		useOf(typeIdx)
	}
	return uses
}

func (m *model) plainOf(i int) (int, bool) {
	if ownerIdx, ok := m.typeOwner[i]; ok {
		i = ownerIdx
	}
	_, ok := m.plain[i]
	return i, ok
}

func (m *model) heads(types []int, nested map[int][]int) []int {
	var out []int
	for _, typeIdx := range types {
		for _, i := range m.plainHead(typeIdx) {
			out = append(append(out, m.heads(nested[i], nested)...), i)
		}
	}
	return out
}

func (m *model) plainHead(typeIdx int) []int {
	spec := soloTypeSpec(m.decls[typeIdx])
	usesType := func(i int) bool { return refersTo(m.decls[i], spec) }
	above := slices.DeleteFunc(slices.Clone(m.typeBlocks[typeIdx]), usesType)
	carried := slices.DeleteFunc(slices.Clone(m.typeBlocks[typeIdx]), func(i int) bool {
		return !usesType(i)
	})
	return slices.Concat(above, []int{typeIdx}, carried)
}

func soloTypeSpec(decl ast.Decl) *ast.TypeSpec {
	genDecl, ok := decl.(*ast.GenDecl)
	if !ok || genDecl.Tok != token.TYPE || len(genDecl.Specs) != 1 {
		return nil
	}
	spec, _ := genDecl.Specs[0].(*ast.TypeSpec)
	return spec
}

func usersOf(spec *ast.TypeSpec, decls []ast.Decl, withConsts bool) []int {
	var users []int
	for i, decl := range decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if ok && !isUserBlock(genDecl.Tok, withConsts) {
			continue
		}
		if refersTo(decl, spec) {
			users = append(users, i)
		}
	}
	return users
}

func isUserBlock(tok token.Token, withConsts bool) bool {
	return tok == token.VAR || withConsts && tok == token.CONST
}

func refersTo(node ast.Node, spec *ast.TypeSpec) bool {
	found := false
	refIdents(node, func(*ast.SelectorExpr) {}, func(id *ast.Ident) bool {
		found = id.Obj != nil && id.Obj.Decl == spec
		return !found
	})
	return found
}
