# Project TODO

## Laundry list

- [ ] tools should have a configuration which says can autoexecute i.e without user consent vs not.
  - [ ] with this config we can have a "agent loop" of sort for file edits/mods etc
  - [ ] major question remains as to what sort of tools should be auto exec vs not. write anything being human in loop is safest anycase. for write ones, we may want to see if we need to have a "keep old file as renamed" with some sessionid/tmp extension so that reverting is easier.
  - [ ] a tool call only and tool output only message may be rendered as a single line after this.
  - [ ] we may want to have a "assistant" like we planned before that has tool sets and autoexec config so that the "agent" loop is kind of autonomous

  - [x] backend to have flag for recommendation per tool
  - [x] at choice time ui can show autoexecute true/false and set default to provided one and then allow to change.
    - [ ] Right now you can set it at insertion time (picker), and see it in details, but you can’t toggle it later from the “Tools” chip menu.
    - [ ] toggle has a delayed view effect, i.e doesnt really show immediately. if you click it doesnt change, but if you click next, then it changes.
    - [ ] tool menu doesnt show tick when selected. also this should be grid based display, the tick, the auto toggle aligned to some col, and version.

  - [ ] rather that drop down in ui for toolcalls we need to show them as visible chips like citations if there is some "content" (not jsut thinking content)
    - [ ] the type of chip shown can be specific to tool call like if toolcall is of type replacetext, we can have chip that shows diff and have run button there too. or generic if no other chip present.
    - [ ] same can be for result but as of now we dont have tool results with special ui reqs.

  - [ ] UI needs to be so that we dont waste a lot of space for tool call only or tool result only cards.

- [ ] editor is starting to look very very heavy, review to see for better refactoring and perf opt.
- [ ] When url cannot fetch content, there is no way of knowing what happened as of now. may want to see how to expose this or disable link only mode in this flow?

## Features

- [ ] there is a reference calculator tool in claude docs

- [ ] valid input output modalities, valid levels, valid reasoning types, etc need to be added to modelpresetspec.

- [ ] Token count in build completion data may be wrong as it doesn't really account for attachments/tool calls etc.
  - [ ] Need to rectify the FilterMessagesByTokenCount function. Look at using gpt5 bpe encoder now rather than regex compilation. [go-tokenizer](https://github.com/tiktoken-go/tokenizer)
  - [ ] Claude has a free token counting api with some rate limits. may need to check it out too.

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
  - [ ] provider and model level allow disallow list of model params/capabilities etc.
  - [ ] Some more additional params in presets and advanced params modal.
    - [ ] tool choice tuning
    - [ ] verbosity tuning
    - [ ] top k
    - [ ] Not sure: Safety parameter, that identifies a user if they violate safety policies.
    - [ ] Not sure: stop strings
    - [ ] Not sure: cache control in claude

- [ ] M2 - Better context
  - [ ] MCP local connections and hooks
  - [ ] MCP options in apis connections and hooks
  - [ ] Doc stores/vector stores connections
    - [ ] Only if MCP cannot serve this.

- [ ] Agent Skills but via local "explorer" or "skills" flow???

- [ ] Deferred.
  - [ ] Image output: See inference-go notes.
  - [ ] audio in/out
