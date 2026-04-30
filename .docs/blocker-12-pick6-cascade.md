# Blocker: Phase 3 Pick-6 Cascade for Issue #12

## Pick

Pick 6: `6ed861589a` -- Fix container auth for public instance (#37290)

## Status

BLOCKED -- cannot land without resolving transitive upstream dependency.

## Root Cause

The upstream commit `6ed861589a` modifies `routers/api/packages/container/container.go`
(an upstream-only file). Taking `--theirs` introduces a reference to `storage.ServeDirectOptions`
which does not exist in the fork's `modules/storage` package.

## Missing Symbol

- Package: `code.gitea.io/gitea/modules/storage`
- Symbol: `storage.ServeDirectOptions` (struct or type)
- Used at: `routers/api/packages/container/container.go:718`

Build error:
```
routers/api/packages/container/container.go:718:105: undefined: storage.ServeDirectOptions
```

## Resolution Options

A. Backport `storage.ServeDirectOptions` from upstream into the fork's `modules/storage`
   package, then re-attempt pick 6.

B. Skip pick 6 entirely. Document missing container auth fix in issue #12.

## Decision

**Option B -- Skip pick 6 entirely.**

Consistent with pick 2's E1 resolution. Backporting `storage.ServeDirectOptions` (option A)
requires understanding the upstream storage refactor, violating design v2 §1. Container auth
surface not confirmed present in our deployed surface.

Decided: 2026-04-30 by Echo (Twin Maintainer) / coordinator decision.

## Date

2026-04-30
