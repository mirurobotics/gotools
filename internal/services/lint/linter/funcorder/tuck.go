package funcorder

import (
	"fmt"
	"maps"
	"slices"
)

func (m *model) tuckTypes(out, roots []int) ([]int, map[int]int) {
	trees := splitTrees(out, roots)
	tucked := map[int]int{}
	for _, typeIdx := range slices.Sorted(maps.Keys(m.held)) {
		from := treeOf(trees, typeIdx)
		user, last := m.lastFreeUser(trees, typeIdx, from)
		if last < 0 {
			continue
		}
		end := m.declsEnd(trees, last+1, from)
		merged := slices.Concat(slices.Concat(trees[last:end]...), trees[from])
		trees = slices.Replace(slices.Delete(trees, from, from+1), last, end, merged)
		tucked[typeIdx] = user
	}
	return slices.Concat(trees...), tucked
}

func (m *model) lastFreeUser(trees [][]int, typeIdx, from int) (int, int) {
	user, last := -1, -1
	for _, candidate := range m.held[typeIdx] {
		userTree := treeOf(trees, candidate)
		isMethod := m.fns[candidate] != nil && m.fns[candidate].recv != ""
		if !isMethod && userTree < from && userTree >= last {
			user, last = candidate, userTree
		}
	}
	return user, last
}

func (m *model) declsEnd(trees [][]int, start, stop int) int {
	isFunc := func(i int) bool { return m.fns[i] != nil }
	for start < stop && !slices.ContainsFunc(trees[start], isFunc) {
		start++
	}
	return start
}

func (m *model) heldMessage(typeIdx, user int, order []int) string {
	pos, prev := slices.Index(order, typeIdx)-1, typeIdx-1
	for pos >= 0 && m.inPlain(order[pos]) {
		pos--
	}
	for prev >= 0 && m.inPlain(prev) {
		prev--
	}
	if pos < 0 || order[pos] == prev {
		return ""
	}
	name := soloTypeSpec(m.decls[typeIdx]).Name.Name
	if order[pos] == user {
		const msg = "type %q should be directly below its user %q"
		return fmt.Sprintf(msg, name, m.declName(user))
	}
	const msg = "type %q should be directly below %q, which follows its user %q"
	return fmt.Sprintf(msg, name, m.declName(order[pos]), m.declName(user))
}

func splitTrees(out, roots []int) [][]int {
	trees := make([][]int, len(roots))
	for treeIdx, start := range roots {
		end := len(out)
		if treeIdx+1 < len(roots) {
			end = roots[treeIdx+1]
		}
		trees[treeIdx] = out[start:end]
	}
	return trees
}

func treeOf(trees [][]int, i int) int {
	inTree := func(tree []int) bool { return slices.Contains(tree, i) }
	return slices.IndexFunc(trees, inTree)
}
