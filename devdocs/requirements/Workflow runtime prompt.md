Implement the pending Workflow Runtime v3 and Session Runtime v3 backend tasks from `devdocs/requirements/workflow_runtime.md`, section `Implementation task list`. Work in the supplied FlexiGPT source tree. Start with functional coverage. Do not spend this pass on speculative guard code, repeated checks, cosmetic cleanup, or broad hardening.

Primary outcomes:

- Session Runtime v3 is a functional backend superset of normal Session v2 behavior.
- Workflow Runtime v3 executes every valid Workflow v1 feature with typed durable semantics.
- Session v2 and Session Store v2 remain unchanged and supported.
- `apiVersion: v1` remains the Workflow language version.
- `artifact.workflow.v1@1` remains the development runtime payload.
- Do not add a new persistence root.

Architecture:

- `internal/agentruntime-go/...` owns typed orchestration, `ProgramFrame`, `WorkItem`, `Attempt`, `Wait`, execution state, command idempotency, timers, recovery, and persistence-safe semantics.
- `internal/llmartifactory-go/...` owns declaration syntax, schema validity, static CEL scope, target relationships, and portable declaration validity.
- `internal/agentruntimesetup/...` owns declaration-to-runtime compilation, Artifact pinning, current binding preparation, provider plans, Tool binding, MCP discovery, Skill resolution, Workspace planning, attachment materialization, Service binding, and Activity binding.
- Session Runtime v3 must not import Artifact packages or domain setup packages.
- Session Runtime v3 must not reimplement provider configuration, Artifact resolution, Workspace file access, MCP discovery, Skill resolution, Tool execution, or attachment materialization.
- Do not add broad facade interfaces. Use concrete types or narrow typed domain ports only where a domain boundary requires one.
- Use `CanonicalValue` only for arbitrary JSON boundaries. Keep IDs, pins, kinds, limits, attempts, wait state, receipts, and errors as concrete types.
- Use lower-case error messages.

Implement in this order:

1. Finish all callable targets listed as pending, including Tool Artifact, MCP Tool, Service, Activity, nested Workflow, Agent, Model, static target sets, resolver-backed target sets, full Agent and Workspace and Model and instruction and Tool and Skill and MCP and attachment and message and port plans, and provider SDK Tool handling.
2. Finish effects and reliability listed as pending, including read and idempotent and keyed and non-idempotent contracts, stable idempotency keys, reconciliation, saga tracking, reverse compensation, compensation errors, cleanup allowance, typed effect state, and no automatic replay for uncertain non-idempotent work.
3. Finish Workflow structured behavior listed as pending, including catch and finally, machines, all parallel policies, foreach, repeat, durable Fork and Join, break, and continue.
4. Finish durable coordination listed as pending, including event triggers, schedule triggers, overlap and missed-occurrence policy, trigger state in Session v3, retention, consumption, channels, cursors, deduplication, human waits, completion waits, signals, and machine serialization.
5. Finish normal V2 backend capability in V3 listed as pending, including attachments, Agent, Workspace, Model, Go and provider and MCP and Skill Tools, Service, Activity, MCP resources and prompts and Apps and approvals, Skill sessions, streaming, usage, debugger, continuation, cancellation, retries, waits, recovery, uncertainty, and command idempotency.

Implementation rules:

- Implement each Workflow v1 feature fully in typed compiler and interpreter paths.
- Replace temporary non-implementation branches with typed runtime behavior in the same feature slice.
- Do not add no-op handling.
- Do not silently omit declaration fields.
- Do not create fallback chat text from arbitrary Workflow JSON.
- Persist `WorkItem` intent before binding.
- Resolve current dependencies and verify Artifact pins before dispatch.
- Persist semantic binding facts before dispatch.
- Persist an `Attempt` before external invocation.
- Persist confirmed output, receipt, failure, or uncertainty after dispatch.
- Use Runtime timers for durable timing.
- Cancel all runtime timers and goroutines in `Runtime.Close`.
- Update Workflow HLD documents truthfully when functional behavior changes.

FULL implementation and no partials. All features done. No tests. No excuses. Mark relevant items in the hlds complete to progress the task list.
