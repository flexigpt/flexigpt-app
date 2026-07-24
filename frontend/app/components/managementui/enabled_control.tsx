interface EnabledControlProps {
	id: string;
	checked: boolean;
	onChange: (checked: boolean) => void;
	disabled?: boolean;
	label?: string;
	title?: string;
	compact?: boolean;
	busy?: boolean;
}

export function EnabledControl({
	id,
	checked,
	onChange,
	disabled = false,
	label = 'Enabled',
	title,
	compact = true,
	busy = false,
}: EnabledControlProps) {
	return (
		<label
			htmlFor={id}
			className={`flex items-center gap-2 text-xs ${disabled || busy ? 'cursor-not-allowed opacity-70' : 'cursor-pointer'}`}
			title={title}
		>
			<span>{label}</span>

			<input
				id={id}
				type="checkbox"
				className={compact ? 'toggle toggle-accent toggle-sm' : 'toggle toggle-accent'}
				checked={checked}
				disabled={disabled || busy}
				onChange={event => {
					onChange(event.currentTarget.checked);
				}}
			/>

			{busy ? <span className="loading loading-spinner loading-xs" aria-label="Updating" /> : null}
		</label>
	);
}
