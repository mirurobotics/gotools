package funcorder

import (
	"maps"
	"slices"
)

func (l *layout) tuckTypes() {
	for l.tuckOnce() {
	}
}

func (l *layout) tuckOnce() bool {
	isFunc := func(i int) bool { return l.m.fns[i] != nil }
	for _, typeIdx := range slices.Sorted(maps.Keys(l.m.tucked)) {
		from := treeOf(l.trees, typeIdx)
		last, tree := l.m.lastUserTree(l.trees, typeIdx, from), l.trees[from]
		if !l.isTuckable(tree, last, from) {
			continue
		}
		l.below[tree[slices.IndexFunc(tree, isFunc)]] = true
		l.trees = slices.Delete(l.trees, from, from+1)
		l.trees[last] = slices.Concat(l.trees[last], tree)
		return true
	}
	return false
}

func (l *layout) isTuckable(tree []int, last, from int) bool {
	isInitVar := func(i int) bool { return l.m.initVars[i] }
	if last < 0 || slices.ContainsFunc(tree, isInitVar) {
		return false
	}
	return !l.hasCallerBetween(tree, last, from)
}

func (l *layout) hasCallerBetween(tree []int, last, from int) bool {
	isBetween := func(caller int) bool {
		callerTree := treeOf(l.trees, caller)
		return callerTree > last && callerTree < from
	}
	return slices.ContainsFunc(tree, func(i int) bool {
		return l.m.fns[i] != nil && slices.ContainsFunc(l.m.fns[i].callers, isBetween)
	})
}

func (m *model) lastUserTree(trees [][]int, typeIdx, from int) int {
	last := -1
	for _, user := range m.tucked[typeIdx] {
		userTree := treeOf(trees, user)
		if userTree >= 0 && userTree < from && m.isFreeUser(trees[userTree], user) {
			last = max(last, userTree)
		}
	}
	return last
}

func (m *model) isFreeUser(tree []int, user int) bool {
	if m.fns[user] == nil {
		isFunc := func(i int) bool { return m.fns[i] != nil }
		root := slices.IndexFunc(tree, isFunc)
		return root >= 0 && m.fns[tree[root]].recv == ""
	}
	return m.fns[user].recv == ""
}

func treeOf(trees [][]int, i int) int {
	inTree := func(tree []int) bool { return slices.Contains(tree, i) }
	return slices.IndexFunc(trees, inTree)
}
