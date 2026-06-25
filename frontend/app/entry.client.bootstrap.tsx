// oxlint-disable import/no-unassigned-import
import { StrictMode } from 'react';

import ReactDOM from 'react-dom/client';
import { HydratedRouter } from 'react-router/dom';

import { renderFatalStartupScreen, reportFrontendError } from '@/lib/frontend_error_reporter';

import '@/apis/baseapi';

import { AppFatalBoundary } from '@/components/error_app_fatal_boundary';

import '@/globals.css';

try {
	ReactDOM.hydrateRoot(
		document,
		<StrictMode>
			<AppFatalBoundary>
				<HydratedRouter />
			</AppFatalBoundary>
		</StrictMode>,
		{
			onRecoverableError(error, info) {
				reportFrontendError(error, {
					phase: 'react.hydration_recoverable',
					componentStack: info.componentStack ?? '',
				});
			},
		}
	);
} catch (error) {
	reportFrontendError(error, {
		phase: 'react.hydrate_root',
	});

	renderFatalStartupScreen(error);
}
