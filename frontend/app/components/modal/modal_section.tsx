import type { ReactNode } from 'react';

interface ModalSectionProps {
	title: ReactNode;
	description?: ReactNode;
	actions?: ReactNode;
	actionsAlign?: 'start' | 'center';
	children: ReactNode;
	className?: string;
}

export function ModalSection({
	title,
	description,
	actions,
	actionsAlign = 'start',
	children,
	className = '',
}: ModalSectionProps) {
	return (
		<section className={`border-base-content/10 rounded-2xl border p-4 ${className}`}>
			<div
				className={`mb-4 flex flex-col gap-2 sm:flex-row sm:justify-between ${
					actionsAlign === 'center' ? 'sm:items-center' : 'sm:items-start'
				}`}
			>
				<div className="min-w-0">
					<h4 className="text-sm font-semibold">{title}</h4>
					{description ? <div className="text-base-content/70 mt-1 text-xs">{description}</div> : null}
				</div>

				{actions ? <div className="flex shrink-0 flex-wrap items-center justify-end gap-2">{actions}</div> : null}
			</div>

			<div className="space-y-4">{children}</div>
		</section>
	);
}
