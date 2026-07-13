import { memo } from 'react';

import { FiChevronDown, FiChevronUp, FiTrash2 } from 'react-icons/fi';

interface OrderedItemControlsProps {
	index: number;
	length: number;
	onMoveUp: (index: number) => void;
	onMoveDown: (index: number) => void;
	onRemove: (index: number) => void;
	disabled?: boolean;
}

export const OrderedItemControls = memo(function OrderedItemControls({
	index,
	length,
	onMoveUp,
	onMoveDown,
	onRemove,
	disabled = false,
}: OrderedItemControlsProps) {
	return (
		<div className="flex shrink-0 items-center gap-1">
			<button
				type="button"
				className="btn btn-square btn-sm btn-ghost rounded-xl"
				onClick={() => {
					onMoveUp(index);
				}}
				disabled={disabled || index === 0}
				title="Move up"
				aria-label="Move up"
			>
				<FiChevronUp size={14} />
			</button>

			<button
				type="button"
				className="btn btn-square btn-sm btn-ghost rounded-xl"
				onClick={() => {
					onMoveDown(index);
				}}
				disabled={disabled || index === length - 1}
				title="Move down"
				aria-label="Move down"
			>
				<FiChevronDown size={14} />
			</button>

			<button
				type="button"
				className="btn btn-square btn-sm btn-ghost rounded-xl"
				onClick={() => {
					onRemove(index);
				}}
				title="Remove"
				aria-label="Remove"
				disabled={disabled}
			>
				<FiTrash2 size={14} />
			</button>
		</div>
	);
});
