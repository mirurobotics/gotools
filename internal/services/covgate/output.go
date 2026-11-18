package covgate

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/mirurobotics/gotools/internal/services/gocover"
)

// writeRunHeader writes the announce line and the progress table
// header when progress is enabled; otherwise it writes the
// standard table header.
func (r *runner) writeRunHeader(w io.Writer, total, parallelism int) {
	if !r.emitProgress {
		printHeader(w)
		return
	}
	_, _ = fmt.Fprintf(
		w, "Running %d packages with parallelism=%d; progress:\n",
		total, parallelism,
	)
	printProgressHeader(w, total)
}

// progressColWidth returns the width of the COUNT column for total
// packages. The value 3 + 2*ndigits is always >= 5 ("COUNT") for
// total >= 0, so no clamp is needed.
func progressColWidth(total int) int { return 3 + 2*len(strconv.Itoa(total)) }

// firstLine returns the first line of s, including the trailing
// newline. s must contain at least one '\n' — every checkResult's
// output is built from a "%...\n" format string, so this holds.
func firstLine(s string) string { return s[:strings.IndexByte(s, '\n')+1] }

// restOfOutput returns everything after the first '\n' in s. Used
// in progress mode to preserve trailing FAIL detail (multi-line
// raw test output) without reprinting the row that was already
// streamed. Same '\n' precondition as firstLine.
func restOfOutput(s string) string { return s[strings.IndexByte(s, '\n')+1:] }

func (r *runner) printResults(
	w io.Writer,
	results []checkResult,
	excluded []string,
	module string,
	totalTime time.Duration,
) error {
	hasFailures := false
	for _, res := range results {
		if r.emitProgress {
			_, _ = fmt.Fprint(w, restOfOutput(res.output))
		} else {
			_, _ = fmt.Fprint(w, res.output)
		}
		if !res.passed {
			hasFailures = true
		}
	}

	indent := ""
	if r.emitProgress {
		indent = strings.Repeat(" ", progressColWidth(len(results))) + "  "
	}
	for _, pkg := range excluded {
		row := skippedRow(gocover.RelPkg(pkg, module))
		_, _ = fmt.Fprintf(w, "%s%s", indent, row)
	}

	_, _ = fmt.Fprintf(w, "\nTotal time: %s\n", fmtDuration(totalTime))
	if hasFailures {
		_, _ = fmt.Fprintln(
			w, "ERROR: One or more packages failed "+
				"tests, are below their minimum coverage, "+
				"or have loose .covgate thresholds",
		)
		return fmt.Errorf("coverage gate failed")
	}

	_, _ = fmt.Fprintln(w, "All packages meet minimum coverage requirement")
	return nil
}

func printHeader(w io.Writer) {
	_, _ = fmt.Fprintf(
		w, "%-7s  %8s  %8s  %8s  %s\n",
		"STATUS", "COVERAGE", "REQUIRED", "TIME", "PACKAGE",
	)
	_, _ = fmt.Fprintf(
		w, "%-7s  %8s  %8s  %8s  %s\n",
		"-------", "--------", "--------", "--------", "-------",
	)
}

// printProgressHeader prints the table header used during the live
// progress stream. It adds a leading COUNT column ahead of the
// standard columns so each progress line carries an [N/total] tag
// aligned under "COUNT".
func printProgressHeader(w io.Writer, total int) {
	cw := progressColWidth(total)
	dashes := strings.Repeat("-", cw)
	_, _ = fmt.Fprintf(
		w, "%-*s  %-7s  %8s  %8s  %8s  %s\n",
		cw, "COUNT",
		"STATUS", "COVERAGE", "REQUIRED", "TIME", "PACKAGE",
	)
	_, _ = fmt.Fprintf(
		w, "%-*s  %-7s  %8s  %8s  %8s  %s\n",
		cw, dashes,
		"-------", "--------", "--------", "--------", "-------",
	)
}

// skippedRow formats a single SKIPPED row using the same column
// widths as PASS/FAIL/LOOSE. relPkg is the module-relative import
// path. Used for packages removed by --exclude so the user can see
// which ones were skipped.
func skippedRow(relPkg string) string {
	return fmt.Sprintf(
		"%-7s  %8s  %8s  %8s  %s\n",
		"SKIPPED", "---", "---", "---", relPkg,
	)
}

func fmtDuration(d time.Duration) string {
	d = d.Round(100 * time.Millisecond)
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := d - time.Duration(m)*time.Minute
	return fmt.Sprintf("%dm%02.0fs", m, s.Seconds())
}
