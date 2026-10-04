# Work Package A completion notes

Scope completed: owner-service split and capability segregation only.

Deliberately not performed here:

- Work Package B aggregate `store/compose` and local deployment collapse.
- Work Package C source lifecycle invalidation publication redesign.
- Work Package D JSON Schema declaration-dispatch and remaining LLM/generic separation.

Any file deletion is blocked in this environment. Use the manual removal block shell commands in the delivery
report after caller migration.

`compose/local` still exists in this Work Package because aggregate assembly collapse is Work Package B. It now uses the named owner constructors above.

## Local opener migration

`compose/local.Open` now returns `*store/compose.Store` directly.

- Replace explicit `*local.Store` declarations with `*store/compose.Store`, or
  rely on type inference from `local.Open`.
- Remove use of `local.Store.components` and all
  `compose/local/internal/assembly` imports; neither is available.
- Continue using the same named API fields (`Roots`, `Sources`, `Artifacts`,
  `Catalog`, `Definitions`, `Refresh`, `Resources`, `ManagedPackages`,
  `Topology`, overlays, and secret capabilities) on the returned generic
  handle.
- Close the returned handle once. It owns locally opened SQLite and any
  caller-provided `SecretValues` only after a successful open.

## Generic Artifact Store assembly

`store/compose.Open` is the sole generic aggregate construction entry. Its
configuration is deliberately a list of named entity and flow contracts:
repositories, refresh publication, installation persistence, Source drivers,
schema factory/codecs, decoders, secret-value storage, policy, clock, and ID
allocation.

It has no local path, SQLite, filesystem-provider, JSON Schema-provider, or
keyring dependency. A deployment opens those resources, passes their narrow
contracts to this package, and supplies a shutdown callback whose ownership
transfers only after successful assembly.

The returned `Store` is the one public aggregate surface. It publishes entity
and flow APIs only; provider repositories remain construction dependencies.

## Local Artifact Store assembly

`compose/local.Open` is the local deployment entrypoint. It owns local layout,
SQLite opening, local source-driver construction, JSON Schema provider
selection, and close ownership for resources opened or explicitly transferred
by the deployment.

It returns `*store/compose.Store` directly. Callers must not depend on a
`local.Store` wrapper or an `internal/assembly.Components` value.

A caller-provided `Config.SecretValues` remains caller-owned when `Open`
fails. Ownership transfers only after `Open` succeeds; the returned aggregate
handle then closes the backend exactly once through `Store.Close`.
