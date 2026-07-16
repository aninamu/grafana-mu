#!/usr/bin/env python3
"""Shared helpers for Grafana Cursor hooks."""

from __future__ import annotations

import json
import re
import sys
from typing import Any


def read_input() -> dict[str, Any]:
    return json.load(sys.stdin)


def emit(payload: dict[str, Any]) -> None:
    print(json.dumps(payload))
    sys.exit(0)


def allow_shell() -> None:
    emit({"permission": "allow"})


def ask_shell(user_message: str, agent_message: str) -> None:
    emit(
        {
            "permission": "ask",
            "user_message": user_message,
            "agent_message": agent_message,
        }
    )


def emit_additional_context(*messages: str) -> None:
    filtered = [message.strip() for message in messages if message and message.strip()]
    if filtered:
        emit({"additional_context": "\n\n".join(filtered)})


def noop() -> None:
    emit({})


SECRET_PATTERNS: list[tuple[str, re.Pattern[str]]] = [
    ("AWS access key", re.compile(r"AKIA[0-9A-Z]{16}")),
    ("private key block", re.compile(r"-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----")),
    ("GitHub personal access token", re.compile(r"ghp_[A-Za-z0-9]{20,}")),
    ("GitHub fine-grained token", re.compile(r"github_pat_[A-Za-z0-9_]{20,}")),
    ("Slack token", re.compile(r"xox[baprs]-[A-Za-z0-9-]{10,}")),
    (
        "generic API key assignment",
        re.compile(
            r"(?i)(?:api[_-]?key|secret|password|token)\s*[:=]\s*['\"]?[A-Za-z0-9/_+=.-]{16,}"
        ),
    ),
]


def find_secret_matches(text: str) -> list[str]:
    matches: list[str] = []
    for label, pattern in SECRET_PATTERNS:
        if pattern.search(text):
            matches.append(label)
    return matches


def relative_path(file_path: str, workspace_roots: list[str]) -> str:
    normalized = file_path.replace("\\", "/")
    for root in workspace_roots:
        root_normalized = root.rstrip("/").replace("\\", "/")
        if normalized.startswith(root_normalized + "/"):
            return normalized[len(root_normalized) + 1 :]
        if normalized == root_normalized:
            return ""
    return normalized


def generated_code_reminders(relative_file_path: str) -> list[str]:
    reminders: list[str] = []

    if relative_file_path.startswith("kinds/") and relative_file_path.endswith(".cue"):
        reminders.append(
            "Edited CUE schema under `kinds/`. Run `make gen-cue` before committing."
        )

    if relative_file_path.startswith("pkg/services/featuremgmt/"):
        reminders.append(
            "Edited feature toggle definitions. Run `make gen-feature-toggles` before committing."
        )

    if relative_file_path == "pkg/server/wire.go" or "/wire.go" in relative_file_path:
        reminders.append(
            "Edited Wire DI configuration. Run `make gen-go` after changing service initialization."
        )

    if relative_file_path.startswith("apps/") and relative_file_path.endswith(".cue"):
        reminders.append(
            "Edited app CUE schema. Run `make gen-apps` if app SDK types changed."
        )

    return reminders


def docs_edit_reminder(relative_file_path: str) -> list[str]:
    if relative_file_path.startswith("docs/") and not relative_file_path.endswith("AGENTS.md"):
        return [
            "Documentation edit detected. Follow `docs/AGENTS.md` for docs style and structure."
        ]
    return []

