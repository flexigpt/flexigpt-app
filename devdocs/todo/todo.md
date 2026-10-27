# Project TODO

## Laundry list

- Testing
  - [ ] test enhanced mcp apps.
  - [ ] test web search etc and pending user args etc after bottom bar migration.
  - [ ] test with some skill that has scripts too
  - [ ] test workspace
  - [ ] test all management pages again properly

- [ ] add wails considerations skill in flexigpt repo.

- [x] as of now in agent declarations we show the md body of skill, but it should ideally show the actual md source i.e frontmatter and everything.
- [x] all skills and other artifacts should always have a copy/export button across the board. this reinforces our artifact driven operation model celanly. the artifacts should be rendered uniformly across the board as a copy/export text which is lazy loaded when someone clicks show or export or copy. may be each artifact domains minimal renderer can be inside that folder and then used everwher as needed. liek skill inlien renders can be in app/skills and then used in skill, composer, agent, workspace etc. focus shoudl be to create proper common components in proper palces and then use it uniformly.
- [x] management ui should be uniform with minimal overrides and ideally consumers shoudl be uniformized to use simialr things rather than doing their own things.
- [x] template args are not rendered for some reason now in template modal. also the modal doesnt show the actual template with placeholders etc.
- [x] code delivery skill should enforce ownership cost sensitivity and max parallelism properly.
- [x] when streaming diff apply paths are not populated.

- [ ] verify new management ui in all pages
- [ ] verify diff applicability in streaming

- [ ] add a default agent selection preference and a default workspace one

- [ ] anthropic released sdk v2 - temperature is not there in sdk, most probably cannot use current api version with new sdk.

## M-3

- [ ] see if artifactory needs to be a repo. same with llm artifacts. and then complete it with agentruntime repo.

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

## M-Future

- [ ] i18n

- [ ] aggregate tokens and may be integration with usage pais will be good

- [ ] Better Text support: extracted other docs input, sheets and docx mainly.
  - [ ] llmtools: may be pdf parsing and any other parsing should be builder hooks that people can input at build time so that adding support etc is easy.

- [ ] Image output: See inference-go notes.
- [ ] audio in/out
- [ ] check the reference calculator tool in claude docs
- [ ] App SDK, etc
- [ ] MCP options in remote server apis connections and hooks.
  - [ ] This app should focus on local first with stateless api behavior rather than stateful, vendor specific server side operations.

- [ ] tool search tool: we may need a tool search tool that does sqlite based bm25 search or regex search like from anthropic
  - [ ] Not needed as of now. May be a progressive disclosure runtime like skills will be better.
  - [ ] May need to check when there are actually a lot of tools.
  - [ ] May be like skills we can also inject a available tools prompt in the sys prompt

- [ ] conversation level tokens would be nice to have.
- [ ] An ai intro tutorial builtin will be good.
- [ ] Tool calls: Some calls like editor replace text, create files, etc can be visually represented better via custom elements representing each.
- [ ] composer: see if grammar rectification or atleast spell check and highlight can be added cleanly

- Multi Model parallel comparison and evals
  - [ ] send same prompt to multiple models
  - [ ] compare responses side-by-side
  - [ ] show token usage
  - [ ] show latency
  - [ ] show estimated cost
  - [ ] save comparison result
  - [ ] reusable evaluation prompts

- Usage and cost dashboard: Because users bring their own keys, help them understand usage.
  - [ ] per-message token usage
  - [ ] per-conversation token usage
  - [ ] per-provider usage
  - [ ] estimated cost
  - [ ] latency
  - [ ] model comparison cost
  - [ ] export usage CSV
