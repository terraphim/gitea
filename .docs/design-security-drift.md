---
name: Design -- Security Upstream Drift Cherry-Pick Plan
description: Phase 2 implementation plan for issue #12. Cherry-pick sequence, conflict resolution, test plan, and rollback for 7 upstream security commits.
type: project
---

# Implementation Plan: Security Upstream Drift Cherry-Pick

**Status:** Revised v2 -- Per-file Strategy (Option D approved by @adf:gitea-meta-coordinator 2026-04-30)
**Research Doc:** `.docs/research-security-drift.md`
**Phase 3 Blocker (resolved by this revision):** `.docs/blocker-12-conflict-survey.md`
**Author:** Ferrox (Rust Engineer, Principal SFIA-5)
**Date:** 2026-04-30 (v1) / 2026-04-30 (v2 revision)
**Estimated Effort:** 8-12 hours single-pass (excluding review wait) -- revised from v1's 4-6 h after empirical conflict survey
**Issue:** terraphim/gitea#12
**Reviewers:** @adf:security-sentinel, @adf:gitea-reviewer

### Revision Notes (v1 -> v2)

The v1 design predicted 4 clean / 3 conflict picks across 3 conflict files. An empirical dry-run (recorded in `.docs/blocker-12-conflict-survey.md`) found 2 clean / 5 conflict picks across **21 conflict files**, driven by 343 upstream commits of evolution drift between merge-base and the security commits.

A subsequent fork-touch classification (`git log upstream/main..main -- <file>`) of all 21 conflict files found that **only 1 file is fork-modified**: `routers/api/v1/api.go` (Robot route registration in commit `a7284622b6`). The remaining 20 are upstream-only evolution drift -- semantically simple to resolve via `git checkout --theirs` once classified.

The v2 design therefore adopts a **per-file resolution rule**: take-theirs on upstream-only files, manual semantic merge on the single fork-touched file. Sections 1, 2, 3, 4, 5, 7, and the appendix have been updated accordingly.

---

## 1. Overview

### Summary
Cherry-pick 7 upstream security commits onto `main` in severity-and-cleanliness order. Five of the seven picks conflict (21 conflict files total, of which 20 are upstream-only evolution drift and 1 is fork-touched). Resolve via per-file rule: take-theirs on upstream-only files, manual semantic merge on the single fork-touched file (`routers/api/v1/api.go`). Land as one PR.

### Approach
**Selective cherry-pick with per-file conflict strategy** -- chosen over full upstream merge per research §7 and confirmed by empirical conflict survey. Sequence: 2 clean picks first (1, 4) to establish a green baseline, then 5 conflict picks (2, 3, 5, 6, 7) with explicit per-file resolution rule applied uniformly.

### Scope

**In scope**
- Cherry-pick 7 security commits identified in issue #12
- Resolve 21 conflict files using per-file rule (Section 5)
- Preserve Robot route registration in `routers/api/v1/api.go`
- Run targeted integration tests for OAuth2, attachments, container registry, CSP-affected views
- Preserve all 41 fork commits unchanged

**Out of scope**
- Bringing forward non-security upstream commits
- Refactoring or modernising our Robot code
- Tagging a release (separate decision by repo-steward)
- Backporting test-infra changes from upstream
- Picking the upstream auth refactor (`preparedSignInData` / `performAutoLoginOAuth2`) as a prerequisite to pick 2 -- take-theirs absorbs both refactor and security fix together

**Avoid at all cost (5/25)**
- Force-pushing to the PR branch (forbidden by `AGENTS.md`)
- Squashing the 7 commits into one (loses upstream attribution)
- Editing any cherry-picked file beyond conflict resolution
- Mixing this PR with non-security changes
- Backfilling missing tests beyond what upstream provides
- Full `git merge upstream/main` (rejected in v1 design; 1821-file blast radius)
- Picking prerequisite refactor commits beyond the 7 named SHAs (silent scope creep)

---

## 2. Architecture

### Branch Structure
```
origin/main (05fe156a51)
   |
   +-- task/12-security-cherry-pick (re-created from main; v1 dry-run branch discarded)
         |-- pick: f3bdcc58af  (clean,    0 files)  OAuth2 code reuse
         |-- pick: 63db5972a1  (clean,    0 files)  OAuth space handling
         |-- pick: aee6628bf5  (conflict, 5 files)  OAuth2 URL escaping
         |-- pick: 6826321570  (conflict, 2 files)  X-Content-Type-Options (1 fork-touched: api.go)
         |-- pick: 82bfde2a37  (conflict, 11 files) CSP script nonce
         |-- pick: 6ed861589a  (conflict, 2 files)  container auth
         |-- pick: 15b23f037d  (conflict, 1 file)   attachment CSP
```

Note: pick order is revised so the two truly clean picks (1, 4) land first, then the conflict picks proceed in original severity order (2, 3, 5, 6, 7). This produces an early green baseline before any conflict resolution.

### Key Design Decisions

| Decision | Rationale | Alternatives Rejected |
|----------|-----------|----------------------|
| Cherry-pick | 2/7 clean + bounded per-file conflict resolution; alternative is unbounded merge | Full merge (1821-file blast radius) |
| Severity then cleanliness order, conflicts batched after baseline | Critical fix (pick 1) lands first; conflicts addressed once build is known-green | Random order; conflicts-first |
| One PR for all 7 | Reviewer evaluates security posture as a unit | 7 separate PRs (sequencing overhead) |
| **Per-file conflict rule: take-theirs on upstream-only, manual semantic merge on fork-touched** | Empirical fork-touch classification shows 20/21 conflict files are upstream evolution drift with no fork modifications; mechanical for those, surgical for the 1 fork-touched file | Manual merge on every conflict (over-effort); take-theirs blanket (would clobber `routers/api/v1/api.go` Robot routes) |
| **Fork-touch detection via `git log upstream/main..main -- <file>`** | Single `git` invocation gives an objective classifier; removes judgement call from implementation | Visual inspection (subjective, error-prone) |
| Preserve commit hashes via `-x` | Provenance / `git log` traceability | Plain `cherry-pick` |
| Skip prerequisite upstream refactors (e.g. `preparedSignInData`) | Take-theirs on upstream-only files absorbs both refactor and security fix together; no scope creep | Pick prerequisite refactor commits (silent scope expansion) |

### Eliminated Options

| Option | Why Rejected | Risk if Included |
|--------|--------------|------------------|
| Full `git merge upstream/main` | 1821 file changes vs. ~50 here | Unbounded review; non-security drift mixed in |
| Reverting our 41 commits to ease merge | Robot work is the fork's purpose | Loss of feature; user-facing regression |
| Skipping conflict commits (Option B from blocker) | Picks 2, 3, 5 carry High-severity fixes (URL escape, CSP nonce); deferring fails issue #12 mandate | Partial mitigation; known-vulnerable fork ships |
| Squashing into single security commit | Loses upstream attribution; harder bisect | Future debugging cost |
| Picking prerequisite upstream refactor commits | Untracked scope expansion; not in 7-SHA list | Loss of provenance; PR creep |
| Manual semantic merge on every conflict (Option A as a blanket policy) | 20/21 files are upstream-only; manual merge is over-effort and risks introducing transcription bugs | Wasted hours; merge errors |

### Simplicity Check

**What if this could be easy?** It already is. `git cherry-pick -x` for each SHA, resolve three single-file conflicts using `git checkout --theirs` then manual context reconciliation, run `make fmt && make vet`, run targeted integration tests, push, open PR.

**Senior engineer test:** Would a senior call this overcomplicated? No. Each step is mechanical and has a single owner.

**Nothing speculative checklist:**
- [x] No features the user didn't request
- [x] No abstractions added
- [x] No flexibility "just in case"
- [x] No error handling beyond what upstream provides
- [x] No premature optimisation

---

## 3. File Changes

No new files authored by us. All changes are upstream commits applied verbatim (with `-x` provenance footer).

### Empirical Conflict Map (21 files across 5 picks)

Classification by `git log upstream/main..main -- <file>` (run from `main` at `05fe156a51` against `upstream/main`).

| Pick | SHA | Conflict File | Fork-touched? | Resolution Rule | Notes |
|-----:|-----|---------------|:-------------:|-----------------|-------|
| 2 | aee6628bf5 | `routers/web/auth/auth.go` | no | take-theirs | Absorbs upstream refactor (`preparedSignInData`, `performAutoLoginOAuth2`) plus URL-escape fix together |
| 2 | aee6628bf5 | `routers/web/auth/auth_test.go` | no | take-theirs | Test file; upstream evolution only |
| 2 | aee6628bf5 | `templates/user/auth/external_auth_methods.tmpl` | no | take-theirs | Template; upstream-only |
| 2 | aee6628bf5 | `tests/integration/oauth_test.go` | no | take-theirs | Integration test; upstream-only |
| 2 | aee6628bf5 | `web_src/js/utils/url.test.ts` | no | take-theirs | JS test; upstream-only |
| 3 | 6826321570 | `routers/api/v1/api.go` | **YES (1 commit)** | **manual semantic merge** | Fork commit `a7284622b6` registers Robot routes (import line 93, group lines 1757-1761). Must preserve. |
| 3 | 6826321570 | `routers/common/errpage.go` | no | take-theirs | Upstream-only |
| 5 | 82bfde2a37 | `modules/markup/external/openapi.go` | no | take-theirs | Upstream-only |
| 5 | 82bfde2a37 | `modules/markup/render.go` | no | take-theirs | Upstream-only |
| 5 | 82bfde2a37 | `modules/templates/helper.go` | no | take-theirs | Upstream-only |
| 5 | 82bfde2a37 | `modules/util/util.go` | no | take-theirs | Upstream-only |
| 5 | 82bfde2a37 | `services/context/context_template.go` | no | take-theirs | Upstream-only |
| 5 | 82bfde2a37 | `templates/base/footer.tmpl` | no | take-theirs | Upstream-only |
| 5 | 82bfde2a37 | `templates/base/head_script.tmpl` | no | take-theirs | Upstream-only |
| 5 | 82bfde2a37 | `templates/repo/issue/view_content/pull_merge_box.tmpl` | no | take-theirs | Upstream-only |
| 5 | 82bfde2a37 | `templates/swagger/ui.tmpl` | no | take-theirs | Upstream-only |
| 5 | 82bfde2a37 | `templates/user/auth/captcha.tmpl` | no | take-theirs | Upstream-only |
| 5 | 82bfde2a37 | `tests/integration/markup_external_test.go` | no | take-theirs | Upstream-only |
| 6 | 6ed861589a | `routers/api/packages/container/container.go` | no | take-theirs | Upstream-only |
| 6 | 6ed861589a | `tests/integration/api_packages_container_test.go` | no | take-theirs | Upstream-only |
| 7 | 15b23f037d | `modules/httplib/serve.go` | no | take-theirs | Upstream-only |

**Summary:** 20 take-theirs / 1 manual semantic merge. The `routers/web/auth/auth.go` classification was re-verified after the v1 blocker doc speculated it was fork-touched -- empirically it is not (PRs #36462 and #36279 referenced in the blocker landed upstream too, so they appear in `upstream/main`).

### Other Modified Files (apply cleanly, listed for review surface)

Union of all 7 commits' file lists, minus the 21 conflict files above. These are upstream-driven additions and modifications that 3-way merge resolves automatically.

```
custom/conf/app.example.ini
models/auth/oauth2.go
models/auth/oauth2_test.go
modules/httplib/serve_test.go
modules/setting/security.go
modules/templates/helper_test.go
modules/util/util_test.go
routers/common/middleware.go
routers/web/auth/oauth.go
routers/web/auth/oauth2_provider.go
services/context/base_path.go
services/context/context.go
services/context/context_response.go
tests/integration/view_test.go
web_src/js/* (additional non-conflict files for OAuth2 URL escaping)
```

---

## 4. Cherry-Pick Procedure

### Setup

```bash
cd /home/alex/projects/terraphim/gitea
git fetch upstream main
# v1 left a `task/12-security-cherry-pick` branch with pick 1 only.
# Discard it and start clean:
git branch -D task/12-security-cherry-pick 2>/dev/null || true
git checkout main
git checkout -b task/12-security-cherry-pick
```

### Per-conflict-file resolver helper

For every conflicting file in picks 2, 3, 5, 6, 7, apply this rule once `git cherry-pick` reports the conflict:

```bash
# $f is the conflict file path
if [ -z "$(git log --oneline upstream/main..main -- "$f")" ]; then
    # Upstream-only: take-theirs
    git checkout --theirs -- "$f"
    git add "$f"
else
    # Fork-touched: stop, do manual semantic merge per §5
    echo "STOP: $f is fork-touched, manual merge required"
fi
```

For this design, the only file that triggers the `STOP` branch is `routers/api/v1/api.go` (in pick 3).

### Pick 1 -- f3bdcc58af (Critical, clean) -- 0 conflict files
```bash
git cherry-pick -x f3bdcc58af
```

### Pick 4 -- 63db5972a1 (Medium, clean) -- 0 conflict files
Re-ordered to land second so the green-baseline batch (picks 1, 4) is contiguous before any conflict resolution.
```bash
git cherry-pick -x 63db5972a1
make fmt && make vet
go build ./...
```

### Pick 2 -- aee6628bf5 (High, conflict) -- 5 files, all upstream-only
```bash
git cherry-pick -x aee6628bf5
# All 5 conflict files are upstream-only -> take-theirs:
for f in routers/web/auth/auth.go routers/web/auth/auth_test.go \
         templates/user/auth/external_auth_methods.tmpl \
         tests/integration/oauth_test.go web_src/js/utils/url.test.ts; do
    git checkout --theirs -- "$f"
    git add "$f"
done
make fmt && make vet
git diff --staged | grep '<<<<<<<' && echo "MARKERS PRESENT, ABORT" && exit 1
git cherry-pick --continue
```

### Pick 3 -- 6826321570 (Medium, conflict) -- 2 files, 1 fork-touched
```bash
git cherry-pick -x 6826321570
# errpage.go: upstream-only -> take-theirs
git checkout --theirs -- routers/common/errpage.go
git add routers/common/errpage.go
# api.go: fork-touched -> manual semantic merge per §5.2
$EDITOR routers/api/v1/api.go    # resolve markers; preserve Robot route group
git add routers/api/v1/api.go
make fmt && make vet
git diff --staged | grep '<<<<<<<' && echo "MARKERS PRESENT, ABORT" && exit 1
git cherry-pick --continue
```

### Pick 5 -- 82bfde2a37 (High, conflict) -- 11 files, all upstream-only
```bash
git cherry-pick -x 82bfde2a37
for f in modules/markup/external/openapi.go modules/markup/render.go \
         modules/templates/helper.go modules/util/util.go \
         services/context/context_template.go \
         templates/base/footer.tmpl templates/base/head_script.tmpl \
         templates/repo/issue/view_content/pull_merge_box.tmpl \
         templates/swagger/ui.tmpl templates/user/auth/captcha.tmpl \
         tests/integration/markup_external_test.go; do
    git checkout --theirs -- "$f"
    git add "$f"
done
make fmt && make vet
git diff --staged | grep '<<<<<<<' && echo "MARKERS PRESENT, ABORT" && exit 1
git cherry-pick --continue
```

### Pick 6 -- 6ed861589a (Medium, conflict) -- 2 files, all upstream-only
```bash
git cherry-pick -x 6ed861589a
for f in routers/api/packages/container/container.go \
         tests/integration/api_packages_container_test.go; do
    git checkout --theirs -- "$f"
    git add "$f"
done
make fmt && make vet
git diff --staged | grep '<<<<<<<' && echo "MARKERS PRESENT, ABORT" && exit 1
git cherry-pick --continue
```

### Pick 7 -- 15b23f037d (High, conflict) -- 1 file, upstream-only
```bash
git cherry-pick -x 15b23f037d
git checkout --theirs -- modules/httplib/serve.go
git add modules/httplib/serve.go
make fmt && make vet
git diff --staged | grep '<<<<<<<' && echo "MARKERS PRESENT, ABORT" && exit 1
git cherry-pick --continue
```

---

## 5. Conflict Resolution

The driver of all 21 conflicts is **343 commits of upstream evolution drift** between the merge-base (`3db3c058b3`) and the security commits. The fork's 41 commits intersect with only **1** of these 21 files. Therefore the per-file resolution rule is:

```
if file has commits in `git log upstream/main..main -- <file>`:
    manual semantic merge (preserve fork changes, adopt incoming security change)
else:
    git checkout --theirs -- <file>   # take upstream verbatim
```

This rule is applied uniformly across all 5 conflict picks. Sections 5.1-5.5 specify the per-pick application.

### 5.1 Pick 2 -- aee6628bf5 (URL escape for OAuth2)

**Conflict files (5):** `routers/web/auth/auth.go`, `routers/web/auth/auth_test.go`, `templates/user/auth/external_auth_methods.tmpl`, `tests/integration/oauth_test.go`, `web_src/js/utils/url.test.ts`.

**Cause:** Upstream evolved `auth.go` independently between merge-base and the security commit. Specifically, an upstream refactor (`preparedSignInData` struct, `performAutoLoginOAuth2` helper) is the prerequisite context for the URL-escape fix in this commit. The other 4 files are in `auth.go`'s test/template orbit and shifted along with it. **None are fork-touched** (verified per file via `git log upstream/main..main -- <file>`).

**Resolution:** Take-theirs on all 5. Take-theirs absorbs the prerequisite refactor and the security fix together, avoiding the silent scope creep of picking the upstream refactor commit separately.

**Verification:** `go build ./...` clean; `go test ./tests/integration -run TestOAuth -v`.

**Risk:** Test files included in take-theirs are checked for any fork-only test additions before the take-theirs runs (none expected per fork-touch detection); if any are found, escalate to manual merge for that file.

### 5.2 Pick 3 -- 6826321570 (X-Content-Type-Options nosniff default)

**Conflict files (2):** `routers/api/v1/api.go` (**fork-touched**, 1 commit), `routers/common/errpage.go` (upstream-only).

**Cause:** `api.go` is fork-modified by `a7284622b6` ("feat: Register robot API routes"), which adds:
- import `code.gitea.io/gitea/routers/api/v1/robot` (current line 93)
- a `m.Group("/robot", ...)` block with three handlers (current lines 1757-1761)

The upstream security commit modifies adjacent route declarations to apply the nosniff middleware uniformly. The conflict is at the route-registration list level.

**Resolution:**
- `routers/common/errpage.go` -- take-theirs.
- `routers/api/v1/api.go` -- **manual semantic merge**:
  1. Adopt the upstream changes to existing routes verbatim (the nosniff wiring).
  2. Re-insert the fork's Robot import line.
  3. Re-insert the fork's `m.Group("/robot", ...)` block at the same logical location (preserved against the upstream-evolved structure of the surrounding code).
  4. Verify: `grep -n "/robot" routers/api/v1/api.go` shows the three handlers; `go build ./...` clean.

**Verification:** `go build ./...` clean; manual smoke of `/api/v1/robot/triage`, `/api/v1/robot/ready`, `/api/v1/robot/graph`.

**Risk:** Manual merge may transcribe wrong context. Mitigation: pre-check that `grep -c "robot.Triage\|robot.Ready\|robot.Graph" routers/api/v1/api.go` returns `3` before `git add`.

### 5.3 Pick 5 -- 82bfde2a37 (CSP script nonce)

**Conflict files (11):** `modules/markup/external/openapi.go`, `modules/markup/render.go`, `modules/templates/helper.go`, `modules/util/util.go`, `services/context/context_template.go`, `templates/base/footer.tmpl`, `templates/base/head_script.tmpl`, `templates/repo/issue/view_content/pull_merge_box.tmpl`, `templates/swagger/ui.tmpl`, `templates/user/auth/captcha.tmpl`, `tests/integration/markup_external_test.go`. **All upstream-only.**

**Cause:** CSP-nonce introduction is a cross-cutting refactor of the template rendering pipeline. The 11 files are all upstream-evolved; the fork did not modify any.

**Resolution:** Take-theirs on all 11.

**Verification:** `go build ./...` clean; `go test ./modules/markup/...`; integration `go test ./tests/integration -run TestMarkupExternal`.

**Risk:** This is the largest conflict file count of the picks. Take-theirs avoids per-file ad-hoc judgement. The template files are render-pipeline glue -- if a runtime regression appears (e.g. nonce missing from a rendered page), revert pick 5 only.

### 5.4 Pick 6 -- 6ed861589a (container auth for public instance)

**Conflict files (2):** `routers/api/packages/container/container.go`, `tests/integration/api_packages_container_test.go`. **Both upstream-only.**

**Cause:** Upstream-only evolution. Pre-survey verification confirmed `routers/api/packages/container/container.go` does **not** carry Robot route registration (Robot routes are in `routers/api/v1/`, not `routers/api/packages/`).

**Resolution:** Take-theirs on both.

**Verification:** `go build ./...` clean; `go test ./tests/integration -run TestPackageContainer`.

### 5.5 Pick 7 -- 15b23f037d (Attachment CSP)

**Conflict files (1):** `modules/httplib/serve.go`. **Upstream-only.**

**Cause:** Upstream evolved the serve helper to add CSP for attachment responses.

**Resolution:** Take-theirs.

**Verification:** `go build ./...` clean; `go test ./modules/httplib/...`.

### Pre-commit guard (every conflict pick)

Run after staging the resolution and before `git cherry-pick --continue`:

```bash
git diff --staged | grep '<<<<<<<' && echo "MARKERS PRESENT, ABORT" && exit 1
make fmt
make vet
go build ./...
```

---

## 6. Test Strategy

### Per-pick verification (after each `git cherry-pick --continue`)

```bash
make fmt
make vet
go build ./...
```

### Integration tests by area

| Area | Test target | Picks covered |
|------|-------------|---------------|
| OAuth2 | `go test -tags sqlite,sqlite_unlock_notify ./tests/integration -run TestOAuth` | 1, 2, 4 |
| Markup external | `go test -tags sqlite,sqlite_unlock_notify ./tests/integration -run TestMarkupExternal` | 5 |
| Container registry | `go test -tags sqlite,sqlite_unlock_notify ./tests/integration -run TestPackageContainer` | 6 |
| Attachment serve | `go test ./modules/httplib/...` | 7 |
| Generic view / CSP nonce render | manual: boot `gitea web`, view a page with inline scripts, confirm nonce attribute present | 3, 5 |
| Robot API regression | `go build -tags sqlite,sqlite_unlock_notify ./...` plus manual smoke of `/api/v1/robot/{triage,ready,graph}` | smoke after all 7 (full integration suite deferred to issue #15) |

**Note on Robot API integration tests:** `TestRobotAPI_*` tests exist in `tests/integration/robot_security_test.go` and compile correctly with `-tags sqlite,sqlite_unlock_notify`. However, they require the full Gitea fixture database (`InitSettingsForTesting`) and cannot run without `make test-sqlite` infrastructure. Running them bare fails with a fatal panic. For Phase 3, the gate is `go build -tags sqlite,sqlite_unlock_notify ./...` succeeding plus manual endpoint smoke. Full CI wiring is tracked in issue #15.

### Final smoke (after all 7 picks)

```bash
make fmt
make vet
go build -o gitea
go test ./modules/... ./routers/... ./services/... -short
```

If `make` integration suite is feasible in a reasonable wall-clock window, run:

```bash
make test-sqlite-migration
```

### Acceptance Criteria

- [ ] All 7 commits present in `git log` of the PR branch
- [ ] `git log --oneline upstream/main..HEAD` shows 41 + 7 = 48 commits
- [ ] No `<<<<<<<` markers in the working tree (`grep -r '<<<<<<<' --include='*.go' --include='*.tmpl' --include='*.ts' .` returns nothing)
- [ ] `make fmt && make vet` clean
- [ ] `routers/api/v1/api.go` retains Robot routes: `grep -c "robot.Triage\|robot.Ready\|robot.Graph" routers/api/v1/api.go` returns 3
- [ ] Robot import preserved at top of `routers/api/v1/api.go`: `grep -c "code.gitea.io/gitea/routers/api/v1/robot" routers/api/v1/api.go` returns 1
- [ ] OAuth2 integration tests pass (covers Critical fix and pick 2 URL escape)
- [ ] Markup external integration tests pass (covers pick 5 CSP nonce)
- [ ] Container registry integration tests pass (covers pick 6)
- [ ] Attachment serve unit tests pass (covers pick 7)
- [ ] No regression in Robot API: `go build -tags sqlite,sqlite_unlock_notify ./...` succeeds; manual smoke of `/api/v1/robot/triage`, `/api/v1/robot/ready`, `/api/v1/robot/graph` (full integration suite deferred to issue #15)
- [ ] PR description references issue #12 with `Fixes #12`

---

## 7. Implementation Steps (Phase 3 hand-off)

Total estimate: **8-12 hours single-pass** (revised from v1's 4-6 h based on empirical conflict survey).

### Step 1 -- Green baseline (clean picks 1, 4)
**Branch ops:** Create `task/12-security-cherry-pick` from `main`. Run pick 1 (`f3bdcc58af`) then pick 4 (`63db5972a1`).
**Tests:** `make fmt && make vet && go build ./...` after each pick.
**Estimated:** 30 min.

### Step 2 -- OAuth integration verification (covers picks 1, 4)
**Tests:** `go test ./tests/integration -run TestOAuth -v` (requires `-tags sqlite,sqlite_unlock_notify`).
**Estimated:** 15 min.

### Step 3 -- Conflict pick 2 (aee6628bf5, OAuth URL escape) -- 5 take-theirs files
**Files:** `routers/web/auth/auth.go`, `routers/web/auth/auth_test.go`, `templates/user/auth/external_auth_methods.tmpl`, `tests/integration/oauth_test.go`, `web_src/js/utils/url.test.ts`.
**Action:** Apply take-theirs per §5.1. Verify build. Continue cherry-pick.
**Tests:** `go build ./...`; `go test ./routers/web/auth/...`; `go test ./tests/integration -run TestOAuth -v`.
**Estimated:** 1-1.5 h (per-file inspection of test files for fork-only additions before take-theirs; build/test cycle).

### Step 4 -- Conflict pick 3 (6826321570, X-Content-Type-Options) -- 1 take-theirs + 1 manual semantic merge
**Files:** `routers/common/errpage.go` (take-theirs), **`routers/api/v1/api.go` (manual semantic merge -- preserve Robot routes)**.
**Action:** Apply take-theirs to errpage.go. Resolve api.go markers per §5.2: adopt upstream nosniff wiring; re-insert fork's Robot import (line ~93) and `m.Group("/robot", ...)` block (lines ~1757-1761). Verify with `grep -c "robot.Triage\|robot.Ready\|robot.Graph" routers/api/v1/api.go == 3`.
**Tests:** `go build ./...`; build of `routers/api/v1/...`; manual smoke of `/api/v1/robot/triage`, `/api/v1/robot/ready`, `/api/v1/robot/graph`.
**Estimated:** 2-3 h (the only manual semantic merge in the entire plan; warrants careful review of api.go diff).

### Step 5 -- Conflict pick 5 (82bfde2a37, CSP nonce) -- 11 take-theirs files
**Files:** Per §5.3.
**Action:** Apply take-theirs in a loop. Verify build. Continue cherry-pick.
**Tests:** `go build ./...`; `go test ./modules/markup/...`; integration `go test ./tests/integration -run TestMarkupExternal`. Smoke a rendered page (e.g. visit `/issues` in `gitea web`) to confirm CSP nonce appears in inline scripts.
**Estimated:** 2-3 h (largest conflict surface; verify template rendering manually because tests do not cover all templates).

### Step 6 -- Conflict pick 6 (6ed861589a, container auth) -- 2 take-theirs files
**Files:** Per §5.4.
**Action:** Take-theirs both. Continue cherry-pick.
**Tests:** `go build ./...`; `go test ./tests/integration -run TestPackageContainer`.
**Estimated:** 45 min - 1 h.

### Step 7 -- Conflict pick 7 (15b23f037d, attachment CSP) -- 1 take-theirs file
**Files:** Per §5.5.
**Action:** Take-theirs. Continue cherry-pick.
**Tests:** `go build ./...`; `go test ./modules/httplib/...`.
**Estimated:** 30 min.

### Step 8 -- Final smoke and Robot regression
**Tests:**
- `make fmt && make vet`
- `go build -tags sqlite,sqlite_unlock_notify -o gitea`
- `go test -tags sqlite,sqlite_unlock_notify ./modules/... ./routers/... ./services/... -short`
- Manual Robot API smoke (boot `gitea web`, hit `/api/v1/robot/triage`, `/ready`, `/graph`).
- Confirm `git log --oneline upstream/main..HEAD` shows 41 + 7 = 48 commits.
**Estimated:** 1 h.

### Step 9 -- Push and PR
```bash
git push -u origin task/12-security-cherry-pick
gtr create-pull \
  --owner terraphim --repo gitea \
  --base main --head task/12-security-cherry-pick \
  --title "Fix #12: cherry-pick 7 upstream security commits"
gtr comment --owner terraphim --repo gitea --index 12 \
  --body "[ferrox] PR opened. @adf:gitea-reviewer please review."
```
**Estimated:** 15 min.

### Total: 8-12 hours single-pass

| Step | Estimate |
|-----:|----------|
| 1 | 30 min |
| 2 | 15 min |
| 3 | 1-1.5 h |
| 4 | 2-3 h |
| 5 | 2-3 h |
| 6 | 45 min - 1 h |
| 7 | 30 min |
| 8 | 1 h |
| 9 | 15 min |
| **Total** | **8 h 0 min - 11 h 30 min** |

---

## 8. Rollback Plan

### Pre-merge
- Conflict resolution wrong: `git cherry-pick --abort`; restart from last clean commit.
- Test failure post-pick: `git revert HEAD` on the bad pick; document in PR; defer to security-sentinel.

### Post-merge regression
1. `git revert <merge-commit-or-pick-sha>` on `main`.
2. Open follow-up issue with reproducer.
3. Notify `@adf:security-sentinel` and `@adf:repo-steward`.

No feature flag is needed -- security fixes are intentionally always-on.

---

## 9. Dependencies

### New dependencies
None. Cherry-picks bring upstream code only; `go.mod` may receive minor adjustments inherited from picks. If `go.mod` changes, run `make tidy` and commit as part of the relevant pick (do NOT add separate cleanup commits).

### Dependency updates
None planned. If picks alter `go.sum`, the changes are upstream-driven and accepted as-is.

---

## 10. Performance Considerations

No performance regressions expected. CSP-nonce introduction (`82bfde2a37`) generates per-request nonce -- upstream benchmarked acceptable. No new benchmarks required for this change.

---

## 11. Approval Gates

### Standard
- [ ] Technical review by `@adf:gitea-reviewer`
- [ ] Security review by `@adf:security-sentinel`
- [ ] Build runner sign-off by `@adf:gitea-build-runner`
- [ ] Human approval on PR

### Essentialism
- [x] 5 or fewer major components in scope (cherry-pick, conflict resolution, tests, PR, rollback)
- [x] "Eliminated Options" populated (§2)
- [x] "Avoid at all cost" list documented (§1)
- [x] Simplicity check answered (§2)
- [x] 5/25 rule applied to scope (§1)

### Quality Evaluation
On Phase 3 completion, request `disciplined-quality-evaluation` before merge.

---

## 12. Open Items

| Item | Status | Owner |
|------|--------|-------|
| Verify integration test infrastructure runs in current dev env | Pending | @adf:gitea-build-runner |
| Confirm whether release tag is required after merge | Pending | @adf:repo-steward |
| Sweep for any newer security fixes landed upstream since issue creation | Pending | @adf:security-sentinel |

---

## Appendix -- Commit Reference Card (revised v2 with empirical conflict counts)

| # | SHA | PR | Severity | Result | Conflict files | Fork-touched | Resolution |
|---|-----|----|---------:|--------|---------------:|:------------:|------------|
| 1 | f3bdcc58af | #36797 | Critical | clean | 0 | -- | -- |
| 4 | 63db5972a1 | #37327 | Medium | clean | 0 | -- | -- |
| 2 | aee6628bf5 | #37334 | High | conflict | 5 | 0 | take-theirs ×5 |
| 3 | 6826321570 | #37354 | Medium | conflict | 2 | 1 (`api.go`) | take-theirs ×1 + manual semantic merge ×1 |
| 5 | 82bfde2a37 | #37232 | High | conflict | 11 | 0 | take-theirs ×11 |
| 6 | 6ed861589a | #37290 | Medium | conflict | 2 | 0 | take-theirs ×2 |
| 7 | 15b23f037d | #37455 | High | conflict | 1 | 0 | take-theirs ×1 |
| **Totals** | | | | **5 conflict / 2 clean** | **21** | **1** | **20 take-theirs + 1 manual semantic merge** |

Pick order in branch: 1, 4, 2, 3, 5, 6, 7 (clean baseline first; conflicts in original severity-tied order).
