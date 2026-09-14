---
name: troubleshooting-guide-authoring
description: Turn errors, logs, bug reports, support notes, and fixes into troubleshooting or knowledge-base articles. Distinguishes confirmed causes from possible causes and prefers safe diagnostic steps.
---

# Troubleshooting Guide Authoring

Convert errors, logs, and fixes into KB-quality troubleshooting articles. Distinguish confirmed causes from possible causes. Safe diagnostics first.

## Use when

- producing a troubleshooting or KB article from a confirmed fix
- documenting how to diagnose a recurring error or symptom
- writing a runbook entry that can be executed by the intended audience

## Do not use when

- the user wants a bug diagnosis
- the user wants release notes
- the user wants conceptual docs

## Execution model

Workflow phases:

    symptom -> likely causes -> diagnosis -> fix or workaround -> verification -> prevention

Symptom-first ordering. Readers arrive with a symptom, not a cause. Lead with how to recognize the symptom unambiguously.

Confirmed vs possible causes. Mark each cause `confirmed` (proven by source material) or `possible` (consistent with symptom but not proven). Do not promote without evidence.

Safe diagnostics. Diagnostic steps should be safe to run on the affected system: read-only first, then non-destructive checks, then any action that mutates state. Call out destructive steps explicitly.

Audience-executable. Steps must be runnable by the intended audience. If they require operator access, say so; if customer-runnable, keep commands minimal and explained.

## Hard rules

- Distinguish `confirmed` causes from `possible` causes for every cause listed.
- Prefer safe, read-only diagnostic steps before any mutating action.
- Mark destructive steps clearly; include rollback or recovery notes.
- Do not invent error codes, log lines, or commands.
- Keep steps executable by the intended audience; do not assume access the audience does not have.
- Include verification steps so the reader can confirm the issue is resolved.
- Add a prevention section when source material supports it.
- Works standalone; does not require any companion prompt.

## Artifact format

Use only sections that matter.

symptom: how the user recognizes the problem; concrete error messages or behaviors.

likely-cause: list of causes; each marked `confirmed` or `possible` with source tie.

diagnosis-steps: ordered, safe-first steps to narrow down the cause; call out destructive steps.

fix-or-workaround: per cause, the fix or workaround; required permissions.

verification: how to confirm the issue is resolved; expected output or state.

prevention: configuration, monitoring, or operational changes that reduce recurrence.

escalation-notes: when to escalate, what information to collect first.

## Response format

End each response with: `scope`, `confidence` (high|medium|low), `status`, key assumptions or unverified causes, `next-step` (only if useful).
