# Research Document: Security Cherry-Picks for Issue #30

**Status**: Approved
**Author**: Ferrox (Rust Engineer)
**Date**: 2026-05-05
**Reviewers**: N/A - Single-agent execution

## Executive Summary

Issue #30 reports the terraphim/gitea fork is 199 commits behind upstream go-gitea/gitea, with 7 specific security-relevant commits identified. After forensic analysis of the main branch, **5 of the 7 commits are already present** (either directly cherry-picked or manually adapted). **2 commits remain unapplied**: 63db5972a1 (OAuth auth sources with spaces) and aee6628bf5 (OAuth URL escaping follow-up). These two form a dependent pair where aee6628bf5 supersedes 63db5972a1.

## Essential Questions Check

| Question | Answer | Evidence |
|----------|--------|----------|
| Energizing? | Yes | Security fixes prevent auth bypass and URL injection |
| Leverages strengths? | Yes | Our fork already has partial OAuth improvements; this completes the picture |
| Meets real need? | Yes | OAuth provider names with spaces cause 500 errors; unescaped URLs are injection vectors |

**Proceed**: Yes - 3/3 YES

## Problem Statement

### Description
Two upstream security fixes for OAuth2 URL handling remain unapplied in the terraphim fork:

1. **63db5972a1**: OAuth provider names containing spaces cause authentication failures due to incorrect URL escaping
2. **aee6628bf5**: Follow-up fix that standardises path escaping across backend (Go) and frontend (TypeScript), replacing ad-hoc QueryEscape usage with proper path segment escaping

### Impact
- Users cannot authenticate via OAuth2 providers whose display names contain spaces or special characters
- Inconsistent URL escaping between frontend and backend creates injection vectors
- The fork's current `SignInOAuth` uses `url.QueryUnescape` with error handling (better than upstream's 63db5972a1 which ignores errors), but still uses the wrong escaping approach for path segments

### Success Criteria
- OAuth provider names with spaces/special characters work correctly in both login flow and callback
- Frontend and backend generate identical escaped URLs
- `GetActiveOAuth2SourceByAuthName` returns `NotFound` (404) instead of generic error (500) when provider doesn't exist
- All existing OAuth tests pass; new regression tests added

## Current State Analysis

### Existing Implementation

**Already Applied (5/7 commits):**

| Commit | Description | Status in Main | How Applied |
|--------|-------------|----------------|-------------|
| 48cea1fb79 | Fix basic auth bug | ✅ Present | Cherry-picked in PR #27 |
| 6826321570 | X-Content-Type-Options: nosniff | ✅ Present | Cherry-picked as df7bb50e4d |
| 6ed861589a | Container auth for public instance | ✅ Present | Manually adapted in ab62efe1b3 |
| 15b23f037d | Attachment CSP | ✅ Present | Manually adapted in abaf36e5f2 |
| 82bfde2a37 | CSP script nonce | ✅ Present | Cherry-picked in PR #29 |

**Remaining (2/7 commits):**

| Commit | Description | Status in Main |
|--------|-------------|----------------|
| 63db5972a1 | OAuth auth sources with spaces | ❌ Missing - superseded by aee6628bf5 |
| aee6628bf5 | OAuth URL escaping follow-up | ❌ Missing |

### Code Locations

| Component | Location | Current State |
|-----------|----------|---------------|
| OAuth2 login handler | `routers/web/auth/oauth.go:40` | Uses `url.QueryUnescape(ctx.PathParamRaw("provider"))` with error handling - custom improvement over upstream |
| OAuth2 callback handler | `routers/web/auth/oauth.go:91` | Uses `ctx.PathParam("provider")` - correct |
| OAuth2 source lookup | `models/auth/oauth2.go` | Returns `fmt.Errorf` for missing source - should be `util.NewNotExistErrorf` |
| OAuth2 login template | `templates/user/auth/oauth_container.tmpl:6` | No URL escaping on `DisplayName` - injection risk |
| URL utilities (JS) | `web_src/js/utils/url.ts` | Has `pathEscapeSegments` but missing `pathEscape` |
| OAuth2 provider model | `models/auth/oauth2.go:627` | Generic error instead of NotFound |

### Key Finding: Divergence from Upstream

Our fork independently improved upon upstream's 63db5972a1 by adding error handling to `QueryUnescape`:

```go
// Upstream 63db5972a1 (ignores error):
authName, _ := url.QueryUnescape(ctx.PathParamRaw("provider"))

// Our fork main (better, but still wrong approach):
authName, err := url.QueryUnescape(ctx.PathParamRaw("provider"))
if err != nil {
    ctx.HTTPError(http.StatusBadRequest, "invalid provider name encoding")
    return
}

// Upstream aee6628bf5 (correct approach - path already decoded by router):
authName := ctx.PathParam("provider")
```

The correct fix is to adopt upstream's aee6628bf5 approach: use `ctx.PathParam("provider")` which is already properly decoded by the chi router.

## Constraints

### Technical Constraints
- **Dependent commits**: aee6628bf5 depends on 63db5972a1 conceptually, but our fork's state means we can apply aee6628bf5 directly
- **Template divergence**: Our fork uses `oauth_container.tmpl` instead of upstream's `external_auth_methods.tmpl`
- **JS utilities**: Our `web_src/js/utils/url.ts` already has `pathEscapeSegments` but needs `pathEscape`
- **Test infrastructure**: Must maintain fork-specific test adaptations

### Business Constraints
- Must preserve existing fork functionality (85 commits ahead)
- Must pass existing test suite
- Must not introduce regressions in container auth or CSP fixes already applied

## Vital Few (Essentialism)

### Essential Constraints (Max 3)

| Constraint | Why It's Vital | Evidence |
|------------|----------------|----------|
| Preserve existing OAuth error handling | Our fork's error handling is superior to upstream's 63db5972a1 | `routers/web/auth/oauth.go:40-43` already has proper error handling |
| Template compatibility | Must adapt upstream template changes to fork's `oauth_container.tmpl` | Fork diverged from upstream template structure |
| Frontend-backend parity | Escaped URLs must match exactly between Go templates and TypeScript | Commit aee6628bf5 explicitly tests this parity |

### Eliminated from Scope

| Eliminated Item | Why Eliminated |
|-----------------|----------------|
| Cherry-pick 63db5972a1 directly | Our fork already has superior error handling; aee6628bf5 supersedes it |
| Module-wide refactoring | Only apply OAuth-specific changes, not upstream's broader refactoring |
| Full upstream sync | Scope limited to the 2 identified security commits |

## Dependencies

### Internal Dependencies
| Dependency | Impact | Risk |
|------------|--------|------|
| Existing OAuth error handling | Must be preserved during transition from QueryUnescape to PathParam | Low - both approaches handle missing provider |
| Container auth fix (ab62efe1b3) | Must not be regressed | Low - unrelated code paths |
| CSP fix (abaf36e5f2) | Must not be regressed | Low - unrelated code paths |

### External Dependencies
| Dependency | Version | Risk | Alternative |
|------------|---------|------|-------------|
| chi router | v5 | Low - `ctx.PathParam` is stable API | N/A |
| util.NotExistError | internal | Low - already used elsewhere in codebase | N/A |

## Risks and Unknowns

### Known Risks
| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Template syntax differences | Medium | Medium | Manual review of `oauth_container.tmpl` vs upstream `external_auth_methods.tmpl` |
| OAuth tests fail due to fork divergence | Medium | Low | Run tests incrementally, fix as needed |
| PathParam behaves differently than QueryUnescape+PathParamRaw | Low | High | Add comprehensive test coverage |

### Open Questions
1. Does the fork's `oauth_container.tmpl` support the same template functions as upstream's `external_auth_methods.tmpl`? - Check `PathEscape` availability
2. Are there other OAuth provider display name usages that need escaping? - Search codebase for `DisplayName` in URL contexts

### Assumptions Explicitly Stated

| Assumption | Basis | Risk if Wrong | Verified? |
|------------|-------|---------------|-----------|
| `ctx.PathParam("provider")` returns the same decoded value as `url.QueryUnescape(ctx.PathParamRaw("provider"))` | chi router documentation | OAuth authentication breaks for providers with special characters | Yes - verified in upstream tests |
| `PathEscape` template function exists in fork | Used in other templates | Template compilation fails | Needs verification |
| `util.NewNotExistErrorf` exists and behaves like upstream | Used elsewhere in codebase | Error handling changes | Yes - grep confirmed usage |

### Multiple Interpretations Considered

| Interpretation | Implications | Why Chosen/Rejected |
|----------------|--------------|---------------------|
| Apply 63db5972a1 then aee6628bf5 | Two-step process, matches upstream history | Rejected: Our fork already has better error handling than 63db5972a1; applying both creates unnecessary churn |
| Apply only aee6628bf5 | Single step, captures all URL escaping fixes | **Chosen**: Supersedes 63db5972a1, cleaner diff, same security outcome |
| Keep current QueryUnescape + error handling | Minimal change | Rejected: Diverges from upstream, inconsistent with callback handler which already uses PathParam |

## Research Findings

### Key Insights
1. **5 of 7 security commits already applied**: The issue description is partially stale; most security fixes from the upstream list are already in main
2. **Our fork has superior error handling**: The `QueryUnescape` error handling in `SignInOAuth` is better than upstream's 63db5972a1, but the approach (using QueryUnescape for path segments) is still wrong
3. **Template divergence requires manual adaptation**: Upstream's `external_auth_methods.tmpl` doesn't exist in our fork; changes must be applied to `oauth_container.tmpl`
4. **aee6628bf5 is self-contained**: Despite touching 16 files, the changes are cohesive and address a single concern (URL escaping consistency)

### Relevant Prior Art
- PR #27: Basic auth bug cherry-pick (48cea1fb79) - established pattern for security cherry-picks
- PR #29: CSP nonce cherry-pick (82bfde2a37) - established pattern for template/JS security fixes
- ab62efe1b3: Container auth manual adaptation - established pattern for commits that don't cherry-pick cleanly

### Technical Spikes Needed
| Spike | Purpose | Estimated Effort |
|-------|---------|------------------|
| Verify `PathEscape` template function | Check if fork supports `PathEscape` in templates | 5 minutes |
| Test cherry-pick of aee6628bf5 | Verify conflict resolution approach | 15 minutes |

## Recommendations

### Proceed/No-Proceed
**PROCEED** - Apply commit aee6628bf5 with manual adaptations for template divergence. Skip 63db5972a1 as superseded.

### Scope Recommendations
- Apply aee6628bf5 changes to: `models/auth/oauth2.go`, `routers/web/auth/oauth.go`, `templates/user/auth/oauth_container.tmpl`, `web_src/js/utils/url.ts`, and tests
- Preserve our fork's superior error handling where it doesn't conflict with upstream
- Skip upstream's `external_auth_methods.tmpl` changes (file doesn't exist); apply equivalent to `oauth_container.tmpl`

### Risk Mitigation Recommendations
- Add regression tests BEFORE changing code (per testing skill regression-first rule)
- Verify OAuth login flow manually with provider name containing spaces
- Run full OAuth test suite after changes

## Next Steps

If approved:
1. Create task branch: `task/30-oauth-url-escaping`
2. Add regression tests for current OAuth behavior
3. Apply aee6628bf5 changes with manual adaptations
4. Run tests: `go test ./routers/web/auth/... ./models/auth/...` and integration tests
5. Commit and create PR

## Appendix

### Reference Materials
- Issue #30: https://git.terraphim.cloud/terraphim/gitea/issues/30
- Upstream commit aee6628bf5: https://github.com/go-gitea/gitea/commit/aee6628bf5
- Upstream commit 63db5972a1: https://github.com/go-gitea/gitea/commit/63db5972a1

### Code Snippets

Current fork OAuth login handler (superior error handling but wrong approach):
```go
// routers/web/auth/oauth.go:40
authName, err := url.QueryUnescape(ctx.PathParamRaw("provider"))
if err != nil {
    ctx.HTTPError(http.StatusBadRequest, "invalid provider name encoding")
    return
}
```

Target state (from aee6628bf5):
```go
authName := ctx.PathParam("provider")
```

Current template (unescaped - injection risk):
```html
<!-- templates/user/auth/oauth_container.tmpl:6 -->
<a href="{{AppSubUrl}}/user/oauth2/{{$provider.DisplayName}}">
```

Target state:
```html
<a href="{{AppSubUrl}}/user/oauth2/{{$provider.DisplayName | PathEscape}}">
```
