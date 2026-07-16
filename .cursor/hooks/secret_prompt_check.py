#!/usr/bin/env python3
"""Block prompts that appear to contain secrets."""

from __future__ import annotations

import os
import sys

sys.path.insert(0, os.path.dirname(__file__))

from hook_utils import emit, find_secret_matches, read_input  # noqa: E402


def main() -> None:
    payload = read_input()
    prompt = payload.get("prompt", "")

    matches = find_secret_matches(prompt)
    if matches:
        joined = ", ".join(matches)
        emit(
            {
                "continue": False,
                "user_message": (
                    f"Prompt blocked by hook: possible secret detected ({joined}). "
                    "Remove credentials from the prompt and use environment variables instead."
                ),
            }
        )

    emit({"continue": True})


if __name__ == "__main__":
    main()
