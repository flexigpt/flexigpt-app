import {
	installGlobalFrontendErrorHandlers,
	renderFatalStartupScreen,
	reportFrontendError,
} from '@/lib/frontend_error_reporter';

installGlobalFrontendErrorHandlers();

void import('./entry.client.bootstrap').catch((error: unknown) => {
	reportFrontendError(error, { phase: 'entry.bootstrap_import' });
	renderFatalStartupScreen(error);
});
