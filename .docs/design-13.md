# Implementation Plan: Verify ADF integration for gitea (#13)

**Status**: Draft
**Research Doc**: `.docs/research-13.md`
**Author**: Ferrox (Rust Engineer agent)
**Date**: 2026-04-30
**Estimated Effort**: 30 minutes

## Overview

### Summary

Demonstrate end-to-end ADF round-trip on issue #13 by:

1. Posting a confirmation comment on #13 (Step 2 in issue body)
2. Adding a small "Documentation" subsection to the existing fork-only
   `## Robot API` section of `README.md` that links the four existing
   fork-only docs (Step 3 in issue body)
3. Opening a PR via `gtr create-pull` and tagging `@adf:gitea-reviewer`

### Approach

Markdown-only patch on the *already fork-divergent* `README.md`. No code,
no new files, no upstream-clean files touched.

### Scope

**In Scope:**

- Comment on #13 confirming `@adf:gitea-developer` visibility
- Append a "Documentation" subsection to `README.md` Robot API block
- Branch, commit, push, PR

**Out of Scope:**

- Any Go code change
- Any rewrite of existing README content (surgical-changes protocol)
- Any new top-level files
- Any change to upstream-pristine files

**Avoid At All Cost** (from 5/25 analysis):

- Editing `CHANGELOG.md` -- noisy fork divergence already
- Adding a new `FORK.md` -- proliferates roots
- Touching the existing Robot API CLI / Docker subsections -- not requested
- Linting drive-bys on AGENTS.md or unrelated markdown -- prohibited
- Generating `.docs/research-13.md` follow-ups (already complete)

## Architecture

### Component Diagram

```
[Agent: Ferrox]
   |--> gtr comment    --> Issue #13 (visibility confirmation)
   |--> git branch + edit README.md
   |--> git commit + push
   '--> gtr create-pull --> PR (mentions @adf:gitea-reviewer)
```

### Data Flow

```
Issue #13 body -> agent reads --> agent comments --> validator ticks Step 2
                                |
                                +-> agent edits README --> commit --> PR --> Step 3 ticked on merge
```

### Key Design Decisions

| Decision | Rationale | Alternatives Rejected |
|----------|-----------|----------------------|
| Patch `README.md` Robot API block (additive subsection) | Already fork-divergent (+79 lines vs upstream); section exists; addition genuinely improves docs discoverability | New `FORK.md` (proliferates); CHANGELOG edit (noisy); upstream-clean file edit (cherry-pick debt) |
| Use `gtr create-pull` over browser | Keeps full integration test in scope -- this is the verification | Browser (skips a hop of the round-trip we're verifying) |
| Single commit | One commit = one reviewable unit; trivial change does not warrant multi-step | Multi-commit (overhead with no benefit at this size) |

### Eliminated Options (Essentialism)

| Option Rejected | Why Rejected | Risk of Including |
|-----------------|--------------|-------------------|
| Adding a Robot API HTTP test | #13 is verification of the *agent pipeline*, not the API surface | Scope creep, opens design questions out of #13 |
| Reformatting existing README content | "Surgical changes" protocol -- only touch what is requested | Drive-by churn; reviewer noise |
| Backporting a fix from #16 | Different issue, different gate | Cross-issue contamination |
| Editing `CONTRIBUTING.md` | Not requested; already fork-divergent (+436 lines) -- adding more noise without prompt | Reviewer rejects on scope |

### Simplicity Check

> What if this could be easy?

It is. The change is:

```markdown
### Documentation

- [Robot API security model](docs/ROBOT_SECURITY.md)
- [MCP server reference](docs/MCP_SERVER.md)
- [Robot CLI testing guide](docs/ROBOT_CLI_TESTING.md)
- [End-to-end testing scenario](docs/E2E_TESTING_SCENARIO.md)
```

Six lines of markdown, inserted after an existing subsection header.
A senior engineer would not call this overcomplicated.

**Nothing Speculative Checklist**:

- [x] No features the user did not request
- [x] No abstractions "in case we need them later"
- [x] No flexibility "just in case"
- [x] No error handling for scenarios that cannot occur
- [x] No premature optimisation

## File Changes

### New Files

| File | Purpose |
|------|---------|
| `.docs/research-13.md` | Phase 1 V-Model artefact (already created) |
| `.docs/design-13.md` | Phase 2 V-Model artefact (this document) |

### Modified Files

| File | Changes |
|------|---------|
| `README.md` | Append "Documentation" subsection to existing `## Robot API` block (~6 lines) |

### Deleted Files

None.

## API Design

Not applicable. No public API change.

## Test Strategy

### Unit Tests

Not applicable. Markdown-only patch.

### Integration Tests

The PR opening *is* the integration test for #13. Specifically:

| Test | Location | Purpose |
|------|----------|---------|
| Comment lands on #13 | live Gitea API | Step 2 of issue body |
| Branch pushes successfully | live Gitea API | Confirms write permission |
| PR opens via `gtr create-pull` | live Gitea API | Step 3 of issue body |
| `make fmt` clean | local | Pre-commit hygiene |
| `make vet` clean | local | Pre-commit hygiene |

### Property Tests

Not applicable.

## Implementation Steps

### Step 1: Confirmation comment on #13

**Files:** none
**Description:** Post a comment to #13 starting with `[ferrox]` authorship
attribution per `AGENTS.md`, confirming agent visibility and the planned
PR contents.
**Tests:** `gtr comment` exit code zero; comment visible via
`gtr view-issue --index 13`.
**Estimated:** 2 min

### Step 2: Branch + README patch

**Files:** `README.md`
**Description:** Create branch `task/13-adf-verify`. Append a
`### Documentation` subsection to the existing `## Robot API` block in
`README.md`, listing the four fork-only docs.
**Tests:** Visual diff inspection; rendered preview via raw markdown.
**Estimated:** 5 min

### Step 3: Pre-commit hygiene

**Files:** none (verification only)
**Description:** Run `make fmt` (no Go change but exercises pre-commit
expectation) and `make vet` (sanity).
**Tests:** Both commands exit zero; `git status` shows only the README
change and the two new `.docs/*.md` files.
**Estimated:** 3 min

### Step 4: Commit + push

**Files:** committed
**Description:** Single commit, `[ferrox]` prefix, `Refs #13` trailer.
**Tests:** `git log -1` shows the expected message; `git push` succeeds.
**Estimated:** 2 min

### Step 5: PR + mention

**Files:** none
**Description:** `gtr create-pull --owner terraphim --repo gitea --base main
--head task/13-adf-verify --title "Fix #13: verify ADF integration via
README docs subsection"`. Body mentions `@adf:gitea-reviewer`.
**Tests:** PR visible via `gtr list-pulls`; PR number returned.
**Estimated:** 3 min

## Rollback Plan

If issues are discovered post-merge:

1. `git revert <merge-commit>` on `main`
2. The README change is purely additive -- revert is mechanical and safe
3. No data migration, no feature flag, no consumer impact

## Migration

Not applicable.

## Dependencies

### New Dependencies

None.

### Dependency Updates

None.

## Performance Considerations

Not applicable -- markdown-only patch.

## Open Items

None.

## Approval

- [x] Technical review complete (self-review given trivial scope)
- [x] Test strategy approved (integration test = the PR itself)
- [x] Performance targets agreed (n/a)
- [ ] Human approval received (awaiting reviewer on PR)
