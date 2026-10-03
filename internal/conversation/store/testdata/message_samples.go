package testdata

import (
	"time"

	conversationSpec "github.com/flexigpt/flexigpt-app/internal/conversation/spec"
)

const tmpMessageDetails = `
start of details

` + "```dart" + `
class ChatScreen extends StatefulWidget {
  const ChatScreen({super.key});

  @override
  ChatScreenState createState() => ChatScreenState();
}
` + "```" + `

there is a ` + "`console`" + `

` + "```py" + `
def myFunc() {
  return "me";
}
` + "```" + `
`

func newTimePointer(
	year int,
	month time.Month,
	day, hour, min, sec, nsec int,
	loc *time.Location,
) *time.Time {
	t := time.Date(year, month, day, hour, min, sec, nsec, loc)
	return &t
}

func newStringPointer(s string) *string {
	t := s
	return &t
}

var MessageSamplesListBase = []conversationSpec.ConversationMessage{
	{
		ID:        "1",
		CreatedAt: newTimePointer(2023, 9, 24, 8, 30, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleSystem,
		Content:   "Welcome to our chat application!",
	},
	{
		ID:        "2",
		CreatedAt: newTimePointer(2023, 9, 24, 8, 31, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleUser,
		Content:   "Hello! I need help with my order.",
		Name:      newStringPointer("John Doe"),
	},
	{
		ID:        "3",
		CreatedAt: newTimePointer(2023, 9, 24, 8, 32, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleAssistant,
		Content:   "Of course, John. Can you provide your order number?",
		Name:      newStringPointer("Assistant"),
	},
	{
		ID:        "4",
		CreatedAt: newTimePointer(2023, 9, 24, 8, 33, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleUser,
		Content:   "My order number is 12345.",
		Name:      newStringPointer("John Doe"),
	},
	{
		ID:        "5",
		CreatedAt: newTimePointer(2023, 9, 24, 8, 34, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleAssistant,
		Content:   "Thank you. I found your order. How can I assist you further?",
		Name:      newStringPointer("Assistant"),
	},
	{
		ID:        "6",
		CreatedAt: newTimePointer(2023, 9, 24, 8, 35, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleUser,
		Content:   "I want to change the delivery address.",
		Name:      newStringPointer("John Doe"),
	},
	{
		ID:        "7",
		CreatedAt: newTimePointer(2023, 9, 24, 8, 36, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleAssistant,
		Content:   "Sure, please provide the new address.",
		Name:      newStringPointer("Assistant"),
	},
	{
		ID:        "8",
		CreatedAt: newTimePointer(2023, 9, 24, 8, 37, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleUser,
		Content:   "123 New St, Springfield.",
		Name:      newStringPointer("John Doe"),
	},
	{
		ID:        "9",
		CreatedAt: newTimePointer(2023, 9, 24, 8, 38, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleAssistant,
		Content:   "The address has been updated.",

		Name: newStringPointer("Assistant"),
	},
	{
		ID:        "10",
		CreatedAt: newTimePointer(2023, 9, 24, 8, 39, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleSystem,
		Content:   "Your chat will end in 10 minutes due to inactivity.",
	},
	{
		ID:        "11",
		CreatedAt: newTimePointer(2023, 9, 24, 8, 40, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleUser,
		Content:   "Thank you. Also, can I change the delivery date?",
		Name:      newStringPointer("John Doe"),
	},
	{
		ID:        "12",
		CreatedAt: newTimePointer(2023, 9, 24, 8, 41, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleAssistant,
		Content:   "Yes, when would you like the order to be delivered?",
		Name:      newStringPointer("Assistant"),
	},
	{
		ID:        "13",
		CreatedAt: newTimePointer(2023, 9, 24, 8, 42, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleUser,
		Content:   "On 30th September.",
		Name:      newStringPointer("John Doe"),
	},
	{
		ID:        "14",
		CreatedAt: newTimePointer(2023, 9, 24, 8, 43, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleAssistant,
		Content:   "The delivery date has been updated to 30th September.",
		Name:      newStringPointer("Assistant"),
	},
	{
		ID:        "15",
		CreatedAt: newTimePointer(2023, 9, 24, 8, 44, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleUser,
		Content: `Great, that's all for now.

May be I will see you again? Thanks a bunch!`,
		Name: newStringPointer("John Doe"),
	},
	{
		ID:        "16",
		CreatedAt: newTimePointer(2023, 9, 24, 8, 45, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleAssistant,
		Content: `
# My heading

[reference](#)

If Grommet's Markdown component uses ` + "`markdown-to-jsx`" + ` internally, then the types for your custom renderer function should align with what ` + "`markdown-to-jsx`" + ` expects. The ` + "`markdown-to-jsx`" + ` library allows you to provide custom renderers for different Markdown elements, and these custom renderers receive specific props based on the Markdown element they're rendering.

For a ` + "`code`" + ` element, the props typically include those that are passed to any React component, along with some specific to Markdown rendering. Here's how you can define the type for your ` + "`code`" + ` component in this context:

## Sample code

` + "```typescript" + `
import React, { FC, ReactNode } from 'react';
// other imports remain the same

interface CodeComponentProps {
  node: any;  // This can be more specific if you know the structure
  inline?: boolean;
  className?: string;
  children: ReactNode;
  // Include other props that markdown-to-jsx might pass
}

const CodeBlock: FC<CodeComponentProps> = ({ node, inline, className, children, ...props }) => {
  // CodeBlock implementation remains the same
};

export const ConversationMessageContent: FC<ConversationMessageContentProps> = ({ content }) => {
  return (
    <MemoizedMarkdown
      components={{
        code: CodeBlock
      }}
    >
      {content}
    </MemoizedMarkdown>
  );
};
` + "```" + `

In this setup:

`,
		Name: newStringPointer("Assistant"),
	},
	{
		ID:        "17",
		CreatedAt: newTimePointer(2023, 9, 24, 9, 45, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleUser,
		Content: `
		# Out of Breath

		You know, sometimes in life it seems like there's no way out. Like
		a sheep trapped in a maze designed by wolves. See all the
		options [here](https://github.com/probablyup/markdown-to-jsx/)

		[reference](#)

	` + "```" + `
	import { Grommet } from 'grommet';
	` + "```" + `

		> i carry your heart with me

		![alt text](//v2.grommet.io/assets/IMG_4245.jpg "Markdown Image")

		| Markdown | Less | Pretty | Long header now | One more for sake of it |
		| --- | --- | --- | --- | --- |
		| Content *still* | ` + "`renders`" + ` | **nicely** in a table | **nicely** in a table | **nicely** in a table |
		| 1 | 2 | 3 | 3 | 3 |
	`,
		Name: newStringPointer("John Doe"),
	},
	{
		ID:        "18",
		CreatedAt: newTimePointer(2023, 9, 24, 9, 45, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleAssistant,
		Content: `

	` + "```python" + `
	def get_openapi_completion_for_integration_sequence_test(intxt, value_type):
		response = openai.FetchCompletion.create(
				model="text-davinci-003",
				prompt=prompts.generate_prompt_integration_sequence_test(intxt, value_type),
				temperature=0,
				max_tokens=2560,
				best_of=1,
				stop=["##", "}}}}}}", "Generate workflow", "func Test"])

	return response
	` + "```" + `
			`,
		Name:    newStringPointer("Assistant"),
		Details: newStringPointer(tmpMessageDetails),
	},
	{
		ID:        "19",
		CreatedAt: newTimePointer(2023, 9, 24, 10, 45, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleUser,
		Content:   MarkDownCheatSheet,
		Name:      newStringPointer("User"),
	},
}

var MessageSamplesListComplex = []conversationSpec.ConversationMessage{
	{
		ID:        "100",
		CreatedAt: newTimePointer(2023, 9, 24, 8, 44, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleUser,
		Content: `Great, that's all for now.

May be I will see you again? Thanks a bunch!`,
		Name: newStringPointer("John Doe"),
	},
	{
		ID:        "101",
		CreatedAt: newTimePointer(2023, 9, 24, 8, 45, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleAssistant,
		Content: `
	# My heading

	[reference](#)

	If Grommet's Markdown component uses ` + "`markdown-to-jsx`" + ` internally, then the types for your custom renderer function should align with what ` + "`markdown-to-jsx`" + ` expects. The ` + "`markdown-to-jsx`" + ` library allows you to provide custom renderers for different Markdown elements, and these custom renderers receive specific props based on the Markdown element they're rendering.

	For a ` + "`code`" + ` element, the props typically include those that are passed to any React component, along with some specific to Markdown rendering. Here's how you can define the type for your ` + "`code`" + ` component in this context:

	## Sample code

	` + "```typescript" + `
	import React, { FC, ReactNode } from 'react';
	// other imports remain the same

	interface CodeComponentProps {
		node: any;  // This can be more specific if you know the structure
		inline?: boolean;
		className?: string;
		children: ReactNode;
		// Include other props that markdown-to-jsx might pass
	}

	const CodeBlock: FC<CodeComponentProps> = ({ node, inline, className, children, ...props }) => {
		// CodeBlock implementation remains the same
	};

	export const ConversationMessageContent: FC<ConversationMessageContentProps> = ({ content }) => {
		return (
			<MemoizedMarkdown
				components={{
					code: CodeBlock
				}}
			>
				{content}
			</MemoizedMarkdown>
		);
	};
	` + "```" + `

	In this setup:

	`,
		Name: newStringPointer("Assistant"),
	},
	{
		ID:        "102",
		CreatedAt: newTimePointer(2023, 9, 24, 9, 45, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleUser,
		Content: `
		# Out of Breath

		You know, sometimes in life it seems like there's no way out. Like
		a sheep trapped in a maze designed by wolves. See all the
		options [here](https://github.com/probablyup/markdown-to-jsx/)

		[reference](#)

	` + "```" + `
	import { Grommet } from 'grommet';
	` + "```" + `

		> i carry your heart with me

		![alt text](//v2.grommet.io/assets/IMG_4245.jpg "Markdown Image")

		| Markdown | Less | Pretty | Long header now | One more for sake of it |
		| --- | --- | --- | --- | --- |
		| Content *still* | ` + "`renders`" + ` | **nicely** in a table | **nicely** in a table | **nicely** in a table |
		| 1 | 2 | 3 | 3 | 3 |
	`,
		Name: newStringPointer("John Doe"),
	},
	{
		ID:        "103",
		CreatedAt: newTimePointer(2023, 9, 24, 9, 45, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleAssistant,
		Content: `

	` + "```python" + `
	def get_openapi_completion_for_integration_sequence_test(intxt, value_type):
		response = openai.FetchCompletion.create(
				model="text-davinci-003",
				prompt=prompts.generate_prompt_integration_sequence_test(intxt, value_type),
				temperature=0,
				max_tokens=2560,
				best_of=1,
				stop=["##", "}}}}}}", "Generate workflow", "func Test"])

	return response
	` + "```" + `
			`,
		Name:    newStringPointer("Assistant"),
		Details: newStringPointer(tmpMessageDetails),
	},
	{
		ID:        "104",
		CreatedAt: newTimePointer(2023, 9, 24, 10, 45, 0, 0, time.UTC),
		Role:      conversationSpec.ConversationRoleUser,
		Content:   MarkDownCheatSheet,
		Name:      newStringPointer("User"),
	},
}

// func getSampleConversations() []spec.Conversation {
// 	convo1 := spec.InitConversation("Sample conversation base")
// 	convo1.Messages = messageSamplesListBase
// 	convo2 := spec.InitConversation("Sample conversation complex")
// 	convo2.Messages = messageSamplesListComplex
// 	return []spec.Conversation{convo1, convo2}
// }

const TempCodeString = `def get_openapi_completion_for_integration_sequence_test(intxt, value_type):
response = openai.FetchCompletion.create(
    model="text-davinci-003",
    prompt=prompts.generate_prompt_integration_sequence_test(intxt, value_type),
    temperature=0,
    max_tokens=2560,
    best_of=1,
    stop=["##", "}}}}}}", "Generate workflow", "func Test"])

return response`
