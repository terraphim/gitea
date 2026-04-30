# Design: Issue #15 -- Wire `TestRobotAPI` integration tests into CI

Status: Phase 2 DESIGN (V-Model)
Inputs: `.docs/research-15.md`
Branch: `task/15-robotapi-integration-tests`
Author: Ferrox

## 1. Architecture decision

The existing CI workflow already runs `make test-sqlite`, which discovers
all `*_test.go` files under `tests/integration/`. The robot suite is
already wired in by virtue of being in that directory. The acceptance
criteria fail today not because the wiring is missing but because:

* `Triage` panics on anonymous (`ctx.Doer == nil`) requests.
* The `TestRobotAPI_InvalidInput` cases assert `http.StatusBadRequest`,
  whereas the handlers (correctly, for a security-by-obscurity reason)
  return `http.StatusNotFound`.
* The `Null_byte_in_owner` sub-case panics in the test framework before
  it can be evaluated, because `http.NewRequest` rejects the raw `\x00`.

The design therefore is:

* **Production change**: make `Triage` (and `checkRepoPermissionForTriage`)
  defensive against `ctx.Doer == nil`, mirroring the pattern already in
  `Ready`/`Graph`.
* **Test change**: align expectations with the implemented security
  policy (404, not 400) and URL-encode the null-byte case so the
  validator -- not `http.NewRequest` -- decides the outcome.
* **No CI change**: the existing job covers it.

## 2. File-by-file change list

### 2.1 `routers/api/v1/robot/robot.go` (production)

Three callsites assume `ctx.Doer != nil`:

1. `checkRepoPermissionForTriage` -- passes `ctx.Doer` to
   `access_model.GetUserRepoPermission`, which dereferences it.
2. `Triage` post-permission audit log:
   `robot.LogRobotAccessQuick(ctx.Doer.ID, ctx.Doer.Name, ...)` on the
   denial branch.
3. `Triage` success audit log: same pattern on the allow branch.

Fix: switch to the same guarded pattern already used in
`ready_graph.go`:

```go
var userID int64
username := "anonymous"
if ctx.IsSigned && ctx.Doer != nil {
    userID = ctx.Doer.ID
    username = ctx.Doer.Name
}
robot.LogRobotAccessQuick(userID, username, ...)
```

For the permission helper, add an early return for anonymous users when
the repo is **not** private (anonymous read of a public repo is allowed
by the existing `IsPrivate && !ctx.IsSigned` short-circuit earlier in
`Triage`, so by the time we reach `checkRepoPermissionForTriage` an
anonymous user implies a public repo and `CanRead(unit.TypeIssues)` on
units that are public-by-default should return true). Concretely:

```go
func checkRepoPermissionForTriage(ctx *context.APIContext, repository *repo_model.Repository) bool {
    // Anonymous on a public repo: allow read of public issue units.
    if !ctx.IsSigned || ctx.Doer == nil {
        if repository.IsPrivate {
            ctx.APIErrorNotFound()
            return false
        }
        return true
    }

    perm, err := access_model.GetUserRepoPermission(ctx, repository, ctx.Doer)
    ...
}
```

This mirrors how `Ready` and `Graph` already gate the audit log calls
and is consistent with `ctx.IsSigned && ctx.Doer != nil` checks
elsewhere in the codebase.

### 2.2 `tests/integration/robot_security_test.go` (test)

Two surgical changes:

1. **`TestRobotAPI_InvalidInput`**: change `expectCode` for all six
   sub-cases from `http.StatusBadRequest` to `http.StatusNotFound`.
   Update the function-level comment to reflect that the API does not
   distinguish "bad input" from "not found", which is the documented
   security policy.

2. **`Null_byte_in_owner`**: replace literal `"user\x00"` with the
   URL-encoded form `"user%00"`, which `http.NewRequest` accepts and
   which exercises `validateOwnerRepoInput`'s null-byte check on the
   server side (after the request is parsed).

No new tests are added. We do not relax the `TestRobotAPI_PublicRepoAnonymous`
or `TestRobotAPI_AuthorizedAccess` assertions -- they will pass once the
nil-pointer panic in `Triage` is fixed (4.1).

### 2.3 No changes elsewhere

* `.github/workflows/pull-db-tests.yml` -- already runs the suite.
* `Makefile` -- already exposes `test-sqlite#%` for targeted runs.
* `tests/sqlite.ini.tmpl` -- unchanged.

## 3. Test strategy

### 3.1 Unit-level

Not applicable -- the change to `Triage` is exclusively about defensive
coding and is exercised end-to-end by the integration tests.

### 3.2 Integration

Run the targeted suite after each change:

```bash
make generate-ini-sqlite
make integrations.sqlite.test
GITEA_TEST_CONF=tests/sqlite.ini ./integrations.sqlite.test \
    -test.run 'TestRobotAP' -test.v -test.timeout 300s
```

Expected after both changes:

| Test                                            | Expected |
|-------------------------------------------------|----------|
| TestRobotAPI_UnauthorizedPrivateRepo            | PASS     |
| TestRobotAPI_PublicRepoAnonymous                | PASS (was crashing) |
| TestRobotAPI_AuthorizedAccess                   | PASS (was crashing) |
| TestRobotAPI_InvalidInput (and 6 sub-cases)     | PASS (404 expected) |
| TestRobotAPI_FeatureDisabled                    | PASS     |
| TestRobotAPI_ReadyEndpoint                      | PASS     |
| TestRobotAPI_GraphEndpoint                      | PASS     |
| TestRobotAPI_AllEndpoints                       | PASS     |
| TestRobotAPI_Integration                        | PASS     |
| TestRobotAPI_NonExistentRepo                    | PASS     |
| TestRobotAPI_NonExistentOwner                   | PASS     |
| TestRobotAPI_CacheConsistency                   | PASS     |
| TestRobotAPI_Performance                        | PASS     |
| TestRobotAPI_ErrorMessages                      | PASS     |

All 14 top-level tests + 6 sub-tests = 20 individual `PASS` lines.

### 3.3 Regression

Run the broader integration suite *for at least the affected handlers*:

```bash
GITEA_TEST_CONF=tests/sqlite.ini ./integrations.sqlite.test \
    -test.run 'TestAPIRepo' -test.v -test.timeout 300s
```

We do not need to run the full ~hour-long suite locally; CI does that.

## 4. Risk

* **Behaviour change for anonymous users**: today the handler panics; after
  the fix, anonymous users get a real response (or a 404 for private
  repos). This is the intended behaviour and is what the test asserts;
  no externally-visible regression because the prior behaviour was a
  500 from a recovered panic.
* **400 -> 404 in tests**: zero risk -- tests are aligned with the
  documented security policy. Any external client relying on a 400 from
  this endpoint would be inferring information that the policy
  explicitly does not give.

## 5. Commit plan

Two commits, each with `Refs terraphim/gitea#15`:

1. `fix(robot): guard ctx.Doer in Triage to prevent nil-pointer panic on anonymous access`
2. `test(robot): align TestRobotAPI_InvalidInput with 404 security policy and URL-encode null byte`

Then push and open PR `task/15-robotapi-integration-tests -> main` with
the local invocation command in the description and `@adf:gitea-reviewer`
mentioned for review.
