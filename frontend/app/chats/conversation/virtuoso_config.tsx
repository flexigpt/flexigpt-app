/**
 * @public
 */
export const VIRTUOSO_AT_BOTTOM_THRESHOLD = 128;

/**
 * @public
 */
export const VIRTUOSO_SCROLL_SEEK = {
	enter: (velocity: number) => Math.abs(velocity) > 1000,
	exit: (velocity: number) => Math.abs(velocity) < 500,
};

/**
 * @public
 */
export function VirtuosoScrollSeekPlaceholder({ height }: { height: number }) {
	return (
		<div className="mx-auto w-11/12 overflow-hidden py-1 xl:w-5/6">
			<div
				className="bg-base-200/70 border-base-300/40 rounded-2xl border"
				style={{ height: height }}
				aria-hidden="true"
			/>
		</div>
	);
}
