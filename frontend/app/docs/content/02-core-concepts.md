# Core concepts

FlexiGPT is much easier to use once a few core terms are clear.

## The main terms

| Term                    | Meaning                                                                                                                                                   |
| ----------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Provider**            | The API family or endpoint you are talking to, such as OpenAI, Anthropic, Gemini, OpenRouter, `llama.cpp`, or a compatible custom endpoint.               |
| **Assistant Preset**    | A higher-level starting setup that can select a model preset and also preload instructions, tools, and skills.                                            |
| **Model Preset**        | A saved model choice plus defaults such as streaming, token limits, temperature or reasoning, timeout, output settings, and provider-specific parameters. |
| **Prompt Template**     | Reusable prompt content, sometimes with variables that must be filled before send.                                                                        |
| **System Prompts**      | The combined instructions that shape model behavior for the request.                                                                                      |
| **Previous user turns** | The number of earlier user messages (and attached context) that should be resent with the next request.                                                   |
| **Tool**                | A callable capability that can be attached to a LLM request, either with autoexecute and submit, or manual review, execute and submit.                    |
| **Skill**               | A reusable workflow or behavior package for a conversation.                                                                                               |
| **Attachment**          | A file, image, PDF, folder-derived file set, or URL attached to a message.                                                                                |

## Assistant presets and Model presets and are different on purpose

These two layers solve different problems.

### Assistant Presets answer

"What complete starting workspace do I want for this kind of task?"

An assistant preset can capture:

- a starting model preset
- instruction-style prompts
- tool selections
- skill selections

That makes assistant presets useful when you repeat the same kind of work and want the same starting setup each time.

### Model Presets answer

"What model and inference defaults should I use?"

A model preset is about execution details such as:

- provider and model identity
- streaming behavior
- token limits
- temperature or reasoning defaults
- timeout
- output format or verbosity controls
- provider-specific advanced parameters

## What happens when you click Send

When you send a message, FlexiGPT builds a request from the current chat state.

That request can include:

- the current user message
- selected model preset values
- resolved system prompt content
- selected prompt template output
- earlier user context allowed by **Previous user turns**
- attachments that belong to included messages
- tool choices and tool outputs
- skill session state, when relevant

The exact payload depends on the selected provider family because different providers expose different capabilities.

## Human-in-loop and agentic tool use

FlexiGPT supports both manual and more agentic tool workflows.

- In a **human-in-loop** setup, the model can propose tool calls and you review or trigger execution yourself and submit.
- In an **agentic** setup, you can mark a tool for **auto-execute**.

When an auto-execute tool is called, FlexiGPT can:

1. run the tool automatically
2. capture the tool result
3. submit that result back to the model without requiring a separate manual send

This lets you build more automated tool-driven flows inside the normal chat workspace, while still keeping manual control available when execution should be reviewed first.

## How memory and context work

FlexiGPT does not blindly resend the entire conversation every time.

The **Previous user turns** control is one of the most important context settings in the app:

- use a small value when the thread has drifted
- use a larger value when the current task depends on earlier decisions
- use `0` when you want an isolated turn in the same conversation
- use `All` only when the whole thread is still relevant

A key rule is that attachments belong to the message they were sent with. They are not stored in a hidden global bucket. If an older user turn is not included next time, its attachments are not automatically resent either.

## A useful mental model

You can think of FlexiGPT as four layers:

1. **Transport layer**: provider and endpoint compatibility
2. **Inference layer**: model preset and advanced request settings
3. **Behavior layer**: assistant preset, system instructions, prompt templates, and skills
4. **Context layer**: current message, previous user turns, attachments, and tool outputs

When results change, it is usually because one of those layers changed.
