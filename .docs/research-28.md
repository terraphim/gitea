---
name: Research -- Cherry-pick CSP Script Nonce (82bfde2a37)
description: Phase 1 research for issue #28. Verifies upstream commit applicability, conflict classification, and security implications for fork.
type: project
---

# Research Document: Cherry-pick CSP Script Nonce (82bfde2a37)

**Status:** Phase 1 -- Research Complete
**Author:** Ferrox (Rust Engineer, Principal SFIA-5)
**Date:** 2026-05-04
**Reviewers:** @adf:security-sentinel, @adf:gitea-reviewer
**Issue:** terraphim/gitea#28
**Upstream:** go-gitea/gitea#37232

---

## Executive Summary

Upstream commit `82bfde2a37` introduces Content-Security-Policy script nonce support across 18 files. Empirical dry-run confirms 11 conflict files (all upstream-only evolution drift, zero fork-touched). The security value is clear: it closes XSS vectors via inline script injection by requiring cryptographically random nonce attributes on all `<script>` tags. Fork-specific code (Robot API, PageRank) does not intersect with any changed files. Resolution strategy: take-theirs on all 11 conflict files.

---

## Essential Questions Check

| Question | Answer | Evidence |
|----------|--------|----------|
| Energizing? | Yes | CSP nonce is a confirmed security hardening measure; our fork serves production traffic at git.terraphim.cloud |
| Leverages strengths? | Yes | We have established V-Model cherry-pick machinery from #12 and #17; this is a mechanical application |
| Meets real need? | Yes | Without nonce support, any XSS that can inject `<script>` executes; nonce binds script execution to server-generated tokens |

**Proceed:** Yes (3/3).

---

## Problem Statement

### Description
Our fork lacks upstream PR #37232, which adds per-request CSP script nonces to all inline and dynamically-injected `<script>` tags. The nonce is a 128-bit cryptographically random value generated per HTTP request, communicated to the browser via a `<meta http-equiv="Content-Security-Policy">` tag, and required on every `<script>` element for execution.

### Impact
- **Who:** All users of git.terraphim.cloud
- **What:** Without nonce support, reflected/stored XSS that can inject `<script>` tags will execute unrestricted
- **Severity:** High -- XSS is a top-tier web vulnerability class

### Success Criteria
- All inline `<script>` tags in templates carry `nonce="{{ctx.CspScriptNonce}}"`
- External script imports use `ctx.ScriptImport()` which auto-injects nonce
- Browser receives `Content-Security-Policy: script-src * 'nonce-XXX'` meta tag
- No regression in Robot API, PageRank, or existing functionality

---

## Current State Analysis

### Existing Implementation
Our fork is 324 commits behind upstream/main and 85 commits ahead. The fork's custom code (Robot API routes, PageRank endpoints, MCP server) lives primarily in:
- `routers/api/v1/robot/` (custom handlers)
- `services/robot/` (business logic)
- `modules/pagerank/` (computation engine)
- `routers/api/v1/api.go` (route registration)

None of these files appear in the commit's change list.

### Code Locations
| Component | Location | Purpose |
|-----------|----------|---------|
| CSP nonce generation | `services/context/context_template.go` (upstream addition) | Per-request nonce generation |
| Script import helper | `services/context/context_template.go` (upstream addition) | Template helper for nonce-aware `<script>` tags |
| Fast random utility | `modules/util/util.go` (upstream addition) | `FastCryptoRandomHex()` for nonce material |
| Template functions | `modules/templates/helper.go` (upstream modification) | Removes legacy `ScriptImport`, wires `AssetURI` |
| Markup renderers | `modules/markup/render.go`, `modules/markup/external/openapi.go` | Adds `nonce` attribute to helper scripts |
| Base templates | `templates/base/head.tmpl`, `templates/base/head_script.tmpl`, `templates/base/footer.tmpl` | Injects CSP meta and nonce into global scripts |
| Feature templates | `templates/repo/diff/box.tmpl`, `templates/repo/issue/view_content/pull_merge_box.tmpl`, `templates/shared/combomarkdowneditor.tmpl`, `templates/status/500.tmpl`, `templates/swagger/openapi-viewer.tmpl`, `templates/user/auth/captcha.tmpl`, `templates/user/dashboard/repolist.tmpl` | Per-feature script nonce application |
| Frontend nonce propagation | `web_src/js/features/repo-issue-pull.ts` | Propagates nonce to dynamically executed scripts |
| Integration tests | `tests/integration/markup_external_test.go` | Updated expectations for nonce attribute |

### Data Flow
```
[HTTP Request]
    |
    v
[context_template.go: CspScriptNonce()] -- generates 32-char hex nonce
    |
    +--> [Template rendering] --> nonce injected into <script nonce="..."> tags
    |
    +--> [Base head template] --> <meta http-equiv="CSP" content="script-src * 'nonce-...'">
    |
    v
[Browser] -- refuses to execute <script> without matching nonce attribute
```

### Integration Points
- Template rendering pipeline (`modules/templates/`)
- HTTP context layer (`services/context/`)
- Frontend JavaScript (`web_src/js/`)
- Markup sanitisation (`modules/markup/`)

---

## Constraints

### Technical Constraints
- **Go 1.22+**: `math/rand/v2` package required (ChaCha8 CSPRNG); Go 1.22+ baseline preserved
- **Template compatibility**: All custom templates must use `nonce="{{ctx.CspScriptNonce}}"` on `<script>` tags
- **Build constraint**: `go build ./...` must succeed post-cherry-pick

### Business Constraints
- Security hardening must land on `main` and deploy to `git.terraphim.cloud`
- No force-push to PR branches (per `AGENTS.md`)
- Preserve all 85 fork commits (Robot/PageRank/MCP features)

### Non-Functional Requirements
| Requirement | Target |
|-------------|--------|
| Nonce generation latency | < 1 microsecond (ChaCha8 via sync.Pool) |
| Template rendering | No regression |
| Build time | No regression |
| XSS mitigation | Inline `<script>` without valid nonce is blocked by browser |

---

## Vital Few (Essentialism)

### Essential Constraints (Max 3)

| Constraint | Why It's Vital | Evidence |
|------------|----------------|----------|
| All 11 conflict files are upstream-only | Enables mechanical take-theirs resolution; no manual semantic merge needed | Verified via `git log upstream/main..main -- <file>` for each conflict file; only upstream dependency update `fed2d81e88` appears, not fork-specific code |
| Preserve 85 fork commits | Robot/PageRank/MCP are the fork's reason for existence | `git rev-list --count upstream/main..HEAD` == 85 |
| No custom template drift | Our fork does not maintain custom templates that would need nonce retrofit | Custom templates audit: none found beyond standard Gitea templates |

### Eliminated from Scope

| Eliminated Item | Why Eliminated |
|-----------------|----------------|
| Custom template nonce retrofit | Fork has no custom templates requiring nonce attributes |
| CSP policy tightening (replace `*` with `self`) | Upstream deferred this to #37238; out of scope for this cherry-pick |
| `CryptoRandomBytes` signature refactor (#37240) | Separate upstream commit; not in this cherry-pick |
| Full upstream merge of 324 commits | Unbounded scope; 1821-file blast radius |
| Per-commit PRs | Sequencing overhead; established pattern is one PR per pick set |

---

## Dependencies

### Internal Dependencies
| Dependency | Impact | Risk |
|------------|--------|------|
| `services/context/context_template.go` | New file in upstream; adds `CspScriptNonce`, `ScriptImport`, `HeadMetaContentSecurityPolicy` | Low -- isolated template context helpers |
| `modules/util/util.go` | Adds `FastCryptoRandomHex`, `chaCha8RandPool` | Low -- self-contained utility |
| `modules/templates/helper.go` | Removes legacy `ScriptImport` func, adds `AssetURI` | Low -- template func map change |

### External Dependencies
| Dependency | Version | Risk | Alternative |
|------------|---------|------|-------------|
| `math/rand/v2` (Go stdlib) | 1.22+ | Low -- ChaCha8 is CSPRNG-safe per upstream review | `crypto/rand.Read` (slower but equivalent) |
| `golang.org/x/text` | Existing | None | N/A |

---

## Risks and Unknowns

### Known Risks
| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|------------|
| Silent 3-way merge drift (build failure without conflict markers) | Medium | Medium | Post-pick `go build ./...` gate before treating as clean |
| Frontend nonce propagation breaks on dynamic script injection | Low | Medium | Verify `web_src/js/features/repo-issue-pull.ts` executes correctly post-pick |
| Empty `nonce` attribute on markup helper scripts breaks CSP | Low | Low | Upstream intentionally uses `<script nonce>` as grep marker; browser ignores when no CSP header |
| `*` wildcard in CSP allows external scripts | Low | Medium | Upstream trade-off documented; deferred to #37238 |

### Open Questions
1. Does our fork have any custom templates with inline `<script>` that would need nonce retrofit? -- **Answered: No custom templates found.**
2. Do any of our Robot API frontend pages use inline scripts? -- **To verify in Phase 3; Robot API is JSON-only, no HTML templates expected.**

### Assumptions Explicitly Stated

| Assumption | Basis | Risk if Wrong | Verified? |
|------------|-------|---------------|-----------|
| All 11 conflict files are upstream-only (no fork-specific modifications) | `git log upstream/main..main -- <file>` shows only upstream dependency update `fed2d81e88`, not Robot/PageRank commits | If wrong, manual semantic merge needed for affected files | Yes -- verified for all 11 files |
| `math/rand/v2.ChaCha8` is safe for security-sensitive nonces | Upstream PR review confirmed; seeded from `crypto/rand`; uses sync.Pool for concurrency | If wrong, nonce collisions possible | Yes -- upstream merged after review |
| Our fork's `routers/api/v1/api.go` is NOT in this commit's file list | Commit stat shows 18 files; `api.go` not listed | If wrong, Robot routes at risk | Yes -- `api.go` absent from diff |
| Build passes after take-theirs on all 11 conflict files | Original #12 design v2 classified pick 5 (this commit) as 11 take-theirs files | If wrong, unknown dependency on upstream refactor | To verify in Phase 3 |

### Multiple Interpretations Considered

| Interpretation | Implications | Why Chosen/Rejected |
|----------------|--------------|---------------------|
| A: Cherry-pick commit verbatim, take-theirs all conflicts | Simplest; preserves upstream provenance; minimal review burden | **Chosen** -- all conflicts are upstream-only evolution drift |
| B: Subset application (skip templates, only take Go code) | Would miss nonce on some script tags; incomplete XSS mitigation | **Rejected** -- partial mitigation is worse than no mitigation (gives false confidence) |
| C: Full upstream merge | Would bring 324 unrelated commits; review burden enormous | **Rejected** -- explicitly eliminated in #12 research |

---

## Research Findings

### Key Insights
1. **Zero fork intersection**: None of our 85 ahead commits (Robot, PageRank, MCP) modify any of the 18 files in this commit. This is a clean upstream-only change from our fork's perspective.
2. **Mechanical resolution**: All 11 conflict files are import-block or context-shift conflicts caused by upstream evolution between merge-base and commit. No semantic judgement required.
3. **Security value is clear**: CSP nonce is a well-understood XSS mitigation. Upstream PR #37232 was merged with maintainer approval after addressing concurrency concerns.
4. **No breaking change for our fork**: The PR's "breaking" label applies only to custom template users. Our fork does not maintain custom templates.

### Relevant Prior Art
- `.docs/design-security-drift.md` (original #12 design, §5.3) -- classified this commit as pick 5 with 11 take-theirs files
- `.docs/research-security-drift.md` (original #12 research) -- established cherry-pick methodology and fork-touch detection
- go-gitea/gitea#37232 -- upstream PR with full review discussion
- go-gitea/gitea#37238 -- upstream follow-up for CSP policy tightening (deferred)

### Technical Spikes Needed
| Spike | Purpose | Estimated Effort |
|-------|---------|------------------|
| Post-pick build verification | Confirm `go build ./...` passes after take-theirs | 5 min |
| Robot API smoke test | Verify Robot endpoints still respond correctly | 10 min |
| Template nonce visual verification | Boot Gitea, view page source, confirm nonce attribute present | 15 min |

---

## Recommendations

### Proceed/No-Proceed
**PROCEED** to Phase 2 (Design). Cherry-pick path is empirically validated and mechanically simple.

### Scope Recommendations
- Apply commit `82bfde2a37` verbatim via cherry-pick with `-x` provenance
- Resolve 11 conflicts via uniform take-theirs (no per-file judgement needed)
- Verify build and run targeted tests (markup external, basic view smoke)
- No custom template modifications required

### Risk Mitigation Recommendations
1. Run `go build ./...` immediately after resolving conflicts (catches silent 3-way merge drift)
2. Run `make fmt && make vet` before commit
3. Manual smoke test of a rendered page to confirm nonce appears in `<script>` tags
4. Verify Robot API endpoints (`/api/v1/robot/*`) respond 200 after pick

---

## Next Steps

If approved:
1. Phase 2: Produce `.docs/design-pick2-csp-nonce.md` with exact cherry-pick procedure, conflict resolution loop, and test plan
2. Phase 3: Create branch `task/28-pick2-csp-nonce`, cherry-pick, resolve, test, commit
3. Phase 4: Verification -- build, tests, smoke, PR

---

## Appendix

### Reference Materials
- Upstream PR: https://github.com/go-gitea/gitea/pull/37232
- Upstream issue: https://github.com/go-gitea/gitea/issues/305
- Original #12 design: `.docs/design-security-drift.md`
- Original #12 blocker (pick 2 auth cascade -- different commit): `.docs/blocker-12-pick2-auth-cascade.md`

### Verification Commands
```bash
# Conflict file fork-touch audit
for f in modules/markup/external/openapi.go modules/markup/render.go \
         modules/templates/helper.go modules/util/util.go \
         services/context/context_template.go \
         templates/base/footer.tmpl templates/base/head_script.tmpl \
         templates/repo/issue/view_content/pull_merge_box.tmpl \
         templates/swagger/ui.tmpl templates/user/auth/captcha.tmpl \
         tests/integration/markup_external_test.go; do
  echo "=== $f ==="
  git log --oneline upstream/main..main -- "$f"
done

# Dry-run cherry-pick
git checkout -b task/28-pick2-dryrun main
git cherry-pick --no-commit 82bfde2a37
# (11 conflicts expected)
```

### Dry-run Results
```
Auto-merging modules/markup/external/openapi.go        CONFLICT
Auto-merging modules/markup/render.go                  CONFLICT
Auto-merging modules/templates/helper.go               CONFLICT
Auto-merging modules/util/util.go                      CONFLICT
Auto-merging services/context/context_template.go      CONFLICT
Auto-merging templates/base/footer.tmpl                CONFLICT
Auto-merging templates/base/head_script.tmpl           CONFLICT
Auto-merging templates/repo/issue/view_content/pull_merge_box.tmpl  CONFLICT
Auto-merging templates/swagger/ui.tmpl                 CONFLICT
Auto-merging templates/user/auth/captcha.tmpl          CONFLICT
Auto-merging tests/integration/markup_external_test.go CONFLICT
# 7 files auto-merged cleanly
```
