# Prompts Backend - Functional Specification

## Terminology

| Term                        | Meaning                                                                               |
| --------------------------- | ------------------------------------------------------------------------------------- |
| Prompt Template             | A multi-message prompt with variables and optional pre-processors.                    |
| Prompt Bundle               | A named group of templates that can be enabled / disabled together.                   |
| Slug                        | Human-readable identifier used in chat (`/slug`).                                     |
| Version                     | Opaque label that distinguishes revisions of the same slug.                           |
| Built-in bundle or template | Content shipped by the application, cannot be edited or deleted, but may be disabled. |

## Objectives

- End-users can save, manage and invoke reusable prompts in chat.
- A single switch can enable / disable a whole bundle or an individual template version.
- The system remains offline-friendly and loose-coupled: removing a model/tool never mutates stored templates.
- Feature is exposed in two surfaces:
  - Chat input palette (`/` autocomplete)
  - `Prompt Templates` admin page (CRUD + search).

## Data Model and Constraints

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

- Built-in

  - Every Bundle and PromptTemplate object returned should mark if it is builtin or not.
  - The flag is server controlled.
  - When `isBuiltIn == true` the object can only be enabled/disabled, everything else is read-only.

## API Surface

All routes are relative to `/prompts`.

### Bundles

| Verb   | Path                  | Body / Query                                                    | Notes                |
| ------ | --------------------- | --------------------------------------------------------------- | -------------------- |
| PUT    | `/bundles/{bundleID}` | Body: `{slug, displayName, isEnabled, description}`             | Create or replace.   |
| PATCH  | `/bundles/{bundleID}` | Body: `{isEnabled}`                                             | Toggle enabled flag. |
| DELETE | `/bundles/{bundleID}` | --                                                              | Soft-delete.         |
| GET    | `/bundles`            | Query params: `bundleIDs, includeDisabled, pageSize, pageToken` |                      |

### Templates

| Verb   | Path                                                             | Notes                                                                       |
| ------ | ---------------------------------------------------------------- | --------------------------------------------------------------------------- |
| PUT    | `/bundles/{bundleID}/templates/{templateSlug}/version/{version}` | conflict error if same `<slug,version>` exists.                             |
| PATCH  | `/bundles/{bundleID}/templates/{templateSlug}/version/{version}` | `{isEnabled}` Only enable/disable.                                          |
| DELETE | `/bundles/{bundleID}/templates/{templateSlug}/version/{version}` | Hard-delete local copy.                                                     |
| GET    | `/bundles/{bundleID}/templates/{templateSlug}/version/{version}` | --                                                                          |
| GET    | `/templates`                                                     | global list: `tags,bundleIDs,includeDisabled,recommendedPageSize,pageToken` |
| GET    | `/templates/search`                                              | global search: `q,includeDisabled,pageSize,pageToken`                       |

## Behavioral Rules

- Search & ranking:

  - The client maintains a local FTS index:
  - `prefix` > `whole-word` > `fuzzy` > `recency` (ModifiedAt).
  - Disabled rows are excluded unless `includeDisabled=true` was requested.

- Soft-delete of bundle

  - Marks `softDeletedAt` timestamp.
  - Background task reaps after 2 days if the directory is still empty.
  - Put and patch of things inside disabled bundles should not be allowed.

- Built-in immutability

  - Any attempt to mutate (PUT / DELETE / non-enabled PATCH) a built-in bundle should not be allowed. The enabled/disabled flag is the only mutable attribute.

- Variable substitution pipeline (invocation time)

  - collect defaults -> run pre-processors -> interpolate ${var} placeholders -> send to LLM
  - Undefined, non-required variables fallback to `""`.

- Pre-processor error handling modes: `"empty"` or `"fail"`.

- Backward references
  - Chat messages store the fully rendered prompt, so deleting / disabling a template does not alter existing chats.
  - When a disabled template is referenced, UI shows a warning and can re-enable the template on demand.

## Non-Functional Requirements

- Offline-first storage: flat-file JSON.
- Search - optional SQLite FTS.
- Concurrency: Duplicate writes across processes must be handled with file-locks so that the uniqueness guarantee is global.
- BuiltIns should be served via normal APIs only.

## Open Questions / Future Work

- Pre-Processor graph execution (dependencies between calls).
- Internationalization.
