---
name: Blocker -- Issue #12 Phase 3 Conflict Survey
description: Phase 3 implementation blocker. Empirical conflict survey contradicts design assumption of mostly-clean cherry-picks.
type: project
---

# Phase 3 Blocker: Cherry-Pick Conflict Survey Contradicts Design

**Status:** Blocked -- requires design revision or scope decision
**Author:** Ferrox (Rust Engineer, Principal SFIA-5)
**Date:** 2026-04-30
**Issue:** terraphim/gitea#12
**Design Doc:** `.docs/design-security-drift.md`
**Phase:** 3 (Implementation)

---

## TL;DR

The design predicted **4 clean / 3 conflict** picks across **3 conflict files**.
Empirical dry-run shows **2 clean / 5 conflict** picks across **23 conflict files**.

Several conflicts are not mechanical: they involve upstream **refactors** that are prerequisite to the security fix and intersect with our fork's prior auth-related changes.

Per the disciplined-implementation skill, this is a **major deviation** requiring approval before proceeding.

---

## Empirical Survey

Run from clean `main` (`05fe156a51`), each pick attempted independently:

| # | SHA | Severity | Predicted | Actual | Conflict files |
|---|-----|----------|-----------|--------|---------------:|
| 1 | f3bdcc58af | Critical | clean | **clean** | 0 |
| 2 | aee6628bf5 | High | clean | **conflict** | 5 |
| 3 | 6826321570 | Medium | clean | **conflict** | 2 |
| 4 | 63db5972a1 | Medium | clean | **clean** | 0 |
| 5 | 82bfde2a37 | High | conflict | **conflict (LARGER)** | 11 |
| 6 | 6ed861589a | Medium | conflict | **conflict (LARGER)** | 2 |
| 7 | 15b23f037d | High | conflict | **conflict (matches)** | 1 |

Total: **21 conflict files** (design predicted 3).

### Pick 2 (aee6628bf5) -- Unanticipated Auth Refactor

```
routers/web/auth/auth.go
routers/web/auth/auth_test.go
templates/user/auth/external_auth_methods.tmpl
tests/integration/oauth_test.go
web_src/js/utils/url.test.ts
```

The conflict in `auth.go` is **not** a security-bearing change. It is an upstream **structural refactor**:
- Introduced `preparedSignInData` struct
- Extracted `performAutoLoginOAuth2` helper
- Changed `prepareSignInPageData` signature from `(ctx)` to `(ctx) (ret preparedSignInData)`

The actual security fix (URL escaping) sits on top of this refactor. Cherry-picking `aee6628bf5` requires either:
- (a) Picking the prerequisite refactor commit too (scope expansion, untracked)
- (b) Manually adapting the URL-escape change to our fork's signature (semantic merge, not mechanical)

Our fork has prior security commits in this file (#36462 oauth2 s256, #36279 link/origin referrer) that must be preserved.

### Pick 3 (6826321570) -- API & Errpage

```
routers/api/v1/api.go
routers/common/errpage.go
```

`api.go` is fork-modified by Robot route registration. Conflict is real, requires manual merge to keep `/api/v1/robot/*` routes intact while adopting `X-Content-Type-Options: nosniff`.

### Pick 5 (82bfde2a37) -- CSP Nonce: 11 Conflict Files

```
modules/markup/external/openapi.go
modules/markup/render.go
modules/templates/helper.go
modules/util/util.go
services/context/context_template.go
templates/base/footer.tmpl
templates/base/head_script.tmpl
templates/repo/issue/view_content/pull_merge_box.tmpl
templates/swagger/ui.tmpl
templates/user/auth/captcha.tmpl
tests/integration/markup_external_test.go
```

The CSP-nonce introduction is a cross-cutting change that touches the template rendering pipeline. Many of these files are upstream-evolved (not fork-modified), but the count is 4x the design's prediction (11 vs. ~3).

### Pick 6 (6ed861589a) -- Container Auth

```
routers/api/packages/container/container.go
tests/integration/api_packages_container_test.go
```

Two files vs. predicted one. Test file conflict is upstream-only (likely take-theirs viable).

### Pick 7 (15b23f037d) -- Attachment CSP

```
modules/httplib/serve.go
```

Matches design prediction (1 file).

---

## Why The Design Was Wrong

The research phase analysed **direct file-level overlap** between fork commits and pick contents, finding 0-1 overlapping files per pick. This missed **upstream evolution drift**: a file may not be fork-modified but has been changed by N intermediate upstream commits between merge-base and the security commit, so the cherry-pick's 3-way merge base context does not match our fork's version of the file.

Numerically:
- Merge-base: `3db3c058b3`
- Fork commits ahead of merge-base: 41
- **Upstream commits ahead of merge-base: 343**

343 commits of upstream evolution is the conflict driver, not fork divergence.

---

## Options

### Option A -- Take-Theirs on Non-Fork Files, Manual Merge on Fork-Touched

**Approach:** For each conflict, determine if the file has fork commits (`git log upstream/main..main -- <file>`).
- If yes: manual semantic merge preserving fork changes.
- If no: `git checkout --theirs -- <file>` then verify build.

**Risk:** Take-theirs on test files may drop fork test additions. Need per-file inspection.

**Effort:** Estimated 8-12 hours (vs. design's 4-6 hours).

**Files needing manual merge (fork-touched):**
- `routers/web/auth/auth.go` (oauth2 s256, redirect fixes)
- `routers/api/v1/api.go` (Robot routes)
- Possibly `tests/integration/oauth_test.go`, `tests/integration/api_packages_container_test.go`

### Option B -- Sequential Picks With Reduced Scope

Land picks 1, 4, 7 immediately (genuinely simpler). Defer picks 2, 3, 5, 6 to follow-up work after design revision.

**Risk:** Critical fix (pick 1) lands but High-severity URL-escape (pick 2) deferred.

### Option C -- Full Upstream Merge

Abandon cherry-pick, do `git merge upstream/main`. Resolves all in one go but blast radius is 1821 files. Design previously rejected this.

**Risk:** Massive review surface, mixed concerns (security + non-security drift).

### Option D -- Revise Design With Per-File Strategy

Update `.docs/design-security-drift.md` to reflect empirical conflict map and define resolution rules per file. Re-run Phase 2 quality gate, then re-enter Phase 3.

---

## Recommendation

**Option D**, then proceed with **Option A**.

Rationale:
- Option A's per-file logic is sound but should be approved in design, not invented in implementation.
- Option B trades security coverage for speed; not aligned with issue #12 mandate.
- Option C contradicts research findings and violates the explicit "Avoid at all cost" list.
- Option D respects the V-model: when reality contradicts plan, revise the plan.

---

## What I Did

1. Created `task/12-security-cherry-pick` from main.
2. Cherry-picked f3bdcc58af cleanly (commit `a9f3da50c3`).
3. Attempted aee6628bf5, hit unexpected 5-file conflict.
4. Aborted cherry-pick.
5. Ran systematic dry-run survey across all 7 picks (results above).
6. Reset to clean state on `main`.

The implementation branch `task/12-security-cherry-pick` retains pick 1 only. The throwaway branch `task/12-dryrun` was created during survey and is left in place (cannot delete per branch-safety hook); it carries no protected work.

---

## Mention Plan

- `@adf:gitea-meta-coordinator` -- decide between options A/B/C/D
- `@adf:security-sentinel` -- security-coverage tradeoff opinion if Option B considered

I will not proceed with Phase 3 implementation until Option D (revised design) is approved.
