# Project TODO

## Laundry list

- Testing
  - [ ] test enhanced mcp apps.
  - [ ] test web search etc and pending user args etc after bottom bar migration.
  - [ ] test with some skill that has scripts too
  - [ ] test workspace
  - [ ] test all management pages again properly

- [x] times on resp etc will be good i.e we should show timings of one round trip that happened. mostly in the footer someplace. just assistant round trip time would be good.
- [x] title rename is not working in ui i.e if i press edit button on title and rename then i do not get any change anyhwere. there is no backend error
- [x] plate doc has weird behavior with cursor. sometimes it goes up and sometimes it doesn't, this is when we use arrows etc and there is already a multi line input. this looks like happening during streaming in other tab most probably. may be normally too, but streaming is confirmed issue.
- [x] if i say cancel a request i.e from stop button, it comes as context cancelled now and not actual error like before. also text or thinking that was streamed is not retained in ui and it shows full assistant replaced with context cancelled. we need to show actual error as appended text with italics and some color error to the bottom of any already streamed thinking or content. that shoudl not go away until user actually closes it. in the backend the streamed one should be retained until either user edits a previous message and sends so we start a new chain or let the partial response be there as empty on error.

- [ ] add as code in md output format skill too
- [ ] add wails considerations etc skill in my repo.

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
