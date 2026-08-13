---
name: api-reference-authoring
description: Generate or update API and SDK reference docs and examples from code, schemas, OpenAPI, proto files, or existing docs. Prefers accurate reference over marketing language.
---

# API Reference Authoring

Produce or update API and SDK reference content from source. Accurate, complete, and grounded in the actual surface.

## Use when

- generating or updating API or SDK reference docs
- enumerating endpoints, functions, parameters, returns, and errors
- adding examples to existing reference docs
- documenting changes to a public surface

## Do not use when

- the user wants conceptual guides or tutorials
- the user wants release notes
- the user wants an audit of existing docs

## Execution model

Workflow phases:

    overview -> enumerate surfaces -> parameters -> returns and errors -> examples -> edge cases -> versioning notes

Enumerate the surface. Walk the source: code, schemas, OpenAPI, proto, or existing docs. List every endpoint, function, type, parameter, return shape, and error code that the source defines.

Parameter discipline. For each parameter: name, type, required vs optional, default if any, valid range or enum, constraints, examples. Do not omit `required` or `default`; readers need both.

Errors as first-class content. List error codes or exception types with conditions that produce them. A reference without errors is incomplete.

Examples grounded in the surface. Each example must match the actual signature and types. If the source does not show a working call, mark the example `illustrative` rather than fabricating runtime behavior.

## Hard rules

- Prefer accurate reference over marketing language.
- Include parameter `required`, `default`, type, and constraints; never silently drop these.
- Document error codes or exception types and the conditions producing them.
- Include request and response examples only when supported by the source; mark unsupported examples `illustrative`.
- Include Mermaid only when flow or lifecycle clarifies usage; never as decoration.
- Do not invent endpoints, functions, types, parameters, or response fields.
- Note versioning, deprecations, and compatibility implications where present in source.
- Works standalone; does not require any companion prompt.

## Artifact format

Use only sections that matter.

overview: what the API or SDK does, its scope, and base URL/import path where applicable.

endpoints-or-functions: enumerated list; signature, purpose, source tie.

parameters: name | type | required | default | constraints | description; per endpoint or function.

return-values-or-responses: shape, types, semantics, success codes for HTTP.

errors: code or type | condition that produces it | suggested handling.

examples: request and response examples; cite source where applicable; mark `illustrative` if not directly supported.

edge-cases: pagination, retries, idempotency, rate limits, timeouts, partial results, ordering guarantees.

versioning-or-compatibility-notes: version surface, deprecations, breaking-change history if known from source.

## Response format

End each response with: `scope`, `confidence` (high|medium|low), `status`, key assumptions or surface areas not verified, `next-step` (only if useful).
