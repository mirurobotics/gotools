package funcorder

import (
	"maps"
	"slices"
)

func (m *model) tuckTypes(out, roots []int, below map[int]bool) []int {
	trees := splitTrees(out, roots)
	isFunc := func(i int) bool { return m.fns[i] != nil }
	for _, typeIdx := range slices.Sorted(maps.Keys(m.held)) {
		from := treeOf(trees, typeIdx)
		last := m.lastUserTree(trees, typeIdx, from)
		if last < 0 {
			continue
		}
		end := last + 1
		for end < from && !slices.ContainsFunc(trees[end], isFunc) {
			end++
		}
		tree := trees[from]
		below[tree[slices.IndexFunc(tree, isFunc)]] = true
		merged := slices.Concat(slices.Concat(trees[last:end]...), tree)
		trees = slices.Replace(slices.Delete(trees, from, from+1), last, end, merged)
	}
	return slices.Concat(trees...)
}

func (m *model) lastUserTree(trees [][]int, typeIdx, from int) int {
	last := -1
	for _, user := range m.held[typeIdx] {
		isMethod := m.fns[user] != nil && m.fns[user].recv != ""
		if userTree := treeOf(trees, user); !isMethod && userTree < from {
			last = max(last, userTree)
		}
	}
	return last
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
