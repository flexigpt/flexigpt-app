# Agent SDK: top-down responsibility model

An agent SDK provides the execution layer for agents: model calls, tool calls, state transitions, events, policies, and completion.

Most systems separate into these layers:

```text
Application / API layer
  -> Orchestration layer
    -> Agent runtime
      -> Capabilities and tools
        -> Data systems, files, external services
```

Policy, observability, and identity apply across every layer.

## 1. Application layer

Owns product-facing concerns.

- User/API request handling
- Authentication and tenant identity
- UI and streaming transport
- Product permissions
- Domain records and business workflows
- Final presentation of agent output

Typical entities:

- `User`
- `Tenant`
- `Request`
- `Conversation`
- `BusinessRecord`
- `Permission`

The application owns the source of truth for business data.

## 2. Orchestration layer

Owns control flow across agents, functions, and humans.

- Select the next step
- Route work to an agent, tool, workflow node, or human
- Run steps sequentially or concurrently
- Handle branches, loops, retries, timeouts, and failure paths
- Pause for approval and resume later
- Coordinate multi-agent work
- Persist workflow progress

Typical entities:

- `Workflow`
- `Node` / `Step`
- `Edge` / `Route`
- `Task`
- `ApprovalRequest`
- `Checkpoint`
- `WorkflowRun`

Common organization patterns:

| Pattern                 | Structure                                                 |
| ----------------------- | --------------------------------------------------------- |
| Single agent loop       | One agent selects tools until completion                  |
| Manager and specialists | A manager calls specialist agents as bounded capabilities |
| Handoff                 | A router transfers ownership to a specialist agent        |
| Deterministic workflow  | Application-defined sequence of model and non-model steps |
| Graph workflow          | Typed nodes, edges, branching, fan-out, fan-in, loops     |
| Event workflow          | Components react to emitted events and external callbacks |

Use orchestration for business processes with required order, approvals, and auditability. Use an agent loop for open-ended reasoning within a bounded task.

## 3. Agent runtime

Owns the lifecycle of one agent invocation.

```text
prepare context
  -> call model
  -> receive proposed tool/action
  -> validate and authorize
  -> execute action
  -> record result/state
  -> call model again
  -> complete, fail, or pause
```

Runtime responsibilities:

- Assemble model-visible context
- Invoke model providers
- Parse tool calls and structured outputs
- Validate tool arguments
- Execute tools
- Add results to the agent history
- Enforce turn, token, time, concurrency, and cost limits
- Handle retries and cancellation
- Stream progress and model output
- Create trace and event records
- Return a final typed result or paused state

Typical entities:

- `AgentSpec`
- `AgentRun`
- `Message`
- `ModelRequest`
- `ModelResponse`
- `ToolCall`
- `ToolResult`
- `RunContext`
- `RunEvent`
- `RunResult`

The agent runtime owns execution. It usually does not own business data or long-term data ingestion pipelines.

## 4. Capability and tool layer

Owns controlled access to external actions and information.

A capability describes an operation available to an agent:

- Search documents
- Query a database
- Fetch a customer record
- Create a ticket
- Send an email
- Read a file
- Write a report
- Run a command in a sandbox
- Call an external API
- Ask a human for approval
- Delegate work to another agent

A well-defined tool contains:

| Field                     | Responsibility                                     |
| ------------------------- | -------------------------------------------------- |
| Name and description      | Helps the model choose the capability              |
| Input schema              | Defines valid arguments                            |
| Output schema             | Defines predictable result shape                   |
| Execution handler         | Performs the actual work                           |
| Authorization policy      | Determines allowed caller and resource scope       |
| Effect classification     | Read, write, destructive, financial, communication |
| Approval policy           | Defines whether human confirmation is required     |
| Timeout/retry/idempotency | Defines operational behavior                       |
| Audit metadata            | Captures who did what, where, and why              |

Tools are the boundary between model decisions and real-world side effects.

## 5. Data responsibility split

Data work usually has four distinct layers.

```text
Source systems
  -> ingestion and parsing
    -> storage and indexing
      -> retrieval/query tools
        -> agent context
```

| Layer            | Responsibility                                                                 |
| ---------------- | ------------------------------------------------------------------------------ |
| Source system    | Owns original documents, databases, SaaS records, repositories, object storage |
| Ingestion        | Connects, fetches, syncs, parses, chunks, enriches, and classifies content     |
| Storage/indexing | Stores documents, metadata, embeddings, search indexes, provenance             |
| Retrieval        | Searches and ranks relevant information for a request                          |
| Agent tool       | Exposes retrieval or domain queries to the agent                               |
| Agent runtime    | Adds selected results into model-visible context                               |

Typical data entities:

- `DataSource`
- `Connector`
- `Document`
- `DocumentVersion`
- `Chunk` / `Node`
- `Metadata`
- `Embedding`
- `Index`
- `Retriever`
- `Citation`
- `QueryResult`

### How SDKs organize data support

| SDK style                      | Data responsibility                                                  |
| ------------------------------ | -------------------------------------------------------------------- |
| Agent-runtime SDK              | Exposes data access through tools, MCP, or custom integrations       |
| RAG/data framework             | Provides connectors, parsing, indexing, retrieval, and query engines |
| Enterprise platform            | Adds managed connectors, identity-aware search, governance, and sync |
| Application-owned architecture | Keeps data services separate and gives agents narrow domain tools    |

LlamaIndex is data-centric: connectors, document processing, indexes, retrieval, and query engines are core concepts. LangGraph, OpenAI Agents SDK, Pydantic AI, and Strands primarily treat retrieval as a capability attached to an agent or workflow.

## 6. Files, artifacts, and workspaces

File handling usually has three separate responsibilities.

| Concept         | Responsibility                                                                   |
| --------------- | -------------------------------------------------------------------------------- |
| Source file     | Original user upload, repository file, object-storage object, or document source |
| Parsed document | Text, tables, images, metadata, chunks, and extracted structure                  |
| Artifact        | A durable output produced or consumed by an agent run                            |
| Workspace file  | Temporary file available during a task                                           |
| Sandbox         | Isolated environment for filesystem, shell, browser, or code execution           |

Typical flow:

```text
uploaded/source file
  -> parser/extractor
  -> structured document representation
  -> retrieval index or domain processing
  -> agent uses retrieval/tool
  -> agent produces artifact
```

Examples:

- A PDF becomes parsed text, tables, page references, and metadata.
- A repository becomes files, symbols, dependencies, and searchable code units.
- A generated report becomes an artifact with a URI, MIME type, version, owner, and provenance.
- A coding agent uses a temporary sandbox workspace and produces a patch or commit as its durable output.

The runtime should reference large files through artifact IDs or URIs. It should place only selected excerpts into model context.

## 7. Schema responsibility split

Schemas appear at several boundaries.

| Schema                     | Owner                   | Purpose                                                      |
| -------------------------- | ----------------------- | ------------------------------------------------------------ |
| Domain schema              | Application/data system | Customer, invoice, ticket, order, repository, policy records |
| Source/document schema     | Ingestion system        | Parsed document fields, metadata, tables, citations          |
| Tool input schema          | Agent SDK               | Valid arguments the model may send                           |
| Tool result schema         | Tool layer              | Structured result returned after execution                   |
| Agent output schema        | Agent SDK/application   | Expected final result for downstream code                    |
| State schema               | Runtime/workflow        | Allowed durable run or workflow fields                       |
| Event schema               | Runtime/orchestration   | Streaming, progress, errors, approvals, state updates        |
| Agent specification schema | SDK                     | Declarative model, instructions, tools, policies, outputs    |

Schemas support three jobs:

- Guide model behavior
- Validate boundaries before execution
- Give downstream systems stable contracts

The application owns domain meaning. The agent SDK owns model-facing action and output contracts.

## 8. State, memory, and source-of-truth split

| Data type            | Owner                   | Scope                            |
| -------------------- | ----------------------- | -------------------------------- |
| Run state            | Agent runtime           | One invocation                   |
| Session/thread state | Runtime/session service | One conversation or task         |
| Workflow state       | Orchestrator            | One long-running process         |
| Checkpoint           | Durable runtime         | Resume after failure or approval |
| Long-term memory     | Memory service          | Across sessions                  |
| Retrieval index      | Data system             | Shared knowledge                 |
| Business data        | Application database    | Authoritative domain state       |

Examples:

- “The user approved this payment” belongs in workflow state and an auditable business record.
- “The user prefers concise answers” belongs in scoped long-term memory.
- “The current tool call is awaiting approval” belongs in run/checkpoint state.
- “The customer’s current invoice balance” belongs in the billing system, accessed through a tool.

## 9. Cross-cutting platform capabilities

These apply across the runtime, workflow, tools, data, and files.

### Identity and policy

- User and tenant identity propagation
- Tool-level authorization
- Read/write permissions
- Resource scoping
- Human approval
- Secret and credential mediation
- Data classification and redaction
- Retention controls

### Observability

- Agent-run traces
- Model-call spans
- Tool-call spans
- Workflow routing events
- Token, latency, and cost metrics
- Artifact and data provenance
- Errors, retries, approvals, and policy decisions

### Evaluation

- Tool selection correctness
- Output schema validity
- Retrieval relevance and citation quality
- Task completion
- Policy compliance
- Cost and latency
- Regression datasets and trace replay

## Core entity model

A broadly useful agent SDK entity set looks like this:

```text
AgentSpec
  -> AgentRun
    -> Messages
    -> ToolCalls / ToolResults
    -> Events
    -> State
    -> Checkpoint
    -> RunResult

Workflow
  -> Nodes / Edges
  -> WorkflowRun
  -> ApprovalRequest
  -> WorkflowState

Capability / ToolSpec
  -> InputSchema
  -> OutputSchema
  -> Policy
  -> Executor

Artifact
  -> SourceFile / WorkspaceFile
  -> URI / Version / Provenance

DataSource
  -> Connector
  -> Document
  -> Chunk
  -> Index
  -> Retriever
  -> Citation
```

## Practical ownership model

| Concern                                        | Primary owner                 |
| ---------------------------------------------- | ----------------------------- |
| Agent instructions, model selection, tool loop | Agent runtime                 |
| Multi-step process and routing                 | Orchestrator/workflow runtime |
| Tool definitions and side effects              | Capability/tool layer         |
| Business records and permissions               | Application/domain services   |
| Source ingestion and parsing                   | Data/RAG pipeline             |
| Indexes, search, and retrieval                 | Data platform                 |
| Temporary file operations                      | Workspace/sandbox             |
| Durable generated files                        | Artifact service              |
| Output/tool/event schemas                      | SDK boundary layer            |
| Audit, tracing, evaluation                     | Platform operations layer     |

## The common architecture in practice

```text
Application
  owns users, permissions, APIs, domain records

Workflow
  owns sequence, routing, approval, retries, checkpoints

Agent runtime
  owns model loop, context assembly, tool execution, output

Tools
  own controlled read/write operations

Data platform
  owns sources, ingestion, parsing, indexes, retrieval

Workspace/artifacts
  own temporary execution files and durable generated outputs

Observability/policy
  govern all layers
```

This separation gives agents flexibility within a task while keeping business state, data pipelines, files, permissions, and high-impact actions under explicit system control.

## Entity-first SDK mapping

- `AgentSpec`
  - Current form: `assistantpreset.Spec.AssistantPreset` is the agent specification seed.
    - Model: `StartingModelPresetRef`
    - Instructions/start state: `StartingText`, `StartingIncludeModelSystemPrompt`
    - Capabilities: `StartingToolSelections`, `StartingSkillSelections`, `StartingMCPContext`
  - Supporting spec entities: `ProviderPreset`, `ModelPreset`, `ToolSelection`, `ArtifactSkillSelection`, `MCPConversationContext`, `WorkspaceSelection`.
  - Gap:
    - No single resolved `AgentSpec` object persisted for execution.
    - No explicit agent identity/version, runtime limits, context policy, output contract, or default execution policy.
  - Need: a resolved `AgentSpec` projection assembled from the existing preset and selected capability references.

- `Thread` / `Conversation`
  - Current form: `conversation.Spec.Conversation`.
  - Current responsibility:
    - Durable transcript.
    - Turn history.
    - Model inputs/outputs.
    - Attachments.
    - Tool choices.
    - MCP mappings/context.
    - Workspace selection/usage.
    - FTS search.
  - Gap:
    - No explicit thread-level ownership, retention, revision, run index, or audit metadata.
  - Need: retain `Conversation` as the SDK `Thread`; add thread metadata only when required.

- `Message` / `Turn`
  - Current form: `conversation.Spec.ConversationMessage`.
  - Current responsibility:
    - User and assistant turns.
    - Persisted `InputUnion` and `OutputUnion`.
    - Model configuration per turn.
    - Attachments, skills, tools, MCP context, workspace context.
  - Gap:
    - Tool-call/result correlation and execution status are distributed across provider unions and MCP-specific fields.
  - Need: normalize a run-facing message/event view over existing persisted turn data.

- `AgentRun`
  - Current form: one `inferencewrapper.Spec.CompletionRequest` and `CompletionResponse`, usually associated with one conversation turn.
  - Current responsibility:
    - Resolve model preset.
    - Hydrate attachments, workspace context, skills, MCP context, and tool choices.
    - Invoke provider.
    - Stream text/thinking.
    - Return provider outputs and hydrated input state.
  - Gap:
    - No durable run record or terminal lifecycle.
    - No explicit `running`, `paused`, `waiting_approval`, `completed`, `failed`, `cancelled` states.
    - No run-owned event log, limits, retry count, cost aggregate, or result record.
  - Need: make the existing completion invocation a persisted `AgentRun`.

- `Agent runtime loop`
  - Current form:
    - `ProviderSetAPI.FetchCompletion` performs context assembly and one model call.
    - `ToolRuntime.InvokeTool`, `MCP.ToolBridge`, and `SkillRuntime.InvokeSkillTool` execute capabilities.
  - Current responsibility:
    - Model calling exists.
    - Tool execution exists.
    - Capability hydration exists.
  - Gap:
    - No generic loop joining them:
      - model output
      - tool-call extraction
      - authorization/approval
      - tool execution
      - tool-result insertion
      - next model call
      - terminal result
  - Need: an `AgentRuntime.Run` state machine over the existing provider and capability runtimes.

- `ModelRequest` / `ModelResponse`
  - Current form: `inferenceSpec.FetchCompletionRequest` and `FetchCompletionResponse`.
  - Current responsibility:
    - Multi-provider calls.
    - Capability resolution from model presets.
    - Streaming.
    - Provider debugging.
    - Native provider output preservation.
  - Gap:
    - Model calls lack a durable parent `AgentRun` record and normalized span/event records.
  - Need: retain inference-go as the model execution layer; wrap calls in run-scoped records.

- `RunContext`
  - Current form:
    - Go `context.Context`.
    - `CompletionRequestBody`.
    - Conversation current turn.
    - Workspace/MCP/skill session fields.
  - Current responsibility:
    - Cancellation/deadlines.
    - Provider/model selection.
    - Capability selections.
  - Gap:
    - No unified principal, tenant, run ID, thread ID, policy scope, workspace scope, budget, or trace context.
  - Need: `RunContext` composed from current request/conversation state plus identity and policy fields.

- `ToolSpec` / `Capability`
  - Current forms:
    - `tool.Spec.Tool` for Go, HTTP, and provider SDK tools.
    - `runtime.MCPToolCapability` for discovered MCP tools.
    - Agent Skills tool definitions from `agentskills-go`.
  - Current responsibility:
    - Versioned tool bundles.
    - Input schema.
    - Tool descriptions.
    - Enablement.
    - Built-in/user distinction.
    - Go and HTTP implementations.
    - MCP dynamic discovery schemas and metadata.
  - Gap:
    - Tool contracts differ across local tools, MCP tools, and skills.
    - Regular tools have no required output schema.
    - Effect class, idempotency, retry policy, caller authorization, and audit requirements are absent from the common tool model.
  - Need: a common SDK-level `ToolSpec` adapter over existing tool, MCP, and skill definitions.

- `ToolCall`
  - Current forms:
    - Provider `FunctionToolCall` / `CustomToolCall` output unions.
    - `runtime.InvokeMCPToolRequestBody`.
    - `skillruntime.InvokeSkillToolRequest`.
  - Current responsibility:
    - Provider tool-call payloads can be preserved in conversation history.
    - MCP mappings bind provider tool names to server/tool/digest/policy.
  - Gap:
    - No common `ToolCall` entity with run ID, call ID, capability identity, attempt number, status, timestamps, and approval state.
  - Need: normalize provider and user-originated invocations into one run-scoped call record.

- `ToolResult`
  - Current forms:
    - `toolruntime.InvokeToolResponseBody`.
    - `runtime.InvokeMCPToolResponseBody`.
    - `skillruntime.InvokeSkillToolResponseBody`.
    - Provider tool-output unions.
  - Current responsibility:
    - Text, file, image, structured MCP content, error state, and MCP provenance are supported.
  - Gap:
    - No common persisted tool-result envelope.
    - No guaranteed insertion of tool results into the next model input by a runtime loop.
  - Need: `ToolResult` with normalized content, error, artifacts, provenance, and model-replay representation.

- Local tool execution
  - Current form: `toolruntime.ToolRuntime`.
  - Current responsibility:
    - Executes persisted Go and HTTP tool definitions.
    - Enforces tool and bundle enablement.
    - Applies timeout overrides.
    - Supports HTTP templating, secret substitution, response classification.
  - Gap:
    - HTTP argument JSON is parsed for templating; schema validation is not enforced before execution.
    - `AutoExecReco` is advisory metadata.
    - Caller/resource authorization is absent.
    - Secret values arrive through invocation options.
  - Need: runtime-owned schema validation, authorization, secret resolution, approval, idempotency, and audit wrapping.

- MCP runtime
  - Current form: `MCPRuntimeManager`, `ToolBridge`, `ApprovalManager`, `mcpbundle.API`.
  - Current responsibility:
    - Server lifecycle: connect, disconnect, refresh, invalidate.
    - stdio and streamable HTTP transport.
    - OAuth and API-key flows.
    - Secret references and redaction.
    - Capability discovery.
    - Tool/resource/prompt access.
    - Tool risk classification.
    - Policy composition.
    - Approval evaluation and one-time approval tokens.
    - Provider-tool mapping and invocation provenance.
  - Gap:
    - Approval requests and remembered decisions are process-local.
    - MCP lifecycle remains separate from generic agent-run lifecycle.
    - MCP notifications are logs/callbacks rather than unified run events.
  - Need: expose MCP as a first-class implementation of the common capability and approval contracts.

- `ApprovalRequest`
  - Current form: MCP `pendingApproval` in `ApprovalManager`.
  - Current responsibility:
    - Allow once, allow always for session, deny once, deny always for session.
    - Expiring approval tokens.
    - Binding to server, tool, digest, risk, source, arguments, app instance.
  - Gap:
    - No persistence across process restart.
    - No generic approval entity for HTTP, Go, shell, git, file-write, or financial actions.
    - No run pause/resume linkage.
  - Need: durable `ApprovalRequest` and `ApprovalDecision`, usable by every capability type.

- `Skill`
  - Current form:
    - Artifact-backed `agent.skill`.
    - `SkillRuntime`.
    - `ArtifactRouter`.
    - Workspace and skill-bundle adapters.
  - Current responsibility:
    - Skill discovery from `SKILL.md`.
    - Source generation/digest verification.
    - Runtime registration.
    - Session allow-lists and active skills.
    - Prompt generation.
    - Skill rendering.
    - Resource reading and optional script execution.
  - Gap:
    - Skill session lifecycle is independent from `AgentRun`.
    - Active-skill state requires caller-managed `SkillSessionID`.
  - Need: attach the existing skill session to the future run/session lifecycle.

- `Workspace`
  - Current form: `workspace.API`, `WorkspaceCollectionV1`, workspace artifact adapters.
  - Current responsibility:
    - Filesystem workspace registration.
    - Project context discovery.
    - `AGENTS.md`, `CLAUDE.md`, `README.md`, markdown context handling.
    - Workspace skill discovery.
    - Explicit artifact selection.
    - Context composition budgets and truncation diagnostics.
    - Runtime-disabled artifact policy.
  - Gap:
    - Workspace serves as a trusted source root and context source.
    - No run-owned writable workspace or disposable task filesystem.
  - Need: add `RunWorkspace` / sandbox lifecycle while retaining current workspace as the source/project layer.

- `Artifact`
  - Current form: `artifactstore.Artifact`.
  - Current responsibility:
    - Durable source-bound resource identity.
    - Artifact state: available, missing, invalid, incompatible.
    - Source binding, definition digest, local enablement, diagnostics, revision.
    - Managed Skill and MCP server/policy installation records.
  - Gap:
    - Generated reports, patches, exports, files, and model-produced outputs have no corresponding durable artifact entity.
  - Need: a separate `RunArtifact` model with URI, MIME type, bytes/size, owner, producer run, version, provenance, and retention.

- Attachments / input files
  - Current form: `attachment.Attachment`, `ContentBlock`, `FileRef`, `ImageRef`, `URLRef`.
  - Current responsibility:
    - Local file, image, PDF, and URL ingestion into provider content blocks.
    - MIME detection.
    - Text/image/file conversion.
    - URL fetching.
    - Snapshot modification detection.
  - Gap:
    - Large content commonly becomes prompt/base64 content.
    - No durable upload ID, object storage reference, extraction record, citation, or file-security pipeline.
  - Need: file/upload service and artifact references for large input/output data.

- `DataSource`
  - Current form: `artifactstore.Source`.
  - Current responsibility:
    - Filesystem source.
    - Embedded source.
    - Managed writable source.
    - Source snapshots, generations, bounded reads, source confirmation.
    - Source configuration isolation from public APIs.
  - Gap:
    - No SaaS, database, repository-hosting, object-store, or enterprise connector family in supplied code.
  - Need: connector adapters under the existing `Source.Adapter` model.

- `Document` / `Definition` / `Catalog`
  - Current form:
    - `catalog.Occurrence`
    - `definition.Definition`
    - `catalog.Snapshot`
    - discovery decoders and refresh service
  - Current responsibility:
    - Source scanning.
    - Content digests.
    - Canonical schema validation.
    - Artifact decoding.
    - Catalog freshness.
    - Source-to-artifact reconciliation.
    - Diagnostics and provenance.
  - Gap:
    - No generic parsed-document, chunk, embedding, or search-ranking model.
  - Need: extend the current source/catalog path with document extraction and retrieval entities.

- `Retriever` / `Citation`
  - Current form:
    - Conversation FTS search.
    - Workspace exact artifact selection.
    - MCP resource selection.
  - Gap:
    - No semantic retrieval, chunk ranking, embedding index, citation references, or retrieval tool.
  - Need: `Retriever.Query` capability returning excerpts, scores, document IDs, locations, and citations.

- `Checkpoint`
  - Current forms:
    - Catalog/source generations and revisions.
    - Hydration markers.
    - Conversation persistence.
    - MCP discovery snapshot digest.
  - Current responsibility:
    - Strong consistency and restart convergence for artifact topology, sources, and catalogs.
  - Gap:
    - No checkpoint for an in-progress agent/tool/approval/workflow execution.
  - Need: run checkpoint with current model turn, pending tool calls, approval state, retry state, and resumable input history.

- `Workflow` / orchestration
  - Current forms:
    - Workspace provision flow.
    - Artifact refresh/reconciliation flow.
    - Built-in topology hydration flow.
    - Managed package publish/remove flow.
    - MCP connect/refresh/invalidation flow.
    - Async rebuild and cleanup loops.
  - Current responsibility:
    - Deterministic multi-step orchestration already exists in several domain services.
    - Concurrency controls, optimistic revisions, cleanup/compensation, and retryable convergence are strong.
  - Gap:
    - No generic workflow entity, workflow run, nodes, edges, task queue, branch state, human task, or durable resume.
  - Need: generalize only when product workflows need reusable graph/process execution; current domain orchestrators can remain domain-owned.

- `WorkflowRun`
  - Current form: implicit execution inside service methods such as refresh, hydration, provisioning, and MCP connect.
  - Gap:
    - No durable run ID, step history, status, retry schedule, or resume token.
  - Need: `WorkflowRun` around long-running or approval-gated domain processes.

- Identity / principal
  - Current form:
    - Protected installer context.
    - Root mutation policy.
    - Secret references.
    - MCP server artifact identity.
  - Current responsibility:
    - Protects built-in topology and MCP installation state.
    - Scopes MCP secrets to server artifact identity.
  - Gap:
    - No user principal, tenant, role, permission, or resource scope propagated through completions and generic tool calls.
  - Need: `Principal` and `AuthorizationContext` in every run and capability invocation.

- Policy
  - Current form:
    - MCP policy composition.
    - Tool enablement.
    - Root protection.
    - Workspace runtime policy.
    - Secret-target restrictions.
    - MCP app visibility and app-call policy.
  - Current responsibility:
    - Strong policy model for MCP and artifact topology.
  - Gap:
    - No unified policy engine across Go/HTTP tools, filesystem tools, shell tools, git tools, attachments, and model output.
  - Need: common policy decision interface: allow, deny, require approval, redact, constrain.

- State / memory
  - Current form:
    - Conversation history.
    - Skill sessions.
    - MCP sessions.
    - Workspace selections.
    - Settings overlays.
  - Current responsibility:
    - Short-term conversation memory and process-local runtime sessions.
  - Gap:
    - No long-term user/project/agent memory model.
    - No memory write policy, scope, expiration, or retrieval.
  - Need: optional memory service after run identity and principal scopes exist.

- `RunEvent`
  - Current forms:
    - Provider streaming callbacks.
    - MCP notifications.
    - `slog` records.
    - diagnostics.
  - Current responsibility:
    - Text/thinking streams and operational notifications exist.
  - Gap:
    - No unified typed event stream for run transitions.
  - Need:
    - `RunStarted`
    - `ContextHydrated`
    - `ModelStarted`
    - `ModelDelta`
    - `ToolProposed`
    - `ApprovalRequired`
    - `ToolCompleted`
    - `RunCompleted`
    - `RunFailed`

- Observability / audit
  - Current form:
    - `slog`.
    - HTTP completion debugger.
    - HTTP tool metadata.
    - MCP tool provenance.
    - Artifact/catalog diagnostics.
  - Current responsibility:
    - Good component-level diagnostics and MCP provenance.
  - Gap:
    - No unified trace tree across conversation, model call, tool call, approval, workflow, and generated artifact.
    - No durable audit record or aggregate run token/cost/latency record.
  - Need: run-scoped tracing, metrics, audit log, and trace replay format.

- Evaluation
  - Current form: substantial unit coverage and deterministic validation.
  - Gap:
    - No agent task datasets, tool-selection scoring, retrieval quality scoring, trace replay, policy regression suite, or cost/latency evaluation.
  - Need: evaluation layer once `AgentRun` traces and normalized `ToolCall` records exist.

## Practical interpretation

- `Conversation` already serves as `Thread`.
- `AssistantPreset` already serves as the source form of `AgentSpec`.
- `CompletionRequest` already serves as the source form of a one-turn `AgentRun`.
- Refresh, hydration, provisioning, and MCP lifecycle already serve as domain orchestration.
- `ToolStore`, `ToolRuntime`, `SkillRuntime`, and MCP runtime already provide a substantial capability layer.
- The main missing SDK layer is the durable, generic `AgentRun` loop that coordinates existing model, tool, approval, state, event, and artifact capabilities.
