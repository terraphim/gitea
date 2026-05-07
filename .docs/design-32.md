# Implementation Plan: Cherry-pick upstream security fixes #37440 and #37455

**Status**: Approved (autonomous execution)
**Research Doc**: `.docs/research-32.md`
**Author**: Ferrox
**Date**: 2026-05-07
**Estimated Effort**: 2 hours

## Overview

### Summary
Apply two missing upstream security commits to `main`: `fedc9dc993` (URL credential sanitization) and `15b23f037d` (attachment CSP). First commit applies cleanly; second requires manual conflict resolution in `modules/httplib/serve.go` due to our prior pick7 (centralised CSP).

### Approach
Sequential cherry-pick with surgical manual resolution. Preserve our audio/video CSP exemption while adopting upstream's improved `serveSetHeaderContentRelated` structure.

### Scope
**In Scope:**
- Cherry-pick `fedc9dc993` (URL sanitization)
- Cherry-pick `15b23f037d` (CSP attachment) with manual serve.go resolution
- Run tests for affected packages

**Out of Scope:**
- Additional security hardening
- Refactoring unrelated code

**Avoid At All Cost:**
- Rewriting serve.go entirely to match upstream (would lose our pick7)
- Manual hunk application (lose upstream commit metadata)
- Skipping tests

## Architecture

### Component Diagram
```
Upstream Commits
  ├── fedc9dc993 (URL Sanitization)
  │     ├── modules/util/sanitize.go [refactor]
  │     ├── modules/git/gitcmd/command.go [delegate always]
  │     └── services/migrations/migrate.go [add sanitize call]
  └── 15b23f037d (CSP Attachment)
        ├── modules/httplib/serve.go [CONFLICT - manual resolution]
        └── modules/httplib/serve_test.go [clean apply]
```

### Key Design Decisions
| Decision | Rationale | Alternatives Rejected |
|----------|-----------|----------------------|
| Cherry-pick sequentially | Preserves upstream authorship and traceability | Manual patch application |
| Preserve audio/video CSP exemption | Required for media playback in some browsers | Drop exemption (breaks playback) |
| Use upstream's `serveSetHeaderContentRelated` name | Follow upstream convention | Keep our `serveSetContentSecurityHeaders` name |

### Simplicity Check
This is a cherry-pick operation. The only complexity is the serve.go conflict, which is bounded to one function. The simplest approach is to accept upstream's refactored structure while ensuring our audio/video exemption logic is preserved within it.

## File Changes

### Modified Files
| File | Changes |
|------|---------|
| `modules/util/sanitize.go` | Refactor SanitizeCredentialURLs to handle schemeless URLs |
| `modules/util/sanitize_test.go` | Add tests for schemeless URL sanitization |
| `modules/git/gitcmd/command.go` | Always sanitize, don't check for `://` |
| `modules/git/gitcmd/command_test.go` | Update test expectations |
| `services/migrations/migrate.go` | Sanitize CloneURL in trace log |
| `modules/httplib/serve.go` | Refactor CSP handling (manual conflict resolution) |
| `modules/httplib/serve_test.go` | Add CSP attachment tests |

## Test Strategy

### Unit Tests
| Test | Location | Purpose |
|------|----------|---------|
| `TestSanitizeCredentialURLs` | `modules/util/sanitize_test.go` | Verify schemeless URL masking |
| `ServeData` tests | `modules/httplib/serve_test.go` | Verify CSP headers for attachments |

### Verification Steps
```bash
# After each commit:
go test ./modules/util/...
go test ./modules/git/gitcmd/...
go test ./modules/httplib/...
go test ./services/migrations/...
```

## Implementation Steps

### Step 1: Cherry-pick URL sanitization (fedc9dc993)
**Files:** `modules/util/sanitize.go`, `sanitize_test.go`, `modules/git/gitcmd/command.go`, `command_test.go`, `services/migrations/migrate.go`
**Description:** Clean cherry-pick of upstream URL sanitization fix
**Tests:** Run `go test ./modules/util/... ./modules/git/gitcmd/... ./services/migrations/...`
**Estimated:** 15 minutes

### Step 2: Cherry-pick CSP fix (15b23f037d) with manual resolution
**Files:** `modules/httplib/serve.go`, `serve_test.go`
**Description:** Cherry-pick with manual conflict resolution. Strategy:
1. Start cherry-pick
2. Resolve serve.go conflict by accepting upstream's refactored structure
3. Ensure `serveHeaderCspAudioVideo = ""` logic is preserved
4. Complete cherry-pick
**Tests:** Run `go test ./modules/httplib/...`
**Estimated:** 45 minutes

### Step 3: Final verification
**Description:** Run all affected tests, `make fmt`, `make lint-go`
**Estimated:** 15 minutes

## Rollback Plan
If issues discovered:
1. `git reset --hard HEAD~2` (removes both cherry-picks)
2. Or revert individual commits: `git revert <commit>`

## Dependencies

### No New Dependencies
Both commits use existing standard library packages only.

## Performance Considerations
No performance impact expected. URL sanitization adds a fast path check; CSP serving is refactored but functionally equivalent.

## Approval
- [x] Technical review complete (autonomous)
- [x] Test strategy approved (autonomous)
- [x] Proceed to implementation
