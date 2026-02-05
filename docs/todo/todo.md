# Project TODO

## Laundry list

- [x] if there is a very large table in md, the ui hangs.
  - [x] scroll conflicts fixed
- [ ] if there are attachments in previous message and then it was changed and then we resend it, the app crashes without trace.
- [ ] mdx is not readable

## Features

- [ ] ModelParams enhancements
  - [ ] valid input output modalities, valid levels, valid reasoning types, etc need to be added to modelpresetspec.
  - [ ] additional params from apis
    - [x] support in inference go api
    - [ ] adapt in model preset spec and provider set

- [ ] Better Attachments
  - [ ] docx, excel support
  - [ ] Drag and drop files as attachments.

- [ ] Tools
  - [ ] tool search tool
  - [ ] skills tools: list, activate, readfile, run, deactivate

- [ ] we may want to have a "assistant" like we planned before that has tool sets and autoexec config so that the "agent" loop is kind of autonomous

- [ ] need to check if anthropic needs explicit caching setting (openai has implicit for 5 mins) so that tool calls loop is better.
  - [ ] this is a "feature" in anthropic and chargable for cache write.
  - [ ] implement this after feature filters support and additional param support features in apis.

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
      - [ ] tool search tool: we may need a tool search tool that does sqlite based bm25 search or regex search like from anthropic

    - [x] Dont: New stateful APIs and its hooks from vendors
      - [x] stored responses, stored conversations, on server memory context, on server prompt templates etc.

  - [ ] i18n

- [ ] M2 - Better context
  - [ ] MCP local connections and hooks
  - [ ] MCP options in apis connections and hooks
  - [ ] Doc stores/vector stores connections
    - [ ] Only if MCP cannot serve this.

- [ ] Agent Skills but via local "explorer" or "skills" flow???

- [ ] Deferred.
  - [ ] Image output: See inference-go notes.
  - [ ] audio in/out
  - [ ] check the reference calculator tool in claude docs
