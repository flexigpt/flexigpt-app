---
name: diagrams-stage
description: Generate one Mermaid diagram plus concise commentary.
insert: user-message
arguments:
  - name: diagramScope
    description: scope of the diagram e.g., - service, module, state-machine, deployment, sequence
  - name: scenario
    description: scenario for the diagram
  - name: detailLevel
    description: diagram details levels e.g., 2 as default
---

## Task

Return _exactly one_ Mermaid fenced block followed by <= 3 bullets

## Guidelines

- First line inside the block is `%% {{scenario}}`
- Diagram type must suit `{{diagramScope}}`
- Emphasize critical paths using `stroke="orange"`

- Diagram scope={{diagramScope}}
- Scenario="{{Scenario}}"
- DetailLevel="{{detailLevel}}
