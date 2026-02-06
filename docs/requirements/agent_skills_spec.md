# Agent Skills Runtime Implementation Spec

- [Terminology](#terminology)
  - [Skill Package](#skill-package)
  - [Skill Registry](#skill-registry)
  - [Runtime Session](#runtime-session)
- [Component Responsibilities](#component-responsibilities)
  - [Orchestrator (Control-Plane)](#orchestrator-control-plane)
  - [State Manager (State-Plane)](#state-manager-state-plane)
  - [Runtime Tools (Data-Plane; LLM-callable)](#runtime-tools-data-plane-llm-callable)
- [Prompt/Instruction Contract](#promptinstruction-contract)
- [Runtime Tools Specification](#runtime-tools-specification)
  - [Skill Load Tool](#skill-load-tool)
  - [Skill Unload Tool](#skill-unload-tool)
  - [Skill Resource/Assets Read Tool](#skill-resourceassets-read-tool)
  - [Skill Script Execution Tool](#skill-script-execution-tool)
- [State Model](#state-model)
- [Linear Turn-by-Turn Runtime Behavior](#linear-turn-by-turn-runtime-behavior)
  - [Per Turn LLM Call Construction](#per-turn-llm-call-construction)
  - [Typical Flow: Choose, Activate, Use](#typical-flow-choose-activate-use)
  - [Skill Switching](#skill-switching)

This document specifies a runtime protocol for using Agent Skills. It defines:

- Orchestrator responsibilities (non-tool, control-plane)
- State manager responsibilities (optional component; may be part of orchestrator)
- LLM-callable tools (data-plane)
- Turn-by-turn linear behavior (append-only chat history; no rewriting)

## Terminology

### Skill Package

A skill package is a directory that conforms to the Agent Skills format:

```shell
skill-name/
  SKILL.md
  scripts/        (optional)
  references/     (optional)
  assets/         (optional)
```

`SKILL.md` contains:

- YAML frontmatter: `name`, `description`, optional fields
- Markdown body: instructions

### Skill Registry

A Skill Registry is a mapping from `skill.name` → `SkillRecord`, built by indexing skill directories.

SkillRecord (functional fields):

- `name` (string)
- `description` (string)
- `location` (string): absolute path to `SKILL.md`. Required for FS agents, Optional for ToolAgents.
- `root_dir` (string): parent directory of `SKILL.md`
- `properties` (object): parsed frontmatter fields (license, compatibility, allowed-tools, metadata)
- `skill_md_body` (string): Markdown body of `SKILL.md` (cached post loading)

### Runtime Session

A Session is a lightweight conversation-scoped state container used to link:

- tool calls (e.g., `skills.load`, `skills.unload`) and their effects
- per-turn prompt injection (active skill instructions)
- authorization checks for skill-scoped reads and script execution

A session has:

- `session_id` (string)
- `active_skills` (string[]): ordered list of currently loaded skills (most-recent last)
- optional: `active_skill_digests` (map): skill name -> digest/version (implementation-defined)

Sessions allow a linear chat history while enabling changing “top-level instructions” per call without rewriting prior messages.

## Component Responsibilities

### Orchestrator (Control-Plane)

The orchestrator is responsible for:

1. Indexing & maintaining the Skill Registry
   - Discover skill folders
   - Parse frontmatter
   - Resolve `location` and `root_dir`

2. Prompt/instructions composition per LLM call
   - Implement skill loading.
   - Inject a representation of `<available_skills>` (recommended: every call)
   - Inject active skills instructions for the current session (if any)

3. Tool dispatch
   - Expose the runtime tools defined next to the LLM
   - Execute tool calls and append tool results to the message stream (chronological)

4. Skill instruction provisioning
   - Provide the loaded skill’s `SKILL.md` body to the LLM _via top-level instructions_ for subsequent calls
     - Can be via tool result if the API does not support mutable top-level instructions.

### State Manager (State-Plane)

A state manager stores session state:

- `active_skill_name` (nullable)
- optionally `active_skill_location`, `active_skill_root_dir`, `allowed-tools`

It may be: a separate component, or integrated into the orchestrator.

### Runtime Tools (Data-Plane; LLM-callable)

Runtime tools are the standardized interface for:

- skill loading
- reading skill resources (references/assets)
- running skill scripts

Tools are _functionally stateless by signature_ but operate against a session context (provided implicitly by the host via `session_id` binding).

## Prompt/Instruction Contract

- Available Skills Exposure:
  - Inject `<available_skills>` XML containing `name`, `description`, `location` (location optional for tool-only agents).
  - Recommended policy: include `<available_skills>` in the top-level instructions of EVERY LLM call to keep selection consistent across turns.

- Loading-Required Rule:
  - The orchestrator MUST include a base instruction rule that specifies that the LLM MUST call `skills.load` before using any skill’s full instructions/resources/scripts.

- Active Skills Instruction Injection:
  - When a session has `active_skills`, the orchestrator SHOULD include each loaded skill’s `SKILL.md` body in the top-level instructions of each LLM call:
    - `<active_skills> <skill name="..."> ... </skill> ... </active_skills>`
  - Ordering rule: inject in `active_skills` order; if instructions conflict, later (more recently loaded) skills take precedence.
  - The runtime SHOULD cap the number of simultaneously loaded skills (implementation-defined) to protect context limits.

- This ensures:
  - Linear message history remains append-only
  - Active instructions can change across turns without editing history
  - Old skill instructions do not automatically persist

## Runtime Tools Specification

### Skill Load Tool

- `skills.load`:
  - Purpose: Load one or more skills (progressive disclosure) and set/update the session’s loaded skill set.

Recommended Input (structured):

```json
{
  "names": ["pdf-processing", "data-analysis"],
  "mode": "replace"
}
```

Where `mode` is:

- `"replace"`: replace the loaded skill set with `names` (recommended default)
- `"add"`: add `names` to the loaded skill set (de-dupe by name; preserve order)

Behavior:

- Resolve each skill by `name` (and optionally `location`) using the Skill Registry.
- Update `active_skills` according to `mode`.
- Load and cache each skill’s `SKILL.md` body for injection on subsequent LLM calls via top-level instructions (recommended: inject on the next model call).
- Enforce a maximum number of loaded skills (implementation-defined); if exceeded, return an error instructing the LLM to load fewer skills.

Output (recommended minimal receipt):

```json
{
  "active_skills": [
    {
      "name": "pdf-processing",
      "location": "/abs/path/pdf-processing/SKILL.md",
      "root_dir": "/abs/path/pdf-processing",
      "digest": "sha256:...",
      "properties": {
        "name": "pdf-processing",
        "description": "...",
        "allowed-tools": "..."
      }
    }
  ]
}
```

### Skill Unload Tool

- `skills.unload`:
  - Purpose: Remove one or more loaded skills from the session, or clear all loaded skills.

Recommended Input:

```json
{ "names": ["pdf-processing"] }
```

or

```json
{ "all": true }
```

Behavior:

- If `all=true`, clear `active_skills`.
- If `names` provided, remove those skills if present.
- Return updated `active_skills` receipt.

### Skill Resource/Assets Read Tool

- `skills.read`
  - Purpose: Read skill-scoped files on demand (progressive disclosure).

Behavior:

- Requires at least one loaded skill (`active_skills.length > 0`).
- If multiple skills are loaded, the tool SHOULD accept a `skill` field to disambiguate; otherwise default to the most recently loaded skill.
- Resolve `path` relative to the selected skill’s `root_dir`.
- Read and return file content as either text or binary as required.
- This single tool covers both `references/` and `assets/` (and optionally other files under skill root, if the runtime allows).

### Skill Script Execution Tool

- `skills.run_script`
  - Purpose: Execute a script from the active skill’s `scripts/` directory.

Behavior:

- Requires at least one loaded skill.
- If multiple skills are loaded, the tool SHOULD accept a `skill` field to disambiguate; otherwise default to the most recently loaded skill.
- Resolve script path relative to the selected skill’s `root_dir`.
- Execute the script according to runtime-defined execution rules (e.g., interpreter mapping by extension, or direct execution if executable).
- Return structured results.
- The runtime MUST define and document how it executes scripts (e.g., `.py` via `python3`, `.sh` via `bash`, `.js` via `node`, etc.). This is needed to determine whether scripts “run seamlessly.”

Example Input (structured):

```json
{
  "skill": "pdf-processing",
  "path": "scripts/extract.py",
  "args": ["--in", "input.pdf", "--out", "out.json"],
  "env": { "KEY": "VALUE" },
  "workdir": "."
}
```

Output:

```json
{
  "path": "scripts/extract.py",
  "exit_code": 0,
  "stdout": "...",
  "stderr": ""
}
```

## State Model

- State Variables per Session
  - `active_skills: string[]`
  - optional: per-skill cached `location`, `root_dir`, `properties`, `digest`

- State Transitions
  - `skills.load` updates `active_skills` (replace/add)
  - `skills.unload` removes skills or clears all
  - `skills.read` and `skills.run_script` require `active_skills.length > 0`

## Linear Turn-by-Turn Runtime Behavior

This is the normative linear flow; it does not require rewriting prior messages.

### Per Turn LLM Call Construction

For each LLM call, orchestrator constructs:

1. Top-level instructions containing:
   - Base rules (loading required)
   - Available skills exposure (recommended: every call)
   - If `active_skills` exists: inject active skills instructions (`SKILL.md` bodies)

2. Messages array containing chronological conversation + tool calls/results so far.

### Typical Flow: Choose, Activate, Use

1. LLM sees available skills (prompt or tool) and decides a skill is needed.
2. LLM calls `skills.load(...)`.
3. Orchestrator appends tool result and updates session state.
4. Next LLM call includes active `SKILL.md` instructions in top-level instructions.
5. LLM uses `skills.read` and `skills.run_script` as needed.

### Skill Switching

1. LLM calls `skills.load(names=["skill-b"], mode="replace")`.
2. Orchestrator updates state (`active_skills=["skill-b"]`).
3. Next LLM call injects skill-b’s `SKILL.md` instructions (not skill-a).
