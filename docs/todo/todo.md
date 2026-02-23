# Project TODO

## Laundry list

- [ ] opus 4.6 has now adaptive thinking which is basically thinking with levels
- [x] when we send attachments, like go files. openai is consuming a lot of tokens, i.e if tiktoken says x, is is almsot 2x generally. check if we are sending things double or something. req debug details doesnt show double atleast, but better print and check.
  - [x] xml encoding caused bloat and resend did not clean hydrated inputs.

## Features

- [ ] Agent skills
  - [x] skills tools: list, activate, readfile, run, deactivate
  - [x] skills runtime
  - [ ] skills discovery/add/remove/management backend
    - [x] store req/resp/code/test
    - [x] embedded fs skill provider for runtime. Approach:
      - [x] materialize the embedded fs containing skills into a read app data dir.
      - [x] no symlinks, skill name and dir name should match
      - [x] then use current fs skill provider.
    - [ ] runtime integration appropriately
      - [x] need agentskills to support allowlist of skills to get a prompt
      - [x] runtime lifecycle integration
      - [ ] runtime tools integration
    - [x] agentgo and httpbackend integration with api exposure
  - [x] skills discovery/add/remove/management ui
    - [x] spec types, skills and skill runtime
    - [x] bundles page, card, add/edit/view modal
    - [x] skills add/edit/view modal
  - [ ] skills in chat ui.
    - [x] most probably like websearch, enable button in bottom, with all available skills from catalog
    - [x] inside the dropdown we can have a select sub skill set for progressive disclosure
    - [ ] then in the prompt and then progressive disclosure
    - [ ] when the skills functionality is enabled we inject the available skills prompt, with the load tool.
    - [ ] From load tool, when something is activated, we inject that skills body in prompt and attach, read file, run script, unload tool.
    - [ ] lifecycle of when all unloaded vs some loaded etc needs to be managed.
    - [ ] also session per convo needs to be managed.
    - [ ] Do we need some builtin skills??
    - [ ] ~~Should ew expose current tools as skills (unnecessary redirection most probably?)?~~

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
    - [ ] Not needed as of now. May be a progressive disclosure runtime like skills will be better.
    - [ ] May be like skills we can also inject a available tools prompt in the sys prompt
  - [ ] We may want to have a explicit prompt saying that use explicit tools rather than shell wherever possible.

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
