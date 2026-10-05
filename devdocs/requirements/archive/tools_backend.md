# Tools Backend - Functional Specification

## Terminology

| Term          | Definition                                                                                 |
| ------------- | ------------------------------------------------------------------------------------------ |
| Tool          | A callable action. The call may execute local Go code or perform an outbound HTTP request. |
| Tool Type     | `"go"` or `"http"`.                                                                        |
| Tool Bundle   | Named container that groups related tools under a single on/off switch.                    |
| Slug          | Short human-friendly identifier (`+readfile`).                                             |
| Version       | Opaque label that distinguishes revisions of the same slug.                                |
| Built-in Tool | Tool compiled into the binary. Content is read-only; only the `isEnabled` flag may change. |

## Objectives

- Let users list, enable/disable and invoke tools from:
  - The "+" invoke option/palette in chat
  - A dedicated "Tools" admin screen (CRUD + live tester(?))
- Support two execution back-ends:
  a. `go` - zero-latency local call (`func(ctx, jsonArgs) (jsonResp, error)`)
  b. `http` - one or many outbound REST calls with templated URLs
- Organize tools into bundles that can be toggled as a group.
- Store all user-defined data offline (flat-files); online connectivity only required at invocation time for HTTP tools.
- Never mutate stored data automatically if an LLM model or Go function disappears; instead mark as "disabled".

## Data Model and Constraints

- `bundleID` and `toolID` are present and are UUIDv7.

- Validation rules

  - `urlTemplate` must start with `http://` or `https://`.
  - Placeholders `${var}` come from call arguments, or app-scoped secrets.
  - Destination host must match `config.allowedHosts`.

- Slug and Version strings

  - Allowed rune categories : Unicode Letter, Digit, ASCII dash.
  - Forbidden characters : underscore, whitespace, slashes, any control / symbol. Dot is allowed in version.
  - Case-sensitive : yes
  - Max length : 64 runes
  - Version strings: No semantic ordering; the backend treats it as an opaque label.
  - Within one bundle: `<slug, version>` must be unique.
  - Across bundles the same pair may repeat.

- Timestamps

  - `CreatedAt` - first insertion (immutable).
  - `ModifiedAt` - last structural change (body, tags, pre-processors).

## API Surface

All routes are relative to `/tools`.

### Bundles

| Verb   | Path                  | Body / Query                                                    | Notes                |
| ------ | --------------------- | --------------------------------------------------------------- | -------------------- |
| PUT    | `/bundles/{bundleID}` | Body: `{slug, displayName, isEnabled, description}`             | Create or replace.   |
| PATCH  | `/bundles/{bundleID}` | Body: `{isEnabled}`                                             | Toggle enabled flag. |
| DELETE | `/bundles/{bundleID}` | --                                                              | Soft-delete.         |
| GET    | `/bundles`            | Query params: `bundleIDs, includeDisabled, pageSize, pageToken` |                      |

### Tools

| Verb   | Path                                                            | Notes                                                                       |
| ------ | --------------------------------------------------------------- | --------------------------------------------------------------------------- |
| PUT    | `/bundles/{bundleID}/tools/{toolSlug}/version/{version}`        | conflict error if same `<slug,version>` exists.                             |
| PATCH  | `/bundles/{bundleID}/tools/{toolSlug}/version/{version}`        | `{isEnabled}` Only enable/disable.                                          |
| DELETE | `/bundles/{bundleID}/tools/{toolSlug}/version/{version}`        | Hard-delete local copy.                                                     |
| GET    | `/bundles/{bundleID}/tools/{toolSlug}/version/{version}`        | --                                                                          |
| POST   | `/bundles/{bundleID}/tools/{toolSlug}/version/{version}/invoke` | `{args}`                                                                    |
| GET    | `/tools`                                                        | global list: `tags,bundleIDs,includeDisabled,recommendedPageSize,pageToken` |
| GET    | `/tools/search`                                                 | global search: `q,includeDisabled,pageSize,pageToken`                       |

## Behavioral Rules

- PUT with existing `<slug,version>`: Must return conflict error; the existing tool is left unchanged.
- Built-in immutability

  - Any attempt to mutate (PUT / DELETE / non-enabled PATCH) a built-in bundle should not be allowed. The enabled/disabled flag is the only mutable attribute.

- Search & ranking:

  - The client maintains a local FTS index:
  - `prefix` > `whole-word` > `fuzzy` > `recency` (ModifiedAt).
  - Disabled rows are excluded unless `includeDisabled=true` was requested.

- Soft-delete of bundle

  - Marks `softDeletedAt` timestamp.
  - Background task reaps after 2 days if the directory is still empty.
  - Put and patch of things inside disabled bundles should not be allowed.

- Invocation flow

  - Validate `args` against `argSchema` (reject with `400` on failure).
  - Dispatch:
    - Go - `ctx, argsJSON` passed to registered func.
    - HTTP - build request from templates, run with retry/back-off.
  - ~~Parse and validate response against `outputSchema`.~~
  - Return `{ok:true, value}` or `{ok:false, error}`.
  - Usage metrics (`lastCalledAt`, `callCount`) updated asynchronously.

## Non-Functional Requirements

- Offline-first storage: flat-file JSON.
- Search - optional SQLite FTS.
- Concurrency: Duplicate writes across processes must be handled with file-locks so that the uniqueness guarantee is global.
- BuiltIns should be served via normal APIs only.

## Pending / Future

- Auth helpers (HMAC, OAuth2 token refresh) for HTTP tools. (May need to do it immediately)
- Tool limits: max sent bytes, max resp bytes, max retries, max no of tools allowed, etc.
- Streaming outputs (Server-Sent Events - chat tokens).
- Import / export bundles as ZIP.
- Rate limiting per tool (`maxCallsPerMinute`).
- UI recipe builder for non-technical users.
- Multi-step tools (sequence or graph of sub-tools).
- Internationalization.
