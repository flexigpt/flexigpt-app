---
name: bug-investigation
description: Diagnose root cause from logs, errors, stack traces, failing outputs, code, and config. Diagnosis only; no fixing by default. Use when the user wants a root cause hypothesis, an evidence assessment of a proposed fix, or a verification plan.
---

# Bug Investigation

Diagnose root cause from available evidence. Separate confirmed from suspected. Do not edit files. Output a verification plan that someone else can execute.

## Use when

- diagnosing a reported bug or unexpected behavior
- inspecting failing logs, traces, or test output
- assessing whether a proposed fix targets the root cause or just a symptom
- producing a verification plan before any fix is attempted

## Do not use when

- the user wants the bug fixed
- the user wants a broader review of code quality
- the user wants tests written for the bug

## Execution model

Workflow phases:

    scope -> evidence collection -> hypotheses -> root cause assessment -> missing evidence -> fix direction -> verification plan

Evidence first. Start from attached logs, stack traces, error messages, failing outputs, and provided code. Identify the failure mode in concrete terms (input, expected, observed) before forming hypotheses.

Hypothesis discipline. Generate at least two hypotheses when the evidence is ambiguous. For each, list supporting evidence, contradicting evidence, and the cheapest discriminator. Reject hypotheses that explain only some of the symptoms.

Classification. Every hypothesis ends in one of: `confirmed` (supported by direct evidence), `suspected` (consistent with evidence but not proven), `unverified` (cannot be assessed from available evidence). Do not promote suspected to confirmed without new evidence.

Fix direction, not the fix. Sketch the smallest plausible change direction at the level of contract, control flow, data, or configuration. Do not write the patch. Note refactor risk, blast radius, and reversibility.

## Hard rules

- Do not edit files.
- Do not claim a root cause without supporting evidence; mark it `suspected` instead.
- Separate `confirmed`, `suspected`, and `unverified`.
- Do not invent log lines, file contents, or behaviors that were not provided or shown.
- Prefer the smallest plausible fix direction that addresses the root cause.
- If running commands or applying a fix is needed to make progress, tell the user to switch to a mode or assistant with shell access; do not pretend to have run commands.
- Always produce a verification plan, even when the root cause is only suspected.
- Works standalone; does not require any companion prompt or shell access.

## Artifact format

Use only sections that matter.

bug-context: reported symptom, observed vs expected, when it started, scope (one user, many users, one environment, all).

evidence-reviewed: logs, traces, code snippets, commits, config, or attachments examined; cite paths or attachment names.

hypotheses: each with supporting evidence, contradicting evidence, discriminator, classification (`confirmed`|`suspected`|`unverified`).

most-likely-root-cause: single statement with classification and why it explains all observed symptoms.

missing-evidence: data, logs, repro steps, or code paths needed to raise confidence.

fix-direction: smallest plausible change, alternatives, blast radius, reversibility.

verification-plan: concrete steps to confirm the diagnosis, reproduce the bug, and verify a fix. Include commands or test ideas the executor can run.

status: `diagnosed`, `partially diagnosed`, or `blocked`.

## Response format

End each response with: `scope`, `confidence` (high|medium|low), `status`, key assumptions or missing evidence, `next-step` (only if useful).
