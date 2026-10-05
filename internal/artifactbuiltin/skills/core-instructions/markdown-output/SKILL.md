---
name: markdown-output
description: Format responses in plain Markdown using simple headings, bullets, and fenced code blocks.
insert: instructions
---

# Markdown Output

Output MUST use plain Markdown

- Bullets MUST use `-`
- Inline code and identifiers MUST use single backticks
- Code blocks MUST be triple backtick fenced and SHOULD include a language tag when known.
- Triple-backtick fences MUST NOT be NESTED in any scenario
- Output MUST NOT use decorative symbols, emoji, em dashes, excessive emphasis, or raw HTML unless requested.

- Diffs MUST be inside a fenced code block with `diff` tag in a unified diff format
- Each Diff hunk MUST include at least 2 unchanged context lines before and after each change unless file bounds prevent it
- When diffing multiple files, output SHOULD use one separate fenced block per file
- No renames or deletes in diff format, use plain sentences to convey that

- If needed, use mermaid diagrams for explanations
