# Implementation Plan: OAuth URL Escaping Security Fix (#30)

**Status**: Draft
**Research Doc**: `.docs/research-30.md`
**Author**: Ferrox
**Date**: 2026-05-05
**Estimated Effort**: 4 hours

## Overview

### Summary
Apply upstream commit aee6628bf5 ("Fix URL related escaping for oauth2") to the terraphim fork, with manual adaptations for template divergence. Skip 63db5972a1 as superseded by aee6628bf5.

### Approach
Cherry-pick aee6628bf5 and resolve conflicts manually. The commit touches 16 files; 4 require manual adaptation due to fork divergence:
- `templates/user/auth/oauth_container.tmpl` (replaces upstream's `external_auth_methods.tmpl`)
- `web_src/js/utils/url.ts` (fork already has `pathEscapeSegments`, needs `pathEscape`)
- `routers/web/auth/oauth.go` (fork has superior error handling to preserve)
- `routers/web/auth/auth_test.go` (fork divergence in test structure)

### Scope

**In Scope:**
- OAuth2 provider name path escaping in backend handlers
- OAuth2 provider name path escaping in frontend templates
- OAuth2 provider name path escaping in frontend TypeScript
- Error type improvement: `fmt.Errorf` → `util.NewNotExistErrorf` for missing OAuth sources
- Comprehensive test coverage for URL escaping parity

**Out of Scope:**
- Direct cherry-pick of 63db5972a1 (superseded)
- Changes to OAuth2 provider architecture (display name as URL component is upstream design debt)
- Full upstream sync beyond these security commits

**Avoid At All Cost:**
- Refactoring unrelated OAuth code
- Changing the OAuth2 provider data model
- Template-wide escaping changes beyond OAuth URLs

## Architecture

### Component Diagram
```
Frontend (Template)          Frontend (JS)              Backend (Go)
-----------------            -------------              ------------
oauth_container.tmpl    →    url.ts (pathEscape)  →    oauth.go (PathParam)
    |                                                    |
    |                                                    ↓
    |                                              models/auth/oauth2.go
    |                                              (NotExistErrorf)
    |                                                    |
    └────────────────────────────────────────────────────┘
                         Tests verify parity
```

### Data Flow
```
User clicks OAuth login button
    ↓
Template renders href with PathEscape(provider.DisplayName)
    ↓
Browser navigates to /user/oauth2/{escaped_name}
    ↓
Chi router decodes path → ctx.PathParam("provider")
    ↓
Handler looks up OAuth source by decoded name
    ↓
If not found: returns 404 (NotExistErrorf)
    ↓
If found: proceeds with OAuth flow
```

### Key Design Decisions

| Decision | Rationale | Alternatives Rejected |
|----------|-----------|----------------------|
| Use `ctx.PathParam` instead of `url.QueryUnescape(ctx.PathParamRaw)` | Chi router already decodes path parameters correctly; QueryEscape/QueryUnescape is for query strings, not path segments | Keep QueryUnescape with error handling - Rejected because it's semantically wrong for path segments |
| Apply to `oauth_container.tmpl` instead of `external_auth_methods.tmpl` | Fork uses different template structure | Rename template to match upstream - Rejected: unnecessary divergence |
| Add `pathEscape` to JS utils alongside existing `pathEscapeSegments` | aee6628bf5 adds both; `pathEscape` is used in admin UI | Only add `pathEscapeSegments` - Rejected: incomplete parity with backend |

### Eliminated Options (Essentialism)

| Option Rejected | Why Rejected | Risk of Including |
|-----------------|--------------|-------------------|
| Cherry-pick both 63db5972a1 and aee6628bf5 | Creates unnecessary churn; aee6628bf5 supersedes 63db5972a1 | Merge conflicts, double the commits, no additional value |
| Keep fork's QueryUnescape + error handling | Diverges from upstream; inconsistent with callback handler | Maintenance burden, misses upstream test coverage |
| Full OAuth2 URL refactoring | Out of scope for security fix | Scope creep, risk of regressions |

### Simplicity Check

**What if this could be easy?**
The simplest design is: use `PathParam` (already decoded) everywhere, escape in templates with `PathEscape`, and ensure frontend JS uses matching escape logic. This is exactly what aee6628bf5 does.

**Senior Engineer Test**: Would a senior engineer call this overcomplicated?
No. The changes are minimal and precisely targeted at the URL escaping inconsistency.

**Nothing Speculative Checklist**:
- [x] No features the user didn't request
- [x] No abstractions "in case we need them later"
- [x] No flexibility "just in case"
- [x] No error handling for scenarios that cannot occur
- [x] No premature optimization

## File Changes

### Modified Files
| File | Changes |
|------|---------|
| `models/auth/oauth2.go` | Change `fmt.Errorf` to `util.NewNotExistErrorf` for missing OAuth source |
| `routers/web/auth/oauth.go` | Replace `url.QueryUnescape(ctx.PathParamRaw("provider"))` with `ctx.PathParam("provider")` in `SignInOAuth` |
| `routers/web/auth/auth.go` | Update URL escaping if provider name used in URLs |
| `routers/web/auth/auth_test.go` | Update tests for new escaping behavior |
| `routers/web/auth/oauth.go` | Ensure consistent escaping in callback handler |
| `services/context/base_path.go` | Verify path escaping consistency |
| `services/context/context_response.go` | Verify path escaping consistency |
| `templates/user/auth/oauth_container.tmpl` | Add `PathEscape` to provider URL |
| `templates/user/settings/security/accountlinks.tmpl` | Add `PathEscape` to provider URL |
| `web_src/js/utils/url.ts` | Add `pathEscape` function |
| `web_src/js/utils/url.test.ts` | Add tests for `pathEscape` |
| `web_src/js/utils.test.ts` | Remove old `pathEscape` tests if present |
| `web_src/js/utils.ts` | Remove old `pathEscape` if present |
| `web_src/js/features/admin/common.ts` | Update to use `pathEscape` |
| `tests/integration/oauth_test.go` | Add comprehensive escaping tests |
| `modules/templates/helper_test.go` | Add `PathEscape` template tests |
| `modules/util/util_test.go` | Add `PathEscapeSegments` tests |

### Files Requiring Manual Adaptation
1. **oauth_container.tmpl**: Upstream changes `external_auth_methods.tmpl` which doesn't exist in fork
2. **oauth.go**: Fork has error handling around QueryUnescape that must be removed cleanly
3. **auth_test.go**: Fork may have divergent test structure
4. **url.ts**: Fork already has `pathEscapeSegments`; needs `pathEscape` added alongside

## API Design

### Template Functions
```go
// Already exists in modules/templates/helper.go:43
"PathEscape": url.PathEscape,
"PathEscapeSegments": util.PathEscapeSegments,
```

### Go Functions
```go
// routers/web/auth/oauth.go
func SignInOAuth(ctx *context.Context) {
    authName := ctx.PathParam("provider")  // Changed from url.QueryUnescape
    // ... rest unchanged
}
```

### JavaScript Functions
```typescript
// web_src/js/utils/url.ts
export function pathEscape(s: string): string {
  return encodeURIComponent(s).replace(/%2F/g, '/');
}

// Already exists:
export function pathEscapeSegments(s: string): string {
  return s.split('/').map(encodeURIComponent).join('/');
}
```

## Test Strategy

### Regression Tests (Add FIRST)
Before changing code, add tests that pass with current behavior:
- Test OAuth login with provider name containing spaces
- Test OAuth callback with provider name containing spaces
- Test 404 returned for non-existent OAuth provider

### Unit Tests
| Test | Location | Purpose |
|------|----------|---------|
| `TestPathEscape` | `web_src/js/utils/url.test.ts` | Verify JS pathEscape matches Go url.PathEscape |
| `TestPathEscapeSegments` | `modules/util/util_test.go` | Verify Go PathEscapeSegments |
| `TestPathEscapeTemplate` | `modules/templates/helper_test.go` | Verify template PathEscape function |

### Integration Tests
| Test | Location | Purpose |
|------|----------|---------|
| `TestOAuthSourceWithSpace` | `tests/integration/oauth_test.go` | Provider names with spaces work end-to-end |
| `TestOAuthProviderNotFound` | `tests/integration/oauth_test.go` | Non-existent provider returns 404 |

### Parity Tests
Critical: Frontend and backend must generate identical escaped URLs.
```typescript
// Test cases to cover:
// "provider name" -> "provider%20name"
// "provider/name" -> "provider%2Fname" (pathEscape) or "provider/name" (pathEscapeSegments)
// "provider+name" -> "provider+name" (NOT "provider%2Bname" - this is key difference from QueryEscape)
```

## Implementation Steps

### Step 1: Regression Tests
**Files:** `tests/integration/oauth_test.go`
**Description:** Add tests that pass with current code to prevent regressions
**Tests:** OAuth with spaces, OAuth not found
**Estimated:** 30 minutes

### Step 2: Backend Changes
**Files:** `models/auth/oauth2.go`, `routers/web/auth/oauth.go`
**Description:** Apply error type fix and PathParam change
**Tests:** Unit tests for oauth handlers
**Dependencies:** Step 1
**Estimated:** 30 minutes

### Step 3: Template Changes
**Files:** `templates/user/auth/oauth_container.tmpl`, `templates/user/settings/security/accountlinks.tmpl`
**Description:** Add PathEscape to provider URLs
**Tests:** Template rendering tests
**Dependencies:** Step 2
**Estimated:** 20 minutes

### Step 4: Frontend Changes
**Files:** `web_src/js/utils/url.ts`, `web_src/js/utils/url.test.ts`, `web_src/js/features/admin/common.ts`
**Description:** Add pathEscape function and update usages
**Tests:** JS unit tests
**Dependencies:** Step 2
**Estimated:** 30 minutes

### Step 5: Test Updates
**Files:** `tests/integration/oauth_test.go`, `routers/web/auth/auth_test.go`, `modules/templates/helper_test.go`, `modules/util/util_test.go`
**Description:** Update/add tests for new escaping behavior
**Tests:** All tests must pass
**Dependencies:** Steps 2-4
**Estimated:** 45 minutes

### Step 6: Integration Verification
**Description:** Run full test suite for affected packages
**Command:** `go test ./routers/web/auth/... ./models/auth/... ./modules/...`
**Dependencies:** Steps 1-5
**Estimated:** 30 minutes

## Rollback Plan

If issues discovered:
1. Revert commit: `git revert HEAD`
2. Restore branch: `git checkout main`

No feature flags required - this is a bug fix.

## Dependencies

### New Dependencies
None.

### Dependency Updates
None.

## Performance Considerations

No performance impact. Changes are to URL escaping logic only.

## Open Items

| Item | Status | Owner |
|------|--------|-------|
| Verify `PathEscape` template function behavior with special characters | Pending | Ferrox |
| Confirm `oauth_container.tmpl` is the only OAuth provider template | Pending | Ferrox |

## Approval

- [ ] Technical review complete
- [ ] Test strategy approved
- [ ] Human approval received

**Note**: As a single-agent execution, approval gates are self-verified. Quality assurance through:
1. Test-first implementation
2. Incremental verification at each step
3. Full test suite before PR
