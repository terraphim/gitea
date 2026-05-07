# Verification Report: Cherry-pick upstream security fixes #37440 and #37455

**Status**: Verified
**Date**: 2026-05-07
**Author**: Echo (Twin Maintainer)
**Phase 2 Doc**: `.docs/design-32.md`
**Phase 1 Doc**: `.docs/research-32.md`
**PR**: [!33](https://git.terraphim.cloud/terraphim/gitea/pulls/33)
**Branch**: `task/32-security-upstream-picks`

## Summary

| Metric | Target | Actual | Status |
|--------|--------|--------|--------|
| Unit Test Coverage (affected packages) | All PASS | All PASS | PASS |
| Integration Points Tested | 4 packages | 4 packages | PASS |
| Design Elements Covered | 7 files | 7 files | PASS |
| Critical/High Defects | 0 | 0 | PASS |
| fmt clean | Yes | Yes | PASS |

Both implementation commits (`179ec06233` URL sanitization, `3a3e708271` attachment CSP) verify
correctly against the design. All affected packages pass their test suites.

## Commits Verified

| Commit | Title | Upstream | Risk |
|--------|-------|----------|------|
| `179ec06233` | FIX: URL sanitization to handle schemeless credentials | `fedc9dc993` (#37440) | Medium |
| `3a3e708271` | fix(security): apply upstream attachment CSP fix | `15b23f037d` (#37455) | Medium-High |

## Unit Test Results

### Test Execution

| Package | Command | Result | Evidence |
|---------|---------|--------|---------|
| `modules/util` | `go test ./modules/util/...` | PASS | `ok code.gitea.io/gitea/modules/util 0.061s` |
| `modules/git/gitcmd` | `go test ./modules/git/...` | PASS | `ok code.gitea.io/gitea/modules/git/gitcmd 0.009s` |
| `modules/httplib` | `go test ./modules/httplib/...` | PASS | `ok code.gitea.io/gitea/modules/httplib 0.005s` |
| `services/migrations` | `go test -tags sqlite,sqlite_unlock_notify ./services/migrations/...` | PASS | `ok code.gitea.io/gitea/services/migrations 0.719s` |

Note: `services/migrations` requires `-tags sqlite,sqlite_unlock_notify`; this is pre-existing
environment constraint, not introduced by this change.

### Traceability: Commit 1 (URL Sanitization -- fedc9dc993)

| Function | Test | Design Ref | Status |
|----------|------|------------|--------|
| `SanitizeCredentialURLs` (schemeless) | `TestSanitizeCredentialURLs` | Design Step 1 | PASS |
| `logArgSanitize` always-sanitize | `TestCommandString` | Design Step 1 | PASS |
| `migrate.go` CloneURL trace log | `TestMigrate*` (integration) | Design Step 1 | PASS |

### Traceability: Commit 2 (Attachment CSP -- 15b23f037d)

| Function | Test | Design Ref | Status |
|----------|------|------------|--------|
| `serveSetHeaderContentRelated` (CSP logic) | `ServeData` tests | Design Step 2 | PASS |
| Audio/video CSP exemption preserved | `serveHeaderCspAudioVideo` coverage | Design Decision | PASS |
| `MimeTypeImageSvg` case (upstream addition) | Added test case | Design Step 2 | PASS |

## Code Quality

| Check | Command | Result |
|-------|---------|--------|
| Formatting | `make fmt` | Clean (no diff) |
| Lint | `make lint-go` | Environment mismatch (go1.25 vs go1.26 target) -- pre-existing, not introduced by this PR |

## Integration Points

| Source | Target | Verified By | Status |
|--------|--------|-------------|--------|
| `gitcmd.Command.LogArgSanitize` | `util.SanitizeCredentialURLs` | `TestCommandString` | PASS |
| `migrate.go` -> `util.SanitizeCredentialURLs` | migration trace logging | `TestMigrate*` | PASS |
| `httplib.serveSetHeaderContentRelated` | CSP header on attachment response | `ServeData` tests | PASS |

## Data Flow Verification

| Flow | Design Ref | Verified | Status |
|------|------------|----------|--------|
| Schemeless URL `user:pass@host` -> masked in log | Design 2.1 | `TestSanitizeCredentialURLs` | PASS |
| `git@host:path` credential URL -> masked | Design 2.1 | `TestSanitizeCredentialURLs` | PASS |
| Attachment served -> CSP header present | Design 2.2 | `ServeData` tests | PASS |
| Audio/video attachment -> audio-specific CSP | Design Decision | `ServeData` tests | PASS |

## Defect Register

No defects found. All test suites pass against the implementation.

## Gate Checklist

- [x] All public functions have unit tests
- [x] Edge cases from Phase 2 covered (schemeless URLs, audio/video CSP exemption)
- [x] All module boundaries tested
- [x] Data flows verified against design
- [x] All critical/high defects resolved (none found)
- [x] `make fmt` clean
- [x] No new dependencies introduced
- [x] Upstream authorship preserved in commits

## Approval

| Approver | Role | Decision | Date |
|----------|------|----------|------|
| Echo (Twin Maintainer) | Phase 4 Verifier | Verified -- proceed to validation | 2026-05-07 |

## Next Steps

Proceed to Phase 5 Validation. The PR (!33) is mergeable. Stakeholder sign-off required before merge.
