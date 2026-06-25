import type { ReactNode } from 'react';

const groupedMenuSeparatorClasses = 'border-base-300 my-2 border-t';
const groupedMenuSectionHeaderClasses = 'px-1 pt-1';
const groupedMenuSectionTitleClasses = 'min-w-0 truncate text-[11px] font-semibold tracking-wide uppercase opacity-70';

interface GroupedMenuSectionProps {
	title: string;
	ariaLabel?: string;
	meta?: ReactNode;
	children: ReactNode;
	separatorBefore?: boolean;
	className?: string;
	headerClassName?: string;
}

export function GroupedMenuSection({
	title,

	ariaLabel,
	meta,
	children,
	separatorBefore = false,
	className = 'space-y-1',
	headerClassName = groupedMenuSectionHeaderClasses,
}: GroupedMenuSectionProps) {
	return (
		<>
			{separatorBefore ? <div role="separator" className={groupedMenuSeparatorClasses} /> : null}

			<div role="group" aria-label={ariaLabel ?? title} className={className}>
				<div className={headerClassName}>
					<div className="flex min-w-0 items-center justify-between gap-2">
						<div className="min-w-0">
							<div className={groupedMenuSectionTitleClasses}>{title}</div>
						</div>

						{meta ? <div className="flex shrink-0 items-center gap-1">{meta}</div> : null}
					</div>
				</div>

				{children}
			</div>
		</>
	);
}

interface GroupedMenuSubheadingProps {
	children: ReactNode;
	tone?: 'muted' | 'warning';
	separated?: boolean;
}

export function GroupedMenuSubheading({ children, tone = 'muted', separated = false }: GroupedMenuSubheadingProps) {
	const toneClass = tone === 'warning' ? 'text-warning opacity-80' : 'opacity-50';

	return (
		<div
			className={`px-1 text-[10px] font-medium tracking-wide uppercase ${toneClass} ${
				separated ? 'border-base-300 mt-1 border-t pt-2' : ''
			}`}
		>
			{children}
		</div>
	);
}
