import type { Key, ReactNode } from 'react';
import { Fragment } from 'react';

import { groupByDateBuckets } from '@/lib/date_utils';

interface GroupedDropdownProps<T> {
	items: T[];
	focused: number;
	getDate: (item: T) => Date;
	getKey: (item: T) => Key;
	getLabel: (item: T) => ReactNode;
	onPick: (item: T) => void;
	renderItemExtra?: (item: T) => ReactNode;
}

export function GroupedDropdown<T>({
	items,
	focused,
	getDate,
	getKey,
	getLabel,
	onPick,
	renderItemExtra,
}: GroupedDropdownProps<T>) {
	const buckets = groupByDateBuckets(items, getDate);

	let globalIndex = 0; // single running index for keyboard focus

	return (
		<ul className="w-full text-sm">
			{Object.entries(buckets)
				.filter(([, arr]) => arr.length)
				.map(([label, arr]) => (
					<Fragment key={label}>
						<li className="app-text-neutral px-12 py-2 text-xs">{label}</li>

						{arr.map(item => {
							const isFocused = globalIndex === focused;

							const li = (
								<li
									key={getKey(item)}
									data-index={globalIndex}
									className={`hover:bg-base-100 flex cursor-pointer items-center justify-between px-12 py-2 ${
										isFocused ? 'bg-base-100' : ''
									}`}
								>
									<button
										type="button"
										className="flex min-w-0 flex-1 cursor-pointer items-center text-left"
										onClick={() => {
											onPick(item);
										}}
									>
										<span className="truncate">{getLabel(item)}</span>
									</button>

									{renderItemExtra && (
										<span className="app-text-neutral hidden text-xs lg:block">{renderItemExtra(item)}</span>
									)}
								</li>
							);

							globalIndex++;
							return li;
						})}
					</Fragment>
				))}
		</ul>
	);
}
