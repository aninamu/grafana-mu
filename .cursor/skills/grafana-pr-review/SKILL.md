---
name: grafana-pr-review
description: Orchestrates a thorough, Grafana-specific review of a PR or branch, covering Go backend, TypeScript/React frontend, enterprise impact, security, breaking changes, and database migrations. Use when asked to review a PR, review a branch, review changes before merge, or do a "full review" of work in the Grafana repo.
---

# Grafana PR Review

Top-level entry point for reviewing a PR or branch in the Grafana monorepo. This skill scopes the diff, dispatches specialized reviewer subagents, and assembles one prioritized review. Do not review the whole diff yourself line-by-line — your job is scoping, dispatch, and synthesis.

## Companion skills and subagents

| Piece | Role |
|-------|------|
| `grafana-review-conventions` skill | Repo-wide conventions checklist (read it now, before dispatching) |
| `grafana-enterprise-impact` skill | Enterprise risk knowledge (read by `enterprise-impact-reviewer`) |
| `grafana-breaking-changes` skill | Breaking-change detection rules (read by `breaking-change-detector`) |
| `pr-review-orchestrator` subagent | Spawns `backend-pr-reviewer`, `frontend-pr-reviewer`, `enterprise-impact-reviewer`, `security-pr-reviewer` |
| `backend-pr-reviewer` | May spawn `sql-migration-reviewer` for migration diffs |
| `enterprise-impact-reviewer` | May spawn `breaking-change-detector` for API/schema/config diffs |

## Workflow

```
Progress:
- [ ] 1. Resolve the diff (PR number, branch, or uncommitted changes)
- [ ] 2. Read the grafana-review-conventions skill
- [ ] 3. Classify changed paths and read directory-scoped AGENTS.md files
- [ ] 4. Dispatch the pr-review-orchestrator subagent with explicit scope
- [ ] 5. Do the orchestration-level checks yourself (section below)
- [ ] 6. Synthesize one prioritized review
```

### 1. Resolve the diff

- PR number/URL: `gh pr view <n> --json title,body,files,baseRefName` and `gh pr diff <n>`.
- Branch: `git diff --stat origin/main...HEAD` and `git diff --name-only origin/main...HEAD`.
- Uncommitted: `git status --porcelain` and `git diff HEAD --name-only`.

Read the PR description/commit messages — reviewers need the *intent*, not just the diff.

### 2. Read the conventions skill

Read `.cursor/skills/grafana-review-conventions/SKILL.md`. Use it for the orchestration-level checks in step 5 and quote its relevant rules in subagent prompts when a change obviously trips one.

### 3. Classify changed paths

| Paths | Stack |
|-------|-------|
| `pkg/`, `apps/`, `go.mod`, `go.work` | backend |
| `public/app/`, `packages/`, `*.tsx`, `*.ts`, `package.json`, `yarn.lock` | frontend |
| `pkg/services/sqlstore/migrations/` | migrations (backend reviewer handles via its child) |
| `kinds/`, `pkg/apis/`, `apps/*/kinds/` | schema/API — flag for breaking-change review |
| `conf/defaults.ini`, `pkg/setting/` | config — flag for enterprise review |
| `docs/` | check against `docs/AGENTS.md` yourself; no subagent needed |

Directory-scoped agent docs exist and override general guidance — if the diff touches these areas, read them and pass key constraints into the orchestrator prompt:

- `pkg/storage/unified/AGENTS.md` — bidirectional compatibility rules
- `public/app/features/alerting/unified/AGENTS.md` — alerting squad patterns
- `docs/AGENTS.md` — docs style

### 4. Dispatch the orchestrator subagent

Spawn the `pr-review-orchestrator` subagent via the Task tool. Its prompt must include:

- How to reproduce the diff (exact `gh pr diff` or `git diff` command).
- The PR title/description or a one-paragraph summary of intent.
- The path classification from step 3 (which stacks apply, whether migrations/schemas/config are touched).
- Any constraints extracted from directory-scoped AGENTS.md files.
- Any focus areas the user requested.

The orchestrator will spawn `backend-pr-reviewer`, `frontend-pr-reviewer`, `enterprise-impact-reviewer`, and `security-pr-reviewer` as needed (children spawn their own specialists) and return a merged report.

### 5. Orchestration-level checks (do these yourself while the orchestrator runs)

- **PR shape**: frontend and backend changes should ship in separate PRs (repo policy). Flag mixed PRs unless the change is trivially coupled.
- **Generated code drift**: if `wire.go`/service init changed but `wire_gen.go` didn't (or vice versa), flag missing `make gen-go`. Same for `kinds/` vs generated TS/Go (`make gen-cue`), feature toggle registry vs generated files (`make gen-feature-toggles`), and swagger specs.
- **Tests present**: new functionality without any test changes is a finding by itself. New API endpoints must update existing service tests (repo rule).
- **CODEOWNERS**: check `.github/CODEOWNERS` for which teams own the touched paths; note owners the author may need sign-off from.
- **Docs**: user-facing behavior changes with no `docs/` update.

### 6. Synthesize

Merge the orchestrator's report with your own step-5 findings into one review:

```markdown
## Review: <PR title or branch>

### Verdict
One sentence: approve / approve with nits / needs changes / needs enterprise sign-off.

### Blocking
- `file:line` — issue, why it blocks, suggested fix.

### Enterprise & compatibility
- Findings from the enterprise reviewer, breaking-change detector, migration reviewer.

### Non-blocking
- Suggestions and nits, grouped by file.

### Well done
- 1–3 things the PR does right (only if genuine).
```

Deduplicate findings that multiple reviewers reported. Every blocking finding needs `file:line` and a concrete fix. If a specialist found nothing, say so in one line rather than omitting the topic silently.

## Constraints

- Never push, comment on GitHub, or modify the PR — this is a read-only review unless the user explicitly asks for fixes.
- Always dispatch through `pr-review-orchestrator`; do not substitute a single shallow pass for the specialist reviews.
- If the diff is tiny (< ~30 lines, single stack, no migrations/schemas/config), you may skip the orchestrator and spawn the one relevant specialist reviewer directly — say you did so.
