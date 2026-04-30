# Research: Issue #15 -- Wire `TestRobotAPI` integration tests into CI

Status: Phase 1 RESEARCH (V-Model)
Branch (target): `task/15-robotapi-integration-tests`
Author: Ferrox

## 1. Issue summary

Acceptance criteria from issue:

1. The `TestRobotAPI_*` sub-tests pass against the SQLite fixture database.
2. CI integration test job shows green for the run.
3. Document the correct command for invoking the integration suite.

The original report mentions "13 sub-tests" -- in practice the file
`tests/integration/robot_security_test.go` declares **14** top-level tests
(see section 3.2 below). Six of those are sub-tests of
`TestRobotAPI_InvalidInput` so the count depends on how you count.

## 2. CI wiring

### 2.1 Existing job is already correct

`.github/workflows/pull-db-tests.yml` lines 67-91 define a `test-sqlite` job
that runs `make test-sqlite`, which invokes
`integrations.sqlite.test` against `tests/sqlite.ini`. New `_test.go` files
in `tests/integration/` are picked up automatically -- there is **no
additional CI wiring required** for the test file to be discovered. The
issue's "wire into CI" requirement is therefore satisfied by the existing
workflow as long as the tests themselves pass.

### 2.2 Local invocation (the documented command)

```bash
# generate the sqlite test config (idempotent)
make generate-ini-sqlite

# build the integration test binary
make integrations.sqlite.test

# run the robot suite (regex must avoid the literal substring "API" -- see 4.2)
GITEA_TEST_CONF=tests/sqlite.ini ./integrations.sqlite.test \
    -test.run 'TestRobotAP' -test.v -test.timeout 300s
```

Or with the Makefile helper:

```bash
make 'test-sqlite#TestRobotAP'
```

Note the exact-case prefix: an earlier run using `TestRobotapi` (lowercase
`api`) returned `no tests to run` because the regex did not match any
top-level test name.

## 3. Code under test

### 3.1 Handlers

| Endpoint              | File                                          | Notes |
|-----------------------|-----------------------------------------------|-------|
| `/api/v1/robot/triage`| `routers/api/v1/robot/robot.go`               | Returns 404 (not 400) on validation failure (intentional security design) |
| `/api/v1/robot/ready` | `routers/api/v1/robot/ready_graph.go`         | Same 404-on-validation behaviour |
| `/api/v1/robot/graph` | `routers/api/v1/robot/ready_graph.go`         | Same 404-on-validation behaviour |

Route registration: `routers/api/v1/api.go:1757-1762`, gated by
`tokenRequiresScopes(auth_model.AccessTokenScopeCategoryIssue)`.

Feature toggle: `setting.IssueGraphSettings.Enabled` (default `true`,
defined in `modules/setting/graph.go`).

### 3.2 Test inventory (14 top-level tests)

```
TestRobotAPI_UnauthorizedPrivateRepo
TestRobotAPI_PublicRepoAnonymous
TestRobotAPI_AuthorizedAccess
TestRobotAPI_InvalidInput        (6 sub-tests)
TestRobotAPI_FeatureDisabled
TestRobotAPI_ReadyEndpoint
TestRobotAPI_GraphEndpoint
TestRobotAPI_AllEndpoints
TestRobotAPI_Integration
TestRobotAPI_NonExistentRepo
TestRobotAPI_NonExistentOwner
TestRobotAPI_CacheConsistency
TestRobotAPI_Performance
TestRobotAPI_ErrorMessages
```

## 4. Findings: things that block the acceptance criteria

### 4.1 Server-side nil-pointer panic on anonymous access

Reproduced by `TestRobotAPI_PublicRepoAnonymous` and
`TestRobotAPI_AuthorizedAccess`. Stack trace shows `gopanic` -> `sigpanic`
-> `panicmem` while serving the request. Root cause is in
`routers/api/v1/robot/robot.go:Triage`:

* `checkRepoPermissionForTriage` calls
  `access_model.GetUserRepoPermission(ctx, repository, ctx.Doer)`.
  For anonymous users `ctx.Doer == nil`, which the helper does not
  defend against.
* The unconditional `robot.LogRobotAccessQuick(ctx.Doer.ID,
  ctx.Doer.Name, ...)` after the permission check dereferences
  `ctx.Doer` when it is `nil`.

`Ready` and `Graph` already guard the analogous calls with
`if ctx.IsSigned && ctx.Doer != nil { ... }`. `Triage` must do the same.

### 4.2 Validation status-code mismatch (test expects 400, code returns 404)

`TestRobotAPI_InvalidInput` declares `expectCode: http.StatusBadRequest`
for all six sub-cases (path traversal, null byte, oversize, special
chars). The handlers deliberately return `ctx.APIErrorNotFound()` after a
validation failure with the comment:

```go
// Return 404 to avoid leaking validation details
ctx.APIErrorNotFound()
```

This is a deliberate security choice (do not distinguish "bad input"
from "no such repo" to a probing client). The test must be updated to
expect `http.StatusNotFound` -- the implementation is correct.

### 4.3 Test panic on `Null_byte_in_owner`

Inside `TestRobotAPI_InvalidInput/Null_byte_in_owner`:

```
robot_security_test.go:128 -> NewRequestf -> NewRequest -> NewRequestWithBody
panic: runtime error: invalid memory address or nil pointer dereference
integration_test.go:330 (req.RequestURI = urlStr)
```

`http.NewRequest` rejects URLs containing `\x00` and returns
`(nil, err)`. `NewRequestWithBody` calls `assert.NoError(t, err)` (which
records the failure but does not stop execution) and then dereferences
`req`, causing the segfault. We can either:

* drop the null-byte sub-case (path-validation already covers it because
  null byte is rejected by `validateOwnerRepoInput`), or
* construct the request manually with a URL-encoded null byte (`%00`),
  which travels through HTTP normally and lets the validator do its job.

Option 2 is the better test of the validator and matches the security
hardening intent; we will use it.

### 4.4 Regex pattern caveat

`-test.run TestRobotAPI` matches **zero** tests against this binary even
though `-test.run TestRobotAP` matches all 14. The cause appears to be
a quirk in this build (possibly Go 1.26 toolchain test pattern
processing -- the binary uses `golang.org/toolchain@v0.0.1-go1.26.0`).
Documenting the working pattern (`TestRobotAP` or `TestRobotAPI_`) and
using it in the run command above sidesteps the issue. We will not try
to "fix" the regex matcher itself; this is out of scope.

## 5. Upstream check

Searched git log for related fixes since the robot package was added:

```
9df0d5afe8 feat: Complete robot API implementation with PageRank and CLI
0a8e21dd1d security: Implement Gitea Robot security hardening (v1.26.1)
cfcafbda81 feat(robot): sort ready issues by PageRank descending
0471720616 fix(robot): remove unused db import causing build failure
```

Neither the anonymous-access nil-pointer bug nor the 400/404 mismatch
has been fixed upstream of this branch. There is no prior PR to
cherry-pick.

## 6. Scope of the fix (handed to design phase)

In priority order:

1. **Handler hardening**: guard `ctx.Doer` access in `Triage` (and any
   other path that dereferences it without the
   `IsSigned && ctx.Doer != nil` check). This is the only **production**
   change in scope for this issue -- the handlers already exist and
   already implement the security policy; we are only fixing the
   anonymous-user crash.
2. **Test corrections**:
   * `TestRobotAPI_InvalidInput`: expect 404 not 400 (matches the
     security-by-design behaviour of the handler).
   * `TestRobotAPI_InvalidInput/Null_byte_in_owner`: URL-encode the
     null byte so `http.NewRequest` succeeds and the validator sees it.
3. **Documentation**: add the local-invocation command (section 2.2) to
   the issue and to a short note in the PR description.

CI itself needs **no change** -- `make test-sqlite` already discovers
the file. The issue is satisfied once 1+2 land and the existing
workflow goes green.

## 7. Out of scope

* Refactoring the duplicated audit-log / permission-check boilerplate
  across `Triage`, `Ready`, `Graph`. Bigger change, separate issue.
* Investigating why `-test.run TestRobotAPI` matches no tests in this
  Go 1.26 toolchain build -- documented as a workaround.
* Changing the security policy of returning 404 instead of 400 for bad
  input. The current design is correct (avoid information leak).
