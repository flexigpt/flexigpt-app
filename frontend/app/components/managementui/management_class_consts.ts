/**
 * Compatibility layout for legacy modal rows that still use fixed 12-column
 * markup. It makes their fixed 2/10 and 3/9 rows stack below `sm`.
 *
 * Prefer `ModalField` for new and actively refactored sections.
 */
export const MANAGEMENT_MODAL_FORM_CLASS = [
	'space-y-4',
	'[&_.grid-cols-12>.col-span-2]:col-span-12',
	'[&_.grid-cols-12>.col-span-3]:col-span-12',
	'[&_.grid-cols-12>.col-span-6]:col-span-12',
	'[&_.grid-cols-12>.col-span-9]:col-span-12',
	'[&_.grid-cols-12>.col-span-10]:col-span-12',
	'[&_.grid-cols-12>.col-span-3:empty]:hidden',
	'sm:[&_.grid-cols-12>.col-span-2]:col-span-2',
	'sm:[&_.grid-cols-12>.col-span-3]:col-span-3',
	'sm:[&_.grid-cols-12>.col-span-6]:col-span-6',
	'sm:[&_.grid-cols-12>.col-span-9]:col-span-9',
	'sm:[&_.grid-cols-12>.col-span-10]:col-span-10',
	'sm:[&_.grid-cols-12>.col-span-3:empty]:block',
].join(' ');

export type StatusTone = 'neutral' | 'success' | 'warning' | 'error' | 'info';
export type ManagementWidth = 'standard' | 'wide';

export const MANAGEMENT_WIDTH_CLASSES: Record<ManagementWidth, string> = {
	standard: 'w-[calc(100%-1rem)] max-w-5xl',
	wide: 'w-[calc(100%-1rem)] max-w-7xl',
};

export const STATUS_TONE_CLASSES: Record<StatusTone, string> = {
	neutral: 'badge-neutral',
	success: 'badge-success',
	warning: 'badge-warning',
	error: 'badge-error',
	info: 'badge-info',
};
