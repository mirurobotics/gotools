package analysis

import (
	"go/token"
	"sort"
)

// Candidate is a source range that can be replaced with a single-line form.
type Candidate struct {
	Pos       token.Pos // start position
	End       token.Pos // end position (exclusive)
	Collapsed string    // replacement text
}

// Fix applies candidates to src, replacing each Pos–End span with its Collapsed text.
func Fix(fset *token.FileSet, src []byte, candidates []Candidate) []byte {
	if len(candidates) == 0 {
		return src
	}

	sort.Slice(candidates, func(a, b int) bool {
		iOff := fset.Position(candidates[a].Pos).Offset
		jOff := fset.Position(candidates[b].Pos).Offset
		return iOff > jOff
	})

	candidates = filterOverlapping(fset, candidates)

	for _, cand := range candidates {
		startOff := fset.Position(cand.Pos).Offset
		endOff := fset.Position(cand.End).Offset

		out := make([]byte, 0, startOff+len(cand.Collapsed)+(len(src)-endOff))
		out = append(out, src[:startOff]...)
		out = append(out, cand.Collapsed...)
		out = append(out, src[endOff:]...)
		src = out
	}

	return src
}

func filterOverlapping(fset *token.FileSet, candidates []Candidate) []Candidate {
	if len(candidates) <= 1 {
		return candidates
	}
	kept := make([]Candidate, 0, len(candidates))
	for _, cand := range candidates {
		if !overlapsAny(fset, cand, kept) {
			kept = append(kept, cand)
		}
	}
	return kept
}

func overlapsAny(fset *token.FileSet, cand Candidate, kept []Candidate) bool {
	cStart := fset.Position(cand.Pos).Offset
	cEnd := fset.Position(cand.End).Offset
	for _, candidate := range kept {
		kStart := fset.Position(candidate.Pos).Offset
		kEnd := fset.Position(candidate.End).Offset
		if cStart < kEnd && cEnd > kStart {
			return true
		}
	}
	return false
}

// CheckCandidates converts candidates into diagnostics with the given message.
func CheckCandidates(
	fset *token.FileSet,
	filename string,
	candidates []Candidate,
	message string,
) []Diagnostic {
	diags := make([]Diagnostic, 0, len(candidates))
	for _, cand := range candidates {
		pos := fset.Position(cand.Pos)
		diags = append(diags, Diagnostic{
			File:    filename,
			Line:    pos.Line,
			Message: message,
		})
	}
	return diags
}
