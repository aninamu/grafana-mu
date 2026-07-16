#!/usr/bin/env python3
"""Nudge agents toward one-shot Jest runs in this repo."""

from __future__ import annotations

import os
import re
import sys

sys.path.insert(0, os.path.dirname(__file__))

from hook_utils import allow_shell, ask_shell, read_input  # noqa: E402

YARN_TEST = re.compile(r"\byarn\s+test\b")
JEST_ONE_SHOT = re.compile(r"(?:--no-watch|--watchAll=false|--watch=false|\bjest\b)")


def main() -> None:
    payload = read_input()
    command = payload.get("command", "")

    if YARN_TEST.search(command) and not JEST_ONE_SHOT.search(command):
        ask_shell(
            "This repo's default `yarn test` script runs Jest in watch mode. "
            "Prefer `yarn jest --no-watch path/to/file` for agent test runs.",
            "Grafana's `yarn test` uses watch mode by default. "
            "Use `yarn jest --no-watch path/to/file` or add `--watchAll=false` "
            "when you need a one-shot test run.",
        )

    allow_shell()


if __name__ == "__main__":
    main()
