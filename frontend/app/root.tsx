/* eslint-disable no-restricted-exports */
import { type ReactNode, useEffect, useRef } from 'react';

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

import { type AppTheme, ThemeType } from '@/spec/setting';
import { CustomThemeDark, CustomThemeLight } from '@/spec/theme_consts';

import { IS_WAILS_PLATFORM } from '@/lib/features';

import { initBuiltIns } from '@/hooks/use_builtin_provider';
import { ensureWorker } from '@/hooks/use_highlight';
import { getStartupThemeSync, initStartupTheme } from '@/hooks/use_startup_theme';
import { GenericThemeProvider } from '@/hooks/use_theme_provider';

import { attachmentsDropAPI } from '@/apis/baseapi';

import { Sidebar } from '@/components/sidebar';

import '@/globals.css';

// eslint-disable-next-line no-restricted-imports
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

export async function clientLoader() {
	// Wait for DOM content to be loaded and Wails runtime to be injected
	await domReady();
	attachmentsDropAPI.startListener();
	// Now it's safe to call Wails backend functions
	await Promise.all([initBuiltIns(), initStartupTheme()]);
	// console.log('root builtins loaded');
}

// Important! Force the client loader to run during hydration and not just during ssr build.
clientLoader.hydrate = true as const;

export default function Root() {
	const navigate = useNavigate();
	const location = useLocation();
	const pathnameRef = useRef(location.pathname);

	// Init worker on mount.
	useEffect(() => {
		if ('requestIdleCallback' in window) requestIdleCallback(() => ensureWorker());
		else setTimeout(() => ensureWorker(), 300);
	}, []);

	useEffect(() => {
		pathnameRef.current = location.pathname;
	}, [location.pathname]);

	// If a drop occurs and no chat drop-target is registered yet (e.g. user is on Tools/Settings),
	// navigate to /chats. The controller will keep the payload queued and flush it once the
	// chat input registers as drop target.
	useEffect(() => {
		attachmentsDropAPI.setNoTargetHandler(() => {
			const p = pathnameRef.current || '';
			if (p.startsWith('/chats')) return;
			navigate('/chats', { replace: false });
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
	let message = 'Oops!';
	let details = 'An unexpected error occurred.';
	let stack: string | undefined;

	if (isRouteErrorResponse(error)) {
		message = error.status === 404 ? '404' : 'Error';
		details = error.status === 404 ? 'The requested page could not be found.' : error.statusText || details;
	} else if (import.meta.env.DEV && error && error instanceof Error) {
		details = error.message;
		stack = error.stack;
	}

	return (
		<main className="container mx-auto p-4 pt-16">
			<h1>{message}</h1>
			<p>{details}</p>
			{stack && (
				<pre className="w-full overflow-x-auto p-4">
					<code>{stack}</code>
				</pre>
			)}
		</main>
	);
}

export function HydrateFallback() {
	return (
		<div id="loading-splash" className="flex h-screen w-full flex-col items-center justify-center gap-4">
			<div id="loading-splash-spinner" />
			<span className="loading loading-dots loading-xl text-primary-content"></span>
		</div>
	);
}
