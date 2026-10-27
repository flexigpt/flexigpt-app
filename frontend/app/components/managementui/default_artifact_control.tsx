import type { ArtifactRef } from '@/spec/artifact';

import { Dropdown } from '@/components/dropdown';

const AUTOMATIC_KEY = '__automatic_default__';
const UNAVAILABLE_KEY = '__unavailable_default__';

export interface DefaultArtifactOption {
	ref: ArtifactRef;
	displayName: string;
	isSelectable: boolean;
	availabilityReason?: string;
}

interface DefaultArtifactControlProps {
	label: string;
	automaticLabel: string;
	options: DefaultArtifactOption[];
	defaultRef: ArtifactRef | null | undefined;
	pending?: boolean;
	onChange: (ref: ArtifactRef | undefined) => void;
}

function optionKey(ref: ArtifactRef): string {
	return `${ref.rootID}\u0000${ref.artifactID}`;
}

function refsEqual(left: ArtifactRef | null | undefined, right: ArtifactRef | null | undefined): boolean {
	return left?.rootID === right?.rootID && left?.artifactID === right?.artifactID;
}

export function DefaultArtifactControl({
	label,
	automaticLabel,
	options,
	defaultRef,
	pending = false,
	onChange,
}: DefaultArtifactControlProps) {
	const selectedOption = options.find(option => refsEqual(option.ref, defaultRef));
	const defaultUnavailable = Boolean(defaultRef) && !selectedOption;
	const defaultNotSelectable = selectedOption !== undefined && !selectedOption.isSelectable;

	const dropdownItems: Record<string, { isEnabled: boolean; displayName: string }> = {
		[AUTOMATIC_KEY]: {
			isEnabled: true,
			displayName: automaticLabel,
		},
	};
	const orderedKeys = [AUTOMATIC_KEY];

	if (defaultUnavailable) {
		dropdownItems[UNAVAILABLE_KEY] = {
			isEnabled: false,
			displayName: 'Current default is unavailable',
		};
		orderedKeys.push(UNAVAILABLE_KEY);
	}

	for (const option of options) {
		const key = optionKey(option.ref);
		dropdownItems[key] = {
			isEnabled: option.isSelectable,
			displayName: option.displayName,
		};
		orderedKeys.push(key);
	}

	const selectedKey = selectedOption
		? optionKey(selectedOption.ref)
		: defaultUnavailable
			? UNAVAILABLE_KEY
			: AUTOMATIC_KEY;

	return (
		<section className="bg-base-100 border-base-content/10 rounded-2xl border p-4">
			<div className="flex flex-col gap-3 sm:flex-row sm:items-center">
				<div className="text-sm font-semibold sm:w-48">{label}</div>

				<div className="min-w-0 grow">
					<Dropdown<string>
						dropdownItems={dropdownItems}
						orderedKeys={orderedKeys}
						selectedKey={selectedKey}
						onChange={key => {
							if (key === AUTOMATIC_KEY) {
								onChange(undefined);
								return;
							}

							const option = options.find(candidate => optionKey(candidate.ref) === key);

							if (option?.isSelectable) {
								onChange(option.ref);
							}
						}}
						filterDisabled={false}
						title={`Select ${label.toLowerCase()}`}
						getDisplayName={key => dropdownItems[key]?.displayName ?? label}
						disabled={pending}
					/>
				</div>
			</div>

			{options.length === 0 ? (
				<p className="text-base-content/60 mt-2 text-xs">No selectable options are currently available.</p>
			) : null}

			{defaultUnavailable ? (
				<p className="text-warning mt-2 text-xs">
					The saved default is no longer available. Choose another option or use the automatic selection.
				</p>
			) : null}

			{defaultNotSelectable ? (
				<p className="text-warning mt-2 text-xs">
					{selectedOption?.availabilityReason ?? 'The saved default is not currently selectable.'}
				</p>
			) : null}
		</section>
	);
}
