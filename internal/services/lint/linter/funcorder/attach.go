package funcorder

import (
	"go/ast"
	"go/token"
	"maps"
	"slices"
)

func (m *model) attachToTypes() map[int]int {
	a := m.newAttacher(map[int]int{})
	a.claimHeads()
	a.claimTypeConsts()
	return a.owner
}

func (m *model) attach(settled bool) map[int]int {
	a := m.newAttacher(maps.Clone(m.typeOwner))
	if settled {
		a.releaseInitVars()
	}
	a.claimLeads(settled)
	return a.owner
}

func (m *model) newAttacher(owner map[int]int) *attacher {
	return &attacher{
		m:     m,
		owner: owner,
		first: slices.IndexFunc(m.fns, func(x *fn) bool { return x != nil }),
	}
}

func (m *model) declTok(i int) token.Token {
	if gd, ok := m.decls[i].(*ast.GenDecl); ok {
		return gd.Tok
	}
	return token.ILLEGAL
}

type attacher struct {
	m     *model
	owner map[int]int
	first int
}

func (a *attacher) claimHeads() {
	inHead := func(i int) bool {
		return a.m.fns[i] == nil && a.m.declTok(i) != token.TYPE
	}
	for _, x := range a.m.funcs() {
		t, ok := a.m.typeDecl[x.recv]
		if !ok || !a.isFree(t) || soloTypeSpec(a.m.decls[t]) == nil {
			continue
		}
		a.owner[t] = x.idx
		a.claimBelow(t, x.idx, inHead)
	}
}

func (a *attacher) claimTypeConsts() {
	plain := slices.Sorted(maps.Keys(a.m.plain))
	for _, t := range plain {
		a.claimBelow(t, t, a.typeConst(t, true))
	}
	for _, t := range plain {
		a.claimAbove(t, a.typeConst(t, false))
	}
}

func (a *attacher) claimBelow(from, owner int, ok func(int) bool) {
	for i := from + 1; i < len(a.m.decls) && a.isFree(i) && ok(i); i++ {
		a.owner[i] = owner
	}
}

func (a *attacher) typeConst(t int, usesType bool) func(int) bool {
	ts := soloTypeSpec(a.m.decls[t])
	return func(i int) bool {
		isConst := a.m.declTok(i) == token.CONST
		return isConst && refersTo(a.m.decls[i], ts) == usesType
	}
}

func (a *attacher) releaseInitVars() {
	for _, x := range a.m.funcs() {
		if t, ok := a.m.typeDecl[x.recv]; ok && owns(a.owner, x.idx, t) {
			a.release(t, x.idx)
		}
	}
}

func (a *attacher) release(t, f int) {
	if owns(a.owner, f, f-1) {
		return
	}
	released := false
	for i := t + 1; owns(a.owner, f, i); i++ {
		released = released || a.m.initVars[i]
		if released {
			delete(a.owner, i)
		}
	}
}

func (a *attacher) claimLeads(settled bool) {
	leads := func(i int) bool {
		tok := a.m.declTok(i)
		isBlock := tok == token.CONST || tok == token.VAR
		return isBlock && (!settled || !a.m.initVars[i])
	}
	for _, x := range a.m.funcs() {
		if x.recv == "" {
			a.claimAbove(x.idx, leads)
		}
	}
}

func (a *attacher) claimAbove(owner int, ok func(int) bool) {
	for i := owner - 1; i > a.first && a.isFree(i) && ok(i); i-- {
		a.owner[i] = owner
	}
}

func (a *attacher) isFree(i int) bool {
	_, owned := a.owner[i]
	return !owned
}

func owns(owner map[int]int, o, i int) bool {
	got, ok := owner[i]
	return ok && got == o
}

func attached(owner map[int]int) map[int][]int {
	blocks := map[int][]int{}
	for _, i := range slices.Sorted(maps.Keys(owner)) {
		blocks[owner[i]] = append(blocks[owner[i]], i)
	}
	return blocks
}
