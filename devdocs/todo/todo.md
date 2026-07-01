# Project TODO

## Laundry list

- Testing
  - [ ] test enhanced mcp apps.
  - [ ] test web search etc and pending user args etc after bottom bar migration.
  - [ ] test with some skill that has scripts too

## M-2

- [ ] Provider setup polish:
  - [ ] should have properly complete and verified presets
  - [ ] all models from leading providers including chinese ones via openrouter and hugging face as remote servers
  - [ ] local servers support
  - [x] anthropic, openai, google gemini, mistral, xai,
  - [x] hf, openrouter
  - [ ] ollama, vllm, lm studio, tgi, llmcpp, gpt4all

- [ ] better more inbuilt mcps.
  - MCP flows review and harden and more automatic in places where it is a bit jerky today.
  - mcp tools from leading providers presets completions.
    - Management: Atlassian, Notion
    - Documents: Box
    - Design: Canva, figma
    - Cloudflare: multiple like Docs, Workers, Observability, etc
    - Software dev: GitHub, Netlify, Stack Overflow, Vercel, Hugging Face
    - Jobs: Indeed
    - Search: Exa, Brave, BGPT
    - Google: Google Big Query, Google Compute Engine, Google GKE , Google Maps, [Other MCP servers](https://github.com/google/mcp), gmail, calender
    - MS: teams, 365
    - Slack
    - Zoom

- [ ] an attachment/tool and apply diff to code preset will be good. simple template and tool
  - [ ] here enhance the file name detection when we are using tool calls and also when we have openai style patch out rather than unified diff.

- [ ] github pr review preset

- [ ] better simple search inside dropdown selectors for context and presets

## M-3

- [ ] a folder selection/input an be given in context bar to say that your current work folder is so and so, so that any claude.md or skills or anything can be selected and auto injected as a "Starter recipe"

- [ ] llmtools: git tools

- [ ] Workflow sharing: easier add via some files/schemas etc. Assistant presets, Model presets, Skills, Tools, MCPs, Prompt templates etc.
  - [ ] easier preset import bundles and flows. e.g: just import a preset bundle that has assistant, models, prompts, tools etc.
    - [ ] may be as a json import bundle or jsonc format
    - [ ] better thing is to establish a jsonschema format for each thing, and then ship corresponding json as individual outputs.

- [ ] Docs clean: positioning wrt repeatable workflows and enhanced guidance
  - setup steps for local or custom models
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

- [x] Doc stores/vector stores connections. No need - MCPs serve this well.

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
