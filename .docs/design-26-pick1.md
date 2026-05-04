# Design: Issue #26 Pick 1 -- Basic Auth Bug (48cea1fb79)

## Decision

Apply the upstream commit verbatim. No adaptation is needed.

## Rationale

As documented in `research-26-pick1.md`:

1. **No conflict in `services/auth/basic.go`**: The only prior touch to this
   file is a dependency update (`fed2d81e88`) which does not alter the
   `VerifyAuthToken` function. The two-line fix applies cleanly.

2. **No conflict in `tests/integration/oauth_test.go`**: Our custom commit
   `206119f3a0` touches only the `TestOAuth2Provider` area and the bottom of the
   file (the `createMockServer` extraction and `testOAuthSourceWithSpace`). The
   upstream fix touches `TestOAuth2`, `testOAuthGrantScopesReadUserFailRepos`,
   and adds new functions below it. These are disjoint regions.

3. **No cascade changes**: The fix propagates `accessTokenScope` to the data
   store via the existing `ApiTokenScope` key -- the same key already used by
   the Bearer token path. No other files require modification.

## Application Plan

```bash
git cherry-pick 48cea1fb79
```

Expected outcome: clean apply, zero conflicts.

## Verification Plan

1. `make fmt` -- format check
2. `go vet -tags sqlite,sqlite_unlock_notify ./services/auth/...` -- vet the
   changed package
3. `go test -tags sqlite,sqlite_unlock_notify ./services/auth/...` -- unit tests
4. `go build -tags sqlite,sqlite_unlock_notify ./...` -- full build check

Note: `make lint-go` is not run due to known golangci-lint version incompatibility
(Go 1.26.0 vs lint v2.9.0) documented in `.docs/verification-24.md`.
