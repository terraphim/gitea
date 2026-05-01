# Research Document: Issue #17 -- Deferred Upstream Module Sync (Picks 5-7)

**Status**: Draft -- awaiting human approval
**Author**: Ferrox (Rust Engineer / fork sync investigator)
**Date**: 2026-05-01
**Reviewers**: @adf:gitea-reviewer, repo coordinator
**Related**: Issue #12 (parent security cherry-pick sprint), PR #16 (picks 1, 3, 4),
  blocker docs `.docs/blocker-12-pick5-cascade.md`, `.docs/blocker-12-pick6-cascade.md`,
  `.docs/blocker-12-pick7-cascade.md` (currently on PR #16 branch only)

## Executive Summary

Issue #17 proposes syncing 5-6 upstream `modules/*` packages to enable picks 5, 6, 7 to
land. The blocker docs already on PR #16 record a coordinator decision (2026-04-30) to
**skip** these three picks under design v2 §1 ("Avoid at all cost: prerequisite refactor
picks beyond the 7 named SHAs"). There is therefore an unresolved contradiction in the
record between #17's framing and the most recent coordinator decision.

After empirical inspection of the upstream commits and the missing symbols, the picks
split into three different risk classes. **Recommendation**: do not perform a "5-module
sync"; instead, scope down to surgical symbol backports for picks 6 and 7 only, and
explicitly close pick 5 as deferred to upstream-rebase time. Request human decision
before any implementation.

## Essential Questions Check

| Question | Answer | Evidence |
|----------|--------|----------|
| Energizing? | **No** | The work is upstream archaeology against a fork that is 341 commits behind upstream; the design v2 §1 already classified this as "Avoid at all cost". |
| Leverages strengths? | **Partial** | Pick 7 backport is straightforward Go module work (~70 LOC). Pick 5 is template-system archaeology spanning markup + templates + public + JS + 13 templates -- not a fork-sync engineer's strength. |
| Meets validated need? | **Partial** | CSP nonce, container auth, and attachment CSP are all real defence-in-depth wins. But the original coordinator (Echo, 2026-04-30) recorded that the deployed surface does not require pick 6 (container auth) and that pick 5/7 ergonomics did not warrant the cascade cost. No re-validation of that need has been recorded. |

**Result: 1/3 fully YES, 2/3 partial → STOP and challenge essentialism before proceeding.**
The "Avoid at all cost" list in the parent design includes exactly this kind of work.

## Problem Statement

### Description
Picks 5, 6, 7 (CSP nonce, container auth, attachment CSP) from the upstream security
cherry-pick set are blocked because the upstream commits reference symbols that do not
exist in our fork:

- `public.AssetURI`, `markup.RenderIFrame`, `RenderOptions.StandalonePageOptions` (pick 5)
- `storage.ServeDirectOptions` (pick 6)
- `ContentDispositionType`, `ContentDispositionInline`, `ContentDispositionAttachment`,
  `encodeContentDisposition`, `typesniffer.FromContentType` (pick 7)

Upstream introduced these symbols as part of a series of refactors that have not been
backported to the fork.

### Impact
Without these picks, the fork lacks:
- **Pick 5**: CSP `script-src` nonce protection (defence-in-depth against XSS).
- **Pick 6**: Container registry auth fix for public-instance unauthenticated pulls.
- **Pick 7**: CSP refinement for attachment serving (media vs other types).

Pick 3 (`X-Content-Type-Options: nosniff`) is already on PR #16 and provides partial
overlap with picks 5 and 7.

### Success Criteria
A successful resolution of this issue produces one of:
1. A bounded set of surgical symbol backports that allows picks 6 and 7 (and possibly
   5) to land cleanly with `make build` and `make test` green; OR
2. A documented decision to formally close picks 5/6/7 as "deferred to upstream rebase",
   with the security gap recorded in the project risk register.

## Current State Analysis

### Existing Implementation (fork side)
| Module | Files (fork / upstream) | Drift (lines) | Notes |
|--------|------------------------|---------------|-------|
| `modules/public` | 5 / 8 | +497 / -28 | 3 new files upstream incl. `manifest.go`. |
| `modules/markup` | 69 / 71 | +408 / -184 | Refactor incl. `RenderIFrame`, `StandalonePageOptions`. |
| `modules/templates` | 30 / 30 | +256 / -66 | `NewFuncMap` renamed to `newFuncMapWebPage`, `AssetURI` added, `SanitizeHTML` lowercased. |
| `modules/storage` | 10 / 10 | +302 / -101 | New `ServeDirectOptions` struct. |
| `modules/httplib` | 5 / 7 | +287 / -88 | New file `content_disposition.go`. |
| `modules/typesniffer` | 2 / 2 | +4 / -0 | Trivial: 1 new function. |

(Drift measured as `git diff --shortstat origin/main upstream/main -- modules/X/`.)

### Code Locations
| Symbol | Upstream Path | Size |
|--------|---------------|------|
| `AssetURI` | `modules/public/manifest.go:135` | new file (164 lines) |
| `RenderIFrame` | `modules/markup/render.go:210` | ~25 lines, depends on `htmlutil.HTMLFormat`, `setting.PanicInDevOrTesting`, `RenderOptions.Metas`, `ExternalRendererOptions` |
| `StandalonePageOptions` | `modules/markup/renderer.go` (struct field) | small |
| `ServeDirectOptions` | `modules/storage/storage.go:44` | 4-line struct |
| `ContentDispositionType` + constants + `encodeContentDisposition` | `modules/httplib/content_disposition.go` | new file (65 lines) |
| `FromContentType` | `modules/typesniffer/typesniffer.go:187` | 3-line function |

### Pick Touch Surface (upstream side)
- **Pick 5** (`82bfde2a37`): 18 files, 134+/52- -- 6 Go files + 11 templates + 1 JS + 1 test.
  Touches `services/context/context.go` and adds new file `services/context/context_template.go`
  -- this widens the scope beyond `modules/*`.
- **Pick 6** (`6ed861589a`): 2 files, 24+/23- -- only `routers/api/packages/container/container.go`
  + integration test.
- **Pick 7** (`15b23f037d`): 2 files, 64+/14- -- only `modules/httplib/serve.go` + test.

### Cascade Verification
The blocker doc for pick 5 lists `NewFuncMap` as missing. **This is incorrect**: the
fork has `NewFuncMap` at `modules/templates/helper.go:26`. Upstream renamed it to
`newFuncMapWebPage` and added a sibling map for AssetURI. The actual problem is the
*upstream caller* expects the new map structure, not that the old name is missing. The
remaining blocker-doc claims for picks 5/6/7 are accurate.

## Constraints

### Technical Constraints
- The fork is 341 commits behind upstream; full rebase is not in scope of #17.
- Any backported symbol must compile against the fork's existing dependency graph
  (`modules/htmlutil`, `modules/setting`, `services/context`, etc).
- `make build` and `make vet` must pass before any commit.
- `gofumpt` formatting and copyright headers required by AGENTS.md.

### Business Constraints
- Design v2 §1 (parent design): "Avoid at all cost: prerequisite refactor picks beyond
  the 7 named SHAs". This is a hard constraint set by the parent design and not yet
  rescinded in writing.
- PR #16 is **OPEN, not merged** at the time of writing. Picks 1, 3, 4 are not on
  `main`. Re-applying picks 5/7 against a tree that lacks pick 3 creates merge risk.

### Non-Functional Requirements
| Requirement | Target | Current |
|-------------|--------|---------|
| Build | green on `go build ./...` | green on main (without picks 5-7) |
| Vet | clean on `go vet ./...` | clean on main |
| Test | unit + integration suites green | green on main |
| Security | CSP nonce, container auth, attachment CSP defence | absent |

## Vital Few

### Essential Constraints (Max 3)
| Constraint | Why It's Vital | Evidence |
|------------|----------------|----------|
| No prerequisite refactor cascade | Direct rule from parent design v2 §1; cascade work historically blew the sprint scope | PR #16 blocker docs |
| PR #16 must merge first | Picks 5/7 patch files that pick 3 already touched (CSP header path); ordering matters | PR #16 still open, contains pick 3 |
| Compile + vet must stay green | AGENTS.md hard rule; Sentrux quality gate | CI policy |

### Eliminated from Scope
| Eliminated Item | Why Eliminated |
|-----------------|----------------|
| Full sync of `modules/markup` (24 files, 408+/184-) | Cascade scope, hits parent design §1. |
| Full sync of `modules/templates` (11 files, 256+/66-) | Cascade scope. |
| Full sync of `modules/public` (5 files, 497+/28-) | Cascade scope; `manifest.go` brings entire web-asset pipeline. |
| Full sync of `modules/storage` (10 files, 302+/101-) | Cascade scope. |
| Backport of `services/context/context_template.go` (new upstream file) | Required by pick 5; widens beyond `modules/*` -- not in #17 scope as written. |
| Re-applying pick 5 in this issue | Cascade dependencies on markup + templates + public + JS make this NOT bounded. |
| Modifying any code while PR #16 is open | Ordering risk; would race with the in-flight reviewer changes. |

## Dependencies

### Internal Dependencies
| Dependency | Impact | Risk |
|------------|--------|------|
| PR #16 merge | Blocks: picks 5/7 patch the same headers as pick 3 | High -- cannot proceed until merged |
| Echo's Option B decision (2026-04-30) | Contradicts #17 framing | Medium -- needs explicit reconciliation |

### External Dependencies
| Dependency | Version | Risk | Alternative |
|------------|---------|------|-------------|
| `upstream/main` `82bfde2a37` (pick 5) | as-is | High cascade | Skip pick 5 |
| `upstream/main` `6ed861589a` (pick 6) | as-is | Low (4-line struct + helper) | Surgical backport of `ServeDirectOptions` |
| `upstream/main` `15b23f037d` (pick 7) | as-is | Low (single new file + 1 function) | Surgical backport of `content_disposition.go` + `FromContentType` |

## Risks and Unknowns

### Known Risks
| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Backporting `manifest.go` drags in entire upstream asset pipeline | High | High | Do not attempt for pick 5 |
| `RenderIFrame` chain pulls htmlutil refactor, setting changes, RenderOptions struct redesign | High | High | Skip pick 5 |
| Pick 7's `serve.go` itself drifts vs fork (287+/88-) -- backport may not isomorphically merge | Medium | Medium | Verify by attempting cherry-pick post-symbol-backport in a spike |
| Pick 6's `container.go` call site uses helpers (`prepareServeDirectOptions`) we don't have | Medium | Medium | Spike to confirm; may require fork-side adaptation rather than backport |
| PR #16 review changes alter pick 3 in ways that break pick 7 application | Medium | Medium | Wait for PR #16 to merge, then re-baseline |

### Open Questions
1. **Does the Option B decision (skip picks 5/6/7) still hold?** -- @adf:gitea-reviewer / coordinator.
2. **If we override Option B partially, which picks are in scope?** -- coordinator.
3. **What is the deployed surface for pick 6's container auth?** -- referenced in pick 6 blocker but not concluded.
4. **Should pick 5 wait for the bigger upstream rebase (#12 long-term)?** -- coordinator.

### Assumptions Explicitly Stated
| Assumption | Basis | Risk if Wrong | Verified? |
|------------|-------|---------------|-----------|
| PR #16 will merge as-is or with cosmetic changes only | PR is open and assigned to gitea-reviewer | If picks 1/3/4 change semantically, picks 5/7 may need re-derivation | No |
| The fork's `htmlutil.HTMLFormat` is API-compatible with upstream's | Same package name, used elsewhere | Pick 7 backport could fail to compile | No |
| The fork's `setting.PanicInDevOrTesting` exists (used by `RenderIFrame`) | Untested | Pick 5 backport even more cascading | No |
| `NewFuncMap` rename to `newFuncMapWebPage` does not require call-site updates | Untested | If many callers reference `NewFuncMap`, rename ripples broadly | No |

### Multiple Interpretations Considered
| Interpretation | Implications | Why Chosen/Rejected |
|----------------|--------------|---------------------|
| **A**: #17 RESCINDS Option B and mandates the full sync | Implementer should plan a multi-day backport across 5 modules | **Tentatively rejected**: contradicts the design v2 §1 hard constraint and the most recent coordinator decision; needs explicit human override in writing. |
| **B**: #17 is a tracking-only issue; Option B stands | Close #17 with a pointer to the blocker docs; no implementation | Plausible but loses the future opportunity to land picks 6/7 surgically. |
| **C**: Scope #17 down to picks 6 and 7 only via surgical symbol backport | ~70 LOC backport for pick 7, ~10 LOC for pick 6 (TBC), no module sync | **Recommended**: respects the §1 constraint (no module-wide refactor), captures real security value, leaves pick 5 properly deferred. |

## Research Findings

### Key Insights
1. **Picks 5, 6, 7 have wildly different cascade profiles.** Treating them as one work
   item ("sync 5 modules") inflates risk. Pick 5 is a template/markup/public refactor
   cascade; pick 7 is a self-contained 65-line file plus a 3-line function; pick 6 is
   a 4-line struct plus an unverified helper.
2. **The blocker docs contain at least one factual error** (`NewFuncMap` listed as
   missing when it exists). The Option B decision rested on partially incorrect data.
   The decision still defends pick 5 (cascade is real) but the case for skipping 7
   was never tested -- pick 7's actual missing surface is a single new file.
3. **PR #16 is the gating prerequisite.** The task brief explicitly notes #17 is
   "unblocked once #16 merges"; #16 is currently open. No implementation can begin
   safely yet.
4. **The "5 modules" framing in #17 conflates different problems.** `modules/typesniffer`
   needs 3 lines; `modules/storage` needs 4 lines; `modules/markup` would need a multi-week
   refactor. Bundling them implies the same effort which is misleading.

### Relevant Prior Art
- `.docs/blocker-12-pick5-cascade.md` (Echo, 2026-04-30) -- pick 5 cascade analysis
- `.docs/blocker-12-pick6-cascade.md` (Echo, 2026-04-30) -- pick 6 missing symbol
- `.docs/blocker-12-pick7-cascade.md` (Echo, 2026-04-30) -- pick 7 missing symbols
- `.docs/research-13.md`, `.docs/research-15.md` -- prior V-Model research patterns

### Technical Spikes Needed (only if scope C is approved)
| Spike | Purpose | Estimated Effort |
|-------|---------|------------------|
| Pick 7 surgical backport spike | Add `content_disposition.go` (65 LOC) + `FromContentType` (3 LOC), then cherry-pick `15b23f037d` and verify build | 2-4 hours |
| Pick 6 surgical backport spike | Add `ServeDirectOptions` struct + audit container.go call site for other missing helpers | 2-4 hours |

## Recommendations

### Proceed/No-Proceed
**Conditional No-Proceed on the issue as currently written.** The "5-module sync"
framing should not be implemented. Recommend:

1. **Block on PR #16 merge.** The task brief itself states this. No code work on #17
   until then.
2. **Reconcile the contradiction.** The coordinator (Echo) recorded Option B; #17
   asks for the opposite. Get an explicit written decision from the coordinator on
   which stands.
3. **If the coordinator approves overriding Option B**, proceed with **scope C**
   (surgical backport for picks 6 and 7 only; pick 5 stays deferred). This makes
   #17 a bounded ~6-hour task instead of an unbounded refactor sprint.
4. **If Option B stands**, close #17 referencing the blocker docs and add the missing
   security capabilities to a follow-up "upstream rebase" issue.

### Scope Recommendations
- Reduce #17's scope to picks 6 and 7 via surgical backport.
- Explicitly remove `modules/markup`, `modules/templates`, `modules/public` from scope.
- Keep `modules/typesniffer` (3 lines), `modules/httplib` (1 new file), `modules/storage`
  (1 struct) as the only touched modules.

### Risk Mitigation Recommendations
- Run each pick as a separate PR to keep blast radius small.
- Add regression tests around CSP/Content-Disposition behaviour before backporting.
- Verify call-site compatibility before cherry-picking the upstream pick commit.

## Next Steps

If approved (scope C):
1. Wait for PR #16 to merge.
2. Rebase research/design from new `main`.
3. Phase 2 design: produce `.docs/design-17.md` with surgical backport plan and per-pick PRs.
4. Phase 3 implementation: separate branches `task/17-pick7` and `task/17-pick6`.

If approved (close as deferred):
1. Update issue #17 referencing blocker docs.
2. Open follow-up "upstream rebase" issue under #12 lineage.

## Appendix

### Reference Materials
- Issue #17: https://git.terraphim.cloud/terraphim/gitea/issues/17
- Issue #12: https://git.terraphim.cloud/terraphim/gitea/issues/12
- PR #16: https://git.terraphim.cloud/terraphim/gitea/pulls/16
- Upstream commit pick 5: `82bfde2a37` ("Use Content-Security-Policy: script nonce (#37232)")
- Upstream commit pick 6: `6ed861589a` ("Fix container auth for public instance (#37290)")
- Upstream commit pick 7: `15b23f037d` ("Fix attachment Content-Security-Policy (#37455)")

### Symbol-Level Backport Inventory (scope C, for design phase)

**Pick 7 backport surface (~70 LOC):**
- New file `modules/httplib/content_disposition.go` (cherry-pick from upstream)
- Add `FromContentType` to `modules/typesniffer/typesniffer.go`
- Then cherry-pick `15b23f037d` -- expect minor merge in `serve.go` due to fork drift

**Pick 6 backport surface (~10 LOC, unverified):**
- Add `ServeDirectOptions` struct to `modules/storage/storage.go`
- Audit `routers/api/packages/container/container.go:718` upstream call site for
  additional helpers (`prepareServeDirectOptions`, etc) -- may require additional
  surgical backports

**Pick 5: out of scope for #17.** Proper landing requires backporting:
- `public.AssetURI` + `manifest.go` (164 LOC + asset pipeline integration)
- `markup.RenderIFrame` + `RenderOptions.StandalonePageOptions` + `htmlutil` chain
- `services/context/context_template.go` (new file)
- 13 templates and 1 JS file
This is an upstream-rebase-class effort, not a bounded task.
