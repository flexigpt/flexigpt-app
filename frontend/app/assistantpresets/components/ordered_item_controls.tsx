import { memo } from 'react';

import { FiChevronDown, FiChevronUp, FiTrash2 } from 'react-icons/fi';

interface OrderedItemControlsProps {
	index: number;
	length: number;
	onMoveUp: (index: number) => void;
	onMoveDown: (index: number) => void;
	onRemove: (index: number) => void;
}

export const OrderedItemControls = memo(function OrderedItemControls({
	index,
	length,
	onMoveUp,
	onMoveDown,
	onRemove,
}: OrderedItemControlsProps) {
	return (
		<div className="flex items-center gap-2">
			<button
				type="button"
				className="btn btn-sm btn-ghost rounded-xl"
				onClick={() => {
					onMoveUp(index);
				}}
				disabled={index === 0}
				title="Move up"
			>
				<FiChevronUp size={14} />
			</button>

			<button
				type="button"
				className="btn btn-sm btn-ghost rounded-xl"
				onClick={() => {
					onMoveDown(index);
				}}
				disabled={index === length - 1}
				title="Move down"
			>
				<FiChevronDown size={14} />
			</button>

			<button
				type="button"
				className="btn btn-sm btn-ghost rounded-xl"
				onClick={() => {
					onRemove(index);
				}}
				title="Remove"
			>
				<FiTrash2 size={14} />
			</button>
		</div>
	);
});
