---
name: Design -- Security Upstream Drift Cherry-Pick Plan
description: Phase 2 implementation plan for issue #12. Cherry-pick sequence, conflict resolution, test plan, and rollback for 7 upstream security commits.
type: project
---

# Implementation Plan: Security Upstream Drift Cherry-Pick

**Status:** Draft -- Awaiting Approval
**Research Doc:** `.docs/research-security-drift.md`
**Author:** Ferrox (Rust Engineer, Principal SFIA-5)
**Date:** 2026-04-30
**Estimated Effort:** 4-6 hours single-pass (excluding review wait)
**Issue:** terraphim/gitea#12
**Reviewers:** @adf:security-sentinel, @adf:gitea-reviewer

---

## 1. Overview

### Summary
Cherry-pick 7 upstream security commits onto `main` in severity-and-cleanliness order. Resolve 3 mechanical single-file conflicts using upstream-as-truth (none of the conflicts touch our Robot code). Land as one PR.

### Approach
**Selective cherry-pick** -- chosen over full upstream merge per research §7. Sequence: clean picks first to establish a green baseline, then conflict picks with explicit per-file resolution.

### Scope

**In scope**
- Cherry-pick 7 security commits identified in issue #12
- Resolve 3 single-file conflicts
- Run targeted integration tests for OAuth2, attachments, container registry, CSP-affected views
- Preserve all 41 fork commits unchanged

**Out of scope**
- Bringing forward non-security upstream commits
- Refactoring or modernising our Robot code
- Tagging a release (separate decision by repo-steward)
- Backporting test-infra changes from upstream

**Avoid at all cost (5/25)**
- Force-pushing to the PR branch (forbidden by `AGENTS.md`)
- Squashing the 7 commits into one (loses upstream attribution)
- Editing any cherry-picked file beyond conflict resolution
- Mixing this PR with non-security changes
- Backfilling missing tests beyond what upstream provides

---

## 2. Architecture

### Branch Structure
```
origin/main (05fe156a51)
   |
   +-- task/12-security-cherry-pick
         |-- pick: f3bdcc58af  (clean)   OAuth2 code reuse
         |-- pick: aee6628bf5  (clean)   OAuth2 URL escaping
         |-- pick: 6826321570  (clean)   X-Content-Type-Options
         |-- pick: 63db5972a1  (clean)   OAuth space handling
         |-- pick: 82bfde2a37  (conflict) CSP script nonce
         |-- pick: 6ed861589a  (conflict) container auth
         |-- pick: 15b23f037d  (conflict) attachment CSP
```

### Key Design Decisions

| Decision | Rationale | Alternatives Rejected |
|----------|-----------|----------------------|
| Cherry-pick | 4/7 clean; bounded conflict surface | Full merge (1821-file blast radius) |
| Severity then cleanliness order | Critical fix lands first; conflicts batched at end | Random order; conflicts-first |
| One PR for all 7 | Reviewer evaluates security posture as a unit | 7 separate PRs (sequencing overhead) |
| Upstream-as-truth on conflicts | Conflicts are upstream evolution, not Robot code | Manual semantic merge |
| Preserve commit hashes via `-x` | Provenance / `git log` traceability | Plain `cherry-pick` |

### Eliminated Options

| Option | Why Rejected | Risk if Included |
|--------|--------------|------------------|
| Full `git merge upstream/main` | 1821 file changes vs. ~50 here | Unbounded review; non-security drift mixed in |
| Reverting our 41 commits to ease merge | Robot work is the fork's purpose | Loss of feature; user-facing regression |
| Skipping conflict commits | Three are High/Medium severity | Partial mitigation |
| Squashing into single security commit | Loses upstream attribution; harder bisect | Future debugging cost |

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

### Modified files (union of 7 commits, deduped)

```
custom/conf/app.example.ini
models/auth/oauth2.go
models/auth/oauth2_test.go
modules/httplib/serve.go                         (CONFLICT)
modules/httplib/serve_test.go
modules/markup/external/openapi.go               (CONFLICT)
modules/markup/render.go
modules/setting/security.go
modules/templates/helper.go
modules/templates/helper_test.go
modules/util/util.go
modules/util/util_test.go
routers/api/packages/container/container.go      (CONFLICT)
routers/api/v1/api.go
routers/common/errpage.go
routers/common/middleware.go
routers/web/auth/auth.go
routers/web/auth/auth_test.go
routers/web/auth/oauth.go
routers/web/auth/oauth2_provider.go
services/context/base_path.go
services/context/context.go
services/context/context_response.go
services/context/context_template.go
templates/* (multiple template files for CSP nonce wiring)
tests/integration/api_packages_container_test.go
tests/integration/markup_external_test.go
tests/integration/oauth_test.go
tests/integration/view_test.go
web_src/js/* (multiple files for OAuth2 URL escaping)
```

Three conflict files are flagged. All other files apply cleanly.

---

## 4. Cherry-Pick Procedure

### Setup

```bash
cd /home/alex/projects/terraphim/gitea
git fetch upstream main
git checkout -b task/12-security-cherry-pick
```

### Pick 1 -- f3bdcc58af (Critical, clean)
```bash
git cherry-pick -x f3bdcc58af
```

### Pick 2 -- aee6628bf5 (High, clean)
```bash
git cherry-pick -x aee6628bf5
```

### Pick 3 -- 6826321570 (Medium, clean)
```bash
git cherry-pick -x 6826321570
```

### Pick 4 -- 63db5972a1 (Medium, clean)
```bash
git cherry-pick -x 63db5972a1
```

### Pick 5 -- 82bfde2a37 (High, conflict)
```bash
git cherry-pick -x 82bfde2a37
# Conflict: modules/markup/external/openapi.go
# Resolution: see §5 below
make fmt && make vet
git add -A
git cherry-pick --continue
```

### Pick 6 -- 6ed861589a (Medium, conflict)
```bash
git cherry-pick -x 6ed861589a
# Conflict: routers/api/packages/container/container.go
# Resolution: see §5 below
make fmt && make vet
git add -A
git cherry-pick --continue
```

### Pick 7 -- 15b23f037d (High, conflict)
```bash
git cherry-pick -x 15b23f037d
# Conflict: modules/httplib/serve.go
# Resolution: see §5 below
make fmt && make vet
git add -A
git cherry-pick --continue
```

---

## 5. Conflict Resolution

For each conflict, the principle is: **upstream is the source of truth for security-bearing changes; our additions must be preserved verbatim.**

### 5.1 modules/markup/external/openapi.go (pick 5, 82bfde2a37)
- **Cause:** Upstream evolved this file between merge-base and the security commit; our 41 commits did NOT modify it.
- **Resolution:** Take upstream version of the conflict block (`git checkout --theirs -- modules/markup/external/openapi.go` is suitable iff no Robot code touched the file). Verify: `git log --oneline upstream/main..HEAD -- modules/markup/external/openapi.go` returns empty.
- **Verification:** `go build ./...` clean; integration test `tests/integration/markup_external_test.go` passes.

### 5.2 routers/api/packages/container/container.go (pick 6, 6ed861589a)
- **Cause:** Same as 5.1 -- upstream-only evolution.
- **Resolution:** Manual 3-way merge using `<<<<<<<` markers. Take incoming security change verbatim. Confirm no Robot route registration is in this file.
- **Verification:** `tests/integration/api_packages_container_test.go` passes.

### 5.3 modules/httplib/serve.go (pick 7, 15b23f037d)
- **Cause:** Same as 5.1 -- upstream-only evolution.
- **Resolution:** Manual 3-way merge. Adopt upstream CSP-attachment header logic. Verify no Robot endpoints serve attachments via this code path.
- **Verification:** New tests in `modules/httplib/serve_test.go` pass.

### Pre-commit guard (each conflict)

```bash
# Before completing a conflict resolution
git diff --staged | grep '<<<<<<<' && echo "MARKERS PRESENT, ABORT"
make fmt
make vet
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
| OAuth2 | `go test ./tests/integration -run TestOAuth` | 1, 2, 4 |
| Container registry | `go test ./tests/integration -run TestPackageContainer` | 6 |
| Markup external | `go test ./tests/integration -run TestMarkupExternal` | 5 |
| Attachment serve | `go test ./modules/httplib/...` | 7 |
| Generic view / CSP | `go test ./tests/integration -run TestView` | 3 |
| Robot API regression | `go test -tags sqlite,sqlite_unlock_notify ./tests/integration -run TestRobotAPI` (requires full fixture DB; see note) | none directly; smoke after all 7 |

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
- [ ] No `<<<<<<<` markers in the working tree
- [ ] `make fmt && make vet` clean
- [ ] OAuth2 integration tests pass (covers Critical fix)
- [ ] No regression in Robot API: `go build -tags sqlite,sqlite_unlock_notify ./...` succeeds; manual smoke of `/api/v1/robot/triage`, `/api/v1/robot/ready`, `/api/v1/robot/graph` (full integration suite deferred to issue #15)
- [ ] PR description references issue #12 with `Fixes #12`

---

## 7. Implementation Steps (Phase 3 hand-off)

### Step 1 -- Clean stack (4 picks)
**Branch ops:** Create `task/12-security-cherry-pick`. Run picks 1-4.
**Tests:** `make fmt && make vet && go build ./...` after step.
**Estimated:** 30 min.

### Step 2 -- OAuth integration verification
**Tests:** `go test ./tests/integration -run TestOAuth -v`.
**Estimated:** 15 min.

### Step 3 -- Conflict pick (82bfde2a37, CSP nonce)
**Files:** `modules/markup/external/openapi.go` resolution.
**Tests:** `go test ./modules/markup/...` and integration markup-external test.
**Estimated:** 45 min.

### Step 4 -- Conflict pick (6ed861589a, container auth)
**Files:** `routers/api/packages/container/container.go` resolution.
**Tests:** `go test ./tests/integration -run TestPackageContainer`.
**Estimated:** 45 min.

### Step 5 -- Conflict pick (15b23f037d, attachment CSP)
**Files:** `modules/httplib/serve.go` resolution.
**Tests:** `go test ./modules/httplib/...`.
**Estimated:** 30 min.

### Step 6 -- Final smoke and Robot regression
**Tests:** Targeted Robot API integration suite.
**Estimated:** 30 min.

### Step 7 -- Push and PR
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

## Appendix -- Commit Reference Card

| # | SHA | PR | Severity | Result | Files |
|---|-----|----|---------:|--------|------:|
| 1 | f3bdcc58af | #36797 | Critical | clean | 3 |
| 2 | aee6628bf5 | #37334 | High | clean | 16 |
| 3 | 6826321570 | #37354 | Medium | clean | 7 |
| 4 | 63db5972a1 | #37327 | Medium | clean | 2 |
| 5 | 82bfde2a37 | #37232 | High | conflict | 18 |
| 6 | 6ed861589a | #37290 | Medium | conflict | 2 |
| 7 | 15b23f037d | #37455 | High | conflict | 2 |
