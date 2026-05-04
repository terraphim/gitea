# Research: Issue #26 Pick 1 -- Basic Auth Bug (48cea1fb79)

## Upstream Commit

- SHA: `48cea1fb79d8804627cbe828f5f67f0e3204840a`
- Author: Lunny Xiao
- Date: 2026-04-30
- Title: Fix basic auth bug (#37486)
- Files changed: `services/auth/basic.go`, `tests/integration/oauth_test.go`

## What the Fix Does

### services/auth/basic.go

The bug is in `VerifyAuthToken`. When an OAuth2 access token is presented via
HTTP Basic authentication, the method calls `GetOAuthAccessTokenScopeAndUserID`
but discards the returned `accessTokenScope`:

```go
// Before (buggy):
_, uid := GetOAuthAccessTokenScopeAndUserID(req.Context(), authToken)
```

The fix captures and stores the scope in the `DataStore`:

```go
// After (fixed):
accessTokenScope, uid := GetOAuthAccessTokenScopeAndUserID(req.Context(), authToken)
// ...
store.GetData()["ApiTokenScope"] = accessTokenScope
```

Without this fix, when an OAuth2 token is used via Basic auth (the
`token:x-oauth-basic` pattern), the scope is never propagated to the request
data store. This means scope-enforcement middleware never sees the scope, so the
token effectively has no scope restrictions when used via Basic auth -- a
security regression.

### tests/integration/oauth_test.go

The test changes:

1. Extracts a helper `issueOAuthAccessTokenForScope(t, user, scope)` from the
   existing `testOAuthGrantScopesReadUserFailRepos` function.
2. Adds a new test case `testOAuthGrantScopesBasicRespectsWriteUser` that
   verifies the same scope restriction applies when the token is used via Basic
   auth (not just via Bearer header).
3. The refactoring simplifies `testOAuthGrantScopesReadUserFailRepos` to use the
   new helper.
4. Minor cleanups: remove blank lines between statements, use `http.StatusOK`
   constant instead of bare `200`.

## Custom Commit History for Affected Files

### services/auth/basic.go

```
fed2d81e88 Update JS and PY deps (#36708)
```

Only a dependency-update commit touches this file; no custom security changes
overlap with the fix location (lines 77-91, `VerifyAuthToken`).

### tests/integration/oauth_test.go

```
206119f3a0 fix(oauth): Error on auth sources with spaces (#37327)
fed2d81e88 Update JS and PY deps (#36708)
```

The space-fix commit (`206119f3a0`) modifies `oauth_test.go` but only in:
- `testAuthorizeLoginRedirect` area (added `testOAuthSourceWithSpace` subtest)
- Bottom of the file (refactored `TestSignInOauthCallbackSyncSSHKeys` by
  extracting `createMockServer()` helper, and added `testOAuthSourceWithSpace`)

The upstream fix (`48cea1fb79`) touches:
- `TestOAuth2` function (line 78: add new subtest call)
- `testOAuthGrantScopesReadUserFailRepos` function (refactored, lines ~583-700)
- New function `testOAuthGrantScopesBasicRespectsWriteUser` (lines ~583-650)
- New helper `issueOAuthAccessTokenForScope` (lines ~650-695)

These are disjoint regions. No overlap.

## Conflict Risk Assessment

**Risk: Low.**

- `services/auth/basic.go`: Only dep-update commits; fix is a two-line change in
  `VerifyAuthToken`. No conflict expected.
- `tests/integration/oauth_test.go`: The space-fix commit touched the bottom of
  the file and the `TestOAuth2Provider` function. The upstream fix touches
  `TestOAuth2` (different function) and `testOAuthGrantScopesReadUserFailRepos`
  (also disjoint from space-fix changes). No overlap.

## Cascade Risk

None. The change adds one new key (`ApiTokenScope`) to the data store. The
bearer-token path already sets this key (search for `ApiTokenScope` in
`services/auth/oauth2.go`), so the Basic-auth path is being brought in line with
the bearer path. No other files need to change.
