package analysis

import (
	"go/token"
	"sort"
)

// Candidate represents a source range that can be replaced with a collapsed
// single-line form. Used by collapse, funcsig, and funcinline fixers.
type Candidate struct {
	Pos       token.Pos // start position
	End       token.Pos // end position (exclusive)
	Collapsed string    // replacement text
}

// Fix applies candidates to src, replacing each Pos–End span with
// its Collapsed text. Candidates are applied in reverse byte-offset
// order so earlier offsets remain valid.
func Fix(fset *token.FileSet, src []byte, candidates []Candidate) []byte {
	if len(candidates) == 0 {
		return src
	}

	sort.Slice(candidates, func(i, j int) bool {
		iOff := fset.Position(candidates[i].Pos).Offset
		jOff := fset.Position(candidates[j].Pos).Offset
		return iOff > jOff
	})

	// Filter overlapping candidates. Inner (higher-offset)
	// candidates are kept; outer ones that overlap are dropped.
	candidates = filterOverlapping(fset, candidates)

	for _, c := range candidates {
		startOff := fset.Position(c.Pos).Offset
		endOff := fset.Position(c.End).Offset

		out := make([]byte, 0, startOff+len(c.Collapsed)+(len(src)-endOff))
		out = append(out, src[:startOff]...)
		out = append(out, c.Collapsed...)
		out = append(out, src[endOff:]...)
		src = out
	}

	return src
}

// filterOverlapping removes candidates whose byte range overlaps
// with a previously kept candidate. Since candidates are sorted by
// descending start offset, inner (nested) candidates are kept and
// outer candidates that encompass them are dropped.
func filterOverlapping(fset *token.FileSet, candidates []Candidate) []Candidate {
	if len(candidates) <= 1 {
		return candidates
	}
	kept := make([]Candidate, 0, len(candidates))
	for _, c := range candidates {
		cStart := fset.Position(c.Pos).Offset
		cEnd := fset.Position(c.End).Offset
		overlaps := false
		for _, k := range kept {
			kStart := fset.Position(k.Pos).Offset
			kEnd := fset.Position(k.End).Offset
			if cStart < kEnd && cEnd > kStart {
				overlaps = true
				break
			}
		}
		if !overlaps {
			kept = append(kept, c)
		}
	}
	return kept
}

// CheckCandidates converts a slice of candidates into diagnostics,
// using the provided message for each. This is the shared logic
// behind collapse.Check, funcsig.Check, and funcinline.Check.
func CheckCandidates(
	fset *token.FileSet,
	filename string,
	candidates []Candidate,
	message string,
) []Diagnostic {
	diags := make([]Diagnostic, 0, len(candidates))
	for _, c := range candidates {
		pos := fset.Position(c.Pos)
		diags = append(diags, Diagnostic{
			File:    filename,
			Line:    pos.Line,
			Message: message,
		})
	}
	return diags
}
