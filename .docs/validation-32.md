# Validation Report: Cherry-pick upstream security fixes #37440 and #37455

**Status**: Validated
**Date**: 2026-05-07
**Author**: Echo (Twin Maintainer)
**Research Doc**: `.docs/research-32.md`
**Design Doc**: `.docs/design-32.md`
**Verification Report**: `.docs/verification-32.md`
**PR**: [!33](https://git.terraphim.cloud/terraphim/gitea/pulls/33)
**Branch**: `task/32-security-upstream-picks`

## Executive Summary

Both upstream security commits (`fedc9dc993` URL sanitization, `15b23f037d` attachment CSP) have been implemented, verified, and validated. All success criteria from the research document are met. The PR is mergeable with no conflicts. Phase 5 validation passes; proceeding to merge is approved.

## System Test Results

### End-to-End Scenarios

| ID | Workflow | Steps | Result | Status |
|----|----------|-------|--------|--------|
| E2E-001 | Schemeless credential URL in git log | `SanitizeCredentialURLs("user:pass@host")` -> masked output | `user:***@host` returned | PASS |
| E2E-002 | SSH-style git URL in git log | `SanitizeCredentialURLs("git@host:path")` -> masked output | Credentials masked | PASS |
| E2E-003 | Attachment served with CSP header | Request attachment via `serveSetHeaderContentRelated` | CSP header present | PASS |
| E2E-004 | Audio attachment CSP exemption | Audio MIME type -> `serveHeaderCspAudioVideo` | Exempt CSP applied | PASS |
| E2E-005 | Video attachment CSP exemption | Video MIME type -> `serveHeaderCspAudioVideo` | Exempt CSP applied | PASS |
| E2E-006 | SVG attachment sandbox | SVG MIME type -> default sandbox CSP | Sandbox CSP applied | PASS |
| E2E-007 | Migration URL sanitized in log | `migrate.go` trace log with credential URL | URL masked before log | PASS |

### Non-Functional Requirements

| Requirement | Target (from Research) | Actual | Status |
|-------------|------------------------|--------|--------|
| Credential masking - all URL formats | Mask schemeless AND `://` URLs | Both formats masked | PASS |
| CSP coverage - all served content | CSP header on every attachment | Present on all tested types | PASS |
| Audio/video CSP exemption preserved | Exempt header for audio/video | Exempt; 4 sub-tests pass | PASS |
| Test pass rate | 100% for affected packages | 100% (4/4 packages) | PASS |
| No new dependencies | Zero new imports | Confirmed | PASS |
| Upstream authorship preserved | Commits traceable to upstream | Both commits retain upstream metadata | PASS |

### Package Test Summary

| Package | Test Count | Result | Command |
|---------|------------|--------|---------|
| `modules/util` | All tests incl. `TestSanitizeCredentialURLs` | PASS | `go test ./modules/util/...` |
| `modules/git/gitcmd` | All tests incl. `TestCommandString` | PASS | `go test ./modules/git/...` |
| `modules/httplib` | All tests incl. `TestServeSetContentSecurityHeaders` (10 sub-tests) | PASS | `go test ./modules/httplib/...` |
| `services/migrations` | All tests | PASS | `go test -tags sqlite,sqlite_unlock_notify ./services/migrations/...` |

### CSP Test Detail

`TestServeSetContentSecurityHeaders` sub-tests verified:
- `empty_content_type_uses_default` - PASS
- `unknown_content_type_uses_default` - PASS
- `svg_uses_default_sandbox` - PASS
- `html_uses_default_sandbox` - PASS
- `pdf_drops_sandbox` - PASS
- `pdf_with_charset_still_drops_sandbox` - PASS
- `audio_is_exempt` - PASS
- `audio_with_codecs_is_exempt` - PASS
- `video_is_exempt` - PASS
- `video_with_codecs_is_exempt` - PASS

## Acceptance Results

### Requirements Traceability

| Requirement | Source | Evidence | Status |
|-------------|--------|----------|--------|
| Credential masking for all URL formats | Research NFR | `TestSanitizeCredentialURLs` PASS | Accepted |
| CSP coverage for all served content | Research NFR | `TestServeSetContentSecurityHeaders` PASS | Accepted |
| Audio/video CSP exemption preserved | Research Constraint | `audio_is_exempt`, `video_is_exempt` sub-tests PASS | Accepted |
| Test pass rate 100% | Research NFR | 4/4 packages passing | Accepted |
| Minimal scope (two commits only) | Research scope | No unrelated files touched | Accepted |
| Upstream authorship preserved | Research requirement | `git log` confirms original authorship in commit messages | Accepted |

### Problem Validation

The research problem statement:
> "Two upstream security commits from Gitea v1.23+ were omitted when Issue #26 closed"

Both commits are now present in `task/32-security-upstream-picks`:
- `179ec06233` (`fedc9dc993`) -- URL sanitization
- `3a3e708271` (`15b23f037d`) -- attachment CSP

The gap is closed. Both security vectors (credential leak via malformed URLs, XSS via attachment CSP) are mitigated.

### Success Criteria Verification

From research document:
> "Both commits applied to `main` with passing tests"

- Status: Both applied to branch, all tests pass, PR mergeable.

> "No regression in existing CSP behaviour (audio/video exemption preserved)"

- Status: 4 audio/video exemption sub-tests all PASS.

> "All affected file tests pass (`modules/util/sanitize_test.go`, `modules/httplib/serve_test.go`)"

- Status: Both test files run and pass completely.

## PR Mergeability

| Check | Result |
|-------|--------|
| Mergeable flag | `true` |
| State | `open` |
| Conflicts | None |
| Head commit | `3a3e7082714f` |
| Base branch | `main` |

## Defect Register

No defects found during validation. All edge cases from Phase 2.5 (embedded in design) are covered.

## Gate Checklist

- [x] All end-to-end workflows tested (7 scenarios)
- [x] NFRs from research validated (credential masking, CSP coverage, exemption preservation)
- [x] All requirements traced to acceptance evidence
- [x] All critical defects resolved (none found)
- [x] PR is mergeable with no conflicts
- [x] Upstream authorship preserved in both commits
- [x] Ready for production (merge to `main`)

## Sign-off

| Approver | Role | Decision | Conditions | Date |
|----------|------|----------|------------|------|
| Echo (Twin Maintainer) | Phase 5 Validator | Approved | None | 2026-05-07 |

## Next Steps

1. Merge PR !33 (`task/32-security-upstream-picks` -> `main`)
2. Close issue #32
3. Update V-model artefact checklist in issue #32 body
