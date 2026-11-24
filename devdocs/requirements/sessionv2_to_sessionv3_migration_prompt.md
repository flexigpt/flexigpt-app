Complete every open backend task in
`devdocs/requirements/sessionv2_to_sessionv3_migration.md` in one substantial,
end-to-end delivery:

- `MIG-ARCH-003`
- `MIG-ARCH-004`
- `MIG-APP-005`

Work only on SessionV3 backend and setup code. Do not add tests, test fixtures,
test reports, validation work, validation-only hardening, frontend integration,
SessionV2 deletion, SessionV2 cleanup, old-session conversion, data migration,
release work, or version bumps. This is WIP. Do not change any declaration,
schema, Store, package, module, API, ProgramFrame, interpreter, or release
version identifier.

Move Workflow start and trigger preparation out of `SessionV3Wrapper` into one
setup-owned backend command service. Preserve the current caller-owned command
identities and perform duplicate receipt lookup before any preparation. The
service can read current Session defaults, use `workflowruntime.Admission`, and
submit only a prepared `ProgramSnapshot` to Runtime. Keep
`SessionV3Wrapper` transport-only. Runtime owns Runtime state, persistence,
scheduling, recovery, settlement, and projections. Runtime must not depend on
Artifact Store or setup packages. Artifact Store remains passive. Session and
Workflow remain orchestration owners. Model, Tool, MCP, Skill, Workspace,
Attachment, Service, and Activity domains must own their native behavior and
private resources.

Remove only V3-local duplication. Keep SessionV2 coexistence untouched. Make
`attachmentruntime.SessionV3Admitter` the one V3 attachment-admission owner.
Confirm the reported delegate before deleting it. Remove the known duplicate
stop-sequence clear, unreachable MCP read branch, repeated Workflow cleanup
assignment, and hardcoded Action kind. Reuse existing constants and typed
enums. Do not introduce hardcoded strings. Use const declarations where a
stable string is needed. Error strings in touched code must be lowercase.

Correct MCP App lifecycle behavior. App context staging must reject terminal or
cancelling Executions. App Tool execution must follow the same paused and
recovery-review restrictions as normal Tool control. Define and persist the
allowed App-context source-to-active-conversation-owner route. Do not scan
unrelated Executions implicitly. Use Runtime-owned logical time for durable
App-context updates.

Remove redundant behavior instead of adding another implementation. Reduce
repeated admission and validation work. Do not create a new validation project
or duplicate existing domain checks.

Update every affected status and source-evidence entry in
`workflow_runtime.md` and
`sessionv2_to_sessionv3_migration.md` in the same delivery. Mark a task
`Implemented` only when its complete listed behavior exists in source.

Follow the separately attached code delivery instructions. No tools are
attached. Return only contextual unified diffs in the required format. Produce
one large complete result, not incremental fragments.
