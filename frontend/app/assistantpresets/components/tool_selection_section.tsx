import { memo, useMemo } from 'react';

import { FiPlus } from 'react-icons/fi';

import { Dropdown } from '@/components/dropdown';

import { OrderedItemControls } from '@/assistantpresets/components/ordered_item_controls';
import type {
	SimpleSelectableOption,
	ToolSelectionDisplayItem,
	TriStateBoolean,
} from '@/assistantpresets/lib/assistant_preset_editor_types';

interface ToolSelectionSectionProps {
	isViewMode: boolean;

	availableOptions: readonly SimpleSelectableOption[];
	selectedOptionKey: string;
	onSelectedOptionKeyChange: (key: string) => void;
	onAdd: () => void;
	emptyOptionsLabel: string;

	items: readonly ToolSelectionDisplayItem[];
	emptyState: string;

	onMoveUp: (index: number) => void;
	onMoveDown: (index: number) => void;
	onRemove: (index: number) => void;
	onAutoExecuteChange: (index: number, value: TriStateBoolean) => void;
	onUserArgsChange: (index: number, value: string) => void;
}

const AUTO_EXECUTE_OPTIONS: TriStateBoolean[] = ['', 'true', 'false'];
const AUTO_EXECUTE_DROPDOWN_ITEMS: Record<TriStateBoolean, { isEnabled: boolean }> = {
	'': { isEnabled: true },
	true: { isEnabled: true },
	false: { isEnabled: true },
};

function getAutoExecuteDropdownLabel(value: TriStateBoolean): string {
	if (value === 'true') return 'Force On';
	if (value === 'false') return 'Force Off';
	return 'Tool Default';
}

export const ToolSelectionSection = memo(function ToolSelectionSection({
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
	onAutoExecuteChange,
	onUserArgsChange,
}: ToolSelectionSectionProps) {
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
							placeholderLabel={availableOptions.length === 0 ? emptyOptionsLabel : 'Select an option'}
							title="Select a tool to add"
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

						<div className="mt-3 grid grid-cols-12 gap-2">
							<div className="col-span-12 md:col-span-4">
								<label className="label py-1">
									<span className="label-text text-sm">Auto Execute Override</span>
								</label>

								{isViewMode ? (
									<div className="bg-base-300 rounded-xl px-3 py-2 text-sm">{item.autoExecuteLabel}</div>
								) : (
									<Dropdown<TriStateBoolean>
										dropdownItems={AUTO_EXECUTE_DROPDOWN_ITEMS}
										orderedKeys={AUTO_EXECUTE_OPTIONS}
										selectedKey={item.autoExecuteMode}
										onChange={value => {
											onAutoExecuteChange(idx, value);
										}}
										placeholderLabel="Tool Default"
										title="Auto execute override"
										getDisplayName={getAutoExecuteDropdownLabel}
									/>
								)}
							</div>

							<div className="col-span-12 md:col-span-8">
								<label className="label py-1">
									<span className="label-text text-sm">User Args JSON</span>
								</label>
								<textarea
									className="textarea textarea-bordered h-24 w-full rounded-xl font-mono text-xs"
									readOnly={isViewMode}
									value={item.userArgSchemaInstance}
									onChange={e => {
										onUserArgsChange(idx, e.target.value);
									}}
									spellCheck="false"
									placeholder='e.g. {"location":"london"}'
								/>
								<div className="label">
									<span className="label-text-alt text-base-content/70 text-xs">{item.userArgsHint}</span>
								</div>
							</div>
						</div>
					</div>
				))}

				{items.length === 0 && <div className="text-base-content/70 text-sm">{emptyState}</div>}
			</div>
		</>
	);
});
