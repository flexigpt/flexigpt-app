interface MessageStreamingStatusProps {
	streamText?: string;
	streamThinking?: string;
}

export function MessageStreamingStatus({ streamText = '', streamThinking = '' }: MessageStreamingStatusProps) {
	const label = streamText.length > 0 ? 'Streaming' : streamThinking.length > 0 ? 'Thinking' : 'Waiting for response';

	return (
		<output className="text-base-content inline-flex min-h-6 items-center gap-2 text-xs" aria-live="polite">
			<span>{label}</span>
			<span className="loading loading-dots loading-sm" />
		</output>
	);
}
