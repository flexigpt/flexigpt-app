# Work Package completion notes

## Package A checkpoint

Scope completed: owner-service split and capability segregation only.

Any file deletion is blocked in this environment. Use the manual removal block shell commands in the delivery
report after caller migration.

`compose/local` still exists in this Work Package because aggregate assembly collapse is Work Package B. It now uses the named owner constructors above.

## Package B checkpoint: Local opener migration

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

## Package B checkpoint: Generic Artifact Store assembly

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

## Package B checkpoint: Local Artifact Store assembly

`compose/local.Open` is the local deployment entrypoint. It owns local layout,
SQLite opening, local source-driver construction, JSON Schema provider
selection, and close ownership for resources opened or explicitly transferred
by the deployment.

It returns `*store/compose.Store` directly. Callers must not depend on a
`local.Store` wrapper or an `internal/assembly.Components` value.

A caller-provided `Config.SecretValues` remains caller-owned when `Open`
fails. Ownership transfers only after `Open` succeeds; the returned aggregate
handle then closes the backend exactly once through `Store.Close`.

## Work Package C completion notes

Scope delivered in this Artifactory-only change:

- Source lifecycle invalidation is an explicit Source command and a narrow
  injected publication port.
- Refresh owns aggregate preconditions and publication; Artifact owns missing
  state derivation; SQLite atomically applies the explicit command.
- SQLite no longer infers Artifact invalidation from ordinary Source
  `Update`/`Retire` repository calls.
- Definition canonicalization and digest admission are centralized at decoder
  and generated-document admission; immutable reads avoid repeated hashing.
- Root/topology purge evicts Root-local Definition cache entries.
- Ordinary Resource access explicitly includes verification sessions; trusted
  native-path access remains separately supplied.
- Source driver aliases and Source-owned resource-session/native-path helpers
  expose no compatibility surface.

Regression tests were added for lifecycle command validation, Artifact
invalidation derivation, Definition admission, and explicit Resource sessions.
They were intentionally not run, together with all existing tests and lint.

Not performed here:

- Work Package D declaration-language dispatch extraction from the JSON Schema
  provider.
- Any change outside `internal/artifactory-go`.

Files whose obsolete compatibility-only names remain because deletion is
blocked in this environment contain no exported forwarding surface:

- `store/source/driver_aliases.go`
- `store/source/verified_path.go`

They may be removed physically once file deletion is available.
