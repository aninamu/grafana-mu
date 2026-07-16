#!/usr/bin/env python3
"""Inject reminders after agent file writes."""

from __future__ import annotations

import os
import sys

sys.path.insert(0, os.path.dirname(__file__))

from hook_utils import (  # noqa: E402
    docs_edit_reminder,
    emit_additional_context,
    find_secret_matches,
    generated_code_reminders,
    noop,
    read_input,
    relative_path,
)


def extract_write_content(payload: dict) -> tuple[str, str]:
    if payload.get("hook_event_name") == "afterFileEdit":
        file_path = payload.get("file_path", "")
        edits = payload.get("edits", [])
        content = "\n".join(edit.get("new_string", "") for edit in edits)
        return file_path, content

    tool_input = payload.get("tool_input", {})
    file_path = tool_input.get("path") or tool_input.get("file_path", "")
    content = tool_input.get("contents") or tool_input.get("content", "")
    return file_path, content


def main() -> None:
    payload = read_input()
    file_path, content = extract_write_content(payload)
    if not file_path:
        noop()

    rel_path = relative_path(file_path, payload.get("workspace_roots", []))
    messages: list[str] = []
    messages.extend(generated_code_reminders(rel_path))
    messages.extend(docs_edit_reminder(rel_path))

    secret_matches = find_secret_matches(content)
    if secret_matches:
        joined = ", ".join(secret_matches)
        messages.append(
            f"Possible secret material detected in `{rel_path}` ({joined}). "
            "Remove credentials before committing and prefer environment variables."
        )

    if ".env" in rel_path.split("/")[-1] and content.strip():
        messages.append(
            f"Edited env file `{rel_path}`. Do not commit secrets; use local overrides instead."
        )

    if messages:
        emit_additional_context(*messages)

    noop()


if __name__ == "__main__":
    main()
