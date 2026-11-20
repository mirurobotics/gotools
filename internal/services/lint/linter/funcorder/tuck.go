package funcorder

import (
	"maps"
	"slices"
)

func (l *layout) tuckTypes() {
	isFunc := func(i int) bool { return l.m.fns[i] != nil }
	for _, typeIdx := range slices.Sorted(maps.Keys(l.m.held)) {
		from := treeOf(l.trees, typeIdx)
		last := l.m.lastUserTree(l.trees, typeIdx, from)
		if last < 0 {
			continue
		}
		end := last + 1
		for end < from && !slices.ContainsFunc(l.trees[end], isFunc) {
			end++
		}
		tree := l.trees[from]
		l.below[tree[slices.IndexFunc(tree, isFunc)]] = true
		merged := slices.Concat(slices.Concat(l.trees[last:end]...), tree)
		l.trees = slices.Delete(l.trees, from, from+1)
		l.trees = slices.Replace(l.trees, last, end, merged)
	}
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

func treeOf(trees [][]int, i int) int {
	inTree := func(tree []int) bool { return slices.Contains(tree, i) }
	return slices.IndexFunc(trees, inTree)
}
