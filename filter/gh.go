// Package filter implements the prunesh/gh filter logic.
//
// Contract:
//   - id:      prunesh/gh
//   - command: gh
//
// Rewrite: no arg rewrites — gh's own --limit defaults are reasonable.
//
// FilterOutput:
//   - pr list / issue list / run list / release list / workflow list:
//     caps output at maxListLines, adding a truncation notice.
//   - pr view / issue view:
//     keeps the compact header, truncates the body at maxBodyLines,
//     drops the "View this … on GitHub" footer.
//   - pr checks:
//     keeps summary and failing/pending checks unchanged;
//     strips the URL column from passing checks.
//   - run view:
//     drops the "View this run on GitHub" footer;
//     when --log or --log-failed is present, caps at maxRunLogLines.
//   - --json requests and error output: full passthrough.
//   - everything else: passthrough.
package filter

import (
	"fmt"
	"strings"
)

const (
	// ID is the full filter identity following the author/<name> rule.
	ID = "prunesh/gh"

	// Command is the argv0 intercepted by this module.
	Command = "gh"

	maxListLines   = 20
	maxBodyLines   = 30
	maxRunLogLines = 60
)

// Rewrite makes no arg changes for gh.
func Rewrite(args []string) ([]string, bool) {
	return nil, false
}

// FilterOutput dispatches by gh subcommand pair.
func FilterOutput(args []string, output string, exitCode int) string {
	if output == "" || exitCode != 0 {
		return output
	}
	// --json requests are for programmatic use; pass through unchanged.
	if hasFlag(args, "--json") {
		return output
	}

	sub1, sub2 := subcmds(args)
	switch sub1 {
	case "pr":
		switch sub2 {
		case "list":
			return capLines(output, maxListLines)
		case "view":
			return filterView(output)
		case "checks":
			return filterPRChecks(output)
		}
	case "issue":
		switch sub2 {
		case "list":
			return capLines(output, maxListLines)
		case "view":
			return filterView(output)
		}
	case "run":
		switch sub2 {
		case "list":
			return capLines(output, maxListLines)
		case "view":
			return filterRunView(args, output)
		}
	case "release":
		if sub2 == "list" {
			return capLines(output, maxListLines)
		}
	case "workflow":
		if sub2 == "list" {
			return capLines(output, maxListLines)
		}
	}
	return output
}

// capLines caps output to at most maxLines lines, appending a notice when truncated.
func capLines(output string, maxLines int) string {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) <= maxLines {
		return output
	}
	kept := lines[:maxLines]
	dropped := len(lines) - maxLines
	kept = append(kept, fmt.Sprintf("... (%d more, use --limit to see all)", dropped))
	return strings.Join(kept, "\n") + "\n"
}

// filterView filters `gh pr view` / `gh issue view` output.
// Header lines (before the first blank line) are kept in full.
// The body is truncated to maxBodyLines. Footer prompts are dropped.
func filterView(output string) string {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	// Locate the first blank line that separates header from body.
	bodyStart := -1
	for i, l := range lines {
		if strings.TrimSpace(l) == "" {
			bodyStart = i + 1
			break
		}
	}
	// No blank line found or body is empty: just drop footer and return.
	if bodyStart < 0 || bodyStart >= len(lines) {
		return strings.Join(trimFooterLines(lines), "\n") + "\n"
	}

	header := lines[:bodyStart] // includes the blank separator
	body := trimFooterLines(lines[bodyStart:])

	truncated := false
	if len(body) > maxBodyLines {
		body = body[:maxBodyLines]
		truncated = true
	}

	result := append(header, body...)
	if truncated {
		result = append(result, fmt.Sprintf("... (body truncated, %d lines shown)", maxBodyLines))
	}
	return strings.Join(result, "\n") + "\n"
}

// trimFooterLines removes trailing footer lines from a view body.
// Footer patterns: "View this … on GitHub", "--", "Add a comment".
func trimFooterLines(lines []string) []string {
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "View this") ||
			t == "--" ||
			strings.HasPrefix(t, "-- ") ||
			strings.HasPrefix(t, "Add a comment") {
			return lines[:i]
		}
	}
	return lines
}

// filterPRChecks compresses `gh pr checks` output.
// Failing and pending check lines are kept unchanged.
// For passing checks, the URL column (last field) is stripped.
func filterPRChecks(output string) string {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	var result []string
	for _, line := range lines {
		lower := strings.ToLower(line)
		// Summary lines and non-passing checks pass through unchanged.
		if !strings.Contains(lower, "pass") && !strings.Contains(lower, "success") {
			result = append(result, line)
			continue
		}
		// For passing checks, drop the trailing URL field.
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			last := fields[len(fields)-1]
			if strings.HasPrefix(last, "https://") || strings.HasPrefix(last, "http://") {
				line = strings.Join(fields[:len(fields)-1], "  ")
			}
		}
		result = append(result, line)
	}
	if len(result) == 0 {
		return output
	}
	return strings.Join(result, "\n") + "\n"
}

// capLogLines caps log output to at most maxLines lines with a targeted hint.
func capLogLines(output string, maxLines int) string {
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	if len(lines) <= maxLines {
		return output
	}
	kept := lines[:maxLines]
	dropped := len(lines) - maxLines
	kept = append(kept, fmt.Sprintf("... (%d lines truncated; re-run with --log-failed for targeted output)", dropped))
	return strings.Join(kept, "\n") + "\n"
}

// filterRunView handles `gh run view` output.
// Without log flags: drops the promotional footer lines.
// With --log or --log-failed: caps at maxRunLogLines.
func filterRunView(args []string, output string) string {
	hasLog := hasFlag(args, "--log") || hasFlag(args, "--log-failed")
	if hasLog {
		return capLogLines(output, maxRunLogLines)
	}

	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")
	var result []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "View this run on GitHub") ||
			strings.HasPrefix(t, "For more information about") {
			continue
		}
		result = append(result, l)
	}
	if len(result) == 0 {
		return output
	}
	return strings.Join(result, "\n") + "\n"
}

// --- helpers ---

// subcmds returns the first two positional (non-flag) arguments.
// e.g. ["pr", "view", "123"] → ("pr", "view")
func subcmds(args []string) (string, string) {
	var positional []string
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			positional = append(positional, a)
			if len(positional) == 2 {
				return positional[0], positional[1]
			}
		}
	}
	if len(positional) == 1 {
		return positional[0], ""
	}
	return "", ""
}

// hasFlag reports whether args contains the given flag (exact match or prefix "flag=").
func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag || strings.HasPrefix(a, flag+"=") {
			return true
		}
	}
	return false
}
