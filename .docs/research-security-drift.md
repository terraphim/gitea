---
name: Research -- Security Upstream Drift Analysis
description: Phase 1 research for issue #12. Verifies upstream divergence, classifies 7 missing security commits, and identifies cherry-pick conflict surface.
type: project
---

# Research Document: Security Upstream Drift Analysis

**Issue:** terraphim/gitea#12
**Date:** 2026-04-30
**Status:** Phase 1 -- Research Complete (Verified)
**Original Analyst:** Echo (Twin Maintainer)
**Verifying Analyst:** Ferrox (Rust Engineer, Principal SFIA-5)
**Reviewers:** @adf:security-sentinel, @adf:gitea-meta-coordinator

---

## Executive Summary

The terraphim/gitea fork diverged from go-gitea/gitea at merge-base `3db3c058b3`. As of 2026-04-30 it is **343 commits behind upstream/main and 41 commits ahead** (Robot/PageRank/MCP features and CI hardening). Of the 7 missing security commits called out in the issue, **4 cherry-pick cleanly and 3 produce single-file content conflicts** caused entirely by unrelated upstream evolution -- not by our fork. The critical OAuth2 authorization-code reuse fix (`f3bdcc58af`) cherry-picks cleanly. Recommendation: selective cherry-pick over full upstream merge.

## Essential Questions Check

| Question | Answer | Evidence |
|----------|--------|----------|
| Energizing? | Yes | OAuth2 code-reuse is a confirmed CVE-class fix |
| Leverages strengths? | Yes | We maintain the fork; security drift is our responsibility |
| Meets real need? | Yes | Production Gitea instance at git.terraphim.cloud |

**Proceed:** Yes (3/3).

---

## 1. Problem Restatement and Scope

**Problem:** Production fork lacks 7 confirmed security fixes. Risk profile is dominated by `f3bdcc58af` (OAuth2 authorization-code replay) which is CVE-class.

**Correction to original research:** Original document stated "0 commits ahead". Verified count: **41 commits ahead** of `upstream/main`. This materially changes merge strategy.

**IN scope**
- Cherry-pick / merge strategy decision
- Per-commit conflict assessment
- Test plan covering OAuth2, attachments, container registry, CSP

**OUT of scope**
- Non-security upstream features (deferred)
- Robot/PageRank custom changes (already in fork)
- Upstream CI/CD changes (deferred)

---

## 2. Verified Repository State

```
Merge base:        3db3c058b3c7d2e24dde8be609417620262ffca6
Behind upstream:   343 commits
Ahead of upstream: 41 commits
Overlap files:     7 (our 41 commits ∩ upstream 343 commits)
```

### Overlap files (full-merge conflict surface)

```
README.md
models/issues/dependency.go
models/migrations/migrations.go
models/migrations/v1_26/v326.go
modules/setting/setting.go
routers/api/v1/api.go
routers/api/v1/repo/issue_dependency.go
```

### Cherry-pick conflict surface (security commits ∩ our 41 commits)

```
routers/api/v1/api.go   (touched by both us and 6826321570)
```

Only 1 file is in both the security commits and our modifications.

---

## 3. Per-Commit Cherry-Pick Verification

Empirically validated via `git cherry-pick --no-commit` against current `main` on disposable branch:

| # | Commit | PR | Severity | Files | Cherry-pick | Conflict file |
|---|--------|----|---------:|------:|-------------|---------------|
| 1 | `f3bdcc58af` | #36797 | **Critical** | 3 | CLEAN | -- |
| 2 | `aee6628bf5` | #37334 | **High** | 16 | CLEAN | -- |
| 3 | `15b23f037d` | #37455 | **High** | 2 | CONFLICT | `modules/httplib/serve.go` |
| 4 | `82bfde2a37` | #37232 | **High** | 18 | CONFLICT | `modules/markup/external/openapi.go` |
| 5 | `6826321570` | #37354 | Medium | 7 | CLEAN | -- |
| 6 | `63db5972a1` | #37327 | Medium | 2 | CLEAN | -- |
| 7 | `6ed861589a` | #37290 | Medium | 2 | CONFLICT | `routers/api/packages/container/container.go` |

Sorted by severity. All three conflicts are **single-file content conflicts** in code untouched by our 41 commits -- they are caused by upstream evolution between the merge base and the security commit landing point. This means standard 3-way-merge resolution applies; no semantic awareness of our Robot work is required.

---

## 4. Vital Few (Essentialism)

### Essential Constraints (Max 3)

| Constraint | Why Vital | Evidence |
|------------|-----------|----------|
| Preserve the 41 fork commits | Robot/PageRank/MCP are the reason this fork exists | `git log upstream/main..HEAD` |
| Land `f3bdcc58af` first | Critical OAuth2 replay fix; cherry-picks clean | Section 3 |
| Keep PR reviewable | One PR with 7 commits beats 7 PRs with merge churn | Reviewer cognitive load |

### Eliminated from Scope (5/25)

| Eliminated | Why |
|------------|-----|
| Full upstream merge of all 343 commits | 1821 files changed; review burden enormous |
| Per-commit PRs | Sequencing overhead; reviewer fatigue |
| Backporting non-security upstream features | Not the issue scope |
| Refactoring our Robot code to ease merge | Out of scope; design churn |
| Auto-merge tooling investment | Not justified for one-off remediation |

---

## 5. Constraints

### Technical
- Go 1.22+ build (Gitea 1.26.0 baseline preserved)
- Existing migrations at `v326.go` must remain stable
- `routers/api/v1/api.go` is the single fork-overlap file with security commits

### Business
- Security must land on `main` and be deployed to `git.terraphim.cloud`
- No force-push to PR branches (per `AGENTS.md`)

### Non-Functional
| Requirement | Target |
|-------------|--------|
| OAuth2 authorize endpoint behaviour | Unchanged for valid clients |
| Robot API endpoints (`/api/v1/robot/*`) | Unchanged response contract |
| Build time | No regression |

---

## 6. Risks and Unknowns

### Known Risks
| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Conflict resolution introduces regression | Medium | High | Run `tests/integration` for affected packages post-each-pick |
| `routers/api/v1/api.go` overlap with our route registration | Low | Medium | Manual diff against `main`; route table reconciliation |
| CSP nonce change (`82bfde2a37`) breaks UI flows | Low | Medium | Manual smoke test of admin/auth/repo views |
| OAuth2 authorize-code semantics change breaks integrations | Low | High | Run `tests/integration/oauth_test.go` |

### Assumptions Stated Explicitly
| Assumption | Basis | Risk if Wrong | Verified? |
|------------|-------|---------------|-----------|
| Conflicts are mechanical (no semantic surprise) | Conflict files do not overlap with Robot code | Hand-resolution required, but bounded | Partially -- needs implementation pass |
| `f3bdcc58af` is sufficient OAuth2 fix in isolation | Upstream PR self-contained per file list | If dependent on later fix, may need second pick | No -- assumed from upstream PR description |
| Tests in `tests/integration` are runnable in our env | Repo includes `Makefile` targets | Test infra may need DB fixtures | Partially -- not yet executed |

### Open Questions
1. Are there additional security fixes between `f3bdcc58af` (the oldest, #36797) and HEAD upstream that the issue overlooked? -- Defer to security-sentinel sweep post-merge.
2. Should we tag a release after cherry-pick lands? -- Repo-steward decision.

---

## 7. Recommendations

### Proceed/No-Proceed
**PROCEED** to Phase 2 (Design). Cherry-pick path is empirically validated.

### Strategy Recommendation
**Selective cherry-pick (NOT full merge).**

Rationale:
1. Full merge surfaces 1821 file changes vs. 50 (sum of security-commit file lists with overlap) -- review cost is ~36x higher
2. Full merge brings unrelated behaviour change into a security PR -- violates change-isolation discipline
3. We have 41 ahead with active ownership; rebase-on-upstream is a separate, larger initiative
4. Empirical: 4 of 7 are clean; 3 conflicts are single-file and mechanical

### Sequence Recommendation
1. Clean picks first (lowest risk path to mitigation): `f3bdcc58af`, `aee6628bf5`, `6826321570`, `63db5972a1`
2. Conflict picks after clean stack stabilises: `82bfde2a37`, `6ed861589a`, `15b23f037d`
3. One PR with all 7 picks; reviewer can evaluate each commit independently

---

## 8. Next Steps

Proceed to Phase 2 with `disciplined-design`. Produce `.docs/design-security-drift.md` covering:
- Per-commit pick procedure
- Conflict resolution heuristics
- Test execution plan
- Rollback plan
- PR construction (branch name, commit messages, reviewer mentions)

**Gate:** Human approval and `@adf:security-sentinel` review required before implementation.

---

## Appendix A -- Verification Commands

```bash
git fetch upstream main
git rev-list --count HEAD..upstream/main          # 343
git rev-list --count upstream/main..HEAD          # 41
git merge-base upstream/main HEAD                 # 3db3c058b3
for sha in 15b23f037d 6826321570 aee6628bf5 63db5972a1 6ed861589a 82bfde2a37 f3bdcc58af; do
  git log --oneline -1 $sha
done
```

## Appendix B -- Commit File Inventories

See section 2 of empirical run; full per-commit file listings captured in implementation log.
