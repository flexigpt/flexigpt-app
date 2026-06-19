# Unified Diff Apply

FlexiGPT can recognize a unified diff in a rendered code block and offer a local apply flow. Use it when the assistant returns a patch, or when you paste a diff into chat and want to review it before changing files on your machine.

This is a local file action. The model only sees the diff text; the dry run and apply steps happen against your local workspace.

## Table of contents <!-- omit from toc -->

- [When to use it](#when-to-use-it)
- [What the UI recognizes](#what-the-ui-recognizes)
- [Dry run first](#dry-run-first)
- [Fix target paths](#fix-target-paths)
- [Apply safely](#apply-safely)
- [If it does not apply cleanly](#if-it-does-not-apply-cleanly)

## When to use it

Use the diff apply flow when you want to:

- review a patch from the model
- apply a local fix without leaving the chat
- compare a proposed code edit against the files on disk
- avoid copying a patch into a separate editor just to apply it

## What the UI recognizes

The apply controls appear when the code block looks like a unified diff or patch.

Common shapes include:

- `diff --git`
- `Index:`
- `---` and `+++` file headers
- patch-like hunk output with `@@` markers

If the block is not recognized as a diff, it stays a normal code block and only the standard code actions appear.

## Dry run first

Start with a dry run before you apply anything.

Use the `Files` button in the code block header to open the details view, then:

- inspect the parsed file list
- review the status badges
- read the patch and file diagnostics
- check whether the patch is `applicable`, `needs info`, `blocked`, `already applied`, or `applied`

If the dry run is clean, you can move on to apply.

## Fix target paths

If a file target is missing or ambiguous, set the local target path in the details view.

Helpful patterns:

- use the candidate paths if one matches your repo layout
- keep strict matching on when you want fewer fuzzy matches
- prefer one file at a time when a patch touches several unrelated paths

If the target path is wrong, correct it and run the dry run again.

## Apply safely

When the dry run looks good:

- use `Apply` for one file or `Apply all` for the whole patch
- let the app run the validation step before writing files
- reopen the edited files after the apply completes
- treat a partially applied patch as a sign to re-check the paths and diagnostics

## If it does not apply cleanly

Do not force a patch through conflicts.

Instead:

- narrow the patch scope
- ask the model for a cleaner diff
- split one large patch into smaller file-specific patches
- fix the file target and run the dry run again
