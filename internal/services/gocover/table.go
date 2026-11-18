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

// Progress prints per-package rows as they finish when Enabled, else a plain table.
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
	cw := ProgressColWidth(total)
	format := "%-*s  " + p.Table.Format
	names := append([]any{cw, "COUNT"}, strArgs(p.Table.Names)...)
	dashes := append([]any{cw, strings.Repeat("-", cw)}, strArgs(p.Table.Dashes)...)
	_, _ = fmt.Fprintf(w, format, names...)
	_, _ = fmt.Fprintf(w, format, dashes...)
}

// RunEach calls run for each index below n, at most parallelism at a time.
func RunEach[T any](
	w io.Writer,
	progress bool,
	n, parallelism int,
	run func(int) T,
	row func(T) string,
) []T {
	results := make([]T, n)
	sem := make(chan struct{}, parallelism)
	var wg sync.WaitGroup
	var mu sync.Mutex
	countWidth := len(strconv.Itoa(n))
	colWidth := ProgressColWidth(n)
	for i := range n {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = run(idx)
			if progress {
				label := fmt.Sprintf("[%*d/%d]", countWidth, idx+1, n)
				line := FirstLine(row(results[idx]))
				mu.Lock()
				_, _ = fmt.Fprintf(w, "%-*s  %s", colWidth, label, line)
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	return results
}

// ProgressColWidth returns the width of the COUNT column for total packages.
func ProgressColWidth(total int) int { return 3 + 2*len(strconv.Itoa(total)) }

// FirstLine returns the first line of s, including its trailing newline.
func FirstLine(s string) string { return s[:strings.IndexByte(s, '\n')+1] }

func strArgs(ss []string) []any {
	args := make([]any, len(ss))
	for i, s := range ss {
		args[i] = s
	}
	return args
}
