package gh_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/prunesh/gh/filter"
)

// --- Rewrite ---

func TestRewriteIsNoop(t *testing.T) {
	_, ok := filter.Rewrite([]string{"pr", "list"})
	if ok {
		t.Fatal("Rewrite must never change args for gh")
	}
}

// --- FilterOutput: pr list / issue list / run list ---

func buildLongList(n int) string {
	var sb strings.Builder
	for i := 1; i <= n; i++ {
		sb.WriteString("#")
		sb.WriteString(strings.Repeat("0", 3))
		sb.WriteString("  Some PR title  feature/branch  OPEN  about 1 day ago\n")
	}
	return sb.String()
}

func TestPRListCapsAt20(t *testing.T) {
	input := buildLongList(30)
	out := filter.FilterOutput([]string{"pr", "list"}, input, 0)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	// 20 data lines + 1 truncation notice
	if len(lines) != 21 {
		t.Fatalf("expected 21 lines (20 + notice), got %d", len(lines))
	}
	if !strings.Contains(lines[20], "more") {
		t.Errorf("last line must be a truncation notice, got %q", lines[20])
	}
}

func TestPRListUnderLimitPassthrough(t *testing.T) {
	input := buildLongList(10)
	out := filter.FilterOutput([]string{"pr", "list"}, input, 0)
	if out != input {
		t.Error("list under limit must pass through unchanged")
	}
}

func TestIssueListCapsAt20(t *testing.T) {
	input := buildLongList(25)
	out := filter.FilterOutput([]string{"issue", "list"}, input, 0)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 21 {
		t.Fatalf("expected 21 lines, got %d", len(lines))
	}
}

func TestRunListCapsAt20(t *testing.T) {
	input := buildLongList(22)
	out := filter.FilterOutput([]string{"run", "list"}, input, 0)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 21 {
		t.Fatalf("expected 21 lines, got %d", len(lines))
	}
}

// --- FilterOutput: pr view / issue view ---

const prViewShort = `Fix auth bug
Open • octocat wants to merge 1 commit into main from feature/fix • about 2 hours ago
+5 -3 • No reviews • No checks

This PR fixes the auth bug.

View this pull request on GitHub: https://github.com/owner/repo/pull/123
`

func TestPRViewDropsFooter(t *testing.T) {
	out := filter.FilterOutput([]string{"pr", "view", "123"}, prViewShort, 0)
	if strings.Contains(out, "View this pull request on GitHub") {
		t.Error("filtered pr view must not contain footer URL line")
	}
}

func TestPRViewKeepsTitle(t *testing.T) {
	out := filter.FilterOutput([]string{"pr", "view", "123"}, prViewShort, 0)
	if !strings.Contains(out, "Fix auth bug") {
		t.Error("filtered pr view must keep title")
	}
}

func TestPRViewKeepsBody(t *testing.T) {
	out := filter.FilterOutput([]string{"pr", "view", "123"}, prViewShort, 0)
	if !strings.Contains(out, "This PR fixes the auth bug.") {
		t.Error("filtered pr view must keep body content")
	}
}

func buildLongPRView(bodyLines int) string {
	var sb strings.Builder
	sb.WriteString("A long PR title\n")
	sb.WriteString("Open • author wants to merge 1 commit into main from feature/x • about 1 day ago\n")
	sb.WriteString("+100 -50 • No reviews\n")
	sb.WriteString("\n")
	for i := 0; i < bodyLines; i++ {
		sb.WriteString("Body line number ")
		sb.WriteString(strings.Repeat("a", 60))
		sb.WriteString("\n")
	}
	sb.WriteString("\nView this pull request on GitHub: https://github.com/owner/repo/pull/456\n")
	return sb.String()
}

func TestPRViewTruncatesLongBody(t *testing.T) {
	input := buildLongPRView(80)
	out := filter.FilterOutput([]string{"pr", "view", "456"}, input, 0)
	if len(out) >= len(input) {
		t.Errorf("filtered pr view (%d bytes) must be smaller than original (%d bytes)", len(out), len(input))
	}
	if !strings.Contains(out, "truncated") {
		t.Error("truncation notice must appear when body is capped")
	}
}

func TestPRViewSmallerThanOriginal(t *testing.T) {
	input := buildLongPRView(50)
	out := filter.FilterOutput([]string{"pr", "view", "456"}, input, 0)
	if len(out) >= len(input) {
		t.Errorf("filtered pr view must be smaller: got %d, want < %d", len(out), len(input))
	}
}

func TestIssueViewDropsFooter(t *testing.T) {
	const issueView = `Fix the login bug
Open • octocat opened this issue about 1 day ago • 3 comments

Description of the bug goes here.

View this issue on GitHub: https://github.com/owner/repo/issues/99
`
	out := filter.FilterOutput([]string{"issue", "view", "99"}, issueView, 0)
	if strings.Contains(out, "View this issue on GitHub") {
		t.Error("filtered issue view must not contain footer URL line")
	}
	if !strings.Contains(out, "Description of the bug") {
		t.Error("filtered issue view must keep body")
	}
}

// --- FilterOutput: pr checks ---

const prChecksOutput = `All checks were successful
0 failing, 3 successful, and 0 pending checks

build  pass  1m22s  https://github.com/owner/repo/actions/runs/123/job/456
test   pass  2m14s  https://github.com/owner/repo/actions/runs/123/job/457
lint   pass  0m45s  https://github.com/owner/repo/actions/runs/123/job/458
`

func TestPRChecksStripsURLsFromPassing(t *testing.T) {
	out := filter.FilterOutput([]string{"pr", "checks"}, prChecksOutput, 0)
	if strings.Contains(out, "https://") {
		t.Error("filtered pr checks must not contain URLs for passing checks")
	}
}

func TestPRChecksKeepsSummary(t *testing.T) {
	out := filter.FilterOutput([]string{"pr", "checks"}, prChecksOutput, 0)
	if !strings.Contains(out, "All checks were successful") {
		t.Error("filtered pr checks must keep summary line")
	}
}

func TestPRChecksKeepsCheckNames(t *testing.T) {
	out := filter.FilterOutput([]string{"pr", "checks"}, prChecksOutput, 0)
	for _, name := range []string{"build", "test", "lint"} {
		if !strings.Contains(out, name) {
			t.Errorf("filtered pr checks must keep check name %q", name)
		}
	}
}

func TestPRChecksKeepsFailingWithURL(t *testing.T) {
	const withFail = `1 failing, 2 successful, and 0 pending checks

build  pass  1m22s  https://github.com/owner/repo/actions/runs/1/job/1
test   pass  2m14s  https://github.com/owner/repo/actions/runs/1/job/2
deploy fail  0s     https://github.com/owner/repo/actions/runs/1/job/3
`
	out := filter.FilterOutput([]string{"pr", "checks"}, withFail, 0)
	// Failing check must keep its URL (agent may need it to diagnose)
	if !strings.Contains(out, "deploy") {
		t.Error("filtered pr checks must keep failing check name")
	}
	// Passing checks must have URLs stripped
	if strings.Contains(out, "actions/runs/1/job/1") {
		t.Error("filtered pr checks must strip URL for passing build check")
	}
}

func TestPRChecksIsSmaller(t *testing.T) {
	out := filter.FilterOutput([]string{"pr", "checks"}, prChecksOutput, 0)
	if len(out) >= len(prChecksOutput) {
		t.Errorf("filtered pr checks (%d bytes) must be smaller than original (%d bytes)", len(out), len(prChecksOutput))
	}
}

// --- FilterOutput: run view ---

const runViewOutput = `✓ CI · 9876543210
Triggered via push about 1 hour ago

JOBS
✓ build (ID 1234567890)
✓ test (ID 1234567891)
✗ deploy (ID 1234567892)

For more information about a failed job, try: gh run view --log-failed 9876543210
View this run on GitHub: https://github.com/owner/repo/actions/runs/9876543210
`

func TestRunViewDropsFooter(t *testing.T) {
	out := filter.FilterOutput([]string{"run", "view", "9876543210"}, runViewOutput, 0)
	if strings.Contains(out, "View this run on GitHub") {
		t.Error("filtered run view must not contain footer URL line")
	}
	if strings.Contains(out, "For more information about") {
		t.Error("filtered run view must not contain promotional hint line")
	}
}

func TestRunViewKeepsJobsSummary(t *testing.T) {
	out := filter.FilterOutput([]string{"run", "view", "9876543210"}, runViewOutput, 0)
	if !strings.Contains(out, "JOBS") {
		t.Error("filtered run view must keep JOBS section")
	}
	if !strings.Contains(out, "deploy") {
		t.Error("filtered run view must keep job names")
	}
}

func TestRunViewLogFailedCaps(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 100; i++ {
		sb.WriteString("2026-09-21T10:00:00.000Z deploy  Step failed at line 42\n")
	}
	input := sb.String()
	out := filter.FilterOutput([]string{"run", "view", "--log-failed", "9876543210"}, input, 0)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) > 62 { // 60 log lines + 1 notice + possible trailing
		t.Fatalf("capped run log must have at most 61 lines, got %d", len(lines))
	}
	if !strings.Contains(out, "truncated") {
		t.Error("truncation notice must appear when log is capped")
	}
}

// --- passthrough cases ---

func TestJSONFlagPassthrough(t *testing.T) {
	input := `{"number":123,"title":"Fix auth"}`
	out := filter.FilterOutput([]string{"pr", "view", "--json", "number,title"}, input, 0)
	if out != input {
		t.Error("--json output must pass through unchanged")
	}
}

func TestErrorOutputPassthrough(t *testing.T) {
	input := "gh: pull request not found\n"
	out := filter.FilterOutput([]string{"pr", "view", "999"}, input, 1)
	if out != input {
		t.Error("non-zero exit code output must pass through unchanged")
	}
}

func TestUnknownSubcommandPassthrough(t *testing.T) {
	input := "some output\n"
	out := filter.FilterOutput([]string{"auth", "status"}, input, 0)
	if out != input {
		t.Error("unknown subcommand must pass through unchanged")
	}
}

// --- ID constant ---

func TestID(t *testing.T) {
	if filter.ID != "prunesh/gh" {
		t.Fatalf("ID %q does not follow author/<cmd> rule", filter.ID)
	}
}

// --- prunesh.json manifest ---

func TestManifest(t *testing.T) {
	data, err := os.ReadFile("prunesh.json")
	if err != nil {
		t.Fatalf("read prunesh.json: %v", err)
	}

	var manifest struct {
		ID      string   `json:"id"`
		Command string   `json:"command"`
		Platforms []string `json:"platforms"`
		Contract string   `json:"contract"`
		PruneshCoreVersion struct {
			Version    string `json:"version"`
			Constraint string `json:"constraint"`
		} `json:"prunesh-core-version"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse prunesh.json: %v", err)
	}
	if manifest.ID != filter.ID {
		t.Fatalf("manifest id %q != code id %q", manifest.ID, filter.ID)
	}
	if manifest.Command != filter.Command {
		t.Fatalf("manifest command %q != code command %q", manifest.Command, filter.Command)
	}
	if manifest.Contract != "stdin/v1" {
		t.Fatalf("unexpected contract: %q", manifest.Contract)
	}
	if manifest.PruneshCoreVersion.Version == "" {
		t.Fatal("prunesh-core-version.version must not be empty")
	}
	if manifest.PruneshCoreVersion.Constraint != "min" && manifest.PruneshCoreVersion.Constraint != "exact" {
		t.Fatalf("unexpected prunesh-core-version.constraint: %q", manifest.PruneshCoreVersion.Constraint)
	}
	if len(manifest.Platforms) == 0 {
		t.Fatal("platforms must not be empty")
	}
}
