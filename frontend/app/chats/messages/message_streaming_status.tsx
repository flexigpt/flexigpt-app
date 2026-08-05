import { useSyncExternalStore } from 'react';

import type { MessageStreamSource } from '@/chats/messages/message_content_card';

interface MessageStreamingStatusProps {
	source: MessageStreamSource;
}

export function MessageStreamingStatus({ source }: MessageStreamingStatusProps) {
	useSyncExternalStore(source.subscribe, source.getVersionSnapshot, source.getVersionSnapshot);

	// Once answer text exists, the thought phase has ended from the user's
	// perspective even if a provider sends a late reasoning callback.
	const label = source.getText().length > 0 ? 'Streaming' : 'Thinking';

	return (
		<output className="text-base-content inline-flex min-h-6 items-center gap-2 text-xs" aria-live="polite">
			<span>{label}</span>
			<span className="loading loading-dots loading-sm" />
		</output>
	);
}
