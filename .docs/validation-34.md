---
issue: terraphim/gitea#34
phase: 5-validation
status: complete
author: Ferrox (Rust Engineer)
date: 2026-05-09
---

# Validation Document: Issue #34 -- All four upstream picks already applied

## Outcome

**No code changes required.** All four upstream security commits cited in the
issue body are already present on the fork's `main` branch. Direct `git show`
patch comparisons confirm semantic equivalence in every case.

## Per-pick Evidence

### Pick 1 -- #36797 OAuth2 authorisation code expiry/reuse

| Field | Upstream | Fork |
|-------|----------|------|
| SHA | `f3bdcc58af` | `000f3abd6c` |
| Title | Fix OAuth2 authorization code expiry and reuse handling (#36797) | identical |
| Trailer | -- | `cherry picked from commit f3bdcc58aff60b66eba7bd5e9b23457441733dfe` |
| Files | `models/auth/oauth2.go`, `models/auth/oauth2_test.go`, `routers/web/auth/oauth2_provider.go` | identical |

```text
$ diff <(git show f3bdcc58af -- models/auth/oauth2.go ...) \
       <(git show 000f3abd6c -- models/auth/oauth2.go ...)
1c1
< commit f3bdcc58aff60b66eba7bd5e9b23457441733dfe
---
> commit 000f3abd6c84965ce9d71891c822af10ef687ed8
20a21
>     (cherry picked from commit f3bdcc58aff60b66eba7bd5e9b23457441733dfe)
```

Only metadata differs. Diff lines: 0 in patch body. **Equivalent.**

### Pick 2 -- #37290 Container auth for public instance

| Field | Upstream | Fork |
|-------|----------|------|
| SHA | `6ed861589a` | `ab62efe1b3` |
| Title | Fix container auth for public instance (#37290) | `[ferrox] feat(security): conditional Basic realm header in container registry 401` |
| Trailer | -- | `Adapted-from: 6ed861589a (#37290)` |

This was a manual surgical adaptation rather than a clean cherry-pick. The
upstream commit's surrounding `container.go` file uses an upstream-only
`storage.ServeDirectOptions` symbol in unchanged context lines, which produces
a Phase 3 cherry-pick cascade documented in `.docs/blocker-12-pick6-cascade.md`.

The fork commit applies the same ~12 LOC behaviour change (only emit the
`Basic realm` challenge header when sign-in is required) plus an enhanced
inline test, and keeps a `HINT: CONTAINER-AUTH-PUBLIC` marker for future
synchronisers. Behaviour is semantically equivalent.

### Pick 3 -- #37327 OAuth source name with spaces

| Field | Upstream | Fork |
|-------|----------|------|
| SHA | `63db5972a1` | `206119f3a0` |
| Title | fix(oauth): Error on auth sources with spaces (#37327) | identical |
| Trailer | -- | `cherry picked from commit 63db5972a14aed725cfe61e2a192c7a7278cd0a0` |
| Files | `routers/web/auth/oauth.go`, `tests/integration/oauth_test.go` | identical |

Diff vs upstream: only metadata + line-number offsets reflecting fork drift
(36 -> 34, 995 -> 999, 1012 -> 1016, 1087 -> 1091). **Equivalent.**

### Pick 4 -- #37354 X-Content-Type-Options nosniff default

| Field | Upstream | Fork |
|-------|----------|------|
| SHA | `6826321570` | `df7bb50e4d` |
| Title | feat(security): set X-Content-Type-Options: nosniff by default (#37354) | identical |
| Trailer | -- | `cherry picked from commit 68263215704222d83eaaf1349736762a9614e3ad` |

Diff vs upstream: metadata + line-number offsets, plus one upstream hunk in
`routers/common/errpage.go` that was skipped because the surrounding template
context call drifted from upstream
(`NewTemplateContextForWeb` -> `NewTemplateContext`). The skipped hunk removed
a now-redundant `X-Frame-Options` setter; the equivalent middleware-level
setter (in `routers/common/middleware.go`'s `SecurityHeadersHandler`) was
applied. Behaviour: header is still emitted; the codepath is harmlessly
redundant on error pages. Documented in research doc as a follow-up candidate.

## Test Coverage Verification

The existing tests on `main` cover all four picks:

| Pick | Test | Source |
|------|------|--------|
| #36797 | OAuth2 authorisation code TTL + double-invalidation | `models/auth/oauth2_test.go` |
| #37290 | `TestPackageContainer/Authenticate/Anonymous` + `/RequireSignIn` | `tests/integration/api_packages_container_test.go` |
| #37327 | OAuth callback escaping | `tests/integration/oauth_test.go` |
| #37354 | nosniff header default | `tests/integration/view_test.go` and `routers/common/middleware.go` integration |

These run on every CI build of `main`.

## Conclusion and Action

- All four cherry-picks are present and equivalent. No new code is needed.
- Recommend closing issue #34 with this evidence comment.
- Recommend the drift detector be enhanced to read trailers (`cherry picked
  from`, `Adapted-from:`) and PR-number references rather than only matching
  upstream SHAs in fork history.
- Recommend a separate small issue for the `routers/common/errpage.go`
  X-Frame-Options residue if strict semantic equivalence is desired.
