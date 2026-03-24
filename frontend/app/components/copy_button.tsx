import type { ButtonHTMLAttributes } from 'react';
import { useState } from 'react';

import { FiCheck, FiCopy } from 'react-icons/fi';

import { log } from '@/apis/baseapi';

interface CopyButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
	value?: string;
	size: number;
}

export function CopyButton({ value, size, ...buttonProps }: CopyButtonProps) {
	const [copied, setCopied] = useState(false);

	const handleCopy = async () => {
		if (!value) {
			return;
		}
		try {
			await navigator.clipboard.writeText(value);
			setCopied(true);
			setTimeout(() => {
				setCopied(false);
			}, 1500);
		} catch (error) {
			log.error('Failed to copy: ', error);
		}
	};

	return (
		<button aria-label="Copy To Clipboard" title="Copy To Clipboard" onClick={handleCopy} {...buttonProps}>
			{copied ? <FiCheck size={size} /> : <FiCopy size={size} />}
		</button>
	);
}
