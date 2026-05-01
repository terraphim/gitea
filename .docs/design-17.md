# Implementation Plan: Issue #17 -- Surgical Backport of Picks 6 & 7 (Scope C)

**Status**: Draft -- awaiting review
**Research Doc**: `.docs/research-17.md` (on `main`)
**Author**: Ferrox (Rust Engineer, V-Model Phase 2)
**Date**: 2026-05-01
**Estimated Effort**: ~5 hours implementation + tests + review
**Branch**: `task/17-design` (this branch)
**Adjudicated Scope**: C (gitea-reviewer comment 16550, acknowledged by Echo 2026-05-01 18:02 CEST)
**Supersedes**: prior `.docs/design-17.md` draft on this branch (had identifier
  corruption from KG hook substitutions and an inverted pick ordering)

## Overview

### Summary

Backport the security intent of upstream picks 6 (`6ed861589a` -- container auth for
public instance) and 7 (`15b23f037d` -- attachment CSP) into the fork via surgical,
file-bounded patches. Pick 5 is explicitly out of scope and stays deferred.

### Approach

**Surgical adaptation, not raw cherry-pick.** Both upstream commits sit on top of
upstream-only refactors that the fork has not absorbed:

- Pick 6's surrounding `container.go` references `storage.ServeDirectOptions` (an
  upstream-only struct) in *unchanged* lines, which is what produced the cherry-pick
  cascade in the original blocker doc. The pick's *actual change* (~12 LOC in
  `apiUnauthorizedError`) does not need that struct.
- Pick 7's surrounding `serve.go` uses upstream's redesigned `ServeHeaderOptions`
  (with a typed `ContentDisposition` field and `encodeContentDisposition` helper),
  which the fork still has as a bare `Disposition string` plus inline disposition
  formatting. The pick's *actual change* (new CSP constants + a helper applied to
  all served content with audio/video exemption) can be re-expressed against the
  fork's existing structure.

Strategy:

1. Identify each pick's semantic intent (the security fix itself).
2. Manually apply that intent to the fork's current files (no `git cherry-pick`).
3. Add tests covering the new behaviour.
4. Land each pick as its own PR.

### Scope

**In Scope:**

- Adapt pick 6 into `routers/api/packages/container/container.go::apiUnauthorizedError`
- Adapt pick 7 into `modules/httplib/serve.go` (CSP constants + central helper applied
  to all served content, including audio/video exemption)
- New unit tests in `modules/httplib/serve_test.go`
- Extend integration tests in `tests/integration/api_packages_container_test.go`
- Each commit references the upstream SHA via an `Adapted-from:` trailer

**Out of Scope:**

- Pick 5 (CSP `script-src` nonce) -- deferred to upstream-rebase class work
- Backport of `modules/httplib/content_disposition.go` and the `ContentDispositionType`
  refactor (drags in many call-sites; not needed for the security intent)
- Backport of `modules/storage.ServeDirectOptions` (NOT needed by pick 6's actual
  change -- the original blocker doc was wrong about this)
- Backport of `typesniffer.FromContentType` (only used by upstream's
  `serveSetHeaderContentRelated`; the fork's adapted helper switches on `contentType`
  string prefixes, equivalent for the constants we care about)
- Refactoring `ServeHeaderOptions.Disposition` from `string` to a typed enum
- Any change to `modules/markup`, `modules/templates`, `modules/public`,
  `services/context`, or `routers/web/*`

**Avoid At All Cost** (5/25):

- `git cherry-pick --theirs` of either upstream commit (re-creates the original cascade)
- "Just a small refactor" of `ServeHeaderOptions` while we are here (parent design v2 §1)
- Importing any new third-party crate or upstream module wholesale
- Bundling picks 6 and 7 into one PR ("convenience commit") -- separate PRs preserve
  blast-radius isolation for revert and review
- Renaming `apiUnauthorizedError` to `APIUnauthorizedError` (it stays unexported; only
  callers within the package use it)

## Architecture

### Component Diagram

```
PICK 6: container registry auth
    Request -> /v2/{username}/...
        -> apiUnauthorizedError(ctx)
            -> always: emit Bearer realm header
            -> [NEW] look up owner from ctx.PathParam("username")
            -> [NEW] requireSignIn = (owner non-public) OR REQUIRE_SIGNIN_VIEW=true
            -> [NEW] only emit Basic realm header when requireSignIn
            -> apiErrorDefined(ctx, errUnauthorized)

PICK 7: served-content CSP
    Any served content via httplib.ServeSetHeaders(...)
        -> [REFACTORED] CSP set via serveSetContentSecurityHeaders(w, contentType)
            -> [NEW]   default CSP (sandbox)
            -> [NEW]   PDF CSP (no sandbox)
            -> [NEW]   audio/video CSP cleared (the security-fix delta)
        -> existing Content-Type / X-Content-Type-Options / Content-Length / etc.
```

### Data Flow

**Pick 6:** moves the `Basic realm="Gitea Container Registry"` challenge from
*always-emitted* to *conditionally-emitted* based on:

- the URL-encoded owner's visibility (only force auth for non-public owners), OR
- the global `setting.Service.RequireSignInViewStrict` setting

**Pick 7:** moves CSP-setting from the file-serve path
(`setServeHeadersByFile`, where it currently lives) to a shared helper
(`serveSetContentSecurityHeaders`) that also runs from the attachment-serve path
(`ServeSetHeaders`), and adds an explicit empty-CSP exemption for `audio/*` and
`video/*` content types. Today the fork sets the SVG sandbox CSP for those types
when serving by file, breaking media playback in some browsers.

### Key Design Decisions

| Decision | Rationale | Alternatives Rejected |
|----------|-----------|-----------------------|
| Apply each pick as a manual code change, not `git cherry-pick` | The cascade is in unchanged context lines around the actual diff hunks; cherry-picking pulls upstream-only symbols in. Manual application keeps the surgical contract. | (a) `cherry-pick --theirs` -- causes the original cascade. (b) Backport `ContentDispositionType` and `ServeDirectOptions` first -- violates parent design v2 §1. |
| One PR per pick, **pick 6 first, pick 7 second** | Pick 6 is the smaller, fully isolated change (one function body, one file). Pick 7 touches `httplib/serve.go` which has more downstream callers; landing pick 6 first proves the surgical-adaptation pattern is acceptable to reviewers before the larger change. | (a) Bundled PR -- larger blast radius, harder to revert one without the other. (b) Pick 7 first -- the prior draft's reasoning ("self-contained") applies to both; pick 6 has fewer call-sites and shorter review surface. |
| Reuse existing fork symbols (`structs.VisibleTypePublic`, `setting.Service.RequireSignInViewStrict`, `typesniffer.MimeTypeApplicationOctetStream`) | All exist on `main` today; no new module surface. | Backport `typesniffer.FromContentType` -- adds 3 LOC for zero behaviour gain (we already match prefixes). |
| Apply CSP universally in `ServeSetHeaders`, not only in `setServeHeadersByFile` | Matches upstream's intent: attachment paths should also enforce CSP. The fork currently only enforces CSP when serving by file. | Leave attachment path uncovered -- defeats the purpose of pick 7. |
| Audio/video CSP empty (`""`, header deleted) | The actual upstream security fix: media types must not have `default-src 'none'` because that breaks playback. | Setting `media-src 'self'` -- diverges from upstream behaviour. |
| Helper named `serveSetContentSecurityHeaders` | The fork's `ServeSetHeaders` already sets Content-Type and X-Content-Type-Options; the helper's only job is CSP. Narrower naming = clearer responsibility. | Match upstream's `serveSetHeaderContentRelated` -- conflates Content-Type setting which the fork handles differently. |
| Keep `apiUnauthorizedError` unexported (no rename) | It is only called from within the `container` package (`ReqContainerAccess`, `Authenticate`); no external caller. Renaming would be a churn-only diff. | Rename to `APIUnauthorizedError` (the prior draft proposed this; it is unnecessary churn). |
| Swallow the `GetUserByName` error in pick 6 | Matches upstream behaviour; on lookup failure `owner == nil` and `requireSignIn` falls back to the global setting. The 401 response is still well-formed. | Log the error -- adds noise on every miss to a non-existent owner. Return early -- breaks the upstream contract. |

### Eliminated Options

| Option Rejected | Why Rejected | Risk of Including |
|-----------------|--------------|-------------------|
| Backport `ContentDispositionType` enum | Refactor cascade across all call-sites; not in §1 budget | Multi-day refactor, high merge-risk |
| Backport `storage.ServeDirectOptions` | Not needed by pick 6's actual diff (blocker doc was wrong; the symbol appears in unchanged container.go context, not in the pick's hunk) | Drags in storage refactor, contradicts §1 |
| Backport `manifest.go` / `AssetURI` / `RenderIFrame` (pick 5 chain) | Out of scope per Scope C adjudication | Re-opens the 5-module sync framing |
| Add a feature flag to gate the CSP change | Pick 7 is a security bug-fix, not a new feature; conservative default is "apply fix" | Adds dead config surface |
| Restructure `ServeHeaderOptions` to typed `Disposition` | Cosmetic; not required by either pick | Drive-by refactor; violates surgical-changes rule |

### Simplicity Check

> What if this could be easy?

The simplest design that works:

- Pick 6: 12 LOC change in one Go file plus a couple of integration test cases.
- Pick 7: ~50 LOC change in one Go file plus one new unit test function.
- Two PRs, sequenced.

That is the design.

**Senior Engineer Test:** Would a senior engineer call this overcomplicated? No --
the plan deliberately under-builds vs the upstream refactor and matches §1.

**Nothing Speculative Checklist:**

- [x] No features the user did not request (only the two adjudicated picks)
- [x] No abstractions "in case we need them later" (helper is named for its narrow job)
- [x] No flexibility "just in case" (no config flags, no toggles)
- [x] No error handling for scenarios that cannot occur (pick 6 swallows
      `GetUserByName` error deliberately, matching upstream)
- [x] No premature optimization

## File Changes

### New Files

None. (Per scope: no new modules, no new files.)

### Modified Files

| File | Pick | Change Summary | Approx LOC |
|------|------|----------------|------------|
| `routers/api/packages/container/container.go` | 6 | Add `structs` import; rewrite body of `apiUnauthorizedError` to conditionally emit Basic realm header. | +12 / -1 |
| `tests/integration/api_packages_container_test.go` | 6 | Extend `TestPackageContainer/Anonymous` to assert WWW-Authenticate header content; add `TestPackageContainer/RequireSignIn` sub-test. | +30 / -2 |
| `modules/httplib/serve.go` | 7 | Add 3 CSP constants; add `serveSetContentSecurityHeaders` helper; call from `ServeSetHeaders`; remove duplicate inline CSP from `setServeHeadersByFile`. | +35 / -10 |
| `modules/httplib/serve_test.go` | 7 | Add `TestServeSetContentSecurityHeaders` (table-driven) and a small assertion in an existing test confirming `ServeSetHeaders` writes a CSP header for default content. | +35 / -0 |

### Deleted Files

None.

## API Design

### Pick 6: `apiUnauthorizedError` rewrite

```go
// routers/api/packages/container/container.go (~line 123)

// apiUnauthorizedError responds with 401 Unauthorized for OCI/container registry endpoints.
//
// The "Basic realm" challenge header is only emitted when sign-in is actually required,
// because container clients on public instances (and for public-visibility owners) treat
// the Basic challenge as a hard sign-in requirement and prompt the user even when an
// anonymous bearer token would suffice.
//
// HINT: CONTAINER-AUTH-PUBLIC: adapted from upstream commit 6ed861589a (#37290);
// the surrounding container.go file uses upstream-only `storage.ServeDirectOptions`
// in unchanged lines, so a `git cherry-pick` cascades. This is the surgical
// equivalent restricted to the actual auth-fix hunk.
func apiUnauthorizedError(ctx *context.Context) {
    realmURL := httplib.GuessCurrentHostURL(ctx) + "/v2/token"
    ctx.Resp.Header().Add("WWW-Authenticate",
        `Bearer realm="`+realmURL+`",service="container_registry",scope="*"`)

    ownerName := ctx.PathParam("username")
    owner, _ := user_model.GetUserByName(ctx, ownerName)
    requireSignIn := owner != nil && owner.Visibility != structs.VisibleTypePublic
    requireSignIn = requireSignIn || setting.Service.RequireSignInViewStrict
    if requireSignIn {
        // support apple container CLI: container registry login <host> -u
        ctx.Resp.Header().Add("WWW-Authenticate", `Basic realm="Gitea Container Registry"`)
    }

    apiErrorDefined(ctx, errUnauthorized)
}
```

New import block addition:

```go
"code.gitea.io/gitea/modules/structs"
```

Call sites (`ReqContainerAccess` line 133, `Authenticate` lines 159, 173) keep
the lowercase `apiUnauthorizedError(ctx)` -- no rename.

### Pick 7: `serve.go` CSP constants + helper

```go
// modules/httplib/serve.go (insert after the ServeHeaderOptions struct)

const (
    // serveHeaderCspDefault: same-origin sandbox for HTML/SVG/unknown bytes.
    // "style-src 'unsafe-inline'" carries SVG inline styles (see #14101).
    serveHeaderCspDefault = "default-src 'none'; style-src 'unsafe-inline'; sandbox"

    // serveHeaderCspPdf: PDF cannot render in a sandboxed context in some browsers
    // (e.g. Safari), so the sandbox attribute is omitted. Scripts inside PDF cannot
    // escape the document.
    // HINT: PDF-RENDER-SANDBOX: PDF won't render in sandboxed context.
    serveHeaderCspPdf = "default-src 'none'; style-src 'unsafe-inline'"

    // serveHeaderCspAudioVideo: empty -> CSP header is removed for audio/video.
    // The default-src 'none' policy breaks playback; audio/video bytes carry no
    // executable surface, so no CSP is required.
    serveHeaderCspAudioVideo = ""
)

// serveSetContentSecurityHeaders sets the CSP header appropriate for the given
// content type, or removes it for audio/video.
//
// HINT: CONTENT-CSP-MEDIA: adapted from upstream commit 15b23f037d (#37455).
// The fork's ServeHeaderOptions still carries Disposition as a bare string and
// has no encodeContentDisposition helper, so this helper is restricted to CSP
// (Content-Type and X-Content-Type-Options stay in ServeSetHeaders).
func serveSetContentSecurityHeaders(w http.ResponseWriter, contentType string) {
    csp := serveHeaderCspDefault
    switch {
    case strings.HasPrefix(contentType, "application/pdf"):
        csp = serveHeaderCspPdf
    case strings.HasPrefix(contentType, "audio/"),
        strings.HasPrefix(contentType, "video/"):
        csp = serveHeaderCspAudioVideo
    }
    if csp != "" {
        w.Header().Set("Content-Security-Policy", csp)
    } else {
        w.Header().Del("Content-Security-Policy")
    }
}
```

`ServeSetHeaders` change (one new line, after the existing X-Content-Type-Options
line at ~line 58):

```go
header.Set("X-Content-Type-Options", "nosniff")

// Pick 7 (#37455): apply CSP to all served content, with audio/video exemption.
serveSetContentSecurityHeaders(w, contentType)
```

`setServeHeadersByFile` change (remove the inline SVG/PDF CSP block at lines
117-126; the downstream `ServeSetHeaders` call now sets CSP based on the same
`opts.ContentType` that this function has just resolved):

```go
// before (lines 117-126):
if isSVG {
    w.Header().Set("Content-Security-Policy",
        "default-src 'none'; style-src 'unsafe-inline'; sandbox")
} else if sniffedType.IsPDF() {
    w.Header().Set("Content-Security-Policy",
        "default-src 'none'; style-src 'unsafe-inline'")
}

// after: removed; ServeSetHeaders below sets CSP via serveSetContentSecurityHeaders.
```

The `opts.Disposition = "inline"` block at line 128 stays unchanged.

### Error Types

No new error types.

## Test Strategy

### Unit Tests

| Test | Location | Purpose |
|------|----------|---------|
| `TestServeSetContentSecurityHeaders` | `modules/httplib/serve_test.go` | New table-driven test. Cases: empty content type → default CSP; `"any"` → default; `application/pdf` → pdf CSP; `application/pdf; charset=...` → pdf CSP; `audio/mp4` → no CSP header; `video/ogg; codecs=...` → no CSP header; `image/svg+xml` → default CSP. Each case asserts the exact CSP value (or absence). Also asserts the default constant contains `"; sandbox"`. |
| Existing `TestServeUserContentByFile` | `modules/httplib/serve_test.go` | Verify no regression: the test calls `ServeContentByReader` which calls `setServeHeadersByFile` then `ServeSetHeaders`. Add an assertion that for the served PNG content the response includes a CSP header (default value). This is a regression assertion, not a new test. |

Required new test imports for `serve_test.go`: none beyond what is already in
the file (`net/http`, `net/http/httptest`, `testing`, `assert`, `require`).

### Integration Tests

| Test | Location | Purpose |
|------|----------|---------|
| `TestPackageContainer/Anonymous` (extension) | `tests/integration/api_packages_container_test.go` | After the existing `MakeRequest(t, req, http.StatusUnauthorized)` for `/v2`, assert that on a public instance with no `RequireSignInViewStrict`, the response carries **only** the Bearer realm `WWW-Authenticate` header (no Basic realm). |
| `TestPackageContainer/RequireSignIn` (new sub-test) | `tests/integration/api_packages_container_test.go` | New `t.Run("RequireSignIn", ...)`: temporarily set `setting.Service.RequireSignInViewStrict = true` (with `defer` restore), issue the same `/v2` request, and assert both Bearer and Basic realm `WWW-Authenticate` headers are present. |

A `PrivateOwner` sub-test (private-visibility user → both headers) was considered
and dropped from the spec: it requires fixture user creation and adds complexity
beyond the §1 budget. The `RequireSignIn` sub-test exercises the same conditional
branch via the global setting, which is sufficient coverage for the conditional.

### Property Tests

Not applicable -- the changes are deterministic header-setting; table-driven
unit tests are sufficient.

### Regression Coverage

Per the testing skill's regression rule:

- Before changing `serve.go`, confirm `go test ./modules/httplib/...` is green
  on the branch base.
- Before changing `container.go`, confirm `go test ./routers/api/packages/container/...`
  and the integration `TestPackageContainer` are green on the branch base.

## Implementation Steps

### Step 1: Pick 6 -- container auth conditional Basic realm

**Files:** `routers/api/packages/container/container.go`,
`tests/integration/api_packages_container_test.go`

**Description:**

1. Cut `task/17-pick6` from current `main`.
2. Add the `code.gitea.io/gitea/modules/structs` import.
3. Replace the body of `apiUnauthorizedError` per the API Design section above.
   Keep the function name lowercase (no rename) and keep the call sites
   `apiUnauthorizedError(ctx)` unchanged.
4. Extend `TestPackageContainer/Anonymous` with the public-instance assertion.
5. Add `TestPackageContainer/RequireSignIn` sub-test with the global-setting toggle.
6. Run `make fmt && make lint-go`.
7. Run `go test ./routers/api/packages/container/...` and the integration test
   pattern `go test -tags integration ./tests/integration/ -run TestPackageContainer`.

**Pre-conditions:**

- This design (Phase 2) approved.
- `task/17-pick6` cut from current `main`.

**Estimated:** 2 hours

**PR title:** `Fix #17 (pick 6 of 7): conditional Basic realm in container registry 401`

**Commit message template:**

```
[ferrox] feat(security): conditional Basic realm header in container registry 401

Adapt upstream commit 6ed861589a (#37290) for the fork. The upstream commit's
surrounding container.go file uses the upstream-only `storage.ServeDirectOptions`
struct in unchanged context lines, which is what produced the Phase 3 cherry-pick
cascade documented in `.docs/blocker-12-pick6-cascade.md`. The semantic change
itself does not need that struct.

This is a manual application (not `git cherry-pick`) of the actual ~12 LOC
behaviour change: only emit the Basic realm challenge header when sign-in is
required, either via the targeted owner's visibility being non-public or via
the global REQUIRE_SIGNIN_VIEW=true setting.

Refs terraphim/gitea#17
Refs terraphim/gitea#12
Adapted-from: 6ed861589a (#37290)
```

### Step 2: Pick 7 -- attachment CSP central helper

**Files:** `modules/httplib/serve.go`, `modules/httplib/serve_test.go`

**Description:**

1. After step 1's PR merges, cut `task/17-pick7` from `main`.
2. In `modules/httplib/serve.go`:
   a. Add the three CSP constants after the `ServeHeaderOptions` struct.
   b. Add `serveSetContentSecurityHeaders`.
   c. Insert the helper call into `ServeSetHeaders` after the
      `X-Content-Type-Options` line.
   d. Remove the inline SVG/PDF CSP block from `setServeHeadersByFile`.
3. In `modules/httplib/serve_test.go`, add `TestServeSetContentSecurityHeaders`
   (table-driven) and the regression assertion in `TestServeUserContentByFile`.
4. Run `make fmt && make lint-go`.
5. Run `go test ./modules/httplib/...`.

**Pre-conditions:**

- Step 1 PR merged (preserves blast-radius isolation; sequencing only).

**Dependencies:** Step 1 (sequencing only -- no shared files).

**Estimated:** 3 hours

**PR title:** `Fix #17 (pick 7 of 7): centralise served-content CSP, exempt audio/video`

**Commit message template:**

```
[ferrox] feat(security): centralise CSP for served content, exempt audio/video

Adapt upstream commit 15b23f037d (#37455) for the fork. The upstream commit
modifies modules/httplib/serve.go around upstream-only symbols
(ContentDispositionType, encodeContentDisposition, typesniffer.FromContentType)
that the fork has not absorbed; this is what produced the Phase 3 cascade
documented in `.docs/blocker-12-pick7-cascade.md`.

This is a manual adaptation (not `git cherry-pick`) that captures the security
intent without touching the ContentDispositionType refactor:

- Add three CSP constants (default, pdf, audio/video).
- Extract a `serveSetContentSecurityHeaders` helper.
- Call it from ServeSetHeaders so CSP applies to attachment-served content too,
  not only file-served content.
- Exempt audio/* and video/* from the default 'sandbox' policy (they cannot
  render under default-src 'none' and carry no executable surface).
- Drop the duplicate inline CSP block from setServeHeadersByFile (now set
  downstream by ServeSetHeaders with the same content-type derivation).

Refs terraphim/gitea#17
Refs terraphim/gitea#12
Adapted-from: 15b23f037d (#37455)
```

### Step 3: Close issue #17

**Files:** Comment on issue #17.

**Description:** Once both PRs merge, post a final summary comment on issue #17
linking the two merged PRs and acknowledging Echo's adjudication. Then
`gtr close-issue --owner terraphim --repo gitea --index 17`.

**Estimated:** 15 minutes

## Rollback Plan

Each pick is its own PR with a separate commit on `main`. Rollback is `git revert`
of the merge commit:

- Pick 6 revert: pure -- only touches `container.go` and the integration test file.
- Pick 7 revert: pure -- only touches `serve.go` and `serve_test.go`. Reverting
  pick 7 restores the previous behaviour where CSP was set only in
  `setServeHeadersByFile` and audio/video got the SVG/sandbox CSP.

No feature flag is added. `git revert` is the rollback.

## Migration

None. No DB changes. No config changes. Behaviour change for existing clients:

- Container registry clients on public instances stop receiving the spurious
  `Basic realm` challenge header for endpoints that don't require sign-in.
- Audio/video files served via `httplib.ServeSetHeaders` no longer carry
  the `default-src 'none'; sandbox` CSP header (which broke playback in some
  browsers).

Both are bug-fixes; no migration steps required.

## Dependencies

### New Dependencies

None.

### Dependency Updates

None.

## Performance Considerations

| Metric | Target | Measurement |
|--------|--------|-------------|
| `apiUnauthorizedError` latency | +1 DB lookup (`GetUserByName`) on 401 path only | Acceptable -- 401 is an error path |
| `serveSetContentSecurityHeaders` | < 1µs per call (4 string prefix checks + 1 header set/del) | Trivial; no benchmark needed |

The added DB lookup in pick 6 only happens on 401 responses, not on successful
requests. No noticeable user-facing impact.

### Benchmarks to Add

None -- the changes are not performance-relevant.

## Open Items

| Item | Status | Resolution |
|------|--------|------------|
| Verify `prepareServeDirectOptions` helper existence in fork (Echo's Phase 3 spike, deferred from research) | **RESOLVED IN DESIGN** | The helper is *not needed* by either pick. Pick 6's blocker-doc claim that `storage.ServeDirectOptions` was a transitive dependency was correct only for the `--theirs` cherry-pick path; the surgical adaptation drops that dependency entirely. No spike required at Phase 3 start. |
| Pick 7's existing serve-test cases that must keep passing | OPEN -- to verify in Phase 3 | Run `go test ./modules/httplib/...` before and after the change to confirm `TestServeUserContentByFile` and any other existing serve tests still pass. |
| Pick 6 deployed-surface check (referenced in blocker doc but never concluded) | NOT BLOCKING | The change makes the 401 response *less* aggressive (drops the spurious Basic realm header). Worst case: a client that depended on the spurious Basic realm prompt would now skip the sign-in dialogue, which is the correct behaviour for public/anonymous endpoints. |
| Should we add a `PrivateOwner` integration sub-test? | DECIDED: NO | Adds fixture-creation cost beyond §1 budget; the `RequireSignIn` global-setting sub-test exercises the same conditional branch. Can be added in a follow-up if reviewer requests it. |

## Approval

- [ ] Technical review complete (`@adf:gitea-reviewer`)
- [ ] Test strategy approved
- [ ] Ordering approved (pick 6 first, then pick 7)
- [ ] Human / coordinator approval received

---

## Appendix A: Why this is not a `git cherry-pick`

Both upstream commits sit on top of refactors the fork has not absorbed. A
`cherry-pick --theirs` of either commit pulls upstream's *entire* version of the
target file into the fork, which brings in unchanged-in-the-pick references to
upstream-only symbols (`storage.ServeDirectOptions` for pick 6;
`encodeContentDisposition`, `ContentDispositionType`, and
`typesniffer.FromContentType` for pick 7). That is the mechanism that produced
the original Phase 3 blocker docs.

The surgical alternative -- manually applying the *intent* of each pick to the
fork's current files -- is the design here. Each commit references the upstream
SHA via an `Adapted-from:` trailer so the lineage is preserved without implying
a cherry-pick relationship.

This decision is bounded to issue #17. The longer-term answer to "fork is N
commits behind upstream" is the upstream rebase under #12 lineage, which is
explicitly out of scope here.

## Appendix B: Symbol-existence verification (carried from research)

Verified directly against `main` (commit `ca274b2bc6`) on 2026-05-01:

| Symbol used by surgical patch | Pick | Location in fork |
|-------------------------------|------|------------------|
| `structs.VisibleTypePublic` | 6 | `modules/structs/visible_type.go:11` |
| `setting.Service.RequireSignInViewStrict` | 6 | `modules/setting/service.go:46` |
| `user_model.GetUserByName` | 6 | already imported in `container.go` (line 21) |
| `ctx.PathParam("username")` | 6 | route registers `{username}` at `routers/api/packages/api.go:552` |
| `typesniffer.MimeTypeApplicationOctetStream` | 7 | `modules/typesniffer/typesniffer.go:22` |
| `typesniffer.MimeTypeImageSvg` | 7 | `modules/typesniffer/typesniffer.go:19` |
| `http.ResponseWriter`, `strings.HasPrefix` | 7 | stdlib |

No symbol used by the design is missing from the fork. No new import is
required beyond `code.gitea.io/gitea/modules/structs` in pick 6.
