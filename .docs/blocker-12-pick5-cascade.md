# Blocker: Phase 3 Pick-5 Cascade for Issue #12

## Pick

Pick 5: `82bfde2a37` -- Use Content-Security-Policy: script nonce (#37232)

## Status

BLOCKED -- cannot land without resolving multiple transitive upstream dependencies.

## Root Cause

The upstream commit `82bfde2a37` introduces changes that reference symbols not present in
the fork. Taking `--theirs` for upstream-only files introduces these missing symbols.
Keeping `--ours` for `modules/markup/render.go` is insufficient because other upstream-only
files also reference missing symbols.

## Missing Symbols (full list from `go build ./...`)

### `modules/markup/external/openapi.go`
- `ctx.RenderOptions.StandalonePageOptions` -- field not present on `markup.RenderOptions`
- `markup.RenderIFrame` -- function not defined in the fork
- `public.AssetURI` -- function not defined in `modules/public`

### `modules/templates/helper.go`
- `public.AssetURI` -- function not defined in `modules/public`
- `NewFuncMap` -- function not defined (renamed/refactored upstream)

### `modules/templates/mail.go` (indirect cascade)
- `NewFuncMap` -- function not defined

### `modules/templates/page.go` (indirect cascade)
- `NewFuncMap` -- function not defined

## Attempted Resolutions

### Attempt 1: Take theirs for all upstream-only files
Result: build fails with `public.AssetURI` undefined in `modules/markup/render.go`

### Attempt 2: Take theirs for all upstream-only files except render.go (keep ours)
Result: build fails with multiple undefined symbols in `openapi.go` and `helper.go`

## Files Causing Cascade (upstream-only, both cause build failures)

- `modules/markup/external/openapi.go`
- `modules/templates/helper.go`

## Resolution Options

A. Backport missing symbols from upstream into the fork before re-attempting pick 5:
   - Add `public.AssetURI` to `modules/public/public.go`
   - Add `RenderOptions.StandalonePageOptions` struct field
   - Add `markup.RenderIFrame` function
   - Ensure `NewFuncMap` is defined/exported in `modules/templates`
   This requires analysing multiple upstream commits to understand the full refactoring.

B. Skip pick 5 (CSP script nonce) entirely. The nosniff header from pick 3 still lands.
   Document missing CSP nonce security feature in issue #12.

C. Apply pick 5 partially with manual stubs for missing symbols. High risk of introducing
   subtle bugs or incomplete security behaviour.

## Date

2026-04-30
