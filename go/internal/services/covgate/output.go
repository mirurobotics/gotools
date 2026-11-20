package covgate

import (
	"fmt"
	"io"
	"time"

	"github.com/mirurobotics/tools/go/internal/services/gocover"
)

func table() gocover.Table {
	return gocover.Table{
		Format: "%-7s  %8s  %8s  %8s  %s\n",
		Names:  []string{"STATUS", "COVERAGE", "REQUIRED", "TIME", "PACKAGE"},
		Dashes: []string{"-------", "--------", "--------", "--------", "-------"},
	}
}

func (r *runner) printResults(
	w io.Writer,
	results []checkResult,
	excluded []string,
	module string,
	totalTime time.Duration,
) error {
	hasFailures := false
	for _, res := range results {
		_, _ = fmt.Fprint(w, r.progress.Rest(res.output))
		if !res.passed {
			hasFailures = true
		}
	}

	indent := r.progress.Indent(len(results))
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
