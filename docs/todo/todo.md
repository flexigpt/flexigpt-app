# Project TODO

## Laundry list

- [ ] Skill
  - [ ] include some builtin skills and test on all platforms, full flow.
  - [ ] with the number of skills present, maybe we dont want to enable all on click for the enable sills button, may be have a button at bottom on open that says enable all separately. also show collapsed bundles at start and then expanding if needed.
  - [ ] in skills world, especially with artifact driven dev, we can almost get away with without sending a lot of things again and again, latest context and last 1/2 chats. attachments may pose a problem here, but can be thought through properly so that these things are persisted properly.

- [ ] Sys prompt:
  - [ ] may also need a default bundle for editable instructions only sys prompts that people can add to directly from chat.

- [ ] context bar
  - [ ] tooltips are needed. ariakit ones as daisyui ones can get cutoff

- [ ] Tool calls customize
  - [ ] Some calls like editor replace text, create files, etc can be visually represented better via custom elements representing each.

## Features

- [ ] need to check if anthropic needs explicit caching setting (openai has implicit for 5 mins) so that tool calls loop is better.
  - [ ] this is a "feature" in anthropic and chargeable for cache write.
  - [ ] implement this after feature filters support and additional param support features in apis.

- [ ] debug flag/s (maybe one for scrubbing, one for key activate etc etc) for provider in UI settings

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

  - [ ] i18n

- [ ] M2 - Better context
  - [ ] Text support
    - [ ] extracted other docs input, sheets and docx mainly.
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

## Chores: Package upgrade

- [ ] Upgrade sequence:
  - [ ] task checkupgrade; go-mod-upgrade -v ; pnpm up -r

- [ ] eslint 10: pending plugins support.
  - [ ] monitor oxclint for a while. ecosystem seems weak as of now.

- [ ] vite 8: react router dev support. react router has some cleanup planned with vite8. would be better to wait until react router stamps it as ok.

- [ ] jsdom next major
