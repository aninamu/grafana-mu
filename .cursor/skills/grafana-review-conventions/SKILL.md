---
name: grafana-review-conventions
description: Grafana monorepo conventions checklist used during code review — Go backend patterns (Wire DI, sqlstore, services), frontend patterns (RTK, Emotion, i18n, a11y), generated-code pairs, and PR hygiene. Use when reviewing Grafana code changes or when asked what conventions a change must follow. Read by the grafana-pr-review skill and the reviewer subagents.
---

# Grafana Review Conventions

Repo-specific rules a change must satisfy. Each rule lists the signal to look for in a diff. This skill is knowledge, not workflow — the caller decides what to do with violations.

## Generated-code pairs

If the left side changes, the right side must change in the same PR (or the PR must say generation is deferred):

| Hand-written change | Must regenerate |
|---------------------|-----------------|
| Service constructors / `pkg/server/wire.go` | `wire_gen.go` via `make gen-go` |
| `kinds/` CUE schemas | Go + TS types via `make gen-cue` |
| `pkg/services/featuremgmt/registry.go` | toggle files via `make gen-feature-toggles` |
| API handler swagger comments | specs via `make swagger-gen` |
| New Go module / `go.mod` in a workspace member | `go.work.sum` via `make update-workspace` |
| App SDK kinds under `apps/*/kinds/` | via `make gen-apps` |
| New user-facing strings | i18n catalogs via `make i18n-extract` |

## Go backend (`pkg/`, `apps/`)

- **Business logic placement**: logic belongs in `pkg/services/<domain>/`, not in `pkg/api/` handlers. A fat handler in a diff is a finding.
- **Interfaces**: services define their interface in their own package; consumers depend on the interface. New concrete cross-package dependencies are a smell.
- **Database access**: via `sqlstore` sessions with parameterized queries. Any string-concatenated SQL is blocking. Dialect-specific SQL must go through the dialect helpers (`migrator.Dialect`) — Grafana must run on SQLite, MySQL, and Postgres.
- **Errors**: use `errutil` for API-facing errors so status codes and public messages are structured; don't return raw `err.Error()` to clients.
- **Logging**: `pkg/infra/log`, structured key-value pairs. No `fmt.Println`, no secrets/tokens/passwords in log fields.
- **Context**: `context.Context` first parameter, propagated end to end; no `context.Background()` inside request paths.
- **Build tags**: OSS code must compile without `enterprise`/`pro` tags. Anything importing enterprise-only packages from OSS paths is blocking.
- **New API endpoints**: must update the existing tests for that service (repo rule), have RBAC evaluators on the route, and carry swagger annotations.

## Database migrations (`pkg/services/sqlstore/migrations/`)

- Append-only: never edit or reorder an existing migration that may have shipped.
- Migration IDs must be unique and descriptive.
- Must work on SQLite, MySQL, and Postgres — check for dialect-specific syntax used raw.
- Large-table operations (index builds, column type changes on `dashboard`, `alert_rule`, etc.) need a note on lock behavior for big enterprise installs.
- Never assume data volume: a migration that loads a whole table into memory is a finding.

## Frontend (`public/app/`, `packages/`)

- **State**: Redux Toolkit slices and RTK Query. New handwritten thunks/reducers in old-Redux style are a finding.
- **Styling**: Emotion via `useStyles2(getStyles)` with theme tokens. Hardcoded colors/px values that exist as theme tokens are findings. No new `.scss`.
- **i18n**: user-visible strings use `t()`/`<Trans>` from `@grafana/i18n`. Raw string literals in JSX are findings.
- **Components**: prefer `@grafana/ui` primitives over hand-rolled equivalents.
- **a11y**: interactive elements need accessible names; icon-only buttons need `aria-label`; no `div` click handlers where a button belongs.
- **Public API of `packages/`**: `@grafana/data|ui|runtime|schema` are consumed by external plugins — exported-surface changes there are breaking-change territory (see `grafana-breaking-changes` skill).
- **Betterer**: this repo tracks lint debt in `.betterer.results`; new code must not add to it (new `any`, missing a11y attrs).
- **Tests**: React Testing Library, query by role/label not test-id where possible; MSW for API mocking.

## PR hygiene

- Frontend and backend ship as separate PRs (different deploy cadences). Mixed PRs need justification.
- Feature work lands behind a feature toggle when it changes existing behavior.
- No links (Slack/Jira/GitHub) in code comments; comments explain *why*, not *what*.
- User-facing changes need a `docs/` update following `docs/AGENTS.md`.
- Check `.github/CODEOWNERS` for required team sign-off on touched paths.

## Directory-scoped overrides

These AGENTS.md files override anything above for their subtree — read them when the diff touches those paths:

- `pkg/storage/unified/AGENTS.md`
- `public/app/features/alerting/unified/AGENTS.md`
- `docs/AGENTS.md`
