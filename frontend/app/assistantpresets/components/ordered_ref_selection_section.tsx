import { memo } from 'react';

import { FiPlus } from 'react-icons/fi';

import { OrderedItemControls } from '@/assistantpresets/components/ordered_item_controls';
import type { OrderedDisplayItem, SimpleSelectableOption } from '@/assistantpresets/lib/assistant_preset_editor_types';

interface OrderedRefSelectionSectionProps {
	isViewMode: boolean;

	availableOptions: readonly SimpleSelectableOption[];
	selectedOptionKey: string;
	onSelectedOptionKeyChange: (key: string) => void;
	onAdd: () => void;
	emptyOptionsLabel: string;

	items: readonly OrderedDisplayItem[];
	emptyState: string;

	onMoveUp: (index: number) => void;
	onMoveDown: (index: number) => void;
	onRemove: (index: number) => void;

	addButtonLabel?: string;
}

export const OrderedRefSelectionSection = memo(function OrderedRefSelectionSection({
	isViewMode,
	availableOptions,
	selectedOptionKey,
	onSelectedOptionKeyChange,
	onAdd,
	emptyOptionsLabel,
	items,
	emptyState,
	onMoveUp,
	onMoveDown,
	onRemove,
	addButtonLabel = 'Add',
}: OrderedRefSelectionSectionProps) {
	return (
		<>
			{!isViewMode && (
				<div className="grid grid-cols-12 items-center gap-2">
					<div className="col-span-10">
						<select
							className="select select-bordered w-full rounded-xl"
							value={selectedOptionKey}
							onChange={e => {
								onSelectedOptionKeyChange(e.target.value);
							}}
							disabled={availableOptions.length === 0}
						>
							{availableOptions.length === 0 ? (
								<option value="">{emptyOptionsLabel}</option>
							) : (
								availableOptions.map(option => (
									<option key={option.key} value={option.key}>
										{option.label}
									</option>
								))
							)}
						</select>
					</div>
					<div className="col-span-2">
						<button
							type="button"
							className="btn btn-ghost w-full rounded-xl"
							onClick={onAdd}
							disabled={!selectedOptionKey}
						>
							<FiPlus size={14} />
							<span className="ml-1">{addButtonLabel}</span>
						</button>
					</div>
				</div>
			)}

			<div className="space-y-3">
				{items.map((item, idx) => (
					<div key={item.key} className="border-base-content/10 rounded-2xl border p-3">
						<div className="flex items-start justify-between gap-3">
							<div>
								<div className="font-medium">{item.title}</div>
								<div className="text-base-content/70 mt-1 text-xs">{item.subtitle}</div>
								{item.statusLabel && <div className="badge badge-warning mt-2 rounded-xl">{item.statusLabel}</div>}
							</div>

							{!isViewMode && (
								<OrderedItemControls
									index={idx}
									length={items.length}
									onMoveUp={onMoveUp}
									onMoveDown={onMoveDown}
									onRemove={onRemove}
								/>
							)}
						</div>
					</div>
				))}

				{items.length === 0 && <div className="text-base-content/70 text-sm">{emptyState}</div>}
			</div>
		</>
	);
});
