# LLM Artifactory Status

## Current position

Proposal 1 is complete as the generic Artifactory foundation.

The generic layer owns Root, Source, Definition, Artifact, Overlay, Secret,
Refresh, Resource, ManagePackage, Install, ArtifactCleanup, provider execution,
and local deployment lifecycle. It has no LLM declaration vocabulary, runtime
execution policy, product topology, or Wails dependency.

Proposal 2 provides the LLM-specific layer above that foundation:

```text
generic Artifactory
        ↑
LLM declaration, composition, and artifact-family layer
        ↑
application topology, supplied content, runtime adapters, and Wails wrappers
```

`internal/llmartifactory-go` does not import `internal/artifactsetup`, Wails,
application topology, application embedded content, or runtime packages.
Application-selected behavior enters through immutable, narrow support values.

## Proposal status

| Proposal / package | Status                 | Result                                                                                                                                                                                                     |
| ------------------ | ---------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Proposal 1         | Complete               | Generic entity ownership, flow sequencing, provider contracts, verified reads, package management, secret lifecycle, installation, and local deployment are established.                                   |
| Proposal 2 / A     | Complete               | LLM library, application content, and application setup ownership are separated.                                                                                                                           |
| Proposal 2 / B     | Complete               | Generic discovery preparation, verification sessions, package outcomes, local-state cleanup, secret lifecycle, immutable content support, and opener ownership are generic responsibilities.               |
| Proposal 2 / C     | Complete               | Declaration grammar, interpretations, canonical decoding, family contracts, and typed Definition reconstruction are family-owned above generic Artifactory.                                                |
| Proposal 2 / D     | Complete               | Composition, locator resolution, explicit refresh closure, capability plans, and Plugin direct-membership semantics replace Collection ownership.                                                          |
| Proposal 2 / E     | Complete               | Family services own package behavior, family local-state semantics, verified materialization, and runtime-neutral projections.                                                                             |
| Proposal 2 / F     | Complete for this tree | Application support resolution, generated content compilation, runtime adapter injection, wrapper wiring, direct capability setup, and wrapper-local management aggregation are aligned with the proposal. |

## Package F decisions implemented

- Wails wrappers remain transport, recovery, lifecycle, and cross-root
  management boundaries. They were not thinned by moving wrapper work into
  `llmartifactory-go`.
- Cross-root Agent management aggregation remains in
  `AgentStoreWrapper`; `agent.Service` exposes only Root-scoped family
  operations.
- `llmartifactory-go` does not depend on `artifactsetup`, including contract
  tests.
- The LLM aggregate remains narrow: it exposes interpretations and the shared
  composition resolver rather than mirroring the generic Store.
- Generic Store aggregate access remains deployment assembly only. No new
  broad facade was added for LLM consumers.
- Model one-line child-service forwarding facades are removed. The Model
  family service owns its actual public landing operations directly.
- Workspace uses `mcp.Service.ResolveMCPServer` directly through a narrow
  consumer port. The former `ResolveWorkspaceMCPServer` forwarding facade is
  removed.
- Constructor validation establishes mandatory dependencies and immutable
  support values. Internal paths do not repeatedly validate constructed
  services merely because a context or dependency was passed through.
- Startup baseline sequencing no longer performs redundant foreground
  `ctx.Err()` checks between synchronous family calls.
- Document support now includes exact names and declared filename patterns.
  Source decoders can recognize forms such as `reviewer.agent.yaml`,
  `reviewer.agent.yml`, and `reviewer.agent.json` when the application support
  catalog declares them.
- Skill, Agent, and Tool package preparation consume application-selected
  document support rather than calling family-level fixed filename helpers.
- Tool built-in compilation receives Tool and Plugin document support from the
  application support matrix. Static SDK Tool declarations and generated Go
  Tool declarations remain first-class Tool artifacts.
- Workspace source-role logic compares injected source profiles instead of
  hard-coded workspace storage-key strings.
- Catalog-derived decoder pattern collection is deterministic.

## Architectural invariants

1. Generic Artifactory remains the only local deployment owner.
2. LLM Artifactory owns LLM declaration and family semantics, not product
   topology.
3. Artifact families receive only support values they require.
4. Application setup owns concrete source kinds, file patterns, package
   layouts, source profiles, built-in identities, and startup ordering.
5. Plugin membership remains declaration content, never Artifact ownership.
6. Normal composition is read-only; refresh remains explicit.
7. Runtime execution, plaintext secret use, native-path authority, and Wails
   transport remain outside LLM Artifactory.
8. No compatibility aliases remain for Collection, topology, or removed
   forwarding-facade APIs.
9. Read, catalog, direct-membership, and built-in paths do not perform
   preparatory writes or implicit refresh.
10. New source forms are declared through application support catalog entries,
    not by adding filename assumptions to artifact-family code.

## Next extension rule

A new artifact family should add:

1. A family contract and interpretation registration.
2. A family support requirement.
3. Application support declarations.
4. Family package/source-format behavior where required.
5. A runtime adapter only when execution is required.

It must not add generic Artifactory type switches, a global built-in registry,
a broad local-store facade, a Wails dependency, or a filename assumption
outside application support declarations.
