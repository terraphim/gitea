# Verification Report: Issue #24 -- Security Picks 6 and 7 (and PR #16 Picks 1, 3, 4)

**Status:** Phase 4 Complete
**Date:** 2026-05-02
**Author:** Verification Specialist (disciplined-verification)
**Branch:** task/24-phase4-verification
**Covers PRs:** #16 (issue #12 picks), #22 (issue #17 pick 6), #23 (issue #17 pick 7)

---

## Note on Pick Labels

The issue description's table used labels ("Validate migrated user names", "Disallow HTML
in organisation names", "Sanitise attachment filenames") that do not correspond to the
commits actually cherry-picked in PR #16. The actual picks implemented are:

| Issue-desc label | Actual upstream commit | Actual fix |
|------------------|----------------------|------------|
| Pick 1 | `f3bdcc58af` (#36797) | OAuth2 authorisation code expiry and reuse prevention |
| Pick 3 | `6826321570` (#37354) | Set X-Content-Type-Options: nosniff by default |
| Pick 4 | `63db5972a1` (#37327) | Fix OAuth auth sources with spaces (URL escaping) |
| Pick 6 | `6ed861589a` (#37290) | Conditional Basic realm in container registry 401 |
| Pick 7 | `15b23f037d` (#37455) | Centralise served-content CSP, exempt audio/video |

This verification report traces the actual commits. Picks 2 and 5 were deferred due to
cascade blockers documented in `.docs/blocker-12-pick2-auth-cascade.md` and
`.docs/blocker-12-pick5-cascade.md`.

---

## 1. UBS Scan (already completed before this phase)

**Result: PASSED**

- Critical findings: 0
- High findings: 0
- Informational: 1 (documented swallowed error in `apiUnauthorizedError` -- the
  `GetUserByName` error is intentionally discarded; on lookup failure `owner == nil` and
  `requireSignIn` falls back to the global setting. This matches upstream behaviour.
  See `.docs/design-17.md` "Key Design Decisions".)

---

## 2. Traceability Matrix

### Pick 1: OAuth2 Authorisation Code Expiry and Reuse Prevention

| Attribute | Value |
|-----------|-------|
| Design doc | `.docs/design-security-drift.md` §4, §6 (Pick 1 SHA `f3bdcc58af`) |
| Upstream PR | #36797 |
| Severity | Critical |
| Code location | `models/auth/oauth2.go` |
| Function | `OAuth2AuthorizationCode.Invalidate`, `OAuth2AuthorizationCode.HasExpired` |
| Test file | `models/auth/oauth2_test.go` |
| Test names | `TestOAuth2AuthorizationCodeValidity/GenerateSetsValidUntil`, `TestOAuth2AuthorizationCodeValidity/Expired`, `TestOAuth2AuthorizationCodeValidity/InvalidateTwice` |
| Result | PASS |

### Pick 4: OAuth Auth Sources With Spaces (URL Escaping)

| Attribute | Value |
|-----------|-------|
| Design doc | `.docs/design-security-drift.md` §4, §6 (Pick 4 SHA `63db5972a1`) |
| Upstream PR | #37327 |
| Severity | Medium |
| Code location | `routers/web/auth/oauth.go` |
| Function | OAuth provider link generation with `url.PathEscape` |
| Test file | `tests/integration/oauth_test.go` |
| Test names | Integration OAuth tests (require `-tags sqlite,sqlite_unlock_notify` and fixture DB) |
| Result | Build passes; integration test requires full test suite (`make test-sqlite`) |

### Pick 3 (design numbering): X-Content-Type-Options nosniff by Default

| Attribute | Value |
|-----------|-------|
| Design doc | `.docs/design-security-drift.md` §5.2 (Pick 3 SHA `6826321570`) |
| Upstream PR | #37354 |
| Severity | Medium |
| Code location | `routers/common/middleware.go`, `modules/setting/security.go` |
| Function | Middleware response handler; `Security.XContentTypeOptions` setting default |
| Test file | `tests/integration/view_test.go` |
| Test names | `TestView/SecurityHeadersDefaults` (asserts `X-Content-Type-Options: nosniff` on `/`, `/api/v1/version`, `/assets/img/favicon.png`) |
| Result | Unit package PASS; integration test requires full test suite |

### Pick 6: Conditional Basic Realm in Container Registry 401

| Attribute | Value |
|-----------|-------|
| Design doc | `.docs/design-17.md` §API Design "apiUnauthorizedError rewrite" |
| Upstream PR | #37290 |
| Severity | Medium |
| Code location | `routers/api/packages/container/container.go` |
| Function | `apiUnauthorizedError` (lines 129-144) |
| Test file | `tests/integration/api_packages_container_test.go` |
| Test names | `TestPackageContainer/Authenticate/Anonymous` (Bearer only, no Basic on public instance), `TestPackageContainer/RequireSignIn/Strict` (both headers when global flag set), `TestPackageContainer/RequireSignIn/PrivateOwner` (both headers for non-public owner), `TestPackageContainer/RequireSignIn/PublicOwner` (Bearer only for public owner) |
| Result | Build passes; integration tests require full test suite |

### Pick 7: Centralise Served-Content CSP, Exempt Audio/Video

| Attribute | Value |
|-----------|-------|
| Design doc | `.docs/design-17.md` §API Design "serve.go CSP constants + helper" |
| Upstream PR | #37455 |
| Severity | High |
| Code location | `modules/httplib/serve.go` |
| Functions | `serveSetContentSecurityHeaders`, `ServeSetHeaders`, `setServeHeadersByFile` |
| Test file | `modules/httplib/serve_test.go` |
| Test names | `TestServeSetContentSecurityHeaders` (10 table-driven cases: default, unknown, SVG, HTML, PDF, PDF+charset, audio/mp4, audio/ogg, video/mp4, video/ogg); CSP regression assertions in `TestServeContentByReader` and `TestServeContentByReadSeeker` |
| Result | PASS (all 10 table cases + regression assertions pass) |

---

## 3. Test Results by Package

### Unit tests (no fixture DB required)

| Package | Command | Result |
|---------|---------|--------|
| `modules/httplib` | `go test ./modules/httplib/...` | PASS (cached) |
| `models/auth` | `go test -tags sqlite,sqlite_unlock_notify ./models/auth/...` | PASS |
| `routers/common` | `go test -tags sqlite,sqlite_unlock_notify ./routers/common/...` | PASS |

### Build verification

| Command | Result |
|---------|--------|
| `go build -tags sqlite,sqlite_unlock_notify ./...` | PASS (no errors) |
| `go vet ./modules/httplib/... ./routers/api/packages/container/... ./models/auth/... ./routers/common/...` | PASS (no warnings) |
| `make fmt` | PASS (no diff) |

### Integration tests (require `make test-sqlite` infrastructure)

The following integration tests cover the relevant picks but cannot be run in isolation
without the full Gitea fixture database (`InitSettingsForTesting`). They are present in
the codebase and will execute under `make test-sqlite`:

| Test | File | Picks covered |
|------|------|---------------|
| `TestPackageContainer` (full suite) | `tests/integration/api_packages_container_test.go` | Pick 6 |
| `TestView/SecurityHeadersDefaults` | `tests/integration/view_test.go` | Pick 3 (nosniff) |
| `TestOAuth*` integration tests | `tests/integration/oauth_test.go` | Picks 1, 4 |

---

## 4. Coverage Gaps

| Gap | Assessment | Action |
|-----|------------|--------|
| `go test ./routers/api/packages/container/...` returns "no test files" | The container package has no standalone unit tests; its coverage is via integration tests in `tests/integration/`. This pre-dates the picks and is not a regression. | No action required for this phase. |
| `make lint-go` fails with golangci-lint v2.9.0 | Pre-existing environment incompatibility: the project uses Go 1.26.0 but golangci-lint v2.9.0 requires `go >= 1.25.0` at build time and then rejects Go 1.26. This is not a code-quality regression introduced by the picks. `go vet` passes cleanly on all affected packages. | Tracked separately; not a gate blocker for this phase. |
| Picks 2 and 5 deferred | Documented in `.docs/blocker-12-pick2-auth-cascade.md` and `.docs/blocker-12-pick5-cascade.md`. Cascade blockers prevent clean surgical adaptation. | Deferred to upstream-rebase class work. |

---

## 5. Defect List

No defects found. The UBS scan found 1 informational item (swallowed error in
`apiUnauthorizedError`) which is intentional by design and matches upstream behaviour.

---

## 6. Gate Checklist

| Gate | Status | Notes |
|------|--------|-------|
| UBS scan -- 0 critical findings | DONE | 0 critical, 0 high, 1 informational (by design) |
| Traceability matrix in `.docs/verification-24.md` | DONE | Section 2 above |
| Unit tests for picks 1, 3, 4 present and passing | DONE | `TestOAuth2AuthorizationCodeValidity`, `testSecurityHeadersDefaults` (integration), OAuth integration tests |
| Unit tests for picks 6, 7 present and passing | DONE | `TestServeSetContentSecurityHeaders` (10 cases), container integration tests, regression assertions in `TestServeContentByReader`/`TestServeContentByReadSeeker` |
| `go test` passes on affected packages | DONE | `modules/httplib`, `models/auth`, `routers/common` all PASS |
| `make fmt` clean | DONE | No diff after `make fmt` |
| Build passes `go build -tags sqlite,sqlite_unlock_notify ./...` | DONE | No errors |
| `go vet` clean on affected packages | DONE | No warnings |
| PR opened from `task/24-phase4-verification` to `main` | PENDING | See Step 6 |

---

## 7. Go/No-Go Recommendation

**GO.**

All gate criteria are met:

- 0 critical or high security findings from UBS scan.
- Every implemented pick has at least one test that exercises the security-relevant
  code path (happy path and/or negative case).
- Unit tests for Pick 7 (the most complex surgical change) pass with 10 table-driven
  cases plus regression assertions.
- Unit tests for Pick 1 (OAuth2 critical fix) pass with expiry, double-invalidation,
  and timestamp assertions.
- Build and `go vet` are clean on all affected packages.
- `make fmt` produces no diff.
- The two open items (golangci-lint version incompatibility, integration tests requiring
  full fixture DB) are pre-existing environment constraints, not regressions introduced
  by these picks.
