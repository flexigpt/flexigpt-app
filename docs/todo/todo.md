# Project TODO

## Laundry list

- [ ] chats
  - [x] context deadline exceeded error or may be other errors dont show up as content box? it comes out as minimal single line item.
  - [ ] perf:
    - [x] react virtuoso
    - [x] throttle time tuning,
    - [ ] offload some md parsing/rendering etc to worker
    - [ ] use sync external store for some stores and selectors for partial subscriptions.

- [ ] context bar
  - [ ] system prompts need to be non editor specific list i.e across tabs. also we should not have edit, we should have a fork button, figitbranch, and add button that has copy for existing anycase.
  - [ ] multi selection possibility. This needs to evolve as additional/multiple system prompts and a way to send a concatenated thing of all these together.
  - [ ] rather than ignore chat, we can have a combo drop down option, i.e include messages: all, only last message, ignore all, send last n messages i.e last 2/3/n etc including the user one that initiated it. we can make option names based on behaviour.

- [ ] Skill
  - [ ] include some builtin skills and test on all platforms, full flow.
  - [ ] with the number of skills present, maybe we dont want to enable all on click for the enable sills button, may be have a button at bottom on open that says enable all separately. also show collapsed bundles at start and then expanding if needed.
  - [ ] in skills world, especially with artifact driven dev, we can almost get away with without sending a lot of things again and again, latest context and last 1/2 chats. attachments may pose a problem here, but can be thought through properly so that these things are persisted properly.

- [ ] bottom bar
  - [ ] if you start a web search, then change model whose sdk is different that before, there is arbitrary behavior as of now wrt web search selection.

- [x] Drag and drop files as attachments.
  - [ ] while the code is there, on linux, webkitgtk has issue wrt firing drop events with wails currently.
  - [ ] win and mac testing is pending

- [ ] Sys prompt:
  - [ ] We may want to have a explicit prompt saying that use explicit tools rather than shell wherever possible.

## Features

- [ ] need to check if anthropic needs explicit caching setting (openai has implicit for 5 mins) so that tool calls loop is better.
  - [ ] this is a "feature" in anthropic and chargeable for cache write.
  - [ ] implement this after feature filters support and additional param support features in apis.

- [ ] Better Attachments
  - [ ] docx, excel support

- [ ] debug flag/s (maybe one for scrubbing, one for key activate etc etc) for provider in UI settings
- [ ] we may want to have a "assistant" like we planned before that has tool sets and autoexec config so that the "agent" loop is kind of autonomous

## Milestone thoughts

- [ ] M1 - API coverage - Pending items:
  - [ ] Modalities coverage:
    - [x] Text
      - [x] content in/out
      - [x] reasoning in/out
      - [x] extracted web pages input
      - [x] extracted pdf input
      - [ ] extracted other docs input, sheets and docx mainly.
    - [x] Image input
    - [x] Document input
    - [x] Image url input
    - [x] Document url input

  - [ ] Tools
    - [x] built-in tools from apis
      - [x] web search

    - [ ] local replacements for some builtin tools that are very vendor specific
      - [x] bash: yes.
      - [x] apply patch: No. this is very error prone, cosnidering unidiff vs V4A diff formats and compatibility issues.
      - [x] text editor: yes

    - [x] Dont: New stateful APIs and its hooks from vendors
      - [x] stored responses, stored conversations, on server memory context, on server prompt templates etc.

  - [ ] i18n

- [ ] M2 - Better context
  - [ ] MCP local connections and hooks
  - [ ] MCP options in apis connections and hooks
  - [ ] Doc stores/vector stores connections
    - [ ] Only if MCP cannot serve this.

- [x] Agent Skills but via local "skills" flow???

- [ ] Deferred.
  - [ ] Image output: See inference-go notes.
  - [ ] audio in/out
  - [ ] check the reference calculator tool in claude docs
  - [ ] tool search tool: we may need a tool search tool that does sqlite based bm25 search or regex search like from anthropic
    - [ ] Not needed as of now. May be a progressive disclosure runtime like skills will be better.
    - [ ] May need to check when there are actually a lot of tools.
    - [ ] May be like skills we can also inject a available tools prompt in the sys prompt
