---
name: keep-tests-in-sync
description: Updates relevant tests when application code changes—API endpoints, function signatures, return types, or new edge cases. Finds matching test files, updates or adds only cases affected by the change, and runs those tests. Use after modifying application code, handlers, services, hooks, reducers, or utilities when tests may be stale or missing.
---

# Keep Tests in Sync

Apply this skill after changing application code, before considering the task done.

## Scope rule

Only update tests **directly affected by the change**:

- Broken assertions, mocks, or call sites caused by your edit
- New branches, parameters, return values, or error paths you introduced
- API route/handler contract changes

Do **not**:

- Refactor unrelated tests
- Add broad coverage "while you're here"
- Change test style or structure beyond what the diff requires

## Workflow

```
Progress:
- [ ] 1. Identify what changed and what behavior must be verified
- [ ] 2. Find the matching test file(s)
- [ ] 3. Update existing cases or add minimal new ones
- [ ] 4. Run only the affected test(s)
- [ ] 5. Fix failures; stop when targeted tests pass
```

### 1. Identify test impact

From the diff, list:

| Change type | Test action |
|-------------|-------------|
| Function/method signature | Update call sites, mocks, table inputs |
| New parameter or return field | Extend assertions or fixture data |
| New error/edge branch | Add one focused case for that branch |
| Renamed/moved symbol | Update imports and references in tests |
| API endpoint (path, method, status, body) | Update handler/API test request + expectations |
| Deleted behavior | Remove or adjust obsolete cases |

Skip areas with no behavioral change (comments, formatting, renames with no contract change).

### 2. Find matching test files

Search colocated first, then package/directory conventions.

**Go** (`pkg/`, `apps/`):

- `foo.go` → `foo_test.go` (same directory, same package)
- Integration tests may live in `*_integration_test.go` or `pkg/tests/` — only touch these if the change crosses service boundaries

**TypeScript/React** (`public/app/`, `packages/`):

- `Component.tsx` → `Component.test.tsx`
- `utils.ts` → `utils.test.ts`
- Hook/reducer/API module → same basename with `.test.ts` or `.test.tsx`

If no test file exists and the change adds **new behavior worth verifying**, create a colocated test file following neighbors in that directory. If the change is trivial with no new behavior, do not create a file.

### 3. Update or add cases

- Match existing patterns: test helpers, `describe`/`it` structure, table-driven Go tests, RTL queries, mock setup
- Prefer editing an existing case over adding a duplicate
- One new case per new branch or contract change — not a suite rewrite
- For API/handler tests: update route, payload, status code, and response body assertions together

### 4. Run affected tests

Run the **smallest command that covers the changed tests**:

**Go:**

```bash
go test -run TestName ./path/to/package/
go test ./path/to/package/
```

**Frontend** (use `--watchAll=false` or `yarn jest --no-watch` so tests exit):

```bash
yarn jest --no-watch path/to/file.test.ts
yarn jest --no-watch -t "test name pattern"
```

**Snapshots** — only when your change intentionally updates UI output:

```bash
yarn jest --no-watch path/to/file.test.tsx -u
```

Do not run full `make test-go-unit` or the entire frontend suite unless targeted runs pass and the change has wide blast radius.

### 5. Handle failures

- Fix test code when the **implementation is correct** and expectations are stale
- Fix implementation when the test correctly encodes the intended contract
- Do not weaken assertions to make tests pass without confirming intended behavior

## Quick examples

**Signature change** — `func Parse(id string)` → `func Parse(id string, orgID int64)`:

- Update all test call sites and table rows in `parse_test.go`
- Add one case if `orgID` enables a new validation path

**New API field** — POST body gains required `folderUid`:

- Update request fixtures and success/error cases in the handler test
- Add one case for missing `folderUid` if the handler now rejects it

**New edge case** — nil input now returns error instead of panic:

- Add one test: nil input → expected error
- Do not add unrelated nil tests for other functions
