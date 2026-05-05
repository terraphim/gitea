---
name: Design -- Cherry-pick CSP Script Nonce (82bfde2a37)
description: Phase 2 implementation plan for issue #28. Cherry-pick procedure, conflict resolution, test plan, and acceptance criteria.
type: project
---

# Implementation Plan: Cherry-pick CSP Script Nonce (82bfde2a37)

**Status:** Approved (proceeding to Phase 3)
**Research Doc:** `.docs/research-28.md`
**Author:** Ferrox (Rust Engineer, Principal SFIA-5)
**Date:** 2026-05-04
**Estimated Effort:** 1-2 hours
**Issue:** terraphim/gitea#28
**Reviewers:** @adf:security-sentinel, @adf:gitea-reviewer

---

## Overview

### Summary
Cherry-pick upstream commit `82bfde2a37` (PR #37232) onto `main`, resolving 11 upstream-only evolution-drift conflicts via uniform take-theirs. Verify build, run targeted tests, and open PR.

### Approach
**Single-commit cherry-pick with mechanical conflict resolution.** All conflicts are upstream-only; no fork-touched files. No manual semantic merge required.

### Scope

**In scope:**
- Cherry-pick `82bfde2a37` with `-x` provenance
- Resolve 11 conflicts via `git checkout --theirs`
- Verify `go build ./...` clean
- Run `make fmt && make vet`
- Run targeted tests: markup external, basic view smoke
- Verify Robot API endpoints respond 200

**Out of scope:**
- CSP policy tightening (replace `*` with `self`) -- upstream #37238
- Custom template modifications (fork has none)
- Full integration test suite (deferred to CI)

**Avoid at all cost (5/25):**
- Force-push to PR branch
- Squash commit (loses upstream attribution)
- Manual merge on any conflict file (unnecessary; all upstream-only)
- Mix with non-security changes
- Full upstream merge

---

## Architecture

### Component Diagram
```
upstream commit 82bfde2a37
    |
    v
[cherry-pick -x] --> 11 conflict files (all upstream-only)
    |
    v
[git checkout --theirs -- <file>] for each conflict
    |
    v
[go build ./...] --> verify clean
    |
    v
[make fmt && make vet]
    |
    v
[targeted tests] --> markup external, smoke
    |
    v
[git commit --continue] --> preserves upstream SHA in message
    |
    v
[push + PR]
```

### Key Design Decisions

| Decision | Rationale | Alternatives Rejected |
|----------|-----------|----------------------|
| Uniform take-theirs on all 11 conflicts | All conflict files are upstream-only evolution drift; verified via `git log upstream/main..main -- <file>` | Manual semantic merge (over-effort; no fork changes to preserve) |
| Cherry-pick with `-x` | Preserves upstream provenance in commit message | Plain cherry-pick (loses traceability) |
| Single commit (not squashed) | Matches upstream SHA exactly; reviewer can diff against upstream | Squash (loses attribution) |

### Eliminated Options

| Option Rejected | Why Rejected | Risk of Including |
|-----------------|--------------|-------------------|
| Subset application (skip templates) | Incomplete XSS mitigation | False confidence; nonce missing on some script tags |
| Full upstream merge | 324 commits; 1821-file blast radius | Unbounded review; unrelated changes |
| Custom nonce implementation | Upstream solution is reviewed and merged | Re-inventing; diverges from upstream |

### Simplicity Check

**What if this could be easy?** It is. One cherry-pick, one loop over conflict files, build, test, push.

**Senior engineer test:** Would a senior call this overcomplicated? No. Each step is mechanical.

**Nothing speculative checklist:**
- [x] No features not requested
- [x] No abstractions "in case we need them later"
- [x] No flexibility "just in case"
- [x] No error handling for impossible scenarios
- [x] No premature optimisation

---

## File Changes

### New Files (from upstream)
| File | Purpose |
|------|---------|
| `services/context/context_template.go` | Template context helpers: `CspScriptNonce()`, `ScriptImport()`, `HeadMetaContentSecurityPolicy()` |

### Modified Files (from upstream, 17 files)
| File | Changes |
|------|---------|
| `modules/markup/external/openapi.go` | Add `nonce` attribute to swagger script tag |
| `modules/markup/render.go` | Add `nonce="not-needed"` to helper script injection |
| `modules/templates/helper.go` | Remove legacy `ScriptImport` func; add `AssetURI` |
| `modules/util/util.go` | Add `FastCryptoRandomHex()`, `chaCha8RandPool` |
| `services/context/context.go` | Move `TemplateContext` type to `context_template.go` |
| `templates/base/footer.tmpl` | Switch to `ctx.ScriptImport` for main JS entry |
| `templates/base/head.tmpl` | Inject CSP meta tag |
| `templates/base/head_script.tmpl` | Add nonce to global inline config script |
| `templates/repo/diff/box.tmpl` | Add nonces to inline scripts |
| `templates/repo/issue/view_content/pull_merge_box.tmpl` | Add nonce to inline module script |
| `templates/shared/combomarkdowneditor.tmpl` | Add nonce to inline script |
| `templates/status/500.tmpl` | Add CSP meta and nonce to error page |
| `templates/swagger/openapi-viewer.tmpl` | Add CSP meta; switch to `ctx.ScriptImport` |
| `templates/user/auth/captcha.tmpl` | Add nonce to captcha provider scripts |
| `templates/user/dashboard/repolist.tmpl` | Add nonce to inline module script |
| `tests/integration/markup_external_test.go` | Update expected HTML for nonce attribute |
| `web_src/js/features/repo-issue-pull.ts` | Propagate nonce to dynamically executed scripts |

### Deleted Files
None.

---

## API Design

### Public Types (upstream, preserved verbatim)
```go
// services/context/context_template.go

func (c TemplateContext) CspScriptNonce() string
func (c TemplateContext) ScriptImport(path string, typ ...string) template.HTML
func (c TemplateContext) HeadMetaContentSecurityPolicy() template.HTML
```

### Public Functions (upstream, preserved verbatim)
```go
// modules/util/util.go

func FastCryptoRandomHex(length int) string
```

### Template Usage (upstream pattern)
```html
<script nonce="{{ctx.CspScriptNonce}}">
  // inline script
</script>

{{ctx.ScriptImport "js/index.js" "module"}}
```

---

## Test Strategy

### Build Verification
| Check | Command | Purpose |
|-------|---------|---------|
| Go build | `go build ./...` | No compilation errors |
| Format | `make fmt` | No formatting violations |
| Vet | `make vet` | No static analysis issues |

### Integration Tests
| Test | Command | Purpose |
|------|---------|---------|
| Markup external | `go test -tags sqlite,sqlite_unlock_notify ./tests/integration -run TestMarkupExternal` | Verify nonce attribute in rendered HTML |

### Smoke Tests
| Test | Method | Purpose |
|------|--------|---------|
| Nonce presence | Boot `gitea web`, view page source, grep for `nonce=` | Confirm nonce injected into scripts |
| Robot API | `curl http://localhost:3000/api/v1/robot/triage` | Verify Robot endpoints unaffected |

### Acceptance Criteria
- [ ] `go build ./...` clean
- [ ] `make fmt && make vet` clean
- [ ] `git log --oneline -1` shows cherry-pick with `-x` footer referencing `82bfde2a37`
- [ ] `grep -r '<<<<<<<' --include='*.go' --include='*.tmpl' --include='*.ts' .` returns nothing
- [ ] Markup external integration tests pass
- [ ] Rendered page contains `<script nonce="...">` attributes
- [ ] Robot API endpoints respond 200
- [ ] PR references issue #28 with `Fixes #28`

---

## Implementation Steps

### Step 1: Branch and cherry-pick
**Files:** All 18 files in commit
**Command:**
```bash
git checkout main
git pull --ff-only
git checkout -b task/28-pick2-csp-nonce
git cherry-pick -x 82bfde2a37
```
**Expected:** 11 conflicts, 7 clean auto-merges
**Estimated:** 5 min

### Step 2: Resolve conflicts (uniform take-theirs)
**Files:** 11 conflict files
**Command:**
```bash
for f in \
  modules/markup/external/openapi.go \
  modules/markup/render.go \
  modules/templates/helper.go \
  modules/util/util.go \
  services/context/context_template.go \
  templates/base/footer.tmpl \
  templates/base/head_script.tmpl \
  templates/repo/issue/view_content/pull_merge_box.tmpl \
  templates/swagger/ui.tmpl \
  templates/user/auth/captcha.tmpl \
  tests/integration/markup_external_test.go; do
  git checkout --theirs -- "$f"
  git add "$f"
done
```
**Guard:** `git diff --staged | grep '<<<<<<<' && echo "MARKERS PRESENT, ABORT" && exit 1`
**Estimated:** 5 min

### Step 3: Build and quality checks
**Command:**
```bash
go build ./...
make fmt
make vet
```
**Expected:** All clean
**Estimated:** 5 min

### Step 4: Continue cherry-pick
**Command:** `git cherry-pick --continue`
**Expected:** Commit created with original message + `(cherry picked from commit 82bfde2a37...)` footer
**Estimated:** 1 min

### Step 5: Targeted tests
**Command:**
```bash
go test -tags sqlite,sqlite_unlock_notify ./tests/integration -run TestMarkupExternal -v
```
**Expected:** Pass
**Estimated:** 2 min

### Step 6: Smoke verification
**Method:**
```bash
# Build binary
go build -o gitea
# Start server (or inspect template rendering)
# Verify nonce in page source
```
**Estimated:** 10 min

### Step 7: Push and PR
**Command:**
```bash
git push -u origin task/28-pick2-csp-nonce
gtr create-pull \
  --owner terraphim --repo gitea \
  --base main --head task/28-pick2-csp-nonce \
  --title "Fix #28: cherry-pick upstream CSP script nonce (82bfde2a37)"
gtr comment --owner terraphim --repo gitea --index 28 \
  --body "[ferrox] PR opened. @adf:gitea-reviewer please review."
```
**Estimated:** 5 min

### Total Estimate: 33 minutes

| Step | Estimate |
|-----:|----------|
| 1 | 5 min |
| 2 | 5 min |
| 3 | 5 min |
| 4 | 1 min |
| 5 | 2 min |
| 6 | 10 min |
| 7 | 5 min |
| **Total** | **33 min** |

---

## Rollback Plan

### Pre-merge
- Build failure: `git cherry-pick --abort`; report blocker
- Test failure: `git revert HEAD`; document in issue; escalate to security-sentinel

### Post-merge regression
1. `git revert <merge-commit>` on `main`
2. Open follow-up issue
3. Notify `@adf:security-sentinel`

No feature flag -- security fixes are intentionally always-on.

---

## Dependencies

### New Dependencies
None. Upstream code only.

### Dependency Updates
None expected. If `go.mod`/`go.sum` drift, run `make tidy` and commit as part of pick.

---

## Performance Considerations

### Expected Performance
| Metric | Target | Measurement |
|--------|--------|-------------|
| Nonce generation | < 1 microsecond | ChaCha8 via sync.Pool |
| Template render | No regression | Visual smoke test |

### No benchmarks needed
Nonce generation is not a hot path; upstream already optimised with sync.Pool.

---

## Open Items

| Item | Status | Owner |
|------|--------|-------|
| Phase 1 approval | Complete | Ferrox |
| Phase 2 approval | In progress | @adf:gitea-reviewer |

---

## Approval

- [x] Technical review complete (Phase 1)
- [x] Test strategy approved
- [ ] Performance targets agreed (N/A -- no regression expected)
- [ ] Human approval received
