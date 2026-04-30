# Research Document: Verify ADF integration for gitea (#13)

**Status**: Draft
**Author**: Ferrox (Rust Engineer agent)
**Date**: 2026-04-30
**Reviewers**: @adf:gitea-reviewer

## Executive Summary

Issue #13 is a verification ticket -- not a feature ticket. It validates that
the AI Dark Factory (ADF) integration is wired correctly for `terraphim/gitea`
by exercising the agent end-to-end: read issue, post comment, push branch,
open PR. There is no production behaviour to design; the only deliverable is
a small, factual change that demonstrates the integration round-trips.

## Essential Questions Check

| Question | Answer | Evidence |
|----------|--------|----------|
| Energising? | Partial | Verification work itself is mechanical; the *outcome* (proven ADF wiring) unblocks every future agent-driven ticket. |
| Leverages strengths? | Yes | Direct exercise of the gitea-developer/Ferrox path -- exactly the pipeline to validate. |
| Meets real need? | Yes | ADF integration must be proven before higher-PageRank engineering work (#12 cherry-picks, #17 deferred sync) can be confidently delegated to agents. |

**Proceed**: Yes (3/3 YES). Verification is a foundational gate -- it is essential precisely because every later agent-driven change depends on it.

## Problem Statement

### Description

Confirm the agent (Ferrox / `@adf:gitea-developer`) can:

1. Observe an open Gitea issue assigned to it
2. Post a comment to that issue
3. Author a trivial change on a task branch
4. Open a pull request referencing the issue

### Impact

Until proven, every PR opened by an agent against `terraphim/gitea` carries
unverified plumbing risk: silent comment failures, missing assignee mapping,
broken push permissions, or PR-creation 4xx that lands as a mute failure
instead of a loud one.

### Success Criteria

- Issue #13 has a fresh comment from the agent confirming ADF visibility
- A PR is opened on `task/13-adf-verify` referencing `#13`
- The change is small, factual, and reviewable in under 60 seconds
- `make fmt` / `make vet` are clean (no spurious changes leaking in)
- The issue's "Step 2" and "Step 3" checkboxes can be marked complete by the
  validator (Phase 5 close-out)

## Current State Analysis

### Existing Implementation

The issue already has a `root` Phase-5 tracking comment (2026-04-30T02:04 CEST)
listing the three test steps. Step 1 is confirmed; Steps 2 and 3 are open and
explicitly waiting on the agent.

| Step | Status | Action Owner |
|------|--------|--------------|
| 1. Issue visible | Done | -- |
| 2. Agent confirmation comment | Open | This task |
| 3. Trivial PR | Open | This task |

### Code Locations

| Component | Location | Purpose |
|-----------|----------|---------|
| Fork README (Robot API section) | `README.md` | Already lists CLI, Docker; missing pointers to existing docs |
| Robot security doc | `docs/ROBOT_SECURITY.md` | Fork-only; not yet linked from README |
| MCP server doc | `docs/MCP_SERVER.md` | Fork-only; not yet linked from README |
| Robot CLI testing doc | `docs/ROBOT_CLI_TESTING.md` | Fork-only; not yet linked from README |
| E2E testing scenario | `docs/E2E_TESTING_SCENARIO.md` | Fork-only; not yet linked from README |
| AGENTS.md | `AGENTS.md` | Defines pre-commit hygiene; relevant to PR fitness |

### Data Flow

Not applicable. No runtime data flow is changed by this task.

### Integration Points

- Gitea Issues API (read, comment) -- exercised via `gtr` CLI
- Gitea Pulls API (create) -- exercised via `gtr create-pull`
- Local Gitea instance at `http://localhost:3000` (the cloud URL is the
  identifier; the live API responds on the local port)

## Constraints

### Technical Constraints

- **Upstream-pristine doctrine**: Files that already differ from
  `upstream/main` are safe to edit; touching upstream-clean files invites
  future cherry-pick conflicts. `README.md` is already fork-divergent
  (79 added lines, "Robot API" section), so additive changes there are safe.
- **No backwards-incompatible changes**: This is a verification PR, not a
  feature PR -- any change must be additive and reversible.
- **Fork hygiene** (per `AGENTS.md`): no trailing whitespace, no force-push,
  authorship attribution at top of comments and PR body.

### Business Constraints

- **Light touch**: change must be small enough that a human reviewer trusts
  the agent without lengthy analysis.
- **Surfaced visibility**: the comment must clearly state "agent visible,
  proceeding with PR" so the Phase-5 validator can tick Step 2 immediately.

### Non-Functional Requirements

| Requirement | Target | Current |
|-------------|--------|---------|
| Time-to-PR | < 30 min from claim | -- |
| Lines changed | < 30 | -- |
| Files touched | 1-2 (markdown only) | -- |

## Vital Few (Essentialism)

### Essential Constraints (Max 3)

| Constraint | Why It's Vital | Evidence |
|------------|----------------|----------|
| Change must be additive and reversible | A verification PR that breaks something defeats its own purpose | Phase-5 gate logic |
| Touch only fork-divergent files | Upstream-clean files create cherry-pick debt for #12/#17 | `git diff upstream/main --stat` shows 13 fork-only md files; `README.md` already +79 lines |
| Demonstrate the *full* round-trip, not just code | Issue body explicitly enumerates 4 round-trip steps | `## Test Steps` section of #13 |

### Eliminated from Scope

| Eliminated Item | Why Eliminated |
|-----------------|----------------|
| Adding a Robot API HTTP endpoint or test | Out of scope; #13 is verification-only, not feature work |
| Touching `upstream/main`-clean files (e.g. CHANGELOG-archived) | Creates merge debt; violates fork hygiene |
| Documenting MCP server protocol details | Already covered by `docs/MCP_SERVER.md`; only README pointer is missing |
| Refactoring the existing Robot API README block | Out of scope; "drive-by" improvement banned by surgical-changes protocol |
| Backporting fixes from #16 / #17 | Different tickets, blocked dependencies |

## Dependencies

### Internal Dependencies

| Dependency | Impact | Risk |
|------------|--------|------|
| `gtr` CLI | Required for comment + PR | Low -- already proven on #15 |
| Local Gitea API | Required for round-trip | Low -- responding on `:3000` |
| `make fmt` / `make vet` | Pre-commit gate | Low -- markdown-only change |

### External Dependencies

None. Markdown-only patch.

## Risks and Unknowns

### Known Risks

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| README change conflicts with future upstream sync | Low | Low | Already fork-divergent; sync strategy is selective cherry-pick (per #14) |
| Comment posts but PR fails | Low | Medium | `gtr` exit code is checked; manual fallback documented in design |
| Reviewer flags the change as too trivial | Low | Low | Issue body literally requests "trivial change (e.g., update README)" |

### Open Questions

None. Issue body is unambiguous; previous Phase-5 comment confirms expected
deliverables.

### Assumptions Explicitly Stated

| Assumption | Basis | Risk if Wrong | Verified? |
|------------|-------|---------------|-----------|
| `gtr` writes through to the live Gitea instance | Worked on #15; #18 PR merged via `gtr` | Comment / PR not landing | Yes -- #18 round-trip |
| Agent name "Ferrox" maps to assignee `gitea-developer` | Issue is assigned to `gitea-developer`; ferrox commits use `[ferrox]` prefix and were merged | Reviewer rejects on identity mismatch | Partial -- prior PRs accepted |
| Markdown-only change passes `make vet` (no Go files) | `vet` only inspects Go packages | False negative on unrelated breakage | Yes -- standard Go tooling behaviour |

### Multiple Interpretations Considered

| Interpretation | Implications | Why Chosen/Rejected |
|----------------|--------------|---------------------|
| "Trivial change = README typo fix" | Minimal, but no real value | Rejected -- still trivial, but adds genuine signpost value |
| "Trivial change = README docs subsection linking existing fork docs" | Useful and small | **Chosen** -- additive, reversible, improves discoverability |
| "Trivial change = bump CHANGELOG.md note" | Touches upstream-divergent file | Rejected -- CHANGELOG diffs are noisy; subsection is cleaner |
| "Trivial change = new fork-only file (e.g. `FORK.md`)" | Avoids touching README | Rejected -- proliferates top-level files; README already has Robot API section that needs the link |

## Research Findings

### Key Insights

1. The fork's `README.md` already documents Robot API CLI and Docker but
   does not link the four existing fork-only docs (`ROBOT_SECURITY.md`,
   `MCP_SERVER.md`, `ROBOT_CLI_TESTING.md`, `E2E_TESTING_SCENARIO.md`).
   Adding that link block is genuinely useful and naturally trivial.
2. Local Gitea API runs at `http://localhost:3000`; cloud URL is identifier
   only. Direct `curl` against `https://git.terraphim.cloud/api/v1/...`
   returns 404. Documenting this in the implementation log saves the next
   agent a debugging cycle.
3. Phase-5 tracking comment on #13 explicitly enumerates expected steps,
   including assignee confirmation -- so the agent comment must explicitly
   reference `@adf:gitea-developer` visibility.

### Relevant Prior Art

- PR #18 (Fix #15: TestRobotapi suite) -- proves the same Ferrox round-trip
  for a non-trivial code change. This task is the markdown-only counterpart.
- PR #14 (docs Phase 1-2 for #12) -- proves the V-Model artefact path is
  accepted for documentation-only PRs.

### Technical Spikes Needed

None.

## Recommendations

### Proceed/No-Proceed

**Proceed.** Verification of ADF integration is a single-step prerequisite
for unlocking confident agent delegation across the rest of the backlog.
Cost is minimal (<30 minutes); deferral cost compounds across every future
agent ticket.

### Scope Recommendations

Strictly: comment + 1 markdown patch + PR. Nothing else.

### Risk Mitigation Recommendations

- Run `make fmt` and `make vet` despite the markdown-only nature, to catch
  any incidental file-system surprises.
- Verify the rendered Markdown in the README diff before pushing.
- Use `gtr create-pull` (not browser) to keep the integration end-to-end.

## Next Steps

If approved (self-approving here given trivial scope and explicit issue body):

1. Author `.docs/design-13.md` (Phase 2)
2. Comment on #13 confirming visibility
3. Branch `task/13-adf-verify`, patch README, commit
4. `make fmt && make vet`
5. Push branch, open PR via `gtr create-pull`, mention `@adf:gitea-reviewer`

## Appendix

### Reference Materials

- Issue #13 body and Phase-5 tracking comment (root, 2026-04-30T02:04)
- `git diff upstream/main --stat -- '*.md'` (fork-divergence map)
- `AGENTS.md` (commit hygiene rules)

### Code Snippets

Not applicable -- markdown-only change.
