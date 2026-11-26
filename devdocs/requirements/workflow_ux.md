# Session UX proposal for Agents and Workflows

## 1. Terminology and recommended direction

**One tab represents one Session.** A Session contains its messages, configuration, Workflow runs, results, and pending requests.

“Chat” is the page name. “Conversation” is mostly a synonym in the existing source. Neither term represents an additional user-facing object in this proposal. The functional term below is Session.

| Term              | Defined meaning                                                                                                                             |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------- |
| Session           | The persistent unit displayed in one tab. A new tab remains a Session draft until the first accepted action creates the persistent Session. |
| Agent             | A reusable reasoning configuration, including instructions and selected capabilities.                                                       |
| Workflow          | A declared process that coordinates Agents, Models, Tools, and other operations.                                                            |
| Run               | An execution of Agent behavior or a Workflow. A Run belongs to a Session. A normal Agent Run can remain open between messages.              |
| Participant       | An Agent or Model role within a Workflow. The same Agent can serve several distinct roles.                                                  |
| Session responder | The Agent currently receiving ordinary Session messages. Background Workflow participants are not automatically Session responders.         |

A Workflow Run and its child work remain in the same Session. Additional participants do not automatically create tabs.

### Recommended direction

Enhance the existing composer and execution controls into one shared experience for Agents and Workflows.

The proposed changes are:

- Extend the Agent selector to include Workflows.
- Retain the existing Model, parameter, Tool, Skill, MCP, Workspace, and attachment controls.
- Extend those controls to show Workflow contributions, shared defaults, and scoped consumer overrides.
- Enhance current Tool activity and approval handling into one shared runtime review experience.
- Add compact Workflow progress and explicit final results.

A valid Model remains compulsory. This experience supports LLM-oriented Workflows, including Workflows that inherit the Model from the composer.

The result is an extension of the current Session UX, rather than a separate Workflow application.

## 2. Existing foundation and enhancements

| Area                 | Current foundation                                                                                                 | Proposed enhancement                                                                                      |
| -------------------- | ------------------------------------------------------------------------------------------------------------------ | --------------------------------------------------------------------------------------------------------- |
| Selection            | Searchable Agent selector, default Agent, availability, and details.                                               | Agents and Workflows in the same selector.                                                                |
| Configuration        | Inherited Agent contributions remain visible in existing controls. Consumer choices can override inherited values. | Apply the same model to Workflow entry settings and participant settings, with explicit scope.            |
| Input                | Message text, attachments, templates, and structured Tool options.                                                 | Add typed Workflow input within the same composer experience.                                             |
| Active work          | Tool activity chips, manual execution, retry, skip, error submission, and continuation.                            | Extend the active-work area to cover Workflow steps and pending requests.                                 |
| Approvals            | Existing wait modal with MCP details, allowed resolutions, structured input, and expiry.                           | Enhance the same review flow to handle multiple requests, human work, and recovery.                       |
| Progress and history | Streams, partial output, Tool results, message details, tabs, search, reopening, and export.                       | Add participant attribution, concurrent activity, explicit Workflow outcomes, and multiple Run awareness. |

These foundations are present in the supplied frontend source. Workflow execution, participant-scoped configuration, and the complete review experience are proposed enhancements, not existing frontend behavior.

## 3. Shared selection and composer behavior

### 3.1 Agent/Workflow selection

The current Agent selector becomes a grouped selector:

- Agents, including Base and the configured default.
- Workflows.

Selecting an Agent prepares the existing message flow.

Selecting a Workflow prepares a new Workflow Run. Selection alone neither starts execution nor activates declared triggers.

The configured default Agent remains the starting selection for a new Session. Previous Workflow use does not silently become the default action for the next Session.

### 3.2 The composer remains the configuration entry point

Selecting a Workflow retains the current configuration controls.

The composer continues to answer:

- Which Model and parameters are effective?
- Which Tools, Skills, and MCP capabilities are available?
- Which Workspace and attachments are included?
- Which values come from defaults?
- Which values have explicit consumer overrides?

Workflow selection changes the meaning and scope of the displayed configuration. It does not replace configuration with a read-only process description.

For a simple Workflow with inherited settings, the experience remains close to the current Agent experience.

For a Workflow with different participant configurations, the same controls expose those differences through their existing selected-item and details patterns.

### Decision and tradeoff

A shared selector and composer are preferred over separate Workflow launch and configuration screens.

A separate launch screen could present every process setting at once. It would also duplicate the existing Model and capability controls. The shared composer preserves familiar defaults and places additional control inside existing menus and options.

## 4. Defaults, Workflow settings, and consumer overrides

### 4.1 Run entry configuration

Run entry configuration means the execution settings supplied when a Workflow Run starts:

- Model selection and Model parameters.
- Instructions.
- Tools.
- Skills and their active state.
- MCP selections.
- Workspace contributions.
- Applicable history and context settings.

Run entry configuration is separate from Workflow business input.

For example, a release identifier is business input. A selected Model and an additional review Skill are execution configuration.

Both belong to the same preparation experience, but remain distinguishable.

### 4.2 Default inheritance

The normal case requires no manual configuration.

A participant can receive its configuration from:

- Explicit Workflow call settings.
- The selected Agent and its contributions.
- Workflow entry defaults.
- Session defaults.
- Application defaults.

A Workflow that supplies no Model or Model settings uses the applicable composer defaults.

Selected shared Skills, Tools, and MCP capabilities are also available to eligible LLM participants through the shared configuration. Their presence means availability, not guaranteed use.

The Workflow therefore remains understandable as several participants working with shared defaults, with specific configurations where required.

### 4.3 Consumer override precedence

Explicit consumer changes take priority over inherited configurable values.

The proposed precedence is:

| Priority | Configuration source                                                           |
| -------- | ------------------------------------------------------------------------------ |
| Highest  | Explicit consumer override for a specific participant or call                  |
| 2        | Explicit shared consumer override for the Run                                  |
| 3        | Workflow participant/call configuration, including its selected Agent baseline |
| 4        | Workflow entry defaults                                                        |
| 5        | Session defaults                                                               |
| Lowest   | Application defaults                                                           |

Workflow call settings take precedence over the Agent baseline that the call uses.

Required interfaces, authorization rules, and fixed target constraints are boundaries on configuration. Those constraints remain authoritative rather than acting as another default layer.

Two distinctions are essential:

- A displayed inherited value remains a default until explicitly changed.
- An explicit consumer choice remains an override until reset.

Selecting a Workflow preserves applicable consumer choices. It does not silently convert every displayed Session default into a shared override.

Fallback resolves an omitted value. An invalid explicit selection produces an actionable problem rather than silent substitution.

### 4.4 Default edit scope

In Workflow preparation, ordinary composer edits apply as shared overrides for the new Run.

Examples include:

- Selecting another Model.
- Setting a temperature.
- Adding a Skill.
- Removing an inherited optional Tool.
- Changing an applicable MCP selection.

The affected scope is stated in the selected-item area: “Shared for this Run”.

The enhanced existing options modal provides narrower control:

- Shared Run settings.
- A specific participant or call.

Participant-level changes apply to the named role or call, not to the Agent definition globally. This matters when one Agent is used for several different tasks.

Fixed values remain visible with their reason. The scope summary identifies affected participants and any fixed exceptions. Incompatible overrides within the selected scope require correction before start.

### 4.5 Existing controls show the effective configuration

| Existing control | Workflow enhancement                                                                                                                                         |
| ---------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Model            | Show the effective Model, or “Multiple Models” when participant Models differ. Opening the control shows the shared setting and participant selections.      |
| Model parameters | Show the effective value, or “Multiple values” when necessary. An ordinary edit creates a shared Run override.                                               |
| Tools            | Show shared LLM capabilities and participant contributions. Explicit Workflow Tool operations remain distinguishable from Tools available for LLM selection. |
| Skills           | Show shared Skills and participant-specific Skills. Preserve the current enabled, active, and instruction-only distinctions.                                 |
| MCP              | Show selected servers, their contributions, and the participants or steps that use them.                                                                     |
| Workspace        | Show the selected contribution source and its applicable scope.                                                                                              |
| History          | Identify the eligible LLM context affected by the selection.                                                                                                 |

The current source labels, such as Agent and Session, extend to include Workflow and participant/call ownership.

Useful reset actions follow the same pattern:

- Use inherited value.
- Use Workflow defaults.
- Use Agent defaults.
- Remove this Run override.

A reset clears the consumer override. It does not remove the underlying declaration.

### 4.6 Collection behavior

Tools, Skills, MCP selections, and instruction sources require collection semantics rather than a single scalar precedence rule.

The proposed behavior retains the existing approach:

- Contributions combine by identity.
- A more-specific configuration of the same item takes precedence.
- Explicit exclusions remove inherited optional contributions.
- Full replacement remains an advanced choice.
- Required process operations remain distinct from optional LLM capabilities.

For example, removing a Tool from the set available to an Agent does not remove an explicit Workflow step that calls that Tool.

The same distinction applies to automatic execution. A Tool’s automatic-execution setting governs eligible Agent-generated calls. Workflow step scheduling follows the Workflow.

### 4.7 Model requirement

A valid Model is compulsory for this experience.

The requirement can be satisfied by:

- A Workflow-selected Model.
- An Agent-selected Model.
- A consumer-selected Model.
- An applicable Session or application default.

A Workflow with several declared Models satisfies the requirement through its effective Model configuration. A second user selection is unnecessary when those Models are already valid.

Start remains unavailable when a required effective Model is missing, unavailable, or incompatible.

Purely non-LLM Workflows are outside this product scope. Deterministic operations, Tools, Services, and Activities remain valid parts of an LLM-oriented Workflow.

This is a product-scope decision for the proposed UX. The declaration language itself has a wider scope. [6]

### 4.8 Configuration examples

| Situation                                        | Effective behavior                                                                             |
| ------------------------------------------------ | ---------------------------------------------------------------------------------------------- |
| Workflow supplies no Model or Skills             | The composer Model and applicable shared Skills are used.                                      |
| Workflow supplies Researcher and Reviewer Models | Both remain visible and effective without further changes.                                     |
| A shared consumer Model override is selected     | The selected Model applies to configurable participants in that Run.                           |
| A Reviewer-specific override is then added       | Reviewer uses its specific override; other applicable participants retain the shared override. |

### Decision and tradeoff

Shared overrides are preferred over requiring participant-by-participant editing for common changes.

Participant exceptions remain available inside options. This adds some scope information to configuration review, but preserves both convenience and deliberate control.

### 4.9 Configuration availability and context transfer

Shared configuration does not imply unrestricted sharing of Session contents.

- A selected Skill makes the Skill available to eligible LLM participants.
- A selected Tool makes the Tool available for eligible calls.
- History selection applies only to accepted LLM context.
- Attachments and Workspace content follow their declared or supported context destinations.
- MCP App context follows its approved ownership and continuation route.

Capability availability and message-content transfer remain separate facts.

This preserves the HLD’s explicit context boundary while allowing useful shared defaults. [6]

## 5. Workflow input and start behavior

### 5.1 Input within the same composer

The Workflow input contract determines the entry method.

| Input shape                                 | Proposed experience                                                        |
| ------------------------------------------- | -------------------------------------------------------------------------- |
| String input                                | Existing text-entry interaction                                            |
| One main text field with additional options | Labelled primary text entry, with additional fields in options             |
| Structured object                           | Required fields available directly or through the input section of options |
| Complex JSON                                | Full JSON entry with schema information                                    |
| Valid empty input                           | Start without a dummy message                                              |

The same enhanced options experience contains Workflow input and advanced configuration. A separate launch wizard is unnecessary.

The composer clearly separates:

- Input values for the process.
- Execution configuration for the participants.

A business field named `model` does not automatically become a Model override.

### 5.2 Input defaults and control

- Valid declared defaults initialize the draft.
- Required values without defaults remain incomplete.
- Examples are inserted only through an explicit action.
- Omission, zero, `false`, empty collections, and `null` retain their meanings.
- A complete JSON alternative remains available when field-based entry cannot represent the contract.
- Validation errors identify the affected value and preserve the draft.

The current structured-input pattern is enhanced for this purpose, including the Workflow schema’s supported value types.

### 5.3 Attachments

Attachments continue through the existing attachment control.

The launch review identifies the accepted destination, such as a declared input field or an eligible entry context.

An attachment without an accepted destination remains visibly unresolved. The attachment control supports correction or retention for a later message. Start does not silently discard or broadcast the attachment.

### 5.4 Start and acceptance

The primary action becomes “Start workflow”.

The experience distinguishes:

- Preparing.
- Start accepted.
- Running.
- Completed or another terminal outcome.

A failed start preserves input and overrides.

Once accepted:

- The Run retains its submitted input and effective prepared configuration.
- The composer returns to ordinary message entry for the Session responder.
- The Workflow remains accessible through Session activity.
- Another invocation requires “Run again” or another explicit Workflow start.

Run-specific overrides remain attached to the Run. They do not silently become Session defaults.

Later composer changes affect the next eligible message or new Run, rather than rewriting prepared calls.

### Decision and tradeoff

Returning to message entry after acceptance is preferred over leaving the Workflow launch action armed.

Repeated execution requires one additional deliberate action. This prevents a follow-up message from accidentally starting the same external process again.

## 6. Shared Session activity

### 6.1 Enhancement of current active-work controls

The current Run status, Tool activity chips, and pending-Tool controls become one shared Session activity experience.

This experience handles both Agent and Workflow work.

The compact activity area shows:

- Selected Run.
- Current activity.
- Running work count.
- Pending-action count.
- Relevant quick actions.

It remains reachable when the main composer is hidden.

Opening an activity item expands the same shared details and review experience. Agent Tool work and Workflow work use one collection of runtime items, rather than separate queues.

Existing quick actions can remain for simple eligible Tool work. Those actions operate on the same runtime item shown in detailed review.

### 6.2 Progress information

| Runtime behavior           | Default summary                      | Expanded information                               |
| -------------------------- | ------------------------------------ | -------------------------------------------------- |
| Ordered work               | Current activity                     | Earlier steps, outputs, and skipped work           |
| Parallel work              | Running and settled branch counts    | Separate branch activity                           |
| Collection processing      | Completed items and captured total   | Item results and failures                          |
| Repetition                 | Current iteration and relevant limit | Earlier iterations and exhaustion result           |
| State machine              | Current state and awaited condition  | State transitions                                  |
| Nested or independent work | Linked child status                  | Ownership, lifetime, and cancellation relationship |
| Retry                      | Attempt and next retry time          | Earlier attempts and retained error                |
| Recovery or compensation   | Current recovery activity            | Original effect and cleanup outcome                |

A fixed percentage is shown only when the total work is known.

An idle Agent Run is shown as ready for a message. A long-lived open Run is not continuously described as generating.

Several conditions can coexist. For example, one branch can await approval while two other branches continue running.

### 6.3 Multiple Runs

A Session can contain several Runs.

The activity area preserves the selected Run and provides access to the others. A new update does not automatically replace the selected inspection target.

An unrelated Workflow wait does not automatically disable ordinary Session messages. Message availability follows the Session responder’s actual state.

### 6.4 Participant messages and handoff

Participant messages identify the producing Agent and task. Concurrent streams remain separate.

Internal transformations, state changes, and channel records remain in activity details by default. They do not become artificial assistant messages.

A participant invocation leaves the Session responder unchanged.

A handoff of the Session message route changes the Session responder and states the relationship clearly:

“Reviewer is the Session responder. Part of Release review.”

During handoff:

- Ordinary messages go to that responder.
- Eligible Tool and continuation controls follow that responder.
- “Finish conversation” is available when handoff completion is permitted.
- The completion behavior identifies whether the previous responder returns or the current Agent remains.

The action label can retain the backend’s existing handoff terminology, but the functional meaning remains Session responder completion.

Inspecting another Run never changes the message destination.

## 7. Enhanced approval and request review

### 7.1 One superset review flow

The current approval/wait modal is enhanced into the shared runtime review flow.

It handles:

- Existing MCP approvals.
- Existing manual Tool work.
- Human response requests.
- Authentication and configuration requests.
- Recovery review.
- Informational waits.

The enhancement replaces the current first-pending-wait behavior with a navigable set of pending items, grouped by Run.

Each item identifies:

- Owning Run and step.
- Purpose.
- Requested decision or response.
- Assignee restrictions.
- Deadline.
- Available actions and their consequences.

Agent and Workflow requests use this one review flow.

### 7.2 Request-specific interaction

| Request type                                          | Interaction                                                                |
| ----------------------------------------------------- | -------------------------------------------------------------------------- |
| Execution approval                                    | Arguments, risk, and permitted approval or denial scope                    |
| Manual Tool work                                      | Existing eligible Run, Retry, Skip, Use error, and continuation actions    |
| Human response                                        | Schema-based response entry with a JSON alternative                        |
| Authentication or configuration                       | Targeted authorization or repair                                           |
| Recovery review                                       | Available evidence and permitted recovery decisions                        |
| Timer                                                 | Scheduled resumption information                                           |
| External event                                        | Expected event information and authorized event submission where supported |
| Channel, child completion, resource, or capacity wait | Waiting reason and access to the dependency                                |

Informational waits do not receive a generic “Continue” action.

An accepted external event is reported as accepted. Progress is reported only when the Workflow consumes the event and advances.

### 7.3 Decision semantics

Three meanings remain separate:

- Permission to execute an operation.
- A business response, such as approval or rejection.
- Cancellation of the Run.

A business rejection can be a valid response that allows the Workflow to continue normally.

Human response actions require assignment authority. Opening a request does not establish that authority.

For MCP approvals:

- Approve once remains the conservative default.
- Session-scoped approval is explicit.
- The affected approval scope is visible.
- Automatic execution remains separate from authorization.

### 7.4 Deferral and resolution

The review modal can close while requests remain pending.

Closing it preserves the response draft and leaves the pending count visible. Other Session activity remains accessible.

After submission, the request remains visibly resolving until authoritative state confirms the result.

The same rule applies to current continuation behavior: accepted staged input remains accepted even when subsequent continuation fails. A continuation retry does not submit the same content again.

### Decision and tradeoff

A dismissible shared review flow is preferred over a mandatory modal for every wait.

The tradeoff is that a request can remain unanswered longer. Persistent pending counts and deadlines preserve awareness while allowing unrelated work to continue.

## 8. Execution controls and effect safety

### 8.1 Clear control meanings

| Control           | Meaning                                                             |
| ----------------- | ------------------------------------------------------------------- |
| Run pending Tools | Execute eligible calls in the identified Agent Tool batch.          |
| Continue          | Resume an eligible continuation.                                    |
| Hold automation   | Apply the existing scoped hold to automatic Tools and continuation. |
| Pause Run         | Request suspension at the supported Runtime boundary.               |
| Resume Run        | Resume the same paused Run.                                         |
| Cancel Run        | Cancel the named Run under its child and cleanup relationships.     |
| Run again         | Prepare a new invocation from reviewed input.                       |

The existing automation hold and Workflow pause remain distinct actions within the same controls.

Pause does not imply that an external operation has stopped or that deadlines have been extended.

Available actions follow the actual runtime state. Provider-managed Tools remain part of Model execution rather than receiving local execution actions.

### 8.2 Cancellation

Cancellation identifies its scope and consequences:

- Attached work affected.
- Independent child Runs affected.
- Independent work that can remain active.
- External effects that may already have occurred.
- Cleanup that can continue.

The visible states distinguish cancellation requested, cleanup in progress, and terminal cancellation.

Partial output remains available. Cancellation is not presented as rollback.

### 8.3 Retry and uncertain effects

Retry follows effect safety.

An ordinary failure and an unconfirmed external effect receive different explanations. For example:

“Publishing may have completed. The external result is not yet confirmed.”

The shared recovery review retains:

- Original operation and captured input.
- Available receipts and external evidence.
- Confirmed and unconfirmed facts.
- Permitted recovery actions.

An Agent’s “Use error” action does not become a way to confirm an uncertain Workflow effect.

Reconciliation and compensation remain identifiable activities. Compensation failure remains attached to the primary failure or cancellation.

A terminal Run can still carry an unresolved-effect warning. [6]

## 9. Results, history, and continuity

### 9.1 Explicit Workflow result

The final result comes from the Workflow’s explicit output or Workflow-level return.

Supported results include:

- Text.
- Structured values.
- Actual files and media.
- A valid `null` value.
- Failure, cancellation, and uncertain outcomes.

The last participant message is not automatically the final Workflow result.

Process completion and business outcome remain separate. A completed review can legitimately return “Declined”.

“Use result in a message” creates a draft and makes the transfer into the next Session message explicit.

### 9.2 Reuse and editing

“Run again” opens the previous input and configuration as a new launch draft.

Changes in Workflow revision remain visible before another start. The previous Run remains unchanged.

Existing edit-and-resend remains available for eligible Agent messages.

Accepted Workflow launches and human decisions remain historical execution facts. Their reuse creates a new draft or Run rather than rewriting accepted work.

Editing visible history does not imply reversal of external effects.

### 9.3 Reopening

Closing a tab closes the view rather than cancelling work.

Reopening the Session restores:

- Run status.
- Pending requests.
- Session responder.
- Accepted continuation input.
- Results and unresolved-effect information.

Reopening never starts the Workflow again.

Workflow-only Sessions remain searchable and receive a useful title from the Workflow name and available input. Export remains Session-based and includes recorded execution facts, not only visible message text.

### 9.4 Declared triggers

A direct start creates one Workflow Run.

Trigger activation remains a separate explicit runtime action within detailed Session activity. Review includes the declared schedule or event, time zone, overlap behavior, and missed-occurrence behavior.

Trigger-generated Runs identify their origin.

Deactivating a registration and cancelling an existing Run remain separate actions.

Trigger authoring, schedule editing, and catalog management remain within the later management UX.

## 10. Source basis and coverage

Current-state findings are based on the supplied source. The HLDs are treated as implemented target behavior, as requested.

- [1] Agent selection and inherited configuration: `composer/agents/agent_dropdown.tsx`, `composer/agents/use_agent_manager.ts`, `composer/editor/editor_context_bar.tsx`, and `composer/models/model_dropdown.tsx`.
- [2] Composer configuration and submission: `composer/composer_box.tsx`, `composer/editor/editor_area.tsx`, `session/session_intent.ts`, and `conversation/use_send_message.ts`.
- [3] Runtime controls and review: `conversation/conversation_area.tsx`, `composer/editor/session_tool_activity_chips.tsx`, `session/session_composer_state.ts`, and `session/session_wait_modal.tsx`.
- [4] Messages and results: `messages/message.tsx`, `messages/message_attachments_bar.tsx`, `composer/tools/tool_details_modal.tsx`, and `mcpapps/mcp_app_view.tsx`.
- [5] Session navigation and restoration: `tabs/use_chats_controller.ts`, `tabs/tabs_persistence.ts`, `conversation/starter_intent.ts`, and `session/session_projection.ts`.
- [6] Target semantics: `devdocs/requirements/workflow_artifact.md`, `devdocs/requirements/workflow_runtime.md`, and `devdocs/requirements/sessionv2_to_sessionv3_migration.md`.
- [7] V3 operation surface: `cmd/agentgo/wrapper_sessionv3_runtime.go`.

Frontend paths above are relative to `frontend/app/chats/`.

Shared and participant-specific consumer override rules are proposal decisions. The supplied wrapper alone does not establish every editable configuration scope. Exact existing support requires the omitted V3 defaults, override, and action-eligibility definitions. Arbitrary process editing, target replacement, and in-flight state editing are not assumed.
