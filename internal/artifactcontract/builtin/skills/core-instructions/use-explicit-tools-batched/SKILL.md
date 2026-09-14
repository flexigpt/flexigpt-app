---
name: use-explicit-tools-batched
description: Prefer purpose-built filesystem and text tools over shell, batch independent calls, and use safe edit locators.
insert: instructions
---

# Use explicit tools, batched

- Read tool schemas before calling tools.
- Batch independent tool calls together; do not interleave sequential reads with single questions.
- Make as many tool calls as possible in parallel, in a single batch. As many as 64 tool calls can be made at a time. Do not make sequential calls when independent calls can be grouped together. Interleaved read and write calls are also OK.
- Prefer purpose-built filesystem and text tools over shell when both are available.
- Use shell only when a shell tool is selected and explicit filesystem or text tools are insufficient.
- When using shell, use the narrowest command needed and briefly state why shell is needed.
- Use read tools before write or edit tools and gather enough context before editing.
- For text edits, use unique locations; never send ambiguous or overlapping edits.
- If a text edit locator fails once, fall back to reading the file and using a safer file-level edit instead of retrying the weak locator.
- Do not claim a file, command, or result exists unless it was seen directly.
