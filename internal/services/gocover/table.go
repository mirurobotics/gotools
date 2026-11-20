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
func (p Progress) WriteHeader(out io.Writer, total, parallelism int) {
	if !p.Enabled {
		_, _ = fmt.Fprintf(out, p.Table.Format, strArgs(p.Table.Names)...)
		_, _ = fmt.Fprintf(out, p.Table.Format, strArgs(p.Table.Dashes)...)
		return
	}
	_, _ = fmt.Fprintf(
		out, "Running %d packages with parallelism=%d; progress:\n",
		total, parallelism,
	)
	colWidth := progressColWidth(total)
	format := "%-*s  " + p.Table.Format
	names := append([]any{colWidth, "COUNT"}, strArgs(p.Table.Names)...)
	dashes := append(
		[]any{colWidth, strings.Repeat("-", colWidth)}, strArgs(p.Table.Dashes)...,
	)
	_, _ = fmt.Fprintf(out, format, names...)
	_, _ = fmt.Fprintf(out, format, dashes...)
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

func strArgs(strs []string) []any {
	args := make([]any, len(strs))
	for i, str := range strs {
		args[i] = str
	}
	return args
}

// RunEach runs run(0..count-1), parallelism at a time, streaming rows when enabled.
func RunEach[T any](
	progress Progress,
	out io.Writer,
	count, parallelism int,
	run func(int) T,
	row func(T) string,
) []T {
	parallelism = Parallelism(parallelism)
	results := make([]T, count)
	sem := make(chan struct{}, parallelism)
	var wg sync.WaitGroup
	var mu sync.Mutex
	countWidth := len(strconv.Itoa(count))
	colWidth := progressColWidth(count)
	for i := range count {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = run(idx)
			if progress.Enabled {
				label := fmt.Sprintf("[%*d/%d]", countWidth, idx+1, count)
				line := firstLine(row(results[idx]))
				mu.Lock()
				_, _ = fmt.Fprintf(out, "%-*s  %s", colWidth, label, line)
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	return results
}

func progressColWidth(total int) int { return 3 + 2*len(strconv.Itoa(total)) }

func firstLine(text string) string { return text[:strings.IndexByte(text, '\n')+1] }
