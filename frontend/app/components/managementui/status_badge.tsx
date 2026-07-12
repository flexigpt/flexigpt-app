import type { ReactNode } from 'react';

import type { StatusTone } from '@/components/managementui/management_class_consts';
import { STATUS_TONE_CLASSES } from '@/components/managementui/management_class_consts';

interface StatusBadgeProps {
	children: ReactNode;
	tone?: StatusTone;
	title?: string;
	className?: string;
}

export function StatusBadge({ children, tone = 'neutral', title, className = '' }: StatusBadgeProps) {
	return (
		<span
			className={`badge h-auto max-w-full px-2 py-1 text-center leading-tight wrap-break-word whitespace-normal ${STATUS_TONE_CLASSES[tone]} ${className}`}
			title={title}
		>
			{children}
		</span>
	);
}
