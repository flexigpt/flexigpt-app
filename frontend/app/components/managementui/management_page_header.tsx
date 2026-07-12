import type { ReactNode } from 'react';

import type { ManagementWidth } from '@/components/managementui/management_class_consts';
import { MANAGEMENT_WIDTH_CLASSES } from '@/components/managementui/management_class_consts';

interface ManagementPageHeaderProps {
	title: string;
	description?: string;
	leadingActions?: ReactNode;
	actions?: ReactNode;
	width?: ManagementWidth;
}

export function ManagementPageHeader({
	title,
	description,
	leadingActions,
	actions,
	width = 'standard',
}: ManagementPageHeaderProps) {
	return (
		<header
			className={`bg-base-200/95 border-base-content/10 z-20 mt-4 shrink-0 rounded-2xl border px-4 py-3 backdrop-blur-sm ${MANAGEMENT_WIDTH_CLASSES[width]}`}
		>
			<div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
				<div className="flex min-w-0 items-center gap-2">
					{leadingActions}
					<div className="min-w-0">
						<h1 className="text-xl font-semibold tracking-tight">{title}</h1>
						{description ? <p className="text-base-content/70 mt-1 text-xs">{description}</p> : null}
					</div>
				</div>

				{actions ? <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">{actions}</div> : null}
			</div>
		</header>
	);
}
