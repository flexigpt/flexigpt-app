# Project Future notes

## Future feature ideas

- [ ] i18n

- [ ] Better Text support: extracted other docs input, sheets and docx mainly.
  - [ ] llmtools: may be pdf parsing and any other parsing should be builder hooks that people can input at build time so that adding support etc is easy.

- [ ] Image output: See inference-go notes.
- [ ] audio in/out
- [ ] check the reference calculator tool in claude docs

- [ ] MCP options in remote server apis connections and hooks.
  - [ ] This app should focus on local first with stateless api behavior rather than stateful, vendor specific server side operations.

- [ ] tool search tool: we may need a tool search tool that does sqlite based bm25 search or regex search like from anthropic
  - [ ] Not needed as of now. May be a progressive disclosure runtime like skills will be better.
  - [ ] May need to check when there are actually a lot of tools.
  - [ ] May be like skills we can also inject a available tools prompt in the sys prompt

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
  - [ ] aggregate tokens and may be integration with usage page will be good. conversation level tokens would also be nice to have.

- [ ] Token count in build completion data may be wrong as it doesn't really account for attachments/tool calls etc.
  - [ ] Need to rectify the FilterMessagesByTokenCount function. Look at using gpt5 bpe encoder now rather than regex compilation. [go-tokenizer](https://github.com/tiktoken-go/tokenizer)
  - [ ] Claude has a free token counting api with some rate limits. may need to check it out too.

- [ ] Observability: At least tokens and dollars need to be represented someplace
- MCP handling
  - [ ] MCP transport: mcp httpsse, stdio for mcp
  - [ ] long future: add servers for: file, git, simplemapstore
- [ ] Docs inside app
  - [ ] Have docs pages inside app itself
  - [ ] Docs should ideally have course work for very beginners that are users of genai, without any programming knowledge.
  - [ ] MS course is fantastic, but requires lots of tech know how.
- [ ] Add keyboard shortcuts and some tooltips for them to know what are these.
  - [ ] Add AWS bedrock too. May be anthropic api as of now for claude on bedrock models?
  - [x] google dedicated sdk
  - [ ] Check vertex ai for google: needs google dedicated sdk

- [ ] Think through and enhance tools create/edit and prompts create/edit modals
  - [ ] Implement a schema area using CodeMirror for defining the tool's/prompts schema. (deferred)
    - [ ] Configure CodeMirror to support JSON syntax highlighting and validation.
    - [ ] Allow users to define input and output parameters using JSON schema.
  - [ ] Implement a function area using CodeMirror for accepting/implementing the schema.
    - [ ] Configure CodeMirror to support TypeScript syntax highlighting and validation.
    - [ ] Allow users to write an asynchronous TypeScript function that utilizes the defined schema.

- Chat attachments
  - [x] Future: May need to decide later that keeping conversation attachment shapshots is better than reattach etc.
  - [x] Future: Check for race between build and fetch for attachment content fetch and how to resolve. For now this is ok to have as the time between these 2 is very small. If we introduce human feedback in between, we ned to look at this closely.

  - [ ] Future: Attachments support
    - [ ] doc formats can be in one place
    - [ ] specialized urls like github/gitlab pr's
    - [ ] local git diff/commit history/etc attachments

- [x] Drag and drop files as attachments.
  - [ ] while the code is there, on linux, webkitgtk has issue wrt firing drop events with wails currently.
  - [ ] win and mac testing is pending

## Laundry list

- [ ] Better release notes

- [ ] Chat
  - [x] Details of request are not visible immediately. We use a trapper in golang ai provider package to get req/resp details, that means until the completion returns there is no way of getting the request details. Some refactoring of workflow is needed to make this happen.
  - [x] Need a stop streaming / cancel button on assistant
  - [ ] Think through:
    - [x] In AI replies, find a way to mention the model used and special params a bit more explicitly.
    - [x] Check whether current details is good enough UX or need a bit better subtle ux.
  - [ ] Support for image/audio in chat
  - [ ] Slash commands for prompts/tools/kb in chat
    - [ ] Should model selection also be in chat commands?
    - [ ] may be can be done as: @ invokes assistants. inbuilt models with current settings are default assistants.
  - [ ] support for read/attach file in chats and then write/save file (slightly different flow than download)
  - [ ] Think if needed:
    - [ ] Users can mark a query to be not included in the chat message conversation history when talking to llm.
    - [ ] This is query level control which is different from the generic control.
    - [ ] Ignore history command structure:
      - [ ] Global on/off switch like right now
      - [ ] Query level switch overrides the global switch.

- [ ] Windows
  - [ ] Signing for Win
  - [ ] Test Windows build

- [ ] When url cannot fetch content, there is no way of knowing what happened as of now. may want to see how to expose this or disable link only mode in this flow?
  - [ ] as of now content blocks are errored out only during attach tme and not during build time.
  - [ ] whenever this is to be done, we need to see what is best ux and then design thngs accordingly.
  - [ ] early thought is that errors definitly need to be consdiered during attach time, at that time, we should do as much as possible to tell user that something is wrong. but once that stage is done, i.e non error in attach time, may be assumethat things wil lbe correct and log and skip in next stage.

## Stricken out list

- [ ] ~~logger is imported in securejsondb before it is set as ipc file logger in appimage~~
- [ ] ~~Electron: currently supported only via appimagelauncher. Better do snap I suppose, but explore later.~~
- [ ] ~~Electron: see about package distribution and updates~~
- [ ] ~~Code interpreter for few languages baked in. This can be provided as a normal tool so that~~
- [ ] ~~Container build and publish~~
