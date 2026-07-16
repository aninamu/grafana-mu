---
name: grafana-breaking-changes
description: Detection rules for breaking changes in Grafana — HTTP API contracts, plugin-facing packages (@grafana/data, ui, runtime, schema), exported Go APIs, CUE/kind schemas, dashboard JSON compatibility, and config keys. Use when reviewing a diff for backwards compatibility, API stability, or plugin breakage. Read by the breaking-change-detector subagent.
---

# Grafana Breaking Changes

Grafana has an unusually wide compatibility surface: external plugin authors, dashboard JSON in the wild, provisioning files, the HTTP API used by Terraform and scripts, and the enterprise repo compiled against `pkg/`. This skill defines what counts as breaking and how to spot it in a diff.

## Surfaces, in order of blast radius

### 1. Plugin-facing npm packages (`packages/grafana-{data,ui,runtime,schema,e2e-selectors}`)

Thousands of external plugins compile against these. In a diff, breaking means:

- Removed/renamed exported symbol, or removal from the package `index.ts` barrel.
- Changed function signature (new required param, changed return type).
- New required property on an exported interface (plugins implementing it stop compiling); adding *optional* members is fine.
- Narrowed prop types or removed props on exported React components.
- Changed runtime behavior of `@grafana/runtime` services (`getBackendSrv`, `locationService`, plugin extension APIs).

Deprecate with `@deprecated` + release-note migration path first; removal only after a deprecation window. An undeprecated removal is blocking.

### 2. HTTP API (`pkg/api/`, `pkg/services/*/api/`, `pkg/apis/`)

Consumed by Terraform provider, grizzly, curl scripts, and the frontend of *older* versions during rolling upgrades.

- Removed/renamed endpoints, changed methods, changed URL params: breaking.
- Response fields: removing or renaming a JSON field is breaking; adding fields is safe.
- Request handling: a previously-optional field becoming required, or validation tightening that rejects previously-accepted payloads, is breaking.
- Status-code changes (e.g. 400→403) break client error handling.
- For `pkg/apis/` (Kubernetes-style APIs): version discipline applies — breaking changes require a new API version (`v0alpha1` → `v1beta1`), never in-place mutation of a served version.

### 3. Exported Go API under `pkg/`

The enterprise repo and some external tools import these packages directly. Removed/renamed exported symbols, changed interface method sets, and changed constructor signatures are breaking for the enterprise build even if OSS compiles. Flag with "verify against grafana-enterprise" (see the `grafana-enterprise-impact` skill for the seam list).

### 4. Schemas and dashboard JSON (`kinds/`, `apps/*/kinds/`, `@grafana/schema`)

- Dashboard JSON is forever: dashboards exported years ago must still import. Removing/renaming panel option fields without a migration in the frontend (panel `setMigrationHandler`) or schema version migration is breaking.
- CUE kind changes must be additive within a version; check that `make gen-cue` output is committed and that TS/Go stay in sync.
- Datasource query models (`packages/grafana-schema/src/raw/composable/`) are saved inside dashboard JSON — same rules as panel options.

### 5. Config and provisioning

- Renamed/removed ini keys in `conf/defaults.ini` or `pkg/setting/` break fleet-managed configs; require alias + deprecation warning.
- Changed *defaults* silently change behavior on upgrade — call out any default change as a release-note item.
- Provisioning YAML shapes (datasources, dashboards, alerting) are API contracts too.

### 6. Feature toggles

- Removing a toggle that installs have set in config is safe only if the toggle is `GA`/`deprecated` per its registry stage.
- Changing a toggle's default (off→on) is a behavior change for every install; needs a release note.

## Diff-scanning heuristics

- In `packages/`, diff any `src/index.ts` barrel: deletions there are near-certain breaks.
- Grep the diff for removed lines starting with `export` (TS) or matching `^func [A-Z]|^type [A-Z]` (Go).
- JSON tag edits in Go structs (`json:"oldName"` → `json:"newName"`) silently break API responses — search the diff for `json:` on modified lines.
- A route registration file (`pkg/api/api.go`, `apps/*/register.go`) with deletions deserves endpoint-by-endpoint checking.
- Check for `@deprecated` markers being *added* (good, note it) vs deprecated symbols being *removed* (verify the deprecation window: grep the changelog).

## Reporting

Classify each finding as: **breaking** (needs migration/deprecation before merge), **behavior change** (needs release note + toggle consideration), or **compatible**. Always name the consumer that breaks (external plugins / HTTP API clients / enterprise repo / existing dashboards / fleet configs).
