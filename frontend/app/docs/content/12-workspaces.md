# Workspaces

Workspaces help you work with the same repository or folder repeatedly.

An Agent gives you a starting way of working. A Workspace gives you project-aware context. You can use both in the same chat.

## Table of contents <!-- omit from toc -->

- [When to use a Workspace](#when-to-use-a-workspace)
- [Add a repository folder](#add-a-repository-folder)
- [Use a Workspace in Chats](#use-a-workspace-in-chats)
- [Workspace and Agent together](#workspace-and-agent-together)
- [Customize a Workspace](#customize-a-workspace)
- [Troubleshooting](#troubleshooting)
  - [The Workspace does not show up](#the-workspace-does-not-show-up)
  - [Too much context appears](#too-much-context-appears)
  - [The wrong project context is selected](#the-wrong-project-context-is-selected)

## When to use a Workspace

Use a Workspace when you often work with:

- one repository
- one product folder
- one documentation site
- one collection of notes
- one local project with recurring context

A Workspace can help you bring the right local material into Chats without attaching the same files every time.

## Add a repository folder

1. Open **Workspaces**.
2. Select **Add Workspace Directory**.
3. Choose the repository or folder.
4. Let FlexiGPT inspect the folder.
5. Open the Workspace details to see what was found.

If the folder has no Workspace file yet, FlexiGPT provides a sensible starting setup.

## Use a Workspace in Chats

1. Open **Chats**.
2. Open the **Workspace** picker in the composer.
3. Choose the repository setup you want.
4. Review the selected context.
5. Add or remove files, Skills, tools, or connected services as needed.
6. Send your request.

A Workspace is still editable in the composer. It does not prevent you from adding other context.

## Workspace and Agent together

A useful pattern is:

1. choose a Workspace for the repository or folder
2. choose an Agent for the type of work
3. review the draft and selected context
4. remove anything that is not needed
5. send a focused request

Examples:

| Goal                       | Workspace                          | Agent                   |
| -------------------------- | ---------------------------------- | ----------------------- |
| Build a feature            | Project repository                 | Spec Driven Development |
| Review a change            | Project repository                 | Reviewing Code          |
| Investigate a failing test | Project repository                 | Bug Investigator        |
| Update a guide             | Documentation folder               | Docs Writer             |
| Prepare release notes      | Repository or release notes folder | Release Notes Writer    |

## Customize a Workspace

You can keep using the default setup, or add a Workspace file to the repository when you need a repeatable custom setup.

A Workspace file can describe the local material and reusable tools that are useful for that project.

Use the **Workspaces** page to:

- refresh a folder after files change
- see available Workspace choices
- enable or disable discovered items
- remove a folder from FlexiGPT without deleting the folder from disk
- view the default setup when no custom Workspace file exists

## Troubleshooting

### The Workspace does not show up

Check:

- the folder is still available
- the Workspace directory is enabled
- refresh completed successfully
- any Workspace file in the repository is valid

### Too much context appears

Remove unneeded selections in the composer before sending. A Workspace is a starting point, not a requirement to send every available file.

### The wrong project context is selected

Open the Workspace picker and choose the intended repository setup. If needed, remove the old Workspace directory from **Workspaces**.
