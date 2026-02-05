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
  - [Available Skills Exposure](#available-skills-exposure)
  - [Activation-Required Rule](#activation-required-rule)
  - [Active Skill Instruction Injection](#active-skill-instruction-injection)
- [Runtime Tools Specification](#runtime-tools-specification)
  - [Skill Activation Tool](#skill-activation-tool)
  - [Skill DeActivation Tool](#skill-deactivation-tool)
  - [Skill Resource/Assets Read Tool](#skill-resourceassets-read-tool)
  - [Skill Script Execution Tool](#skill-script-execution-tool)
- [State Model](#state-model)
  - [State Variables per Session](#state-variables-per-session)
  - [State Transitions](#state-transitions)
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
- `skill_md_body` (string): Markdown body of `SKILL.md` (loaded on activation or cached)

### Runtime Session

A Session is the unit of conversational continuity for skill activation. A session has:

- `session_id` (string)
- `active_skill` (nullable): the currently active skill name (and optionally version/digest)

Sessions allow a linear chat history while enabling changing “top-level instructions” per call without rewriting prior messages.

## Component Responsibilities

### Orchestrator (Control-Plane)

The orchestrator is responsible for:

1. Indexing & maintaining the Skill Registry
   - Discover skill folders
   - Parse frontmatter
   - Resolve `location` and `root_dir`
2. Prompt/instructions composition per LLM call
   - Inject a representation of `<available_skills>`
   - Inject active skill instructions for the current session (if any)
   - Inject the “activation-required” rule
3. Tool dispatch
   - Expose the runtime tools defined next to the LLM
   - Execute tool calls and append tool results to the message stream (chronological)
4. Skill instruction provisioning
   - Provide the activated skill’s `SKILL.md` body to the LLM _via top-level instructions_ for subsequent calls
     - Can be via tool result if the API does not support mutable top-level instructions.

### State Manager (State-Plane)

A state manager stores session state:

- `active_skill_name` (nullable)
- optionally `active_skill_location`, `active_skill_root_dir`, `allowed-tools`

It may be: a separate component, or integrated into the orchestrator.

### Runtime Tools (Data-Plane; LLM-callable)

Runtime tools are the standardized interface for:

- skill activation
- reading skill resources (references/assets)
- running skill scripts

Tools are _functionally stateless by signature_ but operate against a session context (provided implicitly by the host via `session_id` binding).

## Prompt/Instruction Contract

### Available Skills Exposure

- Inject `<available_skills>` XML containing:
  - `name`, `description`, `location`

### Activation-Required Rule

The orchestrator MUST include a base instruction rule:

- The LLM MUST call `skills.activate` before using any skill’s instructions/resources/scripts.

### Active Skill Instruction Injection

When a session has an `active_skill`, the orchestrator SHOULD include the active skill’s `SKILL.md` body in the top-level instructions of each LLM call, e.g.:

- `<active_skill name="pdf-processing"> ... SKILL.md body ... </active_skill>`

This ensures:

- Linear message history remains append-only
- Active instructions can change across turns without editing history
- Old skill instructions do not automatically persist

## Runtime Tools Specification

### Skill Activation Tool

- `skills.activate`:
  - Purpose: Set the session’s active skill.

Behavior:

- Resolve skill by `name` or `location` using the Skill Registry.
- Set `active_skill` for the current session.
- Load and store `SKILL.md` body for injection on subsequent LLM calls via top-level instructions on the next model call.

Output (recommended minimal receipt):

```json
{
  "active_skill": "pdf-processing",
  "location": "/abs/path/pdf-processing/SKILL.md",
  "root_dir": "/abs/path/pdf-processing",
  "properties": {
    "name": "pdf-processing",
    "description": "...",
    "allowed-tools": "..."
  }
}
```

### Skill DeActivation Tool

- `skills.deactivate`:
  - Purpose: Clear the active skill.

### Skill Resource/Assets Read Tool

- `skills.read`
  - Purpose: Read skill-scoped files on demand (progressive disclosure).

Behavior:

- Requires an active skill.
- Resolve `path` relative to `active_skill.root_dir`.
- Read and return file content as either text or binary as required.
- This single tool covers both `references/` and `assets/` (and optionally other files under skill root, if the runtime allows).

### Skill Script Execution Tool

- `skills.run_script`
  - Purpose: Execute a script from the active skill’s `scripts/` directory.

Input (structured):

```json
{
  "path": "scripts/extract.py",
  "args": ["--in", "input.pdf", "--out", "out.json"],
  "env": { "KEY": "VALUE" },
  "workdir": "."
}
```

Behavior:

- Requires an active skill.
- Resolve script path relative to `active_skill.root_dir`.
- Execute the script according to runtime-defined execution rules (e.g., interpreter mapping by extension, or direct execution if executable).
- Return structured results.
- The runtime MUST define and document how it executes scripts (e.g., `.py` via `python3`, `.sh` via `bash`, `.js` via `node`, etc.). This is needed to determine whether scripts “run seamlessly.”

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

### State Variables per Session

- `active_skill_name: string | null`
- optional: `active_skill_location`, `active_skill_root_dir`, `active_skill_properties`

### State Transitions

- `skills.activate` sets `active_skill_name`
- `skills.deactivate` clears it
- `skills.read` and `skills.run_script` require it to be non-null

## Linear Turn-by-Turn Runtime Behavior

This is the normative linear flow; it does not require rewriting prior messages.

### Per Turn LLM Call Construction

For each LLM call, orchestrator constructs:

1. Top-level instructions containing:
   - Base rules (activation required)
   - Available skills exposure
   - If `active_skill` exists: inject active skill instructions (`SKILL.md` body)

2. Messages array containing chronological conversation + tool calls/results so far.

### Typical Flow: Choose, Activate, Use

1. LLM sees available skills (prompt or tool) and decides a skill is needed.
2. LLM calls `skills.activate(...)`.
3. Orchestrator appends tool result and updates session state.
4. Next LLM call includes active `SKILL.md` instructions in top-level instructions.
5. LLM uses `skills.read` and `skills.run_script` as needed.

### Skill Switching

1. LLM calls `skills.activate(name="skill-b")`.
2. Orchestrator updates state (`active_skill = skill-b`).
3. Next LLM call injects skill-b’s `SKILL.md` instructions (not skill-a).
