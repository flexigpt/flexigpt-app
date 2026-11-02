import type { ButtonHTMLAttributes } from 'react';
import { useEffect, useRef, useState } from 'react';
import { FiAlertCircle, FiCheck, FiCopy } from 'react-icons/fi';

import { log } from '@/apis/baseapi';

type CopyValue = string | null | undefined;
type CopyState = 'idle' | 'copying' | 'copied' | 'failed';

function writeClipboardText(text: string): Promise<void> {
	if (typeof navigator === 'undefined' || !navigator.clipboard?.writeText) {
		throw new Error('Clipboard access is unavailable.');
	}

	return navigator.clipboard.writeText(text);
}

interface CopyButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
	value?: string;
	valueFetcher?: () => CopyValue | Promise<CopyValue>;
	size: number;
	label?: string;
	copyingLabel?: string;
	copiedLabel?: string;
	failedLabel?: string;
	onCopied?: () => void;
	onCopyError?: (error: unknown) => void;
}

export function CopyButton({
	value,
	valueFetcher,
	size,
	label,
	copyingLabel = 'Copying',
	copiedLabel = 'Copied',
	failedLabel = 'Copy failed',
	onCopied,
	onCopyError,
	children,
	disabled = false,
	onClick,
	'aria-label': ariaLabel,
	title,
	...buttonProps
}: CopyButtonProps) {
	const [copyState, setCopyState] = useState<CopyState>('idle');
	const resultTimeoutRef = useRef<number | null>(null);
	const copyInFlightRef = useRef(false);
	const copyOperationRef = useRef(0);
	const mountedRef = useRef(true);

	useEffect(() => {
		// The first click can occur before passive effects run on a busy markdown tree.
		mountedRef.current = true;
		return () => {
			mountedRef.current = false;
			if (resultTimeoutRef.current) {
				window.clearTimeout(resultTimeoutRef.current);
			}
		};
	}, []);

	const clearResultTimeout = () => {
		if (resultTimeoutRef.current) {
			window.clearTimeout(resultTimeoutRef.current);
			resultTimeoutRef.current = null;
		}
	};

	const scheduleResultReset = (operationID: number) => {
		resultTimeoutRef.current = window.setTimeout(() => {
			resultTimeoutRef.current = null;
			if (mountedRef.current && copyOperationRef.current === operationID) {
				setCopyState('idle');
			}
		}, 1500);
	};

	const setTerminalCopyState = (operationID: number, state: Extract<CopyState, 'copied' | 'failed'>) => {
		if (!mountedRef.current || copyOperationRef.current !== operationID) {
			return;
		}

		clearResultTimeout();
		setCopyState(state);
		scheduleResultReset(operationID);
	};

	const finishCopySuccess = (operationID: number) => {
		if (copyOperationRef.current !== operationID) {
			return;
		}

		copyInFlightRef.current = false;
		if (!mountedRef.current) {
			return;
		}

		setTerminalCopyState(operationID, 'copied');
		try {
			onCopied?.();
		} catch (error) {
			log.error('Copy success handler failed:', error);
		}
	};

	const finishCopyFailure = (operationID: number, error: unknown) => {
		if (copyOperationRef.current !== operationID) {
			return;
		}

		copyInFlightRef.current = false;
		log.error('Failed to copy:', error);
		if (!mountedRef.current) {
			return;
		}

		setTerminalCopyState(operationID, 'failed');
		try {
			onCopyError?.(error);
		} catch (callbackError) {
			log.error('Copy error handler failed:', callbackError);
		}
	};

	const startClipboardWrite = (text: CopyValue, operationID: number) => {
		if (copyOperationRef.current !== operationID) {
			return;
		}
		if (text === null || text === undefined) {
			finishCopyFailure(operationID, new Error('No text is available to copy.'));
			return;
		}

		let writeRequest: Promise<void>;
		try {
			writeRequest = Promise.resolve(writeClipboardText(text));
		} catch (error) {
			finishCopyFailure(operationID, error);
			return;
		}

		setCopyState('copying');
		void writeRequest.then(
			() => {
				finishCopySuccess(operationID);
			},
			(error: unknown) => {
				finishCopyFailure(operationID, error);
			}
		);
	};

	const copy = () => {
		// Click-only entry. A new click supersedes a pending write.
		// Do not gate on copying state, otherwise rapid retries are lost.
		if (disabled) {
			return;
		}

		const operationID = copyOperationRef.current + 1;
		copyOperationRef.current = operationID;
		copyInFlightRef.current = true;
		clearResultTimeout();

		let nextValue: CopyValue | Promise<CopyValue>;
		try {
			nextValue = valueFetcher ? valueFetcher() : value;
		} catch (error) {
			finishCopyFailure(operationID, error);
			return;
		}

		if (typeof nextValue === 'string' || nextValue === null || nextValue === undefined) {
			// Message and code-block callers pass a direct value. Start the native write
			// before any await so it retains the click's transient user activation.
			startClipboardWrite(nextValue, operationID);
			return;
		}

		// An asynchronous value fetch cannot preserve browser user activation. Callers
		// with immediately available text should pass value instead of valueFetcher.
		setCopyState('copying');
		void Promise.resolve(nextValue).then(
			text => {
				startClipboardWrite(text, operationID);
			},
			(error: unknown) => {
				finishCopyFailure(operationID, error);
			}
		);
	};

	const copying = copyState === 'copying';
	const buttonLabel = copying
		? copyingLabel
		: copyState === 'copied'
			? copiedLabel
			: copyState === 'failed'
				? failedLabel
				: label;

	return (
		<button
			{...buttonProps}
			type="button"
			aria-label={copyState === 'failed' ? failedLabel : (ariaLabel ?? buttonLabel ?? 'Copy To Clipboard')}
			title={copyState === 'failed' ? failedLabel : (title ?? buttonLabel ?? 'Copy To Clipboard')}
			disabled={disabled}
			aria-busy={copying || undefined}
			onClick={event => {
				onClick?.(event);
				if (!event.defaultPrevented) {
					copy();
				}
			}}
		>
			{copying ? (
				<span className="loading loading-spinner loading-xs" aria-hidden="true" />
			) : copyState === 'copied' ? (
				<FiCheck size={size} aria-hidden="true" />
			) : copyState === 'failed' ? (
				<FiAlertCircle size={size} className="text-error" aria-hidden="true" />
			) : (
				<FiCopy size={size} aria-hidden="true" />
			)}
			{children ?? (label ? <span>{buttonLabel}</span> : null)}
			<output className="sr-only" aria-live="polite">
				{copying ? copyingLabel : copyState === 'copied' ? copiedLabel : copyState === 'failed' ? failedLabel : ''}
			</output>
		</button>
	);
}
