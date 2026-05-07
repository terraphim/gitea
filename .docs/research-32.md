# Research Document: Cherry-pick upstream security fixes #37440 and #37455

**Status**: Completed
**Author**: Ferrox
**Date**: 2026-05-07
**Reviewers**: N/A (autonomous execution)

## Executive Summary

Issue #26 closed without applying two upstream security commits: `fedc9dc993` (URL credential sanitization) and `15b23f037d` (attachment CSP). This research confirms `fedc9dc993` cherry-picks cleanly, while `15b23f037d` conflicts with our prior pick7 (centralised CSP) in `modules/httplib/serve.go`. Manual adaptation is required to integrate the upstream CSP improvements without regressing our audio/video CSP exemption.

## Essential Questions Check

| Question | Answer | Evidence |
|----------|--------|----------|
| Energizing? | Yes | Security fixes; zero-tolerance for credential leakage and XSS vectors |
| Leverages strengths? | Yes | Rust-minded precision in conflict resolution; surgical Go changes |
| Meets real need? | Yes | Issue #32 explicitly tracks missing security picks; Medium-High XSS risk |

**Proceed**: Yes - 3/3 YES

## Problem Statement

### Description
Two upstream security commits from Gitea v1.23+ were omitted when Issue #26 closed:

1. **fedc9dc993** (#37440): URL sanitization to handle schemeless credentials
   - Risk: credential leak via malformed URLs without `://` scheme separator
   - Example: `git@host:path/to/repo.git` or `user:pass@host` could leak credentials in logs

2. **15b23f037d** (#37455): Fix attachment Content-Security-Policy
   - Risk: XSS via attachment serving if CSP headers are missing or incorrect
   - Specifically adds `serveHeaderCspMedia` for audio/video content

### Impact
- **Users**: All users of this Gitea fork; credential leakage in logs and potential XSS via attachments
- **System**: Security audit findings; deviation from upstream security posture
- **Compliance**: Missing security fixes may fail security reviews

### Success Criteria
- Both commits applied to `main` with passing tests
- No regression in existing CSP behaviour (audio/video exemption preserved)
- All affected file tests pass (`modules/util/sanitize_test.go`, `modules/httplib/serve_test.go`)

## Current State Analysis

### Existing Implementation

**URL Sanitization (current `main`):**
- `modules/util/sanitize.go`: `SanitizeCredentialURLs` only handles URLs with `://` scheme separator
- `modules/git/gitcmd/command.go`: `logArgSanitize` delegates to `SanitizeCredentialURLs` only when `://` AND `@` are present
- `services/migrations/migrate.go`: logs `repo.CloneURL` without sanitization

**CSP Serving (current `main`):**
- `modules/httplib/serve.go` was modified by our pick7 (commit `abaf36e5f2`) which centralised CSP for served content
- Our version has custom `serveSetContentSecurityHeaders` function with audio/video exemption
- The upstream commit `15b23f037d` significantly refactors `ServeHeaderOptions`, adds `encodeContentDisposition`, and changes CSP handling structure

### Code Locations

| Component | Location | Purpose |
|-----------|----------|---------|
| URL Sanitizer | `modules/util/sanitize.go` | Remove credentials from URLs in logs/errors |
| Git Command Logger | `modules/git/gitcmd/command.go` | Sanitize git command arguments for logging |
| Migration Logger | `services/migrations/migrate.go` | Log migration URLs safely |
| HTTP Serve | `modules/httplib/serve.go` | Serve files with correct headers including CSP |
| HTTP Serve Tests | `modules/httplib/serve_test.go` | Test serve header behaviour |

### Data Flow

**URL Sanitization:**
```
Git Command / Migration URL
  -> SanitizeCredentialURLs()
    -> Scan for colon + @ pattern
    -> Mask userinfo portion
  -> Safe log output
```

**CSP Serving:**
```
File Serve Request
  -> ServeSetHeaders()
    -> serveSetContentSecurityHeaders() [our pick7]
      -> Set CSP based on content type (default / PDF / audio-video exemption)
  -> HTTP Response with CSP header
```

## Constraints

### Technical Constraints
- **Go language**: All affected files are Go; must follow project Go conventions
- **Cherry-pick preferred**: Upstream commits should be preserved for traceability
- **Conflict resolution**: `modules/httplib/serve.go` has diverged from upstream due to pick7
- **Test compatibility**: Existing tests must pass; upstream adds new tests

### Business Constraints
- **Security priority**: P1-high label; must fix this sprint
- **Minimal disruption**: Only apply the two missing commits, no additional refactoring

### Non-Functional Requirements
| Requirement | Target | Current |
|-------------|--------|---------|
| Credential masking | All URL formats | Only `://` URLs |
| CSP coverage | All served content | All content (with audio/video exemption) |
| Test pass rate | 100% | TBD after cherry-pick |

## Vital Few (Essentialism)

### Essential Constraints (Max 3)

| Constraint | Why It's Vital | Evidence |
|------------|----------------|----------|
| Preserve audio/video CSP exemption | Breaks media playback in some browsers if removed | Issue #32 notes pick7 added this exemption; upstream commit adds `serveHeaderCspMedia` which may conflict |
| Maintain credential masking for all URL formats | Security requirement - no credential leakage | Upstream issue #37435 documents real leak vector |
| Pass all existing and new tests | Regression prevention | `make test` must pass for affected packages |

### Eliminated from Scope

| Eliminated Item | Why Eliminated |
|-----------------|----------------|
| Refactoring other CSP-related code | Not in the two missing commits |
| Additional security hardening | Out of scope for this issue; focus on the two commits |
| Updating unrelated tests | Only test files modified by the commits |

## Dependencies

### Internal Dependencies
| Dependency | Impact | Risk |
|------------|--------|------|
| `modules/util` (sanitize) | Core URL sanitization logic | Low - well-tested |
| `modules/httplib` (serve) | CSP header serving | Medium - conflict with pick7 |
| `modules/git/gitcmd` | Git command logging | Low - simple delegation change |

### External Dependencies
| Dependency | Version | Risk | Alternative |
|------------|---------|------|-------------|
| Upstream Gitea | v1.23+ | Low | N/A - cherry-picking from upstream |

## Risks and Unknowns

### Known Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| CSP regression for audio/video | Medium | High - breaks media playback | Manually verify upstream `serveHeaderCspMedia` aligns with our exemption |
| Test failures after cherry-pick | Low | Medium | Run tests for all affected packages |
| Additional conflicts in serve.go | Medium | Medium | Manual resolution with care for our pick7 changes |

### Open Questions
1. Does upstream's `serveHeaderCspMedia` constant provide the same audio/video exemption as our pick7? - Answer by examining diff
2. Are there any other files referencing `ServeHeaderOptions.Disposition` that need updating? - Search codebase
3. Does `modules/httplib/serve_test.go` have tests for audio/video CSP? - Check current tests

### Assumptions Explicitly Stated

| Assumption | Basis | Risk if Wrong | Verified? |
|------------|-------|---------------|-----------|
| `fedc9dc993` applies cleanly | Test cherry-pick succeeded | Low - would require manual resolution | Yes |
| `15b23f037d` only conflicts in serve.go | Cherry-pick output showed only serve.go conflict | Medium - other files may have subtle issues | Yes |
| Upstream tests cover the security fixes | Commits include test changes | Low | Partial - need to run tests |
| Our pick7 audio/video exemption must be preserved | Issue #32 explicitly notes this | High - would break media playback | Yes - requirement from issue |

### Multiple Interpretations Considered

| Interpretation | Implications | Why Chosen/Rejected |
|----------------|--------------|---------------------|
| Cherry-pick both commits sequentially | Preserves upstream authorship and traceability | **Chosen** - standard practice for upstream picks |
| Manually apply only the security-relevant hunks | More surgical, less risk of unrelated changes | Rejected - loses upstream commit metadata and may miss subtle fixes |
| Rewrite serve.go entirely to match upstream | Cleanest merge, but loses our pick7 changes | Rejected - would regress audio/video CSP exemption |

## Research Findings

### Key Insights

1. **fedc9dc993 applies cleanly**: No conflicts. Changes `SanitizeCredentialURLs` to handle schemeless URLs (e.g., `git@host:path` or `user:pass@host`), updates `logArgSanitize` to always sanitize, and adds sanitization to migration logging.

2. **15b23f037d conflicts in serve.go**: The upstream commit refactors `ServeHeaderOptions` (removes `Disposition` string, adds `ContentDisposition` type, removes `ContentTypeCharset`), adds `encodeContentDisposition` helper, and restructures CSP header setting. Our pick7 added `serveSetContentSecurityHeaders` which overlaps with upstream's renamed `serveSetHeaderContentRelated`.

3. **serve_test.go applies cleanly**: The upstream test additions for CSP don't conflict with our existing tests.

### Relevant Prior Art
- Issue #26: Parent issue tracking 4 upstream security picks (2 of which were missed)
- Issue #28: Cherry-pick CSP script nonce (82bfde2a37) - similar CSP-related upstream pick
- Issue #30: Cherry-pick OAuth2 URL escaping (aee6628bf5) - similar URL sanitization pick
- `.docs/design-security-drift.md`: References the security drift tracking

### Technical Spikes Needed
| Spike | Purpose | Estimated Effort |
|-------|---------|------------------|
| Manual serve.go conflict resolution | Integrate upstream CSP changes with our pick7 | 30 minutes |
| Verify audio/video CSP exemption | Ensure media playback still works after merge | 15 minutes |
| Run affected test suites | Confirm no regressions | 10 minutes |

## Recommendations

### Proceed/No-Proceed
**Proceed** - Both commits are security fixes with clear risk profiles. `fedc9dc993` is straightforward. `15b23f037d` requires manual resolution but the conflict is bounded to one file.

### Scope Recommendations
- Apply `fedc9dc993` first (clean cherry-pick)
- Apply `15b23f037d` second (manual conflict resolution in serve.go)
- Preserve our audio/video CSP exemption during serve.go resolution
- Run tests for all affected packages before committing

### Risk Mitigation Recommendations
1. Before resolving serve.go, save a diff of our current pick7 changes for reference
2. After resolution, verify `serveHeaderCspAudioVideo = ""` logic is preserved
3. Run `go test ./modules/httplib/...` and `go test ./modules/util/...` after each commit

## Next Steps

If approved:
1. Create branch `task/32-security-upstream-picks`
2. Cherry-pick `fedc9dc993` cleanly
3. Cherry-pick `15b23f037d` with manual serve.go resolution
4. Run tests for affected packages
5. Commit and push
6. Create PR

## Appendix

### Cherry-pick Test Results

```bash
# fedc9dc993 - URL sanitization
git cherry-pick --no-commit fedc9dc993
# Result: Clean apply, no conflicts
# Files: modules/git/gitcmd/command.go, command_test.go, modules/util/sanitize.go, sanitize_test.go, services/migrations/migrate.go

# 15b23f037d - CSP attachment
git cherry-pick --no-commit 15b23f037d
# Result: CONFLICT in modules/httplib/serve.go
# Files: modules/httplib/serve.go (conflict), serve_test.go (clean)
```

### Reference Materials
- Upstream commit: https://github.com/go-gitea/gitea/commit/fedc9dc993
- Upstream commit: https://github.com/go-gitea/gitea/commit/15b23f037d
- Issue #32: https://git.terraphim.cloud/terraphim/gitea/issues/32
