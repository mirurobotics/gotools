package gocover

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

// Table describes the header of a per-package results table.
type Table struct {
	Format string
	Names  []string
	Dashes []string
}

// Progress is the output mode and header layout for a per-package run.
type Progress struct {
	Enabled bool
	Table   Table
}

// WriteHeader writes the run announcement and table header for total packages.
func (p Progress) WriteHeader(w io.Writer, total, parallelism int) {
	if !p.Enabled {
		_, _ = fmt.Fprintf(w, p.Table.Format, strArgs(p.Table.Names)...)
		_, _ = fmt.Fprintf(w, p.Table.Format, strArgs(p.Table.Dashes)...)
		return
	}
	_, _ = fmt.Fprintf(
		w, "Running %d packages with parallelism=%d; progress:\n",
		total, parallelism,
	)
	cw := progressColWidth(total)
	format := "%-*s  " + p.Table.Format
	names := append([]any{cw, "COUNT"}, strArgs(p.Table.Names)...)
	dashes := append([]any{cw, strings.Repeat("-", cw)}, strArgs(p.Table.Dashes)...)
	_, _ = fmt.Fprintf(w, format, names...)
	_, _ = fmt.Fprintf(w, format, dashes...)
}

// Rest returns the part of output not already printed by RunEach.
func (p Progress) Rest(output string) string {
	if !p.Enabled {
		return output
	}
	return output[strings.IndexByte(output, '\n')+1:]
}

// Indent pads an extra row past the COUNT column to line up with progress rows.
func (p Progress) Indent(total int) string {
	if !p.Enabled {
		return ""
	}
	return strings.Repeat(" ", progressColWidth(total)) + "  "
}

// RunEach runs run(0..n-1), parallelism at a time, streaming row(result) if p.Enabled.
func RunEach[T any](
	p Progress,
	w io.Writer,
	n, parallelism int,
	run func(int) T,
	row func(T) string,
) []T {
	parallelism = Parallelism(parallelism)
	results := make([]T, n)
	sem := make(chan struct{}, parallelism)
	var wg sync.WaitGroup
	var mu sync.Mutex
	countWidth := len(strconv.Itoa(n))
	colWidth := progressColWidth(n)
	for i := range n {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = run(idx)
			if p.Enabled {
				label := fmt.Sprintf("[%*d/%d]", countWidth, idx+1, n)
				line := firstLine(row(results[idx]))
				mu.Lock()
				_, _ = fmt.Fprintf(w, "%-*s  %s", colWidth, label, line)
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	return results
}

func progressColWidth(total int) int { return 3 + 2*len(strconv.Itoa(total)) }

func firstLine(s string) string { return s[:strings.IndexByte(s, '\n')+1] }

func strArgs(ss []string) []any {
	args := make([]any, len(ss))
	for i, s := range ss {
		args[i] = s
	}
	return args
}
