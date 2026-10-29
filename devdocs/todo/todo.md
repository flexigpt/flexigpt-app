# Project TODO

## Laundry list

- Testing
  - [ ] test enhanced mcp apps.
  - [ ] test web search etc and pending user args etc after bottom bar migration.
  - [ ] test with some skill that has scripts too
  - [ ] test workspace
  - [ ] test all management pages again properly

- [ ] verify diff applicability in streaming
- [ ] anthropic released sdk v2 - temperature is not there in sdk, most probably cannot use current api version with new sdk.

## M-n

- [ ] agent to agent and workflow support backend + ui.
  - [ ] mostly loop and team as a artifact doesn't really make sense as it is not saying anything concrete.
  - [ ] maybe agents and agent to agent communication and a stronger workflow schema that can express a team based workflow (where each agent is a team member anycase), and loop semantics inbuilt supported is better.

- [ ] artifactory, llm artifacts and then agentruntime repo. much tighter api surface.
  - [ ] should have per repo docs + skills + workspace def
  - [ ] the declarations, hierarchy and operational thing should be mentioned in the repos and then in flexigpt repo docs too.

- [ ] FlexiGPT repos automation using workspaces
  - [ ] add wails considerations skill in flexigpt repo.
  - [ ] add workspaces of frontend wails bridge and backend work in flexigpt
  - [x] add skills in inference. and a workspace config too.

## M-n+1

- [ ] App SDK
  - [ ] An ai intro tutorial app builtin will be good.
  - [ ] An learn chess app will also be nice
  - [ ] Start with the noted apps support

## M-n+2

- [ ] Distribution
  - [ ] vscode extension?? May need to decide some policies like - no workspace selection (as current will be default) etc
    - [ ] may need some ui level size handling. should be straightforward enough though
  - [ ] mobile apps (because wails supports it now)?? Same ui clean as vscode
  - [x] cli - No. Not interested.

## M-n+m

- [ ] Docs clean: positioning wrt repeatable workflows and enhanced guidance
  - [x] setup steps for local or custom models
  - recommended models
  - context length notes
  - limitations
  - troubleshooting

- Reliability polish. Mostly done but review:
  - [ ] request cancellation
  - [ ] stream abort behavior
  - [x] attachment stale file behavior
  - [ ] failed tool call recovery
  - [ ] corrupted local data recovery
  - [ ] app startup if one catalog is malformed
  - [ ] safe migration of local files
  - [ ] large conversation performance

- Error and empty-state quality.
  - Mostly done but review.
  - Each empty state should answer: what happened, why it matters, what to do next.
  - Polish the boring parts. Add helpful empty states for:
    - [ ] no provider key
    - [ ] no model preset
    - [ ] no assistant selected
    - [ ] no conversations
    - [ ] no search result
    - [ ] broken imported pack
    - [ ] stale attachment
    - [ ] missing tool arg
    - [ ] disabled provider
    - [ ] incompatible web search
    - [ ] missing skill
