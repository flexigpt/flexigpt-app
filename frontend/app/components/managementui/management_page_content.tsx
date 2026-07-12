import type { ReactNode } from 'react';

import type { ManagementWidth } from '@/components/managementui/management_class_consts';
import { MANAGEMENT_WIDTH_CLASSES } from '@/components/managementui/management_class_consts';

interface ManagementPageContentProps {
	children: ReactNode;
	width?: ManagementWidth;
	className?: string;
}

export function ManagementPageContent({ children, width = 'standard', className = '' }: ManagementPageContentProps) {
	return (
		<div className="app-scrollbar-thin flex min-h-0 w-full grow flex-col items-center overflow-y-auto">
			<main className={`mt-4 space-y-4 pb-8 ${MANAGEMENT_WIDTH_CLASSES[width]} ${className}`}>{children}</main>
		</div>
	);
}
