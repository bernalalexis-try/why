package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kavix/why/internal/model"
)

// ANSI color codes with NO_COLOR awareness
var (
	green  = "\033[32m"
	red    = "\033[31m"
	yellow = "\033[33m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	cyan   = "\033[36m"
	reset  = "\033[0m"
)

func init() {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		green = ""
		red = ""
		yellow = ""
		bold = ""
		dim = ""
		cyan = ""
		reset = ""
	}
}

// Print formats and outputs the diagnostic based on user flags.
func Print(w io.Writer, diag *model.Diagnostic, isJSON bool, isExplain bool) error {
	if isJSON {
		jsonStr, err := diag.ToJSON()
		if err != nil {
			return err
		}
		fmt.Fprintln(w, jsonStr)
		return nil
	}

	renderTerminal(w, diag, isExplain)
	return nil
}

func renderTerminal(w io.Writer, diag *model.Diagnostic, isExplain bool) {
	// Status Header
	protocolUpper := strings.ToUpper(diag.Protocol)
	if diag.Status == "passed" {
		fmt.Fprintf(w, "\n%s%s%s %sSUCCESS%s  %s(%s)%s\n\n", bold, green, "✓", protocolUpper, reset, dim, diag.TotalElapsed, reset)
	} else {
		fmt.Fprintf(w, "\n%s%s%s %sFAILED%s  %s(failed at %s in %s)%s\n\n", bold, red, "✗", protocolUpper, reset, dim, diag.FailedAt, diag.TotalElapsed, reset)
	}

	// Stages Checklist
	for _, check := range diag.Checks {
		symbol := fmt.Sprintf("%s✓%s", green, reset)
		summary := check.Summary

		switch check.Status {
		case model.StatusFailed:
			symbol = fmt.Sprintf("%s✗%s", red, reset)
			if check.Error != "" {
				summary = fmt.Sprintf("%s%s%s", red, check.Error, reset)
			}
		case model.StatusWarning:
			symbol = fmt.Sprintf("%s⚠%s", yellow, reset)
			if check.Error != "" {
				summary = fmt.Sprintf("%s%s%s", yellow, check.Error, reset)
			}
		case model.StatusSkipped:
			symbol = fmt.Sprintf("%s-%s", dim, reset)
		}

		if summary != "" {
			fmt.Fprintf(w, "%s %-28s %s%s%s\n", symbol, check.Name, dim, summary, reset)
		} else {
			fmt.Fprintf(w, "%s %-28s %s(%s)%s\n", symbol, check.Name, dim, check.DurationStr, reset)
		}
	}
	fmt.Fprintln(w)

	// Cause & Evidence Section
	if diag.Status == "failed" {
		fmt.Fprintf(w, "%sCause%s\n", bold, reset)
		fmt.Fprintf(w, "%s\n", strings.Repeat("─", 40))

		if len(diag.Causes) > 0 {
			for _, c := range diag.Causes {
				fmt.Fprintf(w, "%s%s%s\n\n", bold, c.Explanation, reset)

				if len(c.Evidence) > 0 {
					for k, v := range c.Evidence {
						fmt.Fprintf(w, "%s%s:%s\n  %v\n", dim, strings.ReplaceAll(k, "_", " "), reset, v)
					}
					fmt.Fprintln(w)
				}

				if len(c.Remediation) > 0 {
					fmt.Fprintf(w, "%sLikely causes & fixes:%s\n", bold, reset)
					for _, r := range c.Remediation {
						fmt.Fprintf(w, "  • %s\n", r)
					}
					fmt.Fprintln(w)
				}
			}
		} else if diag.Failure != nil {
			fmt.Fprintf(w, "%s\n\n", diag.Failure.Reason)
		}

		// Suggestions / Next Steps
		fmt.Fprintf(w, "%sTry:%s\n", bold, reset)
		fmt.Fprintf(w, "  why %s --deep\n", diag.Target)
		if diag.AIAnalysis == "" {
			fmt.Fprintf(w, "  why %s --ai\n", diag.Target)
		}
		fmt.Fprintf(w, "  why %s --json\n", diag.Target)
		fmt.Fprintln(w)
	}

	// AI Analysis Section if present
	if diag.AIAnalysis != "" {
		fmt.Fprintf(w, "%s%s✦ %s%s\n", bold, cyan, diag.AIAnalysis, reset)
		fmt.Fprintln(w)
	}
}
