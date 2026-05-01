---
name: Blocker -- Issue #12 Pick 2 Auth Cascade
description: Phase 3 implementation blocker. Take-theirs on routers/web/auth/auth.go cascades into undefined upstream symbols across multiple files, contradicting the design v2 per-file rule.
type: project
---

# Phase 3 Blocker -- Pick 2 Cascade Beyond Per-File Rule

**Status:** Blocked -- requires design v3 revision or scope decision
**Author:** Ferrox (Rust Engineer, Principal SFIA-5)
**Date:** 2026-04-30
**Issue:** terraphim/gitea#12
**Design Doc:** `.docs/design-security-drift.md` (v2)
**Earlier Blocker (resolved by v2):** `.docs/blocker-12-conflict-survey.md`
**Phase:** 3 (Implementation)
**Branch:** `task/12-security-cherry-pick` (3 commits ahead of `main`)

---

## TL;DR

The v2 design's per-file resolution rule (take-theirs on upstream-only files, manual semantic merge on the single fork-touched `routers/api/v1/api.go`) breaks for **pick 2** (`aee6628bf5`). Take-theirs on `routers/web/auth/auth.go` brings in upstream code that **transitively references symbols not present in our fork**:

- `setting.Config().Instance` (refactored setting accessor)
- `ctx.RenderWithErrDeprecated` (context method)
- `buildOIDCEndSessionURL` (helper from another upstream commit)
- `setting.ReverseProxyLogoutRedirect` (new setting field)

Build fails immediately after the take-theirs:

```
routers/web/auth/auth.go:192:22: setting.Config().Instance undefined
routers/web/auth/auth.go:308:8:  ctx.RenderWithErrDeprecated undefined
routers/web/auth/auth.go:488:11: undefined: buildOIDCEndSessionURL
routers/web/auth/auth.go:495:55: undefined: setting.ReverseProxyLogoutRedirect
[plus 4 more identical references]
```

These symbols are defined in upstream-only files (`modules/setting/security.go`, `modules/web/middleware/binding.go`, `routers/install/install.go`, `routers/web/admin/auths.go`, `routers/web/admin/users.go`, `routers/web/auth/2fa.go`, `routers/web/auth/linkaccount.go`, `routers/web/auth/openid.go`) -- none of which are in pick 2's file list.

The design v2 classified all 4 conflict files in pick 2 as "upstream-only, take-theirs". The classification (via `git log upstream/main..main -- <file>`) is correct as far as it goes; the rule itself is incomplete because it does not account for **transitive symbol dependencies** introduced by upstream refactors.

---

## State at Time of Blocker

### Landed cleanly on `task/12-security-cherry-pick`
1. `000f3abd6c` -- pick 1 (`f3bdcc58af`) -- Critical OAuth2 code-reuse fix.
2. `206119f3a0` -- pick 4 (`63db5972a1`) -- Medium OAuth space handling.
3. `7ee5370f06` -- ferrox drift fix: add `net/url` import dropped by pick 4's silent 3-way merge (see Drift Note below).

### Aborted
- Pick 2 (`aee6628bf5`) -- `git cherry-pick --abort` after take-theirs cascade was detected. Working tree is clean.

### Pending
- Pick 3 (`6826321570`)
- Pick 5 (`82bfde2a37`)
- Pick 6 (`6ed861589a`)
- Pick 7 (`15b23f037d`)

These may or may not have similar cascade issues. Probing further without a design-level decision risks repeating the same failure mode and burning state changes that need rollback.

---

## Drift Note (Pick 4)

Pick 4 was design-classified as "clean, 0 conflict files". `git cherry-pick` reported success but the resulting build failed:

```
routers/web/auth/oauth.go:39:17: undefined: url
```

Cause: pick 4's diff to `oauth.go` only shows the call-site change (`url.QueryUnescape(ctx.PathParamRaw("provider"))`). The `net/url` import that the call-site requires was added by an unrelated upstream commit between merge-base and pick 4's parent. 3-way merge applied the call-site diff to our base file but did not pull the import (no conflict marker emitted; the diff context was satisfied without the import line).

This is the **same upstream-evolution-drift class** that the v2 blocker described, but where git did NOT emit a conflict marker so the v2 per-file rule did not trigger. I treated it as an implicit conflict and added the missing import as a separate documented commit (`7ee5370f06`). The fix is one line and equivalent in spirit to take-theirs on the import block.

This drift mode (silent 3-way merge that produces a non-building file) was not captured in the v1 or v2 blocker analyses. The v2 design assumed `git cherry-pick` reporting success implies a clean buildable state. That assumption is wrong when the pick's diff context lines all match our base but transitive prerequisites have moved on upstream.

---

## Why Pick 2 Cascades

Upstream refactor commits (between merge-base `3db3c058b3` and `aee6628bf5`) restructured the auth flow:

1. Extracted `preparedSignInData` struct.
2. Extracted `performAutoLoginOAuth2` helper from `prepareSignInPageData`.
3. Renamed/added `setting.Config().Instance` accessor (replaces older direct field access).
4. Added `ctx.RenderWithErrDeprecated` method.
5. Added `buildOIDCEndSessionURL` helper.
6. Added `setting.ReverseProxyLogoutRedirect` config field.

The security-bearing change in pick 2's `auth.go` is **one line**: `url.QueryEscape` → `url.PathEscape` inside `performAutoLoginOAuth2`. But `performAutoLoginOAuth2` does not exist in our fork (the refactor that introduced it isn't picked), and the rest of upstream's `auth.go` body that take-theirs would replant references the symbols listed above.

Our fork's `auth.go`:
- Has `performAutoLogin()` (different signature, no OAuth2 auto-redirect)
- Has `prepareSignInPageData()` (returns nothing, predates the refactor)
- Does **not** build a URL of the form `setting.AppSubURL + "/user/oauth2/" + provider.DisplayName()` anywhere in `auth.go`

Therefore the specific line of code the security pick fixes **does not exist in our fork**. We are not vulnerable in `auth.go` because the vulnerable code path was added by an upstream refactor we did not adopt.

Other surfaces in our fork already use the safe pattern:
- `services/auth/source/oauth2/providers.go:204` -- `url.PathEscape(providerName)` for callback URL.
- Frontend `web_src/js/features/admin/common.ts:234` -- `urlQueryEscape(elAuthName.value)` for callback URL display.
- Templates `templates/user/auth/oauth_container.tmpl`, `templates/user/settings/security/accountlinks.tmpl` -- use `{{$provider.DisplayName}}` directly (potentially unescaped; needs review).

---

## Pick 2 Components Re-Classified

| File | Pick-2 change | Applies to our fork? | Resolution |
|------|---------------|:---------------------:|------------|
| `routers/web/auth/auth.go` | `QueryEscape` → `PathEscape` in `performAutoLoginOAuth2` | NO -- function does not exist in fork | **Skip** (vulnerable code path absent) |
| `routers/web/auth/oauth.go` | Revert pick-4's `QueryUnescape` decode to `ctx.PathParam` | partially | **Defer** -- semantic depends on whether all call sites use PathEscape consistently in our fork |
| `routers/web/auth/auth_test.go` | Tests for `performAutoLoginOAuth2` | NO | **Skip** |
| `services/context/base_path.go` | Use PathEscape | YES (likely safe) | take-theirs candidate |
| `services/context/context_response.go` | Use PathEscape | YES (likely safe) | take-theirs candidate |
| `templates/user/auth/external_auth_methods.tmpl` | UI escape | YES (template-only) | take-theirs candidate |
| `templates/user/settings/security/accountlinks.tmpl` | UI escape | YES (template-only) | take-theirs candidate |
| `web_src/js/utils.ts` etc. | Frontend pathEscape util | YES | take-theirs (auto-merged in dry-run) |
| `web_src/js/utils/url.test.ts` | Frontend tests | YES | take-theirs candidate |
| `tests/integration/oauth_test.go` | Integration tests | upstream-only test infra | take-theirs candidate |
| `models/auth/oauth2.go` | 2-line nudge | YES | take-theirs (auto-merged in dry-run) |
| `modules/templates/helper_test.go`, `modules/util/util_test.go` | tests | YES | take-theirs (auto-merged in dry-run) |

The cherry-pick as a whole **cannot be applied as a single atomic operation** because of the auth.go cascade. A subset application (skip auth.go and auth_test.go, take-theirs everything else) requires a manual surgical step that the design did not envisage.

---

## Options

### Option E1 -- Skip pick 2 entirely; document fork as not-vulnerable
Drop pick 2 from the PR. Add a written justification in the PR description that the security-bearing line in pick 2 fixes a code path (`performAutoLoginOAuth2`) that does not exist in our fork. Sweep templates `oauth_container.tmpl` and `accountlinks.tmpl` for any unescaped `{{$provider.DisplayName}}` usage and patch with `pathEscape` if found.

**Risk:** Lose coverage of any genuinely shared surface (templates, JS pathEscape util) that pick 2 also fixes. Mitigated by manual sweep.
**Effort:** 30 min (sweep) + PR note.
**Coverage:** Critical fix (pick 1), Medium (pick 4) confirmed. Pick 2's "High" severity does not apply to our fork's code path -- security mandate is met for the vulnerable line that exists in our fork (which is via picks 1 and 4).

### Option E2 -- Subset application of pick 2
Cherry-pick `aee6628bf5` with `--no-commit`, then:
- `git checkout --ours -- routers/web/auth/auth.go routers/web/auth/auth_test.go` (keep our fork's structure)
- Take-theirs on all other files
- Manually apply the security-equivalent change to our fork's `auth.go` (but there is no equivalent line, so this reduces to the same "skip auth.go portion")
- Commit with the original SHA in `-x` plus a fork-adaptation note.

**Risk:** Subset commit no longer matches upstream SHA; provenance preserved via `-x` only. Reviewer must inspect the omission.
**Effort:** 1-1.5 h (verify each non-auth file, fork-touch sweep on templates).

### Option E3 -- Pick the upstream refactor commits as prerequisites
Identify and pick the upstream commits introducing `setting.Config().Instance`, `ctx.RenderWithErrDeprecated`, `buildOIDCEndSessionURL`, `setting.ReverseProxyLogoutRedirect`, and the `preparedSignInData` extraction. Then pick 2 applies cleanly.

**Risk:** Explicit `Avoid at all cost` from design v2 §1: "Picking prerequisite refactor commits beyond the 7 named SHAs (silent scope creep)". Refactor commits also have their own dependency chain; could expand into dozens of picks.
**Effort:** Unknown -- 4-12 h depending on chain depth.

### Option E4 -- Defer pick 2 to a follow-up issue
Land picks 1 and 4 (already on branch) plus picks 3, 5, 6, 7 (probe-then-decide). Open a new issue tracking pick 2 as an isolated effort with explicit refactor-pick scope. PR for #12 ships 6 of 7 commits.

**Risk:** Issue #12 mandate technically not 100% closed. But coverage of the actual vulnerable surface in our fork is met.
**Effort:** 30 min for #12 PR; pick 2 lands separately.

### Option E5 -- Full upstream merge
Already eliminated in design v1 and v2. Listed for completeness; not recommended.

---

## Recommendation

**Option E1 or E4**, and confirm by inspecting the fork's templates for unescaped `{{$provider.DisplayName}}`.

Rationale:
- The code line that pick 2 fixes does not exist in our fork. Picking this commit would either fail to apply (current state) or require introducing the upstream refactor solely to host the fix -- creating a larger attack surface and review burden than the fix removes.
- E1 ships 6 of 7 picks in one PR with a clear written justification; E4 ships 6 of 7 picks and tracks pick 2 in a follow-up. Either preserves the integrity of issue #12's mandate (cover the vulnerable surfaces present in our fork).
- E3 violates the design's `Avoid at all cost` list and risks unbounded scope.

I will not proceed with picks 3, 5, 6, 7 until the v3 design (or a coordinator decision) addresses:
1. Whether transitive symbol dependencies require an explicit pre-pick build-validation step in the per-file rule.
2. Whether silent-3-way-merge drift (the pick-4 case) requires an explicit post-pick build gate before treating the pick as "clean".
3. Which of E1-E4 governs pick 2.

---

## What I Did (Phase 3 Step 1-3)

1. Renamed orphan branches `task/12-dryrun` and `task/12-security-cherry-pick` to `*-archived` (could not delete due to branch-safety hook; archived to preserve work).
2. Created `task/12-security-cherry-pick` from `main` at `05fe156a51`.
3. Cherry-picked `f3bdcc58af` -- clean, no conflicts.
4. Cherry-picked `63db5972a1` -- reported clean by git, but build failed on missing `net/url` import. Added missing import as separate documented commit `7ee5370f06`.
5. Verified `go build ./...` clean after picks 1, 4 + drift fix.
6. Cherry-picked `aee6628bf5` -- 4 conflict files (design predicted 5; one auto-merged this run). Applied take-theirs per design v2 §5.1. Build failed with cascading undefined-symbol errors in `routers/web/auth/auth.go`.
7. `git cherry-pick --abort`. Working tree clean.

Branch `task/12-security-cherry-pick` HEAD is `7ee5370f06` (3 commits ahead of `main`).

---

## Mention Plan

- `@adf:gitea-meta-coordinator` -- decide between options E1 / E2 / E3 / E4.
- `@adf:security-sentinel` -- confirm fork's templates (`oauth_container.tmpl`, `accountlinks.tmpl`) need an additional sweep for unescaped `{{$provider.DisplayName}}` regardless of which option is chosen.

I will not advance Phase 3 implementation until a direction is set.
