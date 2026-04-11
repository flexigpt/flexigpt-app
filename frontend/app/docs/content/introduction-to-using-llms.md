Large Language Models, or **LLMs**, are systems that read text and generate text. In FlexiGPT, you interact with LLMs through different providers and models.

## Core terms

| Term            | Meaning                                                                                                                 |
| --------------- | ----------------------------------------------------------------------------------------------------------------------- |
| **LLM**         | Large Language Model. A model that can understand and generate language.                                                |
| **Provider**    | The service or company that hosts a model, such as OpenAI, Anthropic, Google, or a compatible local or remote endpoint. |
| **Model**       | The specific AI system you send messages to.                                                                            |
| **Prompt**      | The instruction or question you send to the model.                                                                      |
| **Context**     | The information included with a request, such as earlier messages, system instructions, and attachments.                |
| **Tokens**      | Units of text used for model limits, generation budgets, and provider billing.                                          |
| **Temperature** | A control for randomness. Lower values are usually more predictable and higher values are usually more exploratory.     |
| **Reasoning**   | Extra model thinking behavior or reasoning budget on models that support it.                                            |
| **Streaming**   | Showing the answer as it is generated instead of waiting for the full response.                                         |
| **Attachment**  | A file, image, PDF, directory, or URL added to the conversation as supporting context.                                  |

## A simple mental model

When you send a message in FlexiGPT, the app packages your prompt, the conversation history, the selected model settings, and any attachments into a request for the chosen provider. The model then generates a response and streams it back into the chat view.

## Good first use cases

- Summarizing notes, articles, or pasted text
- Brainstorming ideas or outlines
- Rewriting text for tone or clarity
- Explaining unfamiliar code or concepts
- Drafting first-pass documentation or emails

## Things to remember

- LLMs can sound confident even when they are wrong.
- Results can change when you switch providers, models, or settings.
- Better context usually leads to better answers.
- Clear instructions often matter more than long instructions.
