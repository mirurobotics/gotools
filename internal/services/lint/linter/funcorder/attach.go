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
		first: slices.IndexFunc(m.fns, func(fnc *fn) bool { return fnc != nil }),
	}
}

func (m *model) declTok(i int) token.Token {
	if decl, ok := m.decls[i].(*ast.GenDecl); ok {
		return decl.Tok
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
	for _, fnc := range a.m.funcs() {
		typeIdx, ok := a.m.typeDecl[fnc.recv]
		if !ok || !a.isFree(typeIdx) || soloTypeSpec(a.m.decls[typeIdx]) == nil {
			continue
		}
		a.owner[typeIdx] = fnc.idx
		a.claimBelow(typeIdx, fnc.idx, inHead)
	}
}

func (a *attacher) claimBelow(start, owner int, ok func(int) bool) {
	for i := start + 1; i < len(a.m.decls) && a.isFree(i) && ok(i); i++ {
		a.owner[i] = owner
	}
}

func (a *attacher) claimTypeConsts() {
	plain := slices.Sorted(maps.Keys(a.m.plain))
	for _, typeIdx := range plain {
		if !slices.ContainsFunc(a.m.plain[typeIdx], a.m.isConst) {
			a.claimAll(typeIdx, a.ownConst(typeIdx))
		}
	}
	for _, typeIdx := range plain {
		a.claimAbove(typeIdx, -1, a.looseConst(typeIdx))
	}
}

func (a *attacher) claimAll(owner int, ok func(int) bool) {
	for i := range a.m.decls {
		if a.isFree(i) && ok(i) {
			a.owner[i] = owner
		}
	}
}

func (a *attacher) ownConst(typeIdx int) func(int) bool {
	isConst := a.typeConst(typeIdx, true)
	plain := slices.Collect(maps.Keys(a.m.plain))
	return func(i int) bool {
		usesOther := func(other int) bool {
			spec := soloTypeSpec(a.m.decls[other])
			return other != typeIdx && refersTo(a.m.decls[i], spec)
		}
		return isConst(i) && !slices.ContainsFunc(plain, usesOther)
	}
}

func (a *attacher) looseConst(typeIdx int) func(int) bool {
	isConst := a.typeConst(typeIdx, false)
	return func(i int) bool {
		isHeader := i < a.first && !a.isPlainUser(i)
		return isConst(i) && !isHeader && !a.leadsTo(i, typeIdx)
	}
}

func (a *attacher) typeConst(typeIdx int, usesType bool) func(int) bool {
	spec := soloTypeSpec(a.m.decls[typeIdx])
	return func(i int) bool {
		isConst := a.m.declTok(i) == token.CONST
		return isConst && refersTo(a.m.decls[i], spec) == usesType
	}
}

func (a *attacher) isPlainUser(i int) bool {
	for _, users := range a.m.plain {
		if slices.Contains(users, i) {
			return true
		}
	}
	return false
}

func (a *attacher) leadsTo(i, typeIdx int) bool {
	for userType, users := range a.m.plain {
		if !slices.Contains(users, i) {
			continue
		}
		if userType == typeIdx || a.headLeadsTo(userType, typeIdx) {
			return true
		}
	}
	return false
}

func (a *attacher) headLeadsTo(owner, typeIdx int) bool {
	for i, got := range a.owner {
		if got == owner && a.leadsTo(i, typeIdx) {
			return true
		}
	}
	return false
}

func (a *attacher) releaseInitVars() {
	for _, fnc := range a.m.funcs() {
		typeIdx, ok := a.m.typeDecl[fnc.recv]
		if ok && owns(a.owner, fnc.idx, typeIdx) {
			a.release(typeIdx, fnc.idx)
		}
	}
}

func (a *attacher) release(typeIdx, owner int) {
	if owns(a.owner, owner, owner-1) {
		return
	}
	released := false
	for i := typeIdx + 1; owns(a.owner, owner, i); i++ {
		released = released || a.m.initVars[i]
		if released && !a.m.isConst(i) {
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
	for _, fnc := range a.m.funcs() {
		if fnc.recv == "" {
			a.claimAbove(fnc.idx, a.first, leads)
		}
	}
}

func (a *attacher) claimAbove(owner, floor int, ok func(int) bool) {
	for i := owner - 1; i > floor && a.isFree(i) && ok(i); i-- {
		a.owner[i] = owner
	}
}

func (a *attacher) isFree(i int) bool {
	_, owned := a.owner[i]
	return !owned
}

func owns(owner map[int]int, ownerIdx, i int) bool {
	got, ok := owner[i]
	return ok && got == ownerIdx
}

func attached(owner map[int]int) map[int][]int {
	blocks := map[int][]int{}
	for _, i := range slices.Sorted(maps.Keys(owner)) {
		blocks[owner[i]] = append(blocks[owner[i]], i)
	}
	return blocks
}
