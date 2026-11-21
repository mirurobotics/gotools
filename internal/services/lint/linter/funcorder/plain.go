package funcorder

import (
	"cmp"
	"fmt"
	"go/ast"
	"go/token"
	"maps"
	"slices"
)

type use struct{ user, at, depth int }

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
	usage, head := uses[typeIdx], m.plainHead(typeIdx)
	if typeIdx > usage.at {
		return false
	}
	if _, ok := m.strayBlock(typeIdx); ok {
		return false
	}
	isAbove := func(owner int) bool {
		return uses[owner].at == usage.at && uses[owner].depth <= usage.depth
	}
	for i := head[len(head)-1] + 1; i < usage.at; i++ {
		if owner, ok := m.plainOf(i); !ok || !isAbove(owner) {
			return false
		}
	}
	return true
}

func (m *model) typeMessage(typeIdx int, usage use) string {
	name := soloTypeSpec(m.decls[typeIdx]).Name.Name
	user := m.declName(usage.user)
	if msg, ok := m.strayMessage(typeIdx); ok {
		return msg
	}
	if owner, ok := m.plainOf(usage.user); ok {
		usage.at = m.plainHead(owner)[0]
	}
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

func (m *model) strayMessage(typeIdx int) (string, bool) {
	stray, ok := m.strayBlock(typeIdx)
	spec := soloTypeSpec(m.decls[typeIdx])
	if !ok || stray > typeIdx || !refersTo(m.decls[stray], spec) {
		return "", false
	}
	const msg = "type %q should be directly above %q, which uses only it"
	return fmt.Sprintf(msg, spec.Name.Name, m.declName(stray)), true
}

func (m *model) strayBlock(typeIdx int) (int, bool) {
	head := m.plainHead(typeIdx)
	at := slices.Index(head, typeIdx)
	for offset, i := range head {
		if i != typeIdx+offset-at {
			return i, true
		}
	}
	return 0, false
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
	uses := m.firstUses(units, func(i int) int { return pos[i] })
	deepest := func(a, b int) int { return headKey(uses, a).compare(headKey(uses, b)) }
	above := map[int][]int{}
	for _, idx := range slices.SortedFunc(maps.Keys(uses), deepest) {
		above[uses[idx].at] = append(above[uses[idx].at], m.plainHead(idx)...)
	}
	var placed []int
	for outPos, i := range out {
		placed = append(append(placed, above[outPos]...), i)
	}
	return placed
}

func (m *model) firstUses(units [][]int, pos func(int) int) map[int]use {
	start := unitStarts(units, pos)
	uses := map[int]use{}
	var useOf func(typeIdx int) use
	useOf = func(typeIdx int) use {
		if usage, ok := uses[typeIdx]; ok {
			return usage
		}
		uses[typeIdx] = use{user: typeIdx, at: 0, depth: 0}
		keyOf := func(i int) emitKey { return m.userKey(i, start, useOf) }
		first := slices.MinFunc(m.plain[typeIdx], func(a, b int) int {
			return keyOf(a).compare(keyOf(b))
		})
		usage := use{user: first, at: keyOf(first).at, depth: 0}
		if owner, ok := m.plainOf(first); ok {
			usage.depth = 1 + useOf(owner).depth
		}
		uses[typeIdx] = usage
		return usage
	}
	for typeIdx := range m.plain {
		useOf(typeIdx)
	}
	return uses
}

func (m *model) userKey(i int, start map[int]int, useOf func(int) use) emitKey {
	owner, ok := m.plainOf(i)
	if !ok {
		return emitKey{at: start[i], free: 1, depth: 0, owner: 0, slot: 0}
	}
	usage := useOf(owner)
	slot := slices.Index(m.plainHead(owner), i)
	return emitKey{at: usage.at, free: 0, depth: usage.depth, owner: owner, slot: slot}
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

func headKey(uses map[int]use, typeIdx int) emitKey {
	return emitKey{at: 0, free: 0, depth: uses[typeIdx].depth, owner: typeIdx, slot: 0}
}

func unitStarts(units [][]int, pos func(int) int) map[int]int {
	start := map[int]int{}
	byPos := func(a, b int) int { return pos(a) - pos(b) }
	for _, unit := range units {
		outPos := pos(slices.MinFunc(unit, byPos))
		for _, i := range unit {
			start[i] = outPos
		}
	}
	return start
}

// emitKey orders users that start at the same place the way Fix emits them:
// heads before free users, deeper heads first, then by owner and slot.
type emitKey struct{ at, free, depth, owner, slot int }

func (r emitKey) compare(other emitKey) int {
	return cmp.Or(
		r.at-other.at, r.free-other.free, other.depth-r.depth,
		r.owner-other.owner, r.slot-other.slot,
	)
}
