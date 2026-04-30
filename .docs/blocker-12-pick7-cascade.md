# Blocker: Phase 3 Pick-7 Cascade for Issue #12

## Pick

Pick 7: `15b23f037d` -- Fix attachment Content-Security-Policy (#37455)

## Status

BLOCKED -- cannot land without resolving multiple transitive upstream dependencies.

## Root Cause

The upstream commit `15b23f037d` modifies `modules/httplib/serve.go` (an upstream-only file).
Taking `--theirs` introduces references to multiple symbols that do not exist in the fork's
`modules/httplib` package.

## Missing Symbols (from `go build ./...`)

- `ContentDispositionType` -- type not defined in `modules/httplib`
- `encodeContentDisposition` -- function not defined in `modules/httplib`
- `typesniffer.FromContentType` -- method/function not defined in `modules/typesniffer`
- `ContentDispositionInline` -- constant/var not defined in `modules/httplib`
- `ContentDispositionAttachment` -- constant/var not defined in `modules/httplib`

Build error:
```
modules/httplib/serve.go:33:21: undefined: ContentDispositionType
modules/httplib/serve.go:91:37: undefined: encodeContentDisposition
modules/httplib/serve.go:136:30: undefined: typesniffer.FromContentType
modules/httplib/serve.go:137:29: undefined: ContentDispositionInline
modules/httplib/serve.go:139:30: undefined: ContentDispositionAttachment
```

## Resolution Options

A. Backport the missing `httplib` types and functions from upstream, then re-attempt pick 7.
   This requires understanding the upstream `httplib` refactoring that introduced
   `ContentDispositionType`, `ContentDispositionInline`, `ContentDispositionAttachment`,
   and `encodeContentDisposition`.

B. Skip pick 7 entirely. Document missing CSP attachment fix in issue #12.

## Date

2026-04-30
