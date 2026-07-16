#!/usr/bin/env python3
"""Ask before destructive shell commands."""

from __future__ import annotations

import os
import re
import sys

sys.path.insert(0, os.path.dirname(__file__))

from hook_utils import allow_shell, ask_shell, read_input  # noqa: E402

DANGEROUS_PATTERNS: list[tuple[str, re.Pattern[str], str]] = [
    (
        "recursive delete",
        re.compile(r"\brm\s+(-[a-zA-Z]*r[a-zA-Z]*|--recursive)\b"),
        "This command recursively deletes files. Confirm the paths before continuing.",
    ),
    (
        "hard git reset",
        re.compile(r"\bgit\s+reset\s+--hard\b"),
        "This will discard uncommitted work in the working tree.",
    ),
    (
        "git push",
        re.compile(r"\bgit\s+push\b"),
        "Pushing requires explicit human approval in this repository.",
    ),
    (
        "git clean",
        re.compile(r"\bgit\s+clean\b"),
        "This may permanently remove untracked files.",
    ),
    (
        "docker prune",
        re.compile(r"\bdocker\s+system\s+prune\b"),
        "This removes unused Docker data and can be destructive.",
    ),
    (
        "devenv shutdown",
        re.compile(r"\bmake\s+devenv-down\b"),
        "This stops local backing services used for development.",
    ),
    (
        "force branch reset",
        re.compile(r"\bgit\s+checkout\s+--force\b|\bgit\s+switch\s+--force\b"),
        "This may overwrite local changes.",
    ),
]


def main() -> None:
    payload = read_input()
    command = payload.get("command", "")

    for label, pattern, message in DANGEROUS_PATTERNS:
        if pattern.search(command):
            ask_shell(
                f"Hook blocked a potentially destructive command ({label}). {message}",
                f"A hook flagged this shell command as potentially destructive ({label}). "
                f"Explain why it is needed or choose a safer alternative.",
            )

    allow_shell()


if __name__ == "__main__":
    main()
