---
name: research-brief-authoring
description: Turn attached or local source material into a research brief with findings, evidence, uncertainty, recommendations, and next questions. Uses local or attached sources only; does not assume live web search.
---

# Research Brief Authoring

Produce a research brief from attached or local source material. Cite sources. Separate findings from recommendations. State uncertainty.

## Use when

- synthesizing attached PDFs, notes, articles, transcripts, or local docs into a brief
- preparing a research summary for a decision or discussion
- combining qualitative and quantitative source material into one structured artifact

## Do not use when

- the request requires live web search (this skill does not assume web access)
- the request is user feedback analysis specifically
- the request is a decision record
- the request is a product spec

## Execution model

Workflow phases:

    sources -> themes -> evidence binding -> disagreements -> recommendations -> next questions

Local and attached sources only. Do not assume access to a live web search. If a question requires fresh external information, list it as a `next question` rather than guessing.

Findings before recommendations. Findings describe what the sources say. Recommendations are derived from findings plus stated goals. Keep them in separate sections.

Disagreements named. When sources disagree, present both positions, the strongest evidence on each side, and the resulting uncertainty. Do not silently pick one.

Uncertainty as a section. Uncertainty is part of the brief, not a footnote. List what is known weakly, what is contested, and what is missing.

## Hard rules

- Use local or attached source material only; do not assume live web search.
- Cite source names, file paths, or attachment names for every finding.
- Separate findings from recommendations.
- Name disagreements between sources; do not silently pick a winner.
- Mark uncertain findings explicitly; do not promote weak signal.
- Do not invent quotes, statistics, or source attribution.
- List next questions for follow-up research clearly.
- Works standalone; does not require any companion prompt.

## Artifact format

Use only sections that matter.

source-table: source name | type | date if known | relevance | one-line summary.

key-findings: F# | finding | source tie | signal strength (`strong`|`medium`|`weak`).

evidence: representative quotes or paraphrases per finding; cite source.

disagreements-or-conflicting-evidence: where sources disagree, both positions, strongest evidence each way.

uncertainty: what is known weakly, contested, or missing.

recommendations: derived from findings plus stated goals; tie each recommendation to one or more F#.

next-questions: research questions worth pursuing to raise confidence.

## Response format

End each response with: `scope`, `confidence` (high|medium|low), `status`, key assumptions or gaps in source coverage, `next-step` (only if useful).
