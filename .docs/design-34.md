---
issue: terraphim/gitea#34
phase: 2-design
status: complete
author: Ferrox (Rust Engineer)
date: 2026-05-09
---

# Design Document: Issue #34 -- No-change resolution

## Overview

### Summary

Phase 1 research established that all four upstream security commits cited in
issue #34 are already present on `main`. This design document records the
conclusion that no implementation is required and specifies the validation
artefacts that close the issue.

### Approach

Document the existing fork SHAs that correspond to each upstream pick and
deliver evidence-based resolution via issue comment + close.

### Scope

**In Scope:**

- Validation artefact (`.docs/validation-34.md`) summarising direct evidence.
- Issue comment with per-pick evidence table.
- Issue close.

**Out of Scope:**

- Re-applying any cherry-pick (would conflict; commits are already present).
- Fixing the residual `routers/common/errpage.go` redundancy left by the
  #37354 pick (separate concern, not in #34).
- Improving the upstream-drift detector (different repo).

**Avoid At All Cost:**

- Reopening any of the previously merged picks.
- Touching unrelated files in service of a false-positive issue.
- Producing churn-only commits to "prove" the work was done.

## Architecture

No code changes. The architecture is unchanged.

### Eliminated Options

| Option Rejected | Why Rejected | Risk of Including |
|-----------------|--------------|-------------------|
| Cherry-pick the four upstream SHAs | They are already on main | Hard conflicts, bad history |
| Add `cherry picked from` trailer to `ab62efe1b3` retroactively | Cannot rewrite merged history | Force-push to merged commits |
| Fix the errpage.go redundancy here | Out of scope for #34 | Scope creep |

### Simplicity Check

What if this could be easy? It is. Two artefacts and one issue close.

## File Changes

### New Files

| File | Purpose |
|------|---------|
| `.docs/research-34.md` | Phase 1 evidence (already authored) |
| `.docs/design-34.md` | This document |
| `.docs/validation-34.md` | Phase 5 evidence (next) |

### Modified Files

None.

### Deleted Files

None.

## Test Strategy

No code change -- no new tests required. Existing tests continue to validate:

| Pick | Test |
|------|------|
| #36797 | `models/auth/oauth2_test.go::TestOAuth2AuthorizationCode_*` |
| #37290 | `tests/integration/api_packages_container_test.go::TestPackageContainer/Authenticate/Anonymous` and `/RequireSignIn` |
| #37327 | `tests/integration/oauth_test.go::TestSignInOauthCallbackSyncSSHKeys` (escape coverage) |
| #37354 | `tests/integration/view_test.go::TestRenderingNoSniff` |

These tests live in `main` and are exercised by every CI run.

## Implementation Steps

| Step | Action | Output |
|------|--------|--------|
| 1 | Author research doc with diff-based evidence | `.docs/research-34.md` (done) |
| 2 | Author design doc (this) confirming no code change | `.docs/design-34.md` (done) |
| 3 | Author validation doc consolidating evidence for reviewers | `.docs/validation-34.md` |
| 4 | Comment on issue #34 with per-pick evidence | Gitea API call |
| 5 | Close issue #34 | Gitea API call |

## Follow-up Recommendations

| Recommendation | Rationale | Owner |
|----------------|-----------|-------|
| Open separate issue for `routers/common/errpage.go` X-Frame-Options residue | Strict semantic equivalence with upstream | Backlog |
| Enhance `@adf:upstream-synchronizer` to parse `cherry picked from` and `Adapted-from:` trailers | Eliminates the false-positive class that produced #34 | `@adf:upstream-synchronizer` repo |

## Approval

- [x] Technical review complete (research evidence is direct `git show` diffs)
- [x] Test strategy approved (no change -- existing tests cover all four picks)
- [x] Performance targets agreed (no change)
- [ ] Human approval received (pending review)
