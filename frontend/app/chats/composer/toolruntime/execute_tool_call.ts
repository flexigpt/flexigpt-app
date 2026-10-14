import type { UIToolCall } from '@/spec/inference';

/**
 * The old Composer executor is intentionally retained only as a diagnostic
 * boundary for stale components. Session Runtime commands own all accepted
 * Tool execution, approval, retries, and continuation.
 */
export async function executeComposerToolCall(_toolCall: UIToolCall): Promise<never> {
	throw new Error(
		'Composer-side tool execution was removed. Execute the Run-owned tool call through Session Runtime controls.'
	);
}
