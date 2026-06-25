import type { ReactNode } from 'react';

interface PageFrameProps {
	children: ReactNode;
	// When true (default), wraps children in an overflow-auto area.
	// Set to false for pages that manage their own scrolling (e.g., the Chat 3-row grid).
	contentScrollable?: boolean;
	// Optional extra classes for the rounded box and the content wrapper
	className?: string;
	contentClassName?: string;
	// Optional: adjust the outer padding (gap around the rounded box)
	padClassName?: string; // default: "p-2 md:p-3"
}

export function PageFrame({
	children,
	contentScrollable = true,
	className = '',
	contentClassName = '',
	padClassName = 'p-0',
}: PageFrameProps) {
	return (
		<div className={`box-border size-full overflow-hidden ${padClassName}`}>
			<div className={`bg-base-200 size-full overflow-hidden ${className}`}>
				{contentScrollable ? <div className={`size-full overflow-auto ${contentClassName}`}>{children}</div> : children}
			</div>
		</div>
	);
}
