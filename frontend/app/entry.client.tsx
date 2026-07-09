import {
	installGlobalFrontendErrorHandlers,
	renderFatalStartupScreen,
	reportFrontendError,
} from '@/lib/frontend_error_reporter';

installGlobalFrontendErrorHandlers();

// oxlint-disable-next-line unicorn/prefer-top-level-await,no-restricted-imports
void import('./entry.client.bootstrap').catch((error: unknown) => {
	reportFrontendError(error, { phase: 'entry.bootstrap_import' });
	renderFatalStartupScreen(error);
});
