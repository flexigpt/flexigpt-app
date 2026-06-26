import type { ReactNode } from 'react';
import { useEffect, useRef, useState } from 'react';

import {
	isRouteErrorResponse,
	Links,
	Meta,
	Outlet,
	Scripts,
	ScrollRestoration,
	useLocation,
	useNavigate,
} from 'react-router';

import type { AppTheme } from '@/spec/setting';
import { ThemeType } from '@/spec/setting';
import { CustomThemeDark, CustomThemeLight } from '@/spec/theme_consts';

import { IS_WAILS_PLATFORM } from '@/lib/features';
import { recordFrontendCrash, reportFrontendError } from '@/lib/frontend_error_reporter';

import { initBuiltIns } from '@/hooks/use_builtin_provider';
import { ensureWorker } from '@/hooks/use_highlight';
import { getStartupThemeSync, initStartupTheme } from '@/hooks/use_startup_theme';
import { GenericThemeProvider } from '@/hooks/use_theme_provider';

import { attachmentsDropAPI } from '@/apis/baseapi';

import { ErrorRecoveryScreen } from '@/components/error_recovery_screen';
import { Sidebar } from '@/components/sidebar';

// oxlint-disable-next-line import/no-unassigned-import
import '@/globals.css';

import type { Route } from './+types/root';

export function CustomThemeProvider({ children }: { children: ReactNode }) {
	const startup: AppTheme = (() => {
		try {
			return getStartupThemeSync();
		} catch {
			return { type: ThemeType.System, name: 'system' } as AppTheme;
		}
	})();

	return (
		<GenericThemeProvider
			storageKey="flexigpt-theme"
			defaultTheme={startup.name}
			lightTheme={CustomThemeLight}
			darkTheme={CustomThemeDark}
		>
			{children}
		</GenericThemeProvider>
	);
}

// oxlint-disable-next-line react/only-export-components
export const meta: Route.MetaFunction = () => [
	{ title: 'FlexiGPT' },
	{ name: 'description', content: 'The FlexiGPT ecosystem agent' },
];

export function Layout({ children }: { children: ReactNode }) {
	return (
		<html lang="en" suppressHydrationWarning>
			<head>
				<meta charSet="utf-8" />
				<meta name="viewport" content="width=device-width, initial-scale=1" />
				{IS_WAILS_PLATFORM && (
					<>
						<meta name="wails-options" content="noautoinject" />
						<script src="/wails/ipc.js" />
						<script src="/wails/runtime.js" />
					</>
				)}
				<Meta />
				<Links />
			</head>
			<body className="m-0 h-full overflow-hidden p-0 antialiased">
				{children}
				<ScrollRestoration />
				<Scripts />
			</body>
		</html>
	);
}

function domReady() {
	if (document.readyState === 'complete' || document.readyState === 'interactive') {
		return Promise.resolve();
	}
	return new Promise<void>(resolve => {
		document.addEventListener(
			'DOMContentLoaded',
			() => {
				resolve();
			},
			{ once: true }
		);
	});
}

// oxlint-disable-next-line react/only-export-components
export async function clientLoader() {
	// Wait for DOM content to be loaded and Wails runtime to be injected
	await domReady();
	try {
		attachmentsDropAPI.startListener();
	} catch (error) {
		reportFrontendError(error, { phase: 'root.client_loader.attachments_drop_start' });
	}

	// These are startup initializers. If one fails, log it but do not blank the app.
	// If you later decide one of these is truly mandatory, rethrow that specific failure.
	const results = await Promise.allSettled([initBuiltIns(), initStartupTheme()]);
	const names = ['initBuiltIns', 'initStartupTheme'];

	results.forEach((result, index) => {
		if (result.status === 'rejected') {
			reportFrontendError(result.reason, { phase: 'root.client_loader', task: names[index] });
		}
	});
}

// Important! Force the client loader to run during hydration and not just during ssr build.
clientLoader.hydrate = true as const;

// oxlint-disable-next-line no-restricted-exports
export default function Root() {
	const navigate = useNavigate();
	const location = useLocation();
	const pathnameRef = useRef(location.pathname);

	// Init worker on mount.
	useEffect(() => {
		let cancelled = false;
		let cancelScheduled: (() => void) | undefined;

		const startWorker = () => {
			if (cancelled) {
				return;
			}

			try {
				void Promise.resolve(ensureWorker()).catch((error: unknown) => {
					reportFrontendError(error, { phase: 'highlight_worker.ensure_async' });
				});
			} catch (error) {
				reportFrontendError(error, { phase: 'highlight_worker.ensure_sync' });
			}
		};

		let requestCallAvailable = false;
		if ('requestIdleCallback' in window) {
			requestCallAvailable = true;
		}
		if (requestCallAvailable) {
			const idleID = window.requestIdleCallback(startWorker);
			cancelScheduled = () => {
				window.cancelIdleCallback(idleID);
			};
		} else {
			const timerID = window.setTimeout(startWorker, 300);
			cancelScheduled = () => {
				window.clearTimeout(timerID);
			};
		}

		return () => {
			cancelled = true;
			cancelScheduled?.();
		};
	}, []);

	useEffect(() => {
		pathnameRef.current = location.pathname;
	}, [location.pathname]);

	// If a drop occurs and no chat drop-target is registered yet (e.g. user is on Tools/Settings),
	// navigate to /chats. The controller will keep the payload queued and flush it once the
	// chat input registers as drop target.
	useEffect(() => {
		attachmentsDropAPI.setNoTargetHandler(() => {
			try {
				const p = pathnameRef.current || '';
				if (p.startsWith('/chats')) {
					return;
				}
				navigate('/chats', { replace: false });
			} catch (error) {
				reportFrontendError(error, { phase: 'attachments_drop.no_target_handler' });
			}
		});

		return () => {
			attachmentsDropAPI.setNoTargetHandler(null);
		};
	}, [navigate]);

	return (
		<CustomThemeProvider>
			<Sidebar>
				<Outlet />
			</Sidebar>
		</CustomThemeProvider>
	);
}

export function ErrorBoundary({ error }: Route.ErrorBoundaryProps) {
	const navigate = useNavigate();
	const [crashCount, setCrashCount] = useState(0);

	useEffect(() => {
		const isNotFound = isRouteErrorResponse(error) && error.status === 404;
		if (isNotFound) {
			return;
		}

		// eslint-disable-next-line react-hooks/set-state-in-effect, react-you-might-not-need-an-effect/no-adjust-state-on-prop-change
		setCrashCount(recordFrontendCrash('react_router.root_error_boundary'));

		reportFrontendError(error, {
			phase: 'react_router.root_error_boundary',
		});
	}, [error]);

	let title = 'Something went wrong';
	let message = 'This page hit an unexpected error.';
	let technicalDetails: string | undefined;

	if (isRouteErrorResponse(error)) {
		title = error.status === 404 ? '404' : 'Error';
		message =
			error.status === 404
				? 'The requested page could not be found.'
				: error.statusText || `The page failed with status ${error.status}.`;

		if (error.status !== 404) {
			technicalDetails = `${error.status} ${error.statusText}`;
		}
	} else if (error instanceof Error) {
		message = error.message || message;
		technicalDetails = error.stack || error.message;
	} else if (error) {
		message = JSON.stringify(error, null, 2);
		technicalDetails = message;
	}

	return (
		<ErrorRecoveryScreen
			title={title}
			message={message}
			technicalDetails={technicalDetails}
			showResetLocalState={crashCount >= 2}
			onGoHome={() => {
				navigate('/', { replace: true });
			}}
		/>
	);
}

export function HydrateFallback() {
	return (
		<div id="loading-splash" className="flex h-screen w-full flex-col items-center justify-center gap-4">
			<div id="loading-splash-spinner" />
			<span className="loading loading-dots loading-xl text-primary-content" />
		</div>
	);
}
