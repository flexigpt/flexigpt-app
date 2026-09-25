# Project TODO

## Laundry list

- [x] in go 1.27 here is some issue related to double escaped strings. you need to clean json first and then decode actually. need to debug deeply
- [ ] check similar anything for other things i.e json string thing.
- [ ] in diff ui, when there is streaming ongoing the diffs in previous message say need info and dont resolve paths or apply.
- [ ] if a new file creation is there in diff of multi files it doesnt work.
- [ ] the modal has bugs when multi file things are done. mostly some files are nto shown at all. some files which are shown may or may not be dry run. the error in file is displayed at top and nto near dry run or same file. apply file similalry at top is problematic in message.
- [ ] copy button in different tab seems to lag when there is some other streaming etc going on.

- [ ] in thinking send in inference, make sure that the first message sent doesnt have encrypted thinking or any such input.
- [ ] submit error is there even if there is no run done yet for tool.
- [ ] loading skills and mcps etc take a lot of time. may be a short circuit path that can actually compile and validate and keep things at compile time itself is needed?
- [ ] mcps that are not really configured are shown. we should not do such thing. only configured and connect pending things should be shown. other can be just a small line saying, configuration pending manage from managements page.
- [ ] mcp chip in messags is taller than others. why it uses different chip i dont know. also some ids etc that are internal are used in ui
- [ ] no user message skills available??
- [ ] system prompts didnt seem to get loaded, shows "no system instruction sources"

- Testing
  - [ ] test enhanced mcp apps.
  - [ ] test web search etc and pending user args etc after bottom bar migration.
  - [ ] test with some skill that has scripts too

## M-3

- [x] a folder selection/input an be given in context bar to say that your current work folder is so and so, so that any claude.md or skills or anything can be selected and auto injected as a "Starter recipe"

- [ ] Workflow sharing: easier add via some files/schemas etc. Assistant presets, Model presets, Skills, Tools, MCPs, Prompt templates etc.
  - [ ] easier preset import bundles and flows. e.g: just import a preset bundle that has assistant, models, prompts, tools etc.
    - [ ] may be as a json import bundle or jsonc format
    - [ ] better thing is to establish a jsonschema format for each thing, and then ship corresponding json as individual outputs.

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

- Multi Model parallell comparison and evals
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
