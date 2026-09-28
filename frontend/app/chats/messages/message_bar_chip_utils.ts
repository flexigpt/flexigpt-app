export type MessageBarChipTone = 'default' | 'info' | 'secondary';

export function getMessageBarChipClassName(tone: MessageBarChipTone, fullWidth: boolean, interactive: boolean): string {
	const toneClass =
		tone === 'info'
			? 'bg-info/10 border-info/50 gap-1'
			: tone === 'secondary'
				? 'bg-secondary/10 border-secondary/40 gap-1'
				: 'border-base-content/20 mx-1 justify-between gap-2 bg-inherit';

	return [
		'text-base-content flex items-center rounded-2xl border px-2 py-0',
		toneClass,
		fullWidth ? 'w-full' : 'shrink-0',
		interactive ? 'cursor-pointer' : '',
	]
		.filter(Boolean)
		.join(' ');
}
