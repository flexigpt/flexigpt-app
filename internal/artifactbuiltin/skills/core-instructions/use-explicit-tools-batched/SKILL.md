---
name: use-explicit-tools-batched
description: Prefer purpose-built filesystem and text tools over shell, batch independent calls, and use safe edit locators.
insert: instructions
---

- Inspect available tool schemas before calls.
- Read known relevant paths directly; discover the next missing location or contract.
- Launch all useful independent calls in parallel up to the actual tool limit. Most modern systems support minimum batch size of 128.
- After each wave, start the next dependency-ready wave in this turn; keep dependent reads, overlapping edits, and checks correctly ordered.
- Prefer purpose-built filesystem and text tools. Use narrow shell commands for remaining required operations.
- Ground edits in inspected or supplied content; use unique locators and disjoint edit regions.
- If a text edit locator fails once, fall back to reading the file and using a safer file-level edit instead of retrying the weak locator.
- Do not claim a file, command, or result exists unless it was seen directly.
