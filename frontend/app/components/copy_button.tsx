import type { ButtonHTMLAttributes } from 'react';
import { useEffect, useRef, useState } from 'react';
import { FiCheck, FiCopy } from 'react-icons/fi';

import { log } from '@/apis/baseapi';

interface CopyButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
	value?: string;
	valueFetcher?: () => Promise<string | null | undefined>;
	size: number;
	label?: string;
	copyingLabel?: string;
	copiedLabel?: string;
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
	onCopied,
	onCopyError,
	children,
	disabled = false,
	onClick,
	'aria-label': ariaLabel,
	title,
	...buttonProps
}: CopyButtonProps) {
	const [copied, setCopied] = useState(false);
	const [copying, setCopying] = useState(false);
	const copiedTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
	const mountedRef = useRef(false);

	useEffect(() => {
		mountedRef.current = true;
		return () => {
			mountedRef.current = false;
			if (copiedTimeoutRef.current) {
				clearTimeout(copiedTimeoutRef.current);
			}
		};
	}, []);

	const copy = async () => {
		if (disabled || copying || (!valueFetcher && (value === null || value === undefined))) {
			return;
		}

		setCopying(true);
		setCopied(false);
		if (copiedTimeoutRef.current) {
			clearTimeout(copiedTimeoutRef.current);
			copiedTimeoutRef.current = null;
		}

		try {
			const text = valueFetcher ? await valueFetcher() : value;
			if (text === null || text === undefined) {
				return;
			}
			if (!navigator.clipboard?.writeText) {
				throw new Error('Clipboard access is unavailable.');
			}

			await navigator.clipboard.writeText(text);
			if (!mountedRef.current) {
				return;
			}

			setCopied(true);
			copiedTimeoutRef.current = setTimeout(() => {
				copiedTimeoutRef.current = null;
				if (mountedRef.current) {
					setCopied(false);
				}
			}, 1500);
			onCopied?.();
		} catch (error) {
			log.error('Failed to copy: ', error);
			if (mountedRef.current) {
				onCopyError?.(error);
			}
		} finally {
			if (mountedRef.current) {
				setCopying(false);
			}
		}
	};

	const buttonLabel = copying ? copyingLabel : copied ? copiedLabel : label;

	return (
		<button
			{...buttonProps}
			type="button"
			aria-label={ariaLabel ?? buttonLabel ?? 'Copy To Clipboard'}
			title={title ?? buttonLabel ?? 'Copy To Clipboard'}
			disabled={disabled || copying}
			onClick={event => {
				onClick?.(event);
				if (!event.defaultPrevented) {
					void copy();
				}
			}}
		>
			{copying ? (
				<span className="loading loading-spinner loading-xs" aria-hidden="true" />
			) : copied ? (
				<FiCheck size={size} aria-hidden="true" />
			) : (
				<FiCopy size={size} aria-hidden="true" />
			)}
			{children ?? (label ? <span>{buttonLabel}</span> : null)}
		</button>
	);
}
