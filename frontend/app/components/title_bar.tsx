import { useCallback, useEffect, useState } from 'react';

import { FiCopy, FiMaximize2, FiMenu, FiMinus, FiX } from 'react-icons/fi';

import { Link } from 'react-router';

import { useTitleBarSlots } from '@/hooks/use_title_bar';

import { backendAPI } from '@/apis/baseapi';

type TitleBarProps = {
	onToggleDrawer?: () => void;
};

export function TitleBar({ onToggleDrawer }: TitleBarProps) {
	const slots = useTitleBarSlots();

	const [version, setVersion] = useState<string>('');
	const [isMax, setIsMax] = useState(false);
	const [isTogglingMax, setIsTogglingMax] = useState(false);

	const syncMaxState = useCallback(async () => {
		const m = await backendAPI.isAppWindowMaximised();
		setIsMax(m);
	}, []);

	useEffect(() => {
		let cancelled = false;

		void (async () => {
			try {
				const v = await backendAPI.getAppVersion();
				if (!cancelled) setVersion(v ?? '');
			} catch {
				// ignore
			}
		})();

		void (async () => {
			try {
				const m = await backendAPI.isAppWindowMaximised();
				if (!cancelled) setIsMax(m);
			} catch {
				// ignore
			}
		})();

		// If the user maximises/restores via OS gestures/shortcuts,
		// at least re-sync when the app regains focus.
		const onFocus = () => {
			void syncMaxState();
		};
		window.addEventListener('focus', onFocus);

		return () => {
			cancelled = true;
			window.removeEventListener('focus', onFocus);
		};
	}, [syncMaxState]);

	const toggleMaximise = useCallback(async () => {
		if (isTogglingMax) return;
		setIsTogglingMax(true);
		try {
			backendAPI.appWindowToggleMaximise();
			setTimeout(() => {
				syncMaxState();
			}, 100);
			// Always re-read actual window state (avoid inverted icon)
		} finally {
			setIsTogglingMax(false);
		}
	}, [isTogglingMax, syncMaxState]);

	return (
		<div
			className="app-drag bg-base-200 border-base-300 flex h-8 w-full items-center gap-2 border-b px-2 py-0"
			onDoubleClick={e => {
				// Don’t toggle when double-clicking on interactive elements
				const t = e.target as HTMLElement;
				if (t.closest('.app-no-drag')) return;
				void toggleMaximise();
			}}
		>
			{/* LEFT */}
			<div className="app-drag flex min-w-0 items-center gap-2">
				{onToggleDrawer && (
					<button
						type="button"
						className="app-no-drag btn btn-ghost btn-xs lg:hidden"
						onClick={onToggleDrawer}
						aria-label="Open menu"
						title="Open menu"
					>
						<FiMenu className="h-4 w-4" />
					</button>
				)}

				<Link to="/" className="flex min-w-0 items-center gap-2">
					<img src="/icon.png" alt="FlexiGPT" className="h-5 w-5" />
					<div className="min-w-0 truncate text-xs">
						<span className="opacity-70">FlexiGPT</span>
						{version ? <span className="ml-2 opacity-70">{version}</span> : null}
					</div>
				</Link>

				{slots.left ? <div>{slots.left}</div> : null}
			</div>

			{/* CENTER */}
			<div className="app-drag m-0 flex min-w-0 flex-1 items-center justify-center px-2 py-0">
				{slots.center ? <div className="w-full min-w-0">{slots.center}</div> : null}
			</div>

			{/* RIGHT */}
			<div className="app-drag flex items-center gap-2">
				{slots.right ? <div className="">{slots.right}</div> : null}

				<div className="app-no-drag flex items-center gap-1">
					<button
						type="button"
						className="btn btn-ghost btn-xs btn-circle shrink-0 p-0 opacity-80 hover:opacity-100"
						onClick={() => {
							backendAPI.appWindowMinimise();
						}}
						aria-label="Minimise"
						title="Minimise"
					>
						<FiMinus size={14} />
					</button>

					<button
						type="button"
						className="btn btn-ghost btn-xs btn-circle shrink-0 p-0 opacity-80 hover:opacity-100"
						disabled={isTogglingMax}
						onClick={() => void toggleMaximise()}
						aria-label={isMax ? 'Restore' : 'Maximise'}
						title={isMax ? 'Restore' : 'Maximise'}
					>
						{isMax ? <FiCopy size={14} /> : <FiMaximize2 size={14} />}
					</button>

					<button
						type="button"
						className="btn btn-ghost btn-xs btn-circle hover:btn-error shrink-0 p-0 opacity-80 hover:opacity-100"
						onClick={() => {
							backendAPI.appQuit();
						}}
						aria-label="Close"
						title="Close"
					>
						<FiX size={14} />
					</button>
				</div>
			</div>
		</div>
	);
}
