import type { ButtonHTMLAttributes } from 'react';

import { FiArrowDownCircle, FiArrowUpCircle } from 'react-icons/fi';

interface ButtonScrollToBottomProps extends ButtonHTMLAttributes<HTMLButtonElement> {
	onScrollToBottom: () => void;
	iconSize: number;
	show: boolean; // new: control visibility via CSS, not mount/unmount
}

export function ButtonScrollToBottom({
	onScrollToBottom,
	iconSize,
	show,
	className = '',
	onClick,
	...props
}: ButtonScrollToBottomProps) {
	return (
		<button
			aria-label="Scroll To Bottom"
			title="Scroll To Bottom"
			disabled={!show}
			onClick={e => {
				onClick?.(e);
				if (!show) return;
				if (e.defaultPrevented) return;
				onScrollToBottom();
			}}
			className={`${className} transition-opacity duration-150 ${show ? 'visible opacity-100' : 'pointer-events-none invisible opacity-0'}`}
			{...props}
		>
			<FiArrowDownCircle size={iconSize} />
		</button>
	);
}

interface ButtonScrollToTopProps extends ButtonHTMLAttributes<HTMLButtonElement> {
	onScrollToTop: () => void;
	iconSize: number;
	show: boolean; // new: control visibility via CSS, not mount/unmount
}

export function ButtonScrollToTop({
	onScrollToTop,
	iconSize,
	show,
	className = '',
	onClick,
	...props
}: ButtonScrollToTopProps) {
	return (
		<button
			aria-label="Scroll To Top"
			title="Scroll To Top"
			disabled={!show}
			onClick={e => {
				onClick?.(e);
				if (!show) return;
				if (e.defaultPrevented) return;
				onScrollToTop();
			}}
			className={`${className} transition-opacity duration-150 ${show ? 'visible opacity-100' : 'pointer-events-none invisible opacity-0'}`}
			{...props}
		>
			<FiArrowUpCircle size={iconSize} />
		</button>
	);
}
