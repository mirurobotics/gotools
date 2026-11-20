package funcorder

import (
	"fmt"
	"maps"
	"slices"
)

func (m *model) heldTypes() map[int][]int {
	held := map[int][]int{}
	for typeIdx, decl := range m.decls {
		spec := soloTypeSpec(decl)
		isMethod := func(fnc *fn) bool {
			return fnc != nil && fnc.recv == spec.Name.Name
		}
		if spec == nil || !slices.ContainsFunc(m.fns, isMethod) {
			continue
		}
		isMethodUser := func(i int) bool {
			return m.fns[i] != nil && m.fns[i].recv != ""
		}
		users := slices.DeleteFunc(usersOf(spec, m.decls), isMethodUser)
		if len(users) > 0 {
			held[typeIdx] = users
		}
	}
	return held
}

func (m *model) tuckTypes(out, roots []int) []int {
	trees := make([][]int, len(roots))
	for treeIdx, start := range roots {
		end := len(out)
		if treeIdx+1 < len(roots) {
			end = roots[treeIdx+1]
		}
		trees[treeIdx] = out[start:end]
	}
	for _, typeIdx := range slices.Sorted(maps.Keys(m.held)) {
		from, last := treeOf(trees, typeIdx), -1
		for _, user := range m.held[typeIdx] {
			if userTree := treeOf(trees, user); userTree < from {
				last = max(last, userTree)
			}
		}
		if last >= 0 {
			tree := trees[from]
			trees = slices.Delete(trees, from, from+1)
			trees[last] = slices.Concat(trees[last], tree)
		}
	}
	return slices.Concat(trees...)
}

func (m *model) heldMessage(typeIdx int, order []int) string {
	above := slices.DeleteFunc(slices.Clone(m.held[typeIdx]), func(i int) bool {
		return i > typeIdx
	})
	pos, prev := slices.Index(order, typeIdx)-1, typeIdx-1
	for pos >= 0 && m.inPlain(order[pos]) {
		pos--
	}
	for prev >= 0 && m.inPlain(prev) {
		prev--
	}
	if len(above) == 0 || pos < 0 || order[pos] == prev {
		return ""
	}
	name, user := soloTypeSpec(m.decls[typeIdx]).Name.Name, above[len(above)-1]
	if order[pos] == user {
		const msg = "type %q should be directly below its user %q"
		return fmt.Sprintf(msg, name, m.declName(user))
	}
	const msg = "type %q should be directly below %q, which ends its user %q"
	return fmt.Sprintf(msg, name, m.declName(order[pos]), m.declName(user))
}

func treeOf(trees [][]int, i int) int {
	inTree := func(tree []int) bool { return slices.Contains(tree, i) }
	return slices.IndexFunc(trees, inTree)
}
