import { memo, useMemo } from 'react';

import { FiPlus } from 'react-icons/fi';

import { Dropdown } from '@/components/dropdown';

import { OrderedItemControls } from '@/assistantpresets/components/ordered_item_controls';
import type {
	SimpleSelectableOption,
	SkillSelectionDisplayItem,
} from '@/assistantpresets/lib/assistant_preset_editor_types';

interface SkillSelectionSectionProps {
	isViewMode: boolean;

	availableOptions: readonly SimpleSelectableOption[];
	selectedOptionKey: string;
	onSelectedOptionKeyChange: (key: string) => void;
	onAdd: () => void;
	emptyOptionsLabel: string;

	items: readonly SkillSelectionDisplayItem[];
	emptyState: string;

	onMoveUp: (index: number) => void;
	onMoveDown: (index: number) => void;
	onRemove: (index: number) => void;
	onPreLoadAsActiveChange: (index: number, next: boolean) => void;
}

export const SkillSelectionSection = memo(function SkillSelectionSection({
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
	onPreLoadAsActiveChange,
}: SkillSelectionSectionProps) {
	const dropdownItems = useMemo<Record<string, { isEnabled: boolean }>>(
		() =>
			Object.fromEntries(availableOptions.map(option => [option.key, { isEnabled: true }])) as Record<
				string,
				{ isEnabled: boolean }
			>,
		[availableOptions]
	);

	const orderedKeys = useMemo(() => availableOptions.map(option => option.key), [availableOptions]);

	return (
		<>
			{!isViewMode && (
				<div className="grid grid-cols-12 items-center gap-2">
					<div className="col-span-10">
						<Dropdown<string>
							dropdownItems={dropdownItems}
							orderedKeys={orderedKeys}
							selectedKey={selectedOptionKey}
							onChange={onSelectedOptionKeyChange}
							disabled={availableOptions.length === 0}
							placeholderLabel={availableOptions.length === 0 ? emptyOptionsLabel : 'Select a skill to add'}
							title="Select a skill to add"
							getDisplayName={key => availableOptions.find(option => option.key === key)?.label ?? emptyOptionsLabel}
						/>
					</div>
					<div className="col-span-2">
						<button
							type="button"
							className="btn btn-ghost w-full rounded-xl"
							onClick={onAdd}
							disabled={!selectedOptionKey}
						>
							<FiPlus size={14} />
							<span className="ml-1">Add</span>
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

						<div className="mt-3">
							<label className="label cursor-pointer justify-start gap-3 py-1">
								<span className="label-text text-sm">Preload as active</span>
								{isViewMode ? (
									<span className="text-sm">{item.preLoadAsActive ? 'Yes' : 'No'}</span>
								) : (
									<input
										type="checkbox"
										className="toggle toggle-accent"
										checked={item.preLoadAsActive}
										onChange={e => {
											onPreLoadAsActiveChange(idx, e.target.checked);
										}}
									/>
								)}
							</label>
						</div>
					</div>
				))}

				{items.length === 0 && <div className="text-base-content/70 text-sm">{emptyState}</div>}
			</div>
		</>
	);
});
