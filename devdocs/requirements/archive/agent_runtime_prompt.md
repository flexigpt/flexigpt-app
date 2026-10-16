You are implementing the attached **Go Agent Runtime Proposal** in the FlexiGPT application.

The Go Agent Runtime Proposal will be provided with this prompt. Treat it as the normative architecture and functional specification. Apply every requirement in the proposal exactly.

The current `internal/agentruntime-go`, the sessionstore, and agent and workspace layers along with wrappers are the implementation are supplied as reference material and migration input. You are "enhancing" this codebase properly as described in the proposal.

This is an implementation task. Do not return a high-level design only. Proper code and other artifacts in described output format below are expected.
You can run things in phases as needed, but no partial things shoudl be in a phase, the sessionruntime can be one phase, store can be other, setup can be other. YOU should not create unnecessary large number of phases. Large and coherent output is expected which is prod ready.

Do not provide a review, implementation plan, partial patch summary, compatibility strategy, etc.

## Deliverable

Deliver a proper and full implementation of the proposed Session-owned runtime architecture.

The implementation must include all required backend, persistence, Wails, frontend, runtime setup, inference refactoring, Agent/Workspace composition, MCP App routing, and migration-removal work necessary for the proposal’s definition of done.

Do not leave required functionality as:

- TODOs
- Stubs
- Placeholder implementations
- Compatibility bridges
- Feature flags that preserve old chat ownership
- Fallback paths to legacy conversation execution
- Frontend-owned execution paths

Management APIs may remain where they are still valid for management UI, but they must not remain part of Session chat orchestration.

## Authority and conflict resolution

Use these rules when implementation details are unclear:

1. The proposal is authoritative over existing code structure and legacy behavior.
2. Existing domain APIs should be reused when they fit the target boundaries.
3. If an existing API violates the target boundaries, refactor or replace it rather than preserving it through an adapter that leaks forbidden semantics.
4. Do not invent compatibility behavior to reduce implementation effort.
5. Do not ask for clarification for ordinary repository-level ambiguity. Make the smallest design decision that cleanly satisfies the proposal.
6. If an external dependency genuinely prevents completion, implement every independent part, make the failure explicit and typed, and report the precise blocker at the end. Do not silently skip work.

## Required implementation behavior

Work directly in the repository and implement the target architecture rather than simulating it.

In particular:

- Create and use `internal/agentruntime-go/runtimesetup/` as the only composition root joining Session, inference, and domain services.
- Refactor inference so it only receives provider-ready completion data.
- Move all domain-aware model-operation composition into runtime setup adapters.
- Introduce typed intent, typed overrides, canonical identities, provenance, diagnostics, and immutable prepared-operation data.
- Ensure Session Runtime remains the only execution coordinator after a command is accepted.
- Move chat Tool, Skill, MCP, approval, continuation, persistence, and automation ownership fully into the backend Session Runtime.
- Fully migrate the frontend chat path to the Session Runtime command, snapshot, and event model.
- Remove legacy direct frontend chat execution ownership rather than leaving an unused parallel path.
- Use the new SessionStore schema and do not migrate or import legacy conversations.
- Preserve management and preview capabilities only where they do not become execution authority.

Follow the proposal’s dependency boundaries literally. Do not solve a boundary problem by adding an import that the proposal forbids.

## Execution discipline

Before changing code:

- Inspect the existing Session Runtime, inference package, Wails wrappers, SessionStore, frontend chat flow, Agent APIs, Workspace APIs, Tool APIs, Skill runtime, MCP runtime, and attachment handling.
- Identify existing types and ports that can be reused without violating the target architecture.
- Identify all current direct frontend completion, tool, Skill, MCP, approval, continuation, and persistence call sites.

During implementation:

- Keep the refactor coherent across backend contracts, Wails bindings, generated or exposed frontend types, frontend callers, persistence projections, and tests.
- Remove obsolete chat-loop ownership rather than merely disconnecting it.
- Use typed contracts at public, persistence, domain, provider, and MCP boundaries.
- Keep runtime reads side-effect free.
- Keep operation preparation separate from command admission.
- Ensure any changed or new background work uses the common panic-safe launcher.
- Do not introduce repeated normal-path nil checks, repeated validation, or context polling prohibited by the proposal.
- Do not use unchecked type assertions on public, persistence, provider, MCP, or external-domain adaptation paths.
- Keep unrelated refactoring to a minimum, except where required to satisfy the target dependency graph.
- Do NOT go into circles about symlink behaviour, root exec and any fs ops in general. Liek todays runtime of attachments etc, anything that runs thorugh mapstore or llmtools is already symling and fs and cross paltform sfe. you have to defer all these tasks to these 2 entities only. given that most of them are already implemented in some way or other in either workspace or agent or attachment use that properly rather than you doing things.

## Required verification before completion

Do not consider the work complete until verification has been performed.

At minimum:

- Run the relevant Go formatter and Go test suites.
- Run repository-wide Go tests where feasible.
- Run frontend type checking, linting, and build or test commands available in the repository where feasible.
- Add or update focused tests for:
  - Intent and override precedence
  - Canonical collection override behavior
  - Stale Agent and Workspace selection handling
  - Immutable prepared model operations
  - Backend-only Tool binding and execution ownership
  - Wait, approval, continuation, and recovery behavior
  - No replay of uncertain external work after recovery
  - Snapshot reads that do not invoke domain resolution or provider I/O
  - SessionStore persistence and restore under the new schema
  - MCP App Session and Run scoping
  - Panic-safe failure transitions where practical
- Search the frontend for forbidden direct chat execution APIs and remove Session-chat use.
- Search imports to verify the required package boundaries.
- Search runtime code for raw background goroutine launches and route them through the safe launcher.
- Search for legacy conversation migration, `conversations_v2` import, and compatibility execution paths.

If a repository command cannot run because of an existing environmental issue, record the exact command, failure, and reason in the final response.

## Final response format

After implementation is complete, provide only:

1. A concise summary of completed changes.
2. The major architectural migrations performed.
3. Any remaining external blocker, if one exists.

Do not describe incomplete required work as complete. Do not claim compliance without verifying the relevant code paths.

## Intermediate Output requirements

### Code delivery format

- For every **new file**, provide the full file contents.
- For every **replacement file**, provide the full file contents.
- For every **existing modified file**, provide a unified diff against the exact repository path.
- Do not use placeholders such as `...`, “existing code unchanged”, or omitted method bodies for required implementation.
- Do not invent APIs not supported by the attached code. Inspect actual types and interfaces first.
- If an essential dependency is not attached, identify that exact missing path before relying on it.
- Do not ask nay user to complete or implement anything. you are implementing this properly in a step by step manner. The users task is only applying your patches and understanding your decisions given.

### End-of-response task list

End each output with exactly these sections as proper task lists:

- Implementation status
- Next steps

The task list must be concise, copy-pastable, and suitable for maintaining in a file. Also it must be properly complete.

After this give a diff for the proposals status sections so that things can be known there.
