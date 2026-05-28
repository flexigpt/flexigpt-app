# Attachments UX Requirements

## Terms

- Attachment: Explicit context added to a message (files, images, URLs, PRs, git context) that the assistant will _read/use_ for that turn.

- Attachment Mode: How the assistant will use a specific attachment, expressed simply (e.g. "Text content", "Page content", "Link only", "Diff").

- Attachment Strip & Chip
  - Strip: horizontal area above the composer showing all attachments for the current unsent message.
  - Chip: a single attachment: type icon, label, and (if applicable) a small mode chip.

## Core Principles

- One UX, scalable: Advanced behavior is revealed via consistent, unobtrusive controls.

- Smart defaults, per-turn:
  - Adding an attachment "just works" with a sensible default mode. Attachments apply only to the current message.
  - Progressive disclosure: Mode controls are always present but visually subtle; users can gradually discover and use them.

## Attach Entry Points

- Attach button (📎): Opens a small menu -

  - From your computer... (file picker)
  - From a link or URL... (paste URLs)
  - From Git... (github/gitlab/local git repo)

- Drag & drop: Dropping files onto the composer area is equivalent to "From your computer...".

- URLs in message body vs Attach
  - URLs typed/pasted into the composer text remain plain text; no auto-fetch.
  - URLs added via "From a link or URL..." become attachments and can be processed.

## Attachment Strip & Chips

- Strip appears only when the current unsent message has ≥1 attachment.

- Chip structure

  - `ICON  Label                        [Mode ▾]  (×)`
    - ICON: type indicator (file, image, URL, PR, Git, etc.).
    - Label: filename or short URL/title (truncated as needed).
    - Mode: visible for configurable types (e.g. `[Text ▾]`, `[Page ▾]`, `[Diff ▾]`).
      - Click opens a small mode menu; selection updates immediately.
    - Remove (×): removes attachment from the current message.

- Keyboard
  - Chips and their mode menus must be operable via keyboard (focus, open, change, remove).

## Attachment Types & Default Modes (UX)

_Default = initial mode when chip is created._

### Files from Computer

- PDF, DOC/DOCX, similar

  - Default: Text content
  - Mode menu example:
    - Text content
    - File (original format), Base64 encoded when api supports it.

- Plain text, Markdown, code

  - Default: Text
  - Usually single mode; no dropdown unless variants are added later.

- Images (PNG/JPG, etc.)

  - Default: Image
  - Local images: "[Image]" (no dropdown).

- Binary/unknown (zip, exe, etc.)
  - Default: Not readable (or similar). Visually show that it cannot be processed.
  - No dropdown initially.

### URLs (via "From a link or URL...")

- HTML web page

  - Default: Page content
  - Modes:
    - Page content
    - Link only

- PDF URL

  - Default: Text content
  - Modes:
    - Text content
    - PDF file
    - Link only

- Image URL

  - Default: Image
  - Modes:
    - Image
    - Link only

- Generic/unknown URL
  - If detected as HTML: treat as "HTML web page".
  - Else: default Link only with no further modes.

### Specialized URLs (PRs, commits)

- Pull / Merge Requests (e.g. GitHub/GitLab PR)

  - Default: PR diff only
  - Modes:
    - PR diff only
    - PR page content
    - Link only

- Commits / compare URLs
  - Default: Diff only
  - Modes:
    - Diff only
    - Commit page content
    - Link only

### Local Git Context (From Git...)

- From Git... dialog

  - Offers high-level options (examples):
    - Current uncommitted changes (diff)
    - Changes vs main
    - Recent commit history (last N)

- Chips
  - Diff options:
    - Label like: `Git diff (working tree)` with a static Diff label initially.
  - History options:
    - Label like: `Git history (last 10)` with a static Log label initially.

## Editing & User Messages

- User messages show:

  - Message text.
  - Attachments under the bubble:
    - Images: inline thumbnail/preview.
    - Others: chips or a small list (icon + label + brief mode text if helpful, e.g. "(Text)", "(Page)", "(Diff)").

- Editing messages
  - Editing a message that had attachments:
    - Restores original text into composer.
    - Restores all attachments as chips with their previous modes.
  - During edit, user can:
    - Change text.
    - Remove/add attachments.
    - Change modes.
  - Resend uses the updated text + attachments.

## Discoverability

- Mode chips always visible for attachments with multiple possible behaviors (Text/Page/Diff/etc.), but styled subtly to avoid clutter.

- First-time hints (per attachment type)
  - On first use of a type (PDF, URL, PR, Git), show a short, dismissible inline hint near the chip, e.g.:
    - "Attached as text content. Change via `[Text ▾]`."
  - Hints do not persist beyond a small number of uses per type.
