# prunesh/gh

Token-reduction plugin for [prunesh](https://github.com/prunesh/prunesh) that filters `gh` (GitHub CLI) output.

`gh pr list` with many open PRs, `gh run view --log-failed` with hundreds of log lines, `gh pr checks` with long CI URLs — all of these generate output that's high-noise for an AI agent. This plugin keeps what's actionable and drops the rest.

## What it filters

| Command | Action |
|---|---|
| `pr list` | Caps at 20 PRs with a truncation notice. |
| `pr view` | Keeps title + status header; truncates body at 30 lines; drops "View this pull request on GitHub" footer. |
| `pr checks` | Keeps summary and failing/pending checks unchanged; strips URL column from passing checks. |
| `issue list` | Caps at 20 issues with a truncation notice. |
| `issue view` | Keeps title + status header; truncates body at 30 lines; drops footer. |
| `run list` | Caps at 20 runs with a truncation notice. |
| `run view` | Drops "View this run on GitHub" and hint lines. |
| `run view --log` / `--log-failed` | Caps log output at 60 lines with a truncation notice. |
| `release list` / `workflow list` | Caps at 20 lines. |
| `--json` requests | Full passthrough — structured output is for programmatic use. |
| Everything else | Full passthrough. |

### Example — pr checks

**Before** (3 passing checks):
```
All checks were successful
0 failing, 3 successful, and 0 pending checks

build  pass  1m22s  https://github.com/owner/repo/actions/runs/123/job/456
test   pass  2m14s  https://github.com/owner/repo/actions/runs/123/job/457
lint   pass  0m45s  https://github.com/owner/repo/actions/runs/123/job/458
```

**After**:
```
All checks were successful
0 failing, 3 successful, and 0 pending checks

build  pass  1m22s
test   pass  2m14s
lint   pass  0m45s
```

URLs for *failing* checks are always kept — those are the ones you need to diagnose.

### Example — run view --log-failed

**Before**: 200+ lines of log output.  
**After**: first 60 lines + `... (N lines truncated; re-run with --log-failed for targeted output)`.

## Install

Requires [prunesh core](https://github.com/prunesh/prunesh) >= 0.12.0.

```bash
prunesh plugin install github.com/prunesh/gh@v0.1.0
```

To replace an existing `gh` plugin:

```bash
prunesh plugin install github.com/prunesh/gh@v0.1.0 --replace
```

## Uninstall

```bash
prunesh plugin uninstall prunesh/gh
```

## License

MIT
