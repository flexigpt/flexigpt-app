# Tools UX Requirements

## Terms

- Tool: A backend capability the assistant can use (e.g. _Read local file_, _Run SQL_, etc.).

- Tool choice (user → assistant): A tool the user attaches to the current message via the existing "tool menu" in the composer. Example: `Tool: Read file`, `Tool: Use DB`.
- Tool call (assistant → tools): An invocation _suggested_ by the assistant, with concrete arguments. Example: `Read file: data.csv (columns: name, amount)`.

- Tool call output (tool → assistant): An execution output representing the result of a tool call, reusable in later messages. The execution can be human approved or run locally; conceptually it also covers built-in capabilities provided by the LLM API, although those are not surfaced as chips in this UX. Example: `Result: data.csv (1,245 rows)`.

## Conceptual Model and Flow

1. Users attach _tool choices_ the same way they attach files.

   - Via the existing tool menu → chip in the strip → read‑only chip under their bubble after send.
   - Composer text is cleared.
   - The live chips for that message are "frozen" as history in the bubble.

2. The assistant may respond with _tool calls_ (e.g. "I want to read file X with these options").

   - E.g. - Text: "I’ll read the file and then summarize its contents." + Tool call: `read_file(file = "data.csv", range = "A1:F500")`
   - Each tool call becomes: A small read‑only "suggested tool" chip under the assistant bubble

3. Also, an interactive _tool call chip_ is now attached at the front of the composer strip in the editor.

   - E.g.: `Read file: data.csv (rows 1–500) [Run] [×]`
   - Tool call chips can be run or discarded.
   - The history is in the bubble (above read only chip), and control in the composer (this chip).
   - States and actions:
     - Pending (default) : Visible label summarizing tool + arguments. Buttons: Run (▶) Discard (×)
     - Running: Shows a small spinner / "Running…" state.
   - Discarding a tool call chip only affects the composer; the read‑only "Suggested tool" chip under the assistant bubble remains as history.
   - Composer strip ordering (left → right):
     - Tool call chips (assistant-suggested, pending/completed)
     - Tool output chips
     - Other attachment chips (files, URLs, etc.) + Tool choice chips (user-chosen via the tool menu)

4. Running a tool transforms the _tool call chip_ to a _tool output chip_ that is attached to the next message by default.

   - The user has an option to see the output (click on the chip to open up a visual modal to see the output).
   - They can discard it via a button too.
   - Once a chip is in "output" state, it is no longer considered "pending" for global actions.

5. The primary button adapts:

   - Normal "Send"
   - Or "Run & send" when there is at least one pending tool call chip in the composer. Icon can be adapted. Behavior:
     - Run all pending tool calls.
     - When all complete:
       - Convert their chips to tool output chips.
       - Immediately send a new request to the assistant including:
         - The last assistant tool calls,
         - The tool outputs (all current tool output chips in the composer),
         - The current editor text (possibly empty).

6. Secondary control: Small "Run tools only" button or icon near the primary:

   - Runs all pending tool calls.
   - Updates chips to outputs.
   - Does not yet send a new message.
   - Lets the user see results first, then type and hit "Send".

7. Once all tool calls (if any) are resolved:

   - Primary button label returns to "Send".
   - User can send with or without additional text, using the available tool output chips.

8. When there are no tool calls: Primary button is always "Send" (current behavior).

## Bubble Behavior Summary

User bubbles:

- Show text.
- Below, read‑only chips for:
  - Files.
  - Tool choices used that turn.
  - Tool outputs the user chose to attach.

Assistant bubbles:

- Show text.
- Below, read‑only "Suggested tool" chips representing the assistant’s tool calls.
- No Run/Discard here; that happens in the composer strip.

## Example: Read Local File Tool

Tool: `read_file`

- Input: attached file reference + optional range/options.
- Output: structured representation of file content, plus any helper summary.

Flow:

1. User attaches `report.csv` as a file and a `Tool: Read file` choice chip.
   Asks: "Use the Read file tool on this CSV and tell me the total revenue."

2. Assistant responds:

   - "I’ll read report.csv, then compute the total revenue."
   - Tool call: `read_file(file = report.csv, sheet = "Summary")`

3. UI:

   - Assistant bubble shows the text and a read‑only chip:
     `Suggested: Read file (report.csv, sheet "Summary")`
   - Composer strip shows a tool call chip at the front:
     `Read file: report.csv (Summary)` [Run] [×]

4. User clicks "Run & send" with no extra text:

   - The Read file tool runs.
   - Chip becomes: `Result: report.csv (Summary, 450 rows)`.
   - Backend sends the new request to LLM including the file contents as tool output.
   - Next assistant bubble gives the human‑readable answer: "Total revenue is \$1.2M."

5. Under the user bubble for that turn:
   - Chips show that `report.csv` and `Result: report.csv (Summary…)` were both used.

## Built‑in Provider Tools (e.g. Web Search)

- Some LLM providers expose _built‑in_ tools such as web search or their own retrieval, which run entirely inside the API. From the client’s perspective:

  - There is no explicit tool call or output to manage.
  - The model simply uses search/retrieval internally and returns a normal answer, possibly with citations.

- UX implications:

  - These built‑in tools do not create tool call or tool output chips.
  - Additional lightweight indicators like "Citations" or "Searched the web" or "Retrieved from knowledge base" in the assistant bubble can be built depending on the capability type.
  - Given that these calls are executed by the LLM provider on their infra directly, we don’t have the intermediate step of compose or run/discard in our UI.
