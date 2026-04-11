## Local storage and keys

- Provider API keys are stored securely in your OS keyring and not as plain text.
- Conversations, settings, and related local app data are stored on your device.

## What gets sent to a provider

When you send a message, the selected model may receive:

- your current prompt
- prior messages included in the conversation context
- system or preset instructions
- attachments added to the current request or conversation
- generation parameters such as temperature or reasoning settings

## Practical checks

- Review attachments before sending.
- Use the intended provider key for the selected model.
- Check that the selected model supports the feature you want, such as attachments or reasoning options.
- Remember that provider capabilities and billing differ across services.
