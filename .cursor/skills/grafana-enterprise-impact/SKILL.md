---
name: grafana-enterprise-impact
description: Knowledge base for assessing how a Grafana OSS change affects Grafana Enterprise and Grafana Cloud — extension points, RBAC, licensing, multi-org/multi-tenancy, HA, provisioning, and scale concerns. Use when reviewing changes for enterprise impact, when a diff touches auth/accesscontrol/licensing/settings, or when asked "will this break enterprise". Read by the enterprise-impact-reviewer subagent.
---

# Grafana Enterprise Impact

Grafana Enterprise is a separate closed-source repo that builds *on top of* this OSS repo: it imports OSS packages, swaps implementations behind interfaces, and registers extra services via hooks. OSS changes can break Enterprise without any test in this repo failing. This skill lists the seams and what to check at each one.

## The OSS/Enterprise seam

Enterprise attaches to OSS at these extension points. Changes here have outsized blast radius:

| Extension point | Where in OSS | Risk when changed |
|-----------------|--------------|-------------------|
| Enterprise imports hook | `pkg/extensions/enterprise_imports.go` | Enterprise registers itself here; signature/registry changes break the enterprise build |
| Wire DI graph | `pkg/server/wire.go`, `pkg/registry/` | Enterprise overrides providers; removing/renaming a provider or changing a constructor signature breaks their overrides |
| Licensing interface | `pkg/services/licensing/` (`oss.go`, `models.go`) | Enterprise supplies the real implementation; any interface change is a cross-repo breaking change |
| Access control | `pkg/services/accesscontrol/` | Enterprise adds fine-grained RBAC, data source permissions, teams sync on top of these interfaces |
| Hooks service | `pkg/services/hooks/` | Enterprise hooks into index/login pages |
| Settings | `pkg/setting/`, `conf/defaults.ini` | Enterprise reads OSS settings structs; renamed/retyped fields break silently |
| Encryption/secrets | `pkg/services/encryption/`, `pkg/services/secrets/`, `pkg/services/kmsproviders/` | Enterprise adds KMS backends behind these interfaces |
| Publicly exported Go API | any exported symbol under `pkg/` | Enterprise imports OSS packages directly; treat exported symbols like a public library API |

**Rule of thumb**: changing an exported interface, struct field, or constructor signature in the packages above is a *cross-repo* breaking change even when all OSS tests pass. The finding should say "verify against grafana-enterprise" explicitly.

## RBAC and permissions

- New API routes must declare access-control evaluators (`middleware/authorize` with `ac.Eval*`). A route with only `reqSignedIn` deserves scrutiny: is org-role gating really enough, and how does it interact with enterprise fine-grained permissions?
- New actions/scopes must follow the `<resource>:<verb>` / `<resource>:<attribute>:<value>` naming conventions and be registered so enterprise role sync picks them up.
- Anything that bypasses the accesscontrol service (direct org-role checks like `c.OrgRole == org.RoleAdmin`) undermines enterprise custom roles — flag it.
- Resource-level permissions (dashboards, folders, data sources, service accounts) have enterprise-managed inheritance; changes to folder/dashboard move/delete logic must consider permission cascade.

## Multi-org and multi-tenancy

- Every new query must filter by `org_id`. A missing org filter is a **cross-tenant data leak** — always blocking.
- New tables need an `org_id` column (or an explicit reason they're global).
- Caches keyed without org/tenant in the key leak data between tenants in Grafana Cloud.
- `namespace`/stack ID handling in apiserver-style code (`pkg/apis/`, `apps/`) must not assume the single-org default.

## Auth

- Changes under `pkg/services/authn/`, `pkg/services/auth/`, `pkg/services/ldap/`, `pkg/login/` affect enterprise SAML/LDAP-sync/OAuth flows that live partly in the enterprise repo.
- Session/token behavior changes (`pkg/services/auth/` token service) affect remote cache HA setups.
- Anything touching signed-in user structs (`identity.Requester`, `user.SignedInUser`) is consumed pervasively by enterprise code.

## HA and scale (Grafana Cloud runs this)

- **Statefulness**: new in-memory state (maps, caches, singletons) breaks multi-replica HA unless it's advisory-only or backed by the database/remote cache. Ask: what happens with 3 replicas behind a load balancer?
- **Background jobs**: new goroutines/tickers run on *every* replica. Do they need leader election or `ServerLockService`?
- **Query cost**: unbounded queries (`SELECT` without limits over dashboards, users, alerts) will be run against installs with 100k+ dashboards and 10k+ users. Pagination is required.
- **Startup cost**: work added to service `Init`/`Run` delays every pod start in cloud rollouts.

## Provisioning and config management

- Enterprise customers manage everything as code: new resources/settings should be provisionable (file provisioning, Terraform provider, or the new `apps/provisioning` path) — a UI-only knob is a finding for enterprise workflows.
- New config options need `conf/defaults.ini` entries with safe defaults, and must not change behavior for existing installs when unset.
- Renaming or repurposing an existing ini key breaks fleet-managed configs; require a deprecation path.

## Unified storage compatibility

`pkg/storage/unified/` deploys as a *separate service* at its own cadence in cloud. Changes spanning API-layer callers and unified storage must be backwards compatible in **both directions** (old API ↔ new storage, new API ↔ old storage). `pkg/storage/unified/AGENTS.md` is the authority — read it whenever the diff touches that tree.

## Reporting

For each finding, state: the seam it hits (from the tables above), worst-case blast radius (enterprise build break / cross-tenant leak / HA incident / config break), and the cheapest verification (e.g. "grep grafana-enterprise for uses of this symbol", "add org_id filter", "guard behind toggle").
