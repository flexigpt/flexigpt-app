import type { ButtonHTMLAttributes } from 'react';
import { useEffect, useRef, useState } from 'react';
import { FiAlertCircle, FiCheck, FiCopy } from 'react-icons/fi';

import { log } from '@/apis/baseapi';

type CopyValue = string | null | undefined;
type CopyState = 'idle' | 'copying' | 'copied' | 'failed';

function getClipboard(): Clipboard {
	if (typeof navigator === 'undefined' || !navigator.clipboard) {
		throw new Error('Clipboard access is unavailable.');
	}

	return navigator.clipboard;
}

function requireClipboardText(text: CopyValue): string {
	if (text === null || text === undefined) {
		throw new Error('No text is available to copy.');
	}

	return text;
}

function writeClipboardText(text: string): Promise<void> {
	const clipboard = getClipboard();
	if (typeof clipboard.writeText !== 'function') {
		throw new TypeError('Clipboard access is unavailable.');
	}

	return clipboard.writeText(text);
}

async function writeFetchedClipboardText(value: Promise<CopyValue>): Promise<void> {
	const clipboard = getClipboard();
	const textValue = Promise.resolve(value).then(requireClipboardText);

	// Begin the native write in the click task. The text representation can
	// resolve after the asynchronous source load completes.
	if (typeof clipboard.write === 'function' && typeof ClipboardItem !== 'undefined') {
		const blobValue = textValue.then(text => new Blob([text], { type: 'text/plain' }));
		void blobValue.catch(() => {});

		try {
			const clipboardItem = new ClipboardItem({ 'text/plain': blobValue });
			await clipboard.write([clipboardItem]);
			return;
		} catch {
			// Fall through for implementations that expose write but do not
			// accept a promised ClipboardItem representation.
		}
	}

	const text_2 = await textValue;
	if (typeof clipboard.writeText !== 'function') {
		throw new TypeError('Clipboard text access is unavailable.');
	}
	await clipboard.writeText(text_2);
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
		mountedRef.current = true;
		return () => {
			mountedRef.current = false;
			if (resultTimeoutRef.current !== null) {
				window.clearTimeout(resultTimeoutRef.current);
				resultTimeoutRef.current = null;
			}
		};
	}, []);

	const clearResultTimeout = () => {
		if (resultTimeoutRef.current !== null) {
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
		console.error('Failed to copy:', error);
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

	const copy = () => {
		// The ref changes synchronously. React state alone cannot prevent two
		// click handlers from starting work before the next render commits.
		if (disabled || copyInFlightRef.current) {
			return;
		}

		const operationID = copyOperationRef.current + 1;
		copyOperationRef.current = operationID;
		copyInFlightRef.current = true;
		clearResultTimeout();

		let nextValue: CopyValue | Promise<CopyValue>;
		try {
			nextValue = value !== undefined ? value : valueFetcher ? valueFetcher() : undefined;
		} catch (error) {
			finishCopyFailure(operationID, error);
			return;
		}

		let writeRequest: Promise<void>;
		try {
			if (typeof nextValue === 'string' || nextValue === null || nextValue === undefined) {
				writeRequest = writeClipboardText(requireClipboardText(nextValue));
			} else {
				// Clipboard.write is invoked in this click task. ClipboardItem
				// carries the source-loading promise without losing activation.
				writeRequest = writeFetchedClipboardText(nextValue);
			}
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
			disabled={disabled || copying}
			aria-busy={copying || undefined}
			onClick={event => {
				// Copy is this button's action, not a cancelable browser default.
				copy();
				onClick?.(event);
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
