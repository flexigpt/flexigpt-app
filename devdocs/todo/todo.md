# Project TODO

## Laundry list

- [ ] Need more cleanup wrt shortcuts, input tips, shortcuts to add mcp, skills etc.
- [ ] also need to update docs for mcp and diff render support.
- [ ] Add diff out guidance to md sys prompt

- [ ] need a file ops only assistant, may be add diff to other text things in assitants.

- [ ] better more inbuilt mcps. test enhanced mcp apps.

- [ ] test web search etc and pending user args etc after bottom bar migration.
- [ ] test with some skill that has scripts too

## Milestone thoughts

- [ ] M-Future
  - [ ] i18n
  - [ ] Better context
    - [ ] Text support
      - [ ] extracted other docs input, sheets and docx mainly.
    - [ ] Doc stores/vector stores connections: Only if MCP cannot serve this.

- [ ] Deferred.
  - [ ] Image output: See inference-go notes.
  - [ ] audio in/out
  - [ ] check the reference calculator tool in claude docs
  - [ ] App SDK, etc
  - [ ] MCP options in server apis connections and hooks.
    - [ ] This app should focus on local first with stateless api behavior rather than stateful, vendor specific server side operations.
  - [ ] tool search tool: we may need a tool search tool that does sqlite based bm25 search or regex search like from anthropic
    - [ ] Not needed as of now. May be a progressive disclosure runtime like skills will be better.
    - [ ] May need to check when there are actually a lot of tools.
    - [ ] May be like skills we can also inject a available tools prompt in the sys prompt

  - [ ] a folder selection/input an be given in context bar to say that your current work folder is so and so, so that any claude.md or skills or anything can be selected and auto injected as a "Starter recipe"

  - easier add via some files/schemas etc
    - [ ] easier preset import bundles and flows. e.g: just import a preset bundle that has assistant, models, prompts, tools etc.
      - [ ] may be as a json import bundle or jsonc format
      - [ ] better thing is to establish a jsonschema format for each thing, and then ship corresponding json as individual outputs.

    - [ ] easier add of skill.
    - [ ] easier add of mcp.

## Chores: Package upgrade

- [ ] Upgrade sequence:
  - [ ] task checkupgrade; go-mod-upgrade -v ; pnpm up -r

- [ ] eslint 10: pending plugins support.
  - [ ] monitor oxclint for a while. ecosystem seems weak as of now.

## Observations

- openai tool loop
  - [ ] openai tool calls does a deep dive a lot, for flexigpt, it goes from backend to frontend to others. i think there needs to be a way to control its spread. most probably need to have a way to say that do step by step, and identify domains and then do things etc.
  - [ ] in a tooling session, there are no parallel tool calls being made for some reason. better force the caller to say that do a bfs kind of tool calls.
  - [ ] running linters tests etc is shell driven, the agents.md or similar file may help to say what are commands. for task file it may be simpler, but ned a way of discovery and exec.
  - [ ] asking for things that are missing should be encouraged in some ways. most probably llms are now more in "agentic" mode so they dont really ask questions, but we can shine a lot with human in loops rather than direct.

- [ ] once nice thing i noticed that may or may not be available in others, in flexigpt, i can remove/add tools if there are errors etc. and llm may work with base tools i.e read/write/delete files are unambiguous
- [ ] i think we should have a "files only" assistant as well.
- [ ] conversation level tokens would be nice to have.

- [ ] I am linking this loop of implementing this using tool template loop (may be skills at some point), and then observing issues, then fixing these, then picking other feature and doing the same.
- [ ] An inbuilt docs pages and then some ai intro tutorial builtin will be good.

- [ ] in skills world, especially with artifact driven dev, we can almost get away with without sending a lot of things again and again, latest context and last 1/2 chats. attachments may pose a problem here, but can be thought through properly so that these things are persisted properly.

- [ ] lots of persona like templates can be added for sys prompt templates.

- [ ] Tool calls
  - [ ] Some calls like editor replace text, create files, etc can be visually represented better via custom elements representing each.

- [ ] an apply diff to code preset will be good. simple template and tool
- [ ] composer: see if grammar rectification or atleast spell check and highlight can be added cleanly

## Other repos thoughts

- [ ] llm tools
  - [ ] may be pdf parsing and any other parsing should be builder hooks that people can input at build time so that adding support etc is easy.

## Model analysis

- [ ] model providers with dedicated platforms:
  - [ ] openai -> best use responses api -> done.
  - [ ] anthropic -> use messages api -> done.
  - [ ] google gemini -> use generate content api -> done.
  - [ ] xai -> responses api, 4.x doesnt support thinking as such, "inbuilt" thinking is what they say, so no control
  - [ ] mistral -> chat completions compatible but looks a bit different. check cleanly again.

  - [ ] cohere -> non sota inference
  - [ ] meta api -> chat completions, but getting api keys is in closed beta as of now.

- [ ] Aggregate API endpoint providers:
  - [ ] HF -> responses beta, completions stable
  - [ ] Openrouter -> responses beta, completions stable

- [ ] Aggregate Cloud services with AI deployments
  - [ ] Amazon -> (non-sota models exist) -> needs aws specific converse/conversestream sdk intergation.
  - [ ] Azure -> (non-sota models exist) -> more chat completions like. needs a adapter.
  - [ ] nvidia nim/dgx cloud -> (non-sota models exist) - ??
  - [ ] Oracle -> ??

- [ ] Chinese model providers with dedicated platforms: GLM (z ai), minimax, qwen (alibaba), kimi (moonshot ai), deepseek, yi models (01.ai), StepFun
  - [ ] some specialize dedicated platforms for these have very slow access/login.
  - [ ] data mostly sent to chinese infra and can be accessed by chinese govt.
  - [ ] most privacy policy dictates that data will be used for training.
  - [ ] better access models via openrouter/hf/other aggregators as needed

- [ ] Local providers:
  - [ ] llama.cpp -> chat completions
  - [ ] LMStudio, Ollama -> anthropic messages, responses, chat completions, all are supported.
  - [ ] vLLM -> responses, chat completions
  - [ ] GPT4All -> chat completions
