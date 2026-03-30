# Project TODO

## Laundry list

- [ ] Skill
  - [ ] include some builtin skills and test on all platforms, full flow.
  - [ ] with the number of skills present, maybe we dont want to enable all on click for the enable sills button, may be have a button at bottom on open that says enable all separately. also show collapsed bundles at start and then expanding if needed.
  - [ ] in skills world, especially with artifact driven dev, we can almost get away with without sending a lot of things again and again, latest context and last 1/2 chats. attachments may pose a problem here, but can be thought through properly so that these things are persisted properly.

## Features

- [ ] need to check if anthropic needs explicit caching setting (openai has implicit for 5 mins) so that tool calls loop is better.
  - [ ] this is a "feature" in anthropic and chargeable for cache write.
  - [ ] implement this after feature filters support and additional param support features in apis.

- [ ] Tool calls
  - [ ] Some calls like editor replace text, create files, etc can be visually represented better via custom elements representing each.

## Milestone thoughts

- [ ] M1 - API coverage - Pending items:
  - [x] Modalities coverage:
    - [x] Text
      - [x] content in/out
      - [x] reasoning in/out
      - [x] extracted web pages input
      - [x] extracted pdf input

    - [x] Image input
    - [x] Document input
    - [x] Image url input
    - [x] Document url input

  - [x] Tools
    - [x] built-in tools from apis
      - [x] web search

    - [x] local replacements for some builtin tools that are very vendor specific
      - [x] bash: yes.
      - [x] apply patch: No. this is very error prone, cosnidering unidiff vs V4A diff formats and compatibility issues.
      - [x] text editor: yes

    - [x] Dont: New stateful APIs and its hooks from vendors
      - [x] stored responses, stored conversations, on server memory context, on server prompt templates etc.

  - [x] Agent Skills but via local "skills" flow

- [ ] M-Future
  - [ ] App SDK, etc
  - [ ] i18n
  - [ ] Better context
    - [ ] Text support
      - [ ] extracted other docs input, sheets and docx mainly.
    - [ ] MCP local connections and hooks
    - [ ] MCP options in apis connections and hooks
    - [ ] Doc stores/vector stores connections: Only if MCP cannot serve this.

- [ ] Deferred.
  - [ ] Image output: See inference-go notes.
  - [ ] audio in/out
  - [ ] check the reference calculator tool in claude docs

  - [ ] tool search tool: we may need a tool search tool that does sqlite based bm25 search or regex search like from anthropic
    - [ ] Not needed as of now. May be a progressive disclosure runtime like skills will be better.
    - [ ] May need to check when there are actually a lot of tools.
    - [ ] May be like skills we can also inject a available tools prompt in the sys prompt

## Chores: Package upgrade

- [ ] Upgrade sequence:
  - [ ] task checkupgrade; go-mod-upgrade -v ; pnpm up -r

- [ ] eslint 10: pending plugins support.
  - [ ] monitor oxclint for a while. ecosystem seems weak as of now.

- [ ] vite 8: react router dev support. react router has some cleanup planned with vite8. would be better to wait until react router stamps it as ok.

- [ ] jsdom next major

## Observations

- take some action
  - [ ] attachments need to include absolute path most probably. think through and see how to fit it. tool calls need some way to say that start from this.

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

## Other repos thoughts

- [ ] llm tools
  - [x] search files (and may be others too), need to explicit setting to exclude hidden files/folders. (may be list dir too.)
  - [x] read/replace text utils, may need to support approx line numbers so that llm can be more specific on where to read / write etc.
    - [x] replace has lots of ambiguous mismatch errors. need a way to narrow down, and have errors better communicated back that are actionable.
  - [ ] may be pdf parsing and any other parsing should be builder hooks that people can input at build time so that adding support etc is easy.
