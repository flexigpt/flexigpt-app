---
name: user-feedback-analysis
description: Analyze interviews, support notes, surveys, NPS comments, and customer notes into themes, needs, pain points, and product opportunities. Maintains traceability from theme to source.
---

# User Feedback Analysis

Turn raw customer voice into themes, needs, pain points, and opportunities. Maintain traceability. Separate strong signal from weak signal.

## Use when

- analyzing interview notes, support tickets, surveys, NPS comments, or customer notes
- identifying themes and needs from qualitative input
- preparing product opportunities from feedback
- assessing what is signal vs noise

## Do not use when

- the request is a product spec
- the request is prioritization across opportunities
- the request is a decision record between options

## Execution model

Workflow phases:

    sources -> themes -> evidence binding -> needs and pain points -> opportunities -> uncertainty

Source inventory. Before theming, list the sources, their dates, sample sizes, and any selection bias (e.g., only paid users, only support contacts). This frames what the analysis can and cannot claim.

Theme then bind. Generate candidate themes, then bind each to source snippets. A theme without binding is a guess. Track snippet counts and whether evidence is unique or duplicated.

Strong vs weak signal. Mark themes as `strong` (multiple independent sources, consistent), `medium` (several sources, some inconsistency), or `weak` (few sources or contradicted). Do not promote weak signal to strong without new evidence.

Need vs request. A user's stated request is not always the underlying need. Note both when they differ; do not silently substitute one for the other.

## Hard rules

- Maintain traceability: every theme cites source snippets or source IDs.
- Do not overcount duplicate notes from the same source as independent evidence.
- Separate strong, medium, and weak signals.
- Distinguish stated requests from inferred needs.
- Do not invent quotes or paraphrases that change meaning.
- Note selection bias and gaps in the source set explicitly.
- Works standalone; does not require any companion prompt.

## Artifact format

Use only sections that matter.

sources-reviewed: source list, date range, approximate counts, selection bias.

themes: theme | one-line summary | signal strength (`strong`|`medium`|`weak`) | source IDs.

pain-points: distilled pain points, severity, who is affected, source tie.

user-needs: underlying needs (vs surface requests); note when they diverge.

evidence-snippets: representative quotes or paraphrases per theme; cite source.

product-opportunities: candidate opportunities derived from themes; tie each to themes and source.

risks-or-uncertainty: known biases, gaps, conflicting signals.

follow-up-questions: questions worth asking next round of research to raise confidence.

## Response format

End each response with: `scope`, `confidence` (high|medium|low), `status`, key assumptions or gaps, `next-step` (only if useful).
