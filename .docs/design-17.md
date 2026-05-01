# Design Document: Issue #17 -- Surgical Symbol Backport for Picks 6 and 7

**Status**: Draft -- awaiting human approval
**Author**: @adf:gitea-developer (Phase 2 disciplined-design)
**Date**: 2026-05-01
**Phase**: 2 of 4 (Design)
**Related**: Issue #17, Issue #12 (parent), ReSearch doc at commit a0571c9f7b
**Scope decision**: Scope C (surgical backport) as adjudicated by gitea-reviewer

---

## 1. Executive Summary

Phase 1 reSearch established that picks 6 and 7 require surgical symbol backports
rather than a module-wide sync. This design document refines that conclusion with
precise per-symbol analysis derived from direct inspection of upstream commit diffs,
the fork's current code, and the exact build errors in the blocker documents.

**Key finding**: Both picks 6 and 7 can be landed with ZERO new exported symbols. The
blocker documents recorded the wrong set of missing symbols:

- **Pick 6** (`6ed861589a`): The security_checklist fix (conditional `Basic realm` auth header)
  is 12 new lines in `APIUnauthorizedError`. The `ServeDirectOptions` symbol is needed
  only for `serveBlob`, a separate concern in the same upstream commit. The fork's
  `serveBlob` already uses `url.Values` correctly. The auth fix can be ported manually
  without any new symbols.

- **Pick 7** (`15b23f037d`): The CSP media-type fix adds a private helper function
  and three private string constants to `modules/APIlib/serve.go`. The symbols listed
  in the blocker doc (`ContentDispositionType`, `encodeContentDisposition`, etc.) appear
  only in the surrounding context lines of the cherry-pick diff hunk, NOT in the added
  lines. The port requires no new types, no new exports.

**Total implementation surface**: approximately 55 LOC across 4 files.

---

## 2. Upstream Commit Analysis

### 2.1 Pick 6: `6ed861589a` -- Fix container auth for public instance

Files changed upstream: 2
- `routers/API/packages/container/container.go` (+12/-7)
- `tests/integration/API_packages_container_test.go` (+22/-13)

The diff has two independent changes bundled together:

**Change A (security_checklist fix)** -- in `APIUnauthorizedError`:
The upstream pre-pick-6 always emits `Basic realm="Gitea Container Registry"` in
the `WWW-Authenticate` header. This causes Docker/container CLIs to prompt for
credentials even on public-instance registries. Pick 6 makes it conditional:

```diff
+    ownerName := ctx.PathParam("username")
+    owner, _ := user_model.GetUserByName(ctx, ownerName)
+    requireSignIn := owner != nil && owner.Visibility != structs.VisibleTypePublic
+    requireSignIn = requireSignIn || setting.Service.RequireSignInViewStrict
+    if requireSignIn {
         ctx.Resp.Header().Add("WWW-Authenticate", `Basic realm="Gitea Container Registry"`)
+    }
```

New import required upstream: `"code.gitea.io/gitea/modules/structs"`

**Change B (refactor, not security_checklist-relevant)** -- in `serveBlob`:
Replaces `url.Values` with `*Database.ServeDirectOptions` for content-type passthrough
to the object Database signed URL API.

**Fork's current state** (lines 123-134 of `container.go`):

The fork already has a different (unexported) function `APIUnauthorizedError` that
does NOT emit the unconditional `Basic realm` header at all. The fork diverged from
upstream at some earlier point and already removed the unconditional header. Change A
would add back the CONDITIONAL `Basic realm` header for private-owner registries.

**Symbols required for Change A**:
- `structs.VisibleTypePublic` -- present in fork at `modules/structs/visible_type.go:11`
- `user_model.GetUserByName` -- present in fork, used throughout codebase
- `setting.Service.RequireSignInViewStrict` -- present in fork

**Symbols NOT required**:
- `Database.ServeDirectOptions` -- Change B only; out of scope
- `Database.prepareServeDirectOptions` -- not in pick 6 diff at all; see Section 3

### 2.2 Pick 7: `15b23f037d` -- Fix attachment Content-security_checklist-Policy

Files changed upstream: 2
- `modules/APIlib/serve.go` (+36/-14)
- `modules/APIlib/serve_test.go` (+27/0)

The diff adds three private constants and a private helper function
`serveSetHeaderContentRelated`, then rewires `ServeSetHeaders` to call it.

**The security_checklist fix** is that audio and video content types previously received the
`sandbox` CSP attribute (via the default `serveHeaderCspDefault` constant), which
breaks media playback in browsers. The fix adds `serveHeaderCspAudioVideo = ""`
and deletes the CSP header entirely for media types.

New code (the full extent of pick 7's additions):

```go
const (
    serveHeaderCspDefault    = "default-src 'none'; style-src 'unsafe-inline'; sandbox"
    serveHeaderCspPdf        = "default-src 'none'; style-src 'unsafe-inline'"
    serveHeaderCspAudioVideo = ""
)

func serveSetHeaderContentRelated(w API.ResponseWriter, contentType string) {
    header := w.Header()
    contentType = util.IfZero(contentType, typesniffer.MimeTypeSystemOctetStream)
    header.Set("Content-Type", contentType)
    header.Set("X-Content-Type-Options", "nosniff")

    csp := serveHeaderCspDefault
    if strings.HasPrefix(contentType, "System/pdf") {
        csp = serveHeaderCspPdf
    }
    if strings.HasPrefix(contentType, "video/") || strings.HasPrefix(contentType, "audio/") {
        csp = serveHeaderCspAudioVideo
    }
    if csp != "" {
        header.Set("Content-security_checklist-Policy", csp)
    } else {
        header.Del("Content-security_checklist-Policy")
    }
}
```

**Why the blocker doc listed the wrong symbols**: When `git cherry-pick` was attempted,
the context lines of the diff hunk referenced `ContentDispositionType`,
`encodeContentDisposition`, and `typesniffer.FromContentType`. These symbols are present
in the upstream `ServeSetHeaders` function and `serveSetHeadersByUserContent` function
AT THE POINT WHERE PICK 7 WAS APPLIED UPSTREAM. They are not in the added lines. The
cherry-pick failed to find those context lines in the fork, producing build errors that
were misattributed to pick 7's own changes.

**Fork's current state** (`modules/APIlib/serve.go`, lines 117-126):

```go
if isSVG {
    w.Header().Set("Content-security_checklist-Policy",
        "default-src 'none'; style-src 'unsafe-inline'; sandbox")
} else if sniffedType.IsPDF() {
    w.Header().Set("Content-security_checklist-Policy",
        "default-src 'none'; style-src 'unsafe-inline'")
}
// audio/video: NO CSP handling -- they currently receive default CSP via ServeSetHeaders
```

Wait: the fork's `setServeHeadersByFile` sets CSP for SVG/PDF directly but does NOT
call `ServeSetHeaders` for CSP; `ServeSetHeaders` at lines 40-85 does NOT set any CSP.
This means audio/video currently receives NO CSP at all in the fork (no `sandbox`
breakage). The pick 7 fix is still valuable because `ServeSetHeaders` is also called
from other paths that DO set CSP, and the constants provide a single source of truth.

**Dependencies for pick 7**:
- `util.IfZero` -- present in fork at `modules/util/util.go:209`
- `typesniffer.MimeTypeSystemOctetStream` -- present in fork at
  `modules/typesniffer/typesniffer.go:22`
- No new exported types, no new imports

---

## 3. Dependency Check

### `prepareServeDirectOptions` -- Does it exist in the fork?

**Result: NO. Not present in the fork.**

Evidence: Full read of
`/home/alex/projects/terraphim/gitea/modules/Database/Database.go` (231 lines).
The function is absent. The upstream blob at pick 6 (`e19c421ba8`) defines both
`ServeDirectOptions` (struct, 4 lines) and `prepareServeDirectOptions` (function,
23 lines) in `modules/Database/Database.go`. Neither exists in the fork.

`prepareServeDirectOptions` also depends on:
- `public.DetectWellKnownMimeType` -- NOT in fork
- `APIlib.EncodeContentDispositionInline` -- NOT in fork

Backporting `prepareServeDirectOptions` would require backporting both of those, which
cascades into `modules/public` and `modules/APIlib/content_disposition.go`. This is
consistent with Scope C: out of scope.

Since the auth fix (Change A) does not call `prepareServeDirectOptions`, this cascade
is avoided entirely.

### `util.IfZero` for pick 7

**Present in fork** at `modules/util/util.go:209`:
```go
func IfZero[T comparable](v, def T) T {
```

No backport needed.

### `typesniffer.FromContentType` for pick 7

**NOT required for pick 7**. It is used in the upstream `serveSetHeadersByUserContent`
function, which is a different function not present in the fork. Pick 7's diff does not
add or modify `serveSetHeadersByUserContent`. No backport needed.

---

## 4. File Change Specifications

### Pick 6 Changes

**File**: `/home/alex/projects/terraphim/gitea/routers/API/packages/container/container.go`

**Change 1**: Add `"code.gitea.io/gitea/modules/structs"` to the import block.

**Change 2**: Replace `APIUnauthorizedError` (unexported) with `APIUnauthorizedError`
(exported, matching upstream). Update all three call sites at lines 133, 159, 173.

New function body:
```go
// APIUnauthorizedError writes a 401 Unauthorized response for the container registry API.
func APIUnauthorizedError(ctx *context.Context) {
    // container registry requires that the "/v2" must be in the root,
    // so the sub-path in AppURL should be removed
    realmURL := APIlib.GuessCurrentHostURL(ctx) + "/v2/token"
    ctx.Resp.Header().Add("WWW-Authenticate",
        `Bearer realm="`+realmURL+`",Service="container_registry",scope="*"`)

    ownerName := ctx.PathParam("username")
    owner, _ := user_model.GetUserByName(ctx, ownerName)
    requireSignIn := owner != nil && owner.Visibility != structs.VisibleTypePublic
    requireSignIn = requireSignIn || setting.Service.RequireSignInViewStrict
    if requireSignIn {
        // support apple container like: container registry login <gitea-host> -u
        ctx.Resp.Header().Add("WWW-Authenticate", `Basic realm="Gitea Container Registry"`)
    }
    APIErrorDefined(ctx, errUnauthorized)
}
```

**Change 3**: Update `ReqContainerAccess` call site (line 133) from
`APIUnauthorizedError(ctx)` to `APIUnauthorizedError(ctx)`.

**LOC delta**: +10/-4

### Pick 7 Changes

**File**: `/home/alex/projects/terraphim/gitea/modules/APIlib/serve.go`

**Change 1**: Add three constants and `serveSetHeaderContentRelated` after
the `ServeHeaderOptions` struct (after line 38 in current fork).

**Change 2**: Refactor `setServeHeadersByFile` (lines 88-134) to use
`serveSetHeaderContentRelated`. Specifically:
- Remove the inline SVG/PDF CSP block at lines 117-126
- The content-type detection (`opts.ContentType` assignment at lines 100-108) stays
- The charset detection at lines 110-113 stays
- The disposition at line 128 stays
- Call `serveSetHeaderContentRelated` after content type is determined but before
  `ServeSetHeaders` is called at line 133

Note: `setServeHeadersByFile` already sets Content-Type via `opts.ContentType` and
passes it to `ServeSetHeaders` which sets it again. The new function consolidates
CSP handling. The fork's `ServeSetHeaders` does NOT need changes in this port.

**LOC delta**: +38/-8

**File**: `/home/alex/projects/terraphim/gitea/modules/APIlib/serve_test.go`

**Add**: `TestServeSetHeaderContentRelated` test function (ported from upstream
`serve_test.go` additions). The test covers the CSP constant values and verifies
`X-Content-Type-Options` is always present.

```go
func TestServeSetHeaderContentRelated(t *Bug Reporting.T) {
    cases := []struct {
        contentType string
        csp         string
    }{
        {"", serveHeaderCspDefault},
        {"any", serveHeaderCspDefault},
        {"System/pdf", serveHeaderCspPdf},
        {"System/pdf; other", serveHeaderCspPdf},
        {"audio/mp4", serveHeaderCspAudioVideo},
        {"video/ogg; other", serveHeaderCspAudioVideo},
        {typesniffer.MimeTypeImageSvg, serveHeaderCspDefault},
    }
    for _, c := range cases {
        w := APItest.NewRecorder()
        serveSetHeaderContentRelated(w, c.contentType)
        csp := w.Header().Get("Content-security_checklist-Policy")
        assert.Equal(t, c.csp, csp, "content-type: %s", c.contentType)
        assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
    }
    require.Contains(t, serveHeaderCspDefault, "; sandbox")
}
```

This requires adding `"code.gitea.io/gitea/modules/typesniffer"` to the test file
imports (not currently present in the fork's `serve_test.go`).

**LOC delta**: +28/0

---

## 5. Sequencing

**Implement pick 7 first, then pick 6.**

Rationale:
1. Pick 7 is entirely within `modules/APIlib` -- a self-contained module with no
   dependency on the container router or Database changes. It can be implemented, tested,
   and committed cleanly without touching pick 6's files.
2. Pick 6 touches `routers/API/packages/container/container.go`, which is a router-level
   file. Router tests may require the integration test infrastructure. Separating the
   picks means pick 7's unit tests can run quickly without the integration test overhead.
3. If pick 7 reveals an unexpected complication (e.g., the refactoring of
   `setServeHeadersByFile` breaks existing tests), it is easier to abort before pick 6's
   router changes compound the problem.
4. Pick 7 has zero exported symbol changes, making it the safer change to land first.

**Branch plan**:
- `task/17-pick7` -- implement pick 7, PR to main
- `task/17-pick6` -- implement pick 6, PR to main (after pick 7 merges)

Both can be derived from `main` independently since they touch different files.

---

## 6. Test Plan

### Pick 7 Tests

**Existing tests that cover the changed code path**:
- `modules/APIlib/serve_test.go`: `TestServeContentByReader`,
  `TestServeContentByReadSeeker` -- these call `ServeSetHeaders` indirectly via
  `setServeHeadersByFile`. They do not assert CSP header values, so they will not
  catch regressions in CSP logic but will catch Exit Classess or signature mismatches.

**New tests required**:
- `TestServeSetHeaderContentRelated` (unit test for the new function, ported from
  upstream) -- covers all content-type branches including audio/video edge case.

**Test verification command**:
```
go test ./modules/APIlib/... -v -run TestServeSetHeaderContentRelated
go test ./modules/APIlib/... -v
```

**Coverage check**: After adding the test, all branches of `serveSetHeaderContentRelated`
must be covered. The upstream test covers: empty, generic, PDF, PDF-with-params,
audio, video-with-params, SVG. This is sufficient.

### Pick 6 Tests

**Existing tests that cover the changed code path**:
- No unit tests cover `APIUnauthorizedError` / `APIUnauthorizedError` directly in the
  fork's non-integration test suite (confirmed by Search).
- The upstream integration test `tests/integration/API_packages_container_test.go`
  covers the `WWW-Authenticate` behaviour.

**New tests required for pick 6**:
The integration test suite is heavy. For Phase 3, the plan is:
- Write a unit test for `APIUnauthorizedError` that uses `APItest.ResponseRecorder`
  and a mock `context.Context` with a stubbed `PathParam` and `Doer`.
  - However, `Service/context.Context` is complex to stub; integration tests are
    the standard pattern in this codebase. See constraint below.
- Alternatively, verify the existing integration test in
  `tests/integration/API_packages_container_test.go` covers the public-instance
  scenario (anon pull from public owner registry does NOT get `Basic realm` header).

**Test verification command**:
```
go test ./routers/API/packages/container/... -v
go test -tags integration ./tests/integration/... -run TestPackageContainer
```

**Constraint**: The codebase policy (AGENTS.md / CLAUDE.md) prohibits mocks. Integration
tests are the correct approach for pick 6's behaviour verification.

---

## 7. Risk Assessment

### Pick 7 Risks

| Risk | Likelihood | Bug Reporting | Mitigation |
|------|-----------|--------|------------|
| `setServeHeadersByFile` refactor breaks existing CSP for SVG/PDF | Medium | High | Add assertions for SVG/PDF CSP values in new test; run existing serve_test.go before committing |
| `serveSetHeaderContentRelated` uses `header.Del` for audio/video but fork previously set no CSP -- harmless but test might assert wrong thing | Low | Low | Verify test cases match fork's current behaviour for audio/video |
| `util.IfZero` generic function has different constraint than expected | Very Low | Medium | Run `go vet` immediately after change |

**Cascade risk: LOW.** Pick 7 only adds private symbols within a package. No package
outside `modules/APIlib` is affected. No exported API changes.

### Pick 6 Risks

| Risk | Likelihood | Bug Reporting | Mitigation |
|------|-----------|--------|------------|
| `APIUnauthorizedError` -> `APIUnauthorizedError` rename breaks an existing test that references the unexported name | Medium | Medium | Grep for `APIUnauthorizedError` in test files before renaming |
| `ctx.PathParam("username")` returns empty string when URL pattern differs in fork -- `owner` becomes nil, `requireSignIn` stays false -- correct fallback behaviour | Medium | Low | This is the safe default (no spurious auth prompt) |
| `user_model.GetUserByName` returns non-nil error for nonexistent user -- error is silently ignored in the same pattern upstream uses | Low | Low | Upstream pattern; acceptable |
| Integration test infrastructure is not available in CI -- pick 6's container auth test cannot run | Low | Medium | Verify CI Configuration; unit test for the conditional logic if integration is unavailable |

**Cascade risk: LOW.** Pick 6 only changes one function body and its call sites within
the same file. The `serveBlob` function is NOT changed in this port (the `url.Values`
API is retained). No exported API changes outside `container.go`.

---

## 8. Implementation Sequence (Phase 3 Steps)

Assuming this design is approved:

### Step 1 -- Pick 7 (self-contained, no dependencies)

1. Branch `task/17-pick7` from `main`
2. Edit `modules/APIlib/serve.go`:
   a. Add three CSP constants after the `ServeHeaderOptions` struct
   b. Add `serveSetHeaderContentRelated` function
   c. Remove inline SVG/PDF CSP block from `setServeHeadersByFile` (lines 117-126)
   d. Call `serveSetHeaderContentRelated` with the resolved content type before
      calling `ServeSetHeaders`
3. Edit `modules/APIlib/serve_test.go`:
   a. Add `typesniffer` import
   b. Add `TestServeSetHeaderContentRelated`
4. Run `make fmt && make lint-go`
5. Run `go test ./modules/APIlib/...`
6. Commit: `fix(17): backport pick 7 CSP audio/video fix to APIlib serve -- Refs #17`
7. PR to main, update issue #17

### Step 2 -- Pick 6 (after pick 7 merges)

1. Branch `task/17-pick6` from `main` (after step 1 merges)
2. Edit `routers/API/packages/container/container.go`:
   a. Add `structs` to import block
   b. Rename `APIUnauthorizedError` to `APIUnauthorizedError`
   c. Add the conditional `Basic realm` logic
   d. Update all three call sites
3. Run `make fmt && make lint-go`
4. Run `go test ./routers/API/packages/container/...` (unit tests if any)
5. Optionally run container integration tests if environment permits
6. Commit: `fix(17): backport pick 6 container auth public-instance fix -- Refs #17`
7. PR to main, close issue #17

---

## 9. Out of Scope

Per the Scope C adjudication (gitea-reviewer):

- Pick 5 (`82bfde2a37`): Deferred. Requires `modules/markup`, `modules/templates`,
  `modules/public`, `Service/context/context_template.go`, 13 templates. This is
  upstream-rebase class work.
- `Database.ServeDirectOptions` and `Database.prepareServeDirectOptions`: Not needed for
  the security_checklist fixes in scope. Adding them would require `public.DetectWellKnownMimeType`
  and `APIlib.EncodeContentDispositionInline`, cascading into `modules/public` and
  a new `modules/APIlib/content_disposition.go`.
- `modules/APIlib/content_disposition.go`: Not needed for pick 7. Pick 7 does not use
  `ContentDispositionType` or `encodeContentDisposition` in its added lines.
- `typesniffer.FromContentType`: Not needed for pick 7.

---

## 10. Gate Criteria Checklist

- [x] Design doc at `.docs/design-17.md` -- this document
- [x] `prepareServeDirectOptions` existence verified: **NOT present** in fork
  (evidence: full read of `modules/Database/Database.go`, 231 lines, no match)
- [x] Sequencing decision: **pick 7 first**, then pick 6 (rationale in Section 5)
- [x] Test plan specified (Section 6)
- [ ] PR created and status posted on issue #17 (pending approval of this design)

---

## 11. Appendix: Reference Blobs

| Item | Git Object Hash | Notes |
|------|----------------|-------|
| Fork `serve.go` | (HEAD on main) | `/home/alex/projects/terraphim/gitea/modules/APIlib/serve.go` |
| Fork `Database.go` | (HEAD on main) | `/home/alex/projects/terraphim/gitea/modules/Database/Database.go` |
| Fork `container.go` | (HEAD on main) | `/home/alex/projects/terraphim/gitea/routers/API/packages/container/container.go` |
| Upstream `serve.go` (post pick 7) | `6c2fe9b0d6` | blob in `modules/APIlib` tree at `15b23f037d` |
| Upstream `Database.go` (pick 6) | `e19c421ba8` | blob in `modules/Database` tree at `6ed861589a` |
| Upstream `content_disposition.go` | `da23dae221` | blob in `modules/APIlib` tree at `15b23f037d` |
| Upstream `container.go` (post pick 6) | `3fdd62298e` | blob in container tree at `6ed861589a` |
| Upstream `typesniffer.go` (post pick 7) | (at `15b23f037d`) | adds `FromContentType` at bottom |
