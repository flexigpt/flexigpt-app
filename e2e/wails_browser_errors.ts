import { test as base, expect } from "@playwright/test";

type WailsBrowserErrorFixtures = {
	filterKnownWailsBrowserOverlayErrors: void;
};

function isKnownWailsV216BrowserOverlayError(error: Error): boolean {
	return error.message === "Cannot read properties of null (reading 'nodes')";
}

/**
 * Wails v2.16.0 browser IPC starts a broken Svelte reconnect-overlay.
 *
 * This fixture ignores only that exact vendor error. Any other pageerror
 * remains a failing test result.
 *
 * Real Wails WebSocket IPC and bound Go calls are still used.
 */
export const test = base.extend<WailsBrowserErrorFixtures>({
	filterKnownWailsBrowserOverlayErrors: [
		async ({ page }, use, testInfo) => {
			const ignoredErrors: string[] = [];
			const unexpectedErrors: string[] = [];

			const onPageError = (error: Error): void => {
				const details = error.stack ?? `${error.name}: ${error.message}`;

				if (isKnownWailsV216BrowserOverlayError(error)) {
					ignoredErrors.push(details);
					return;
				}

				unexpectedErrors.push(details);
			};

			page.on("pageerror", onPageError);

			try {
				await use(undefined);
			} finally {
				page.off("pageerror", onPageError);

				if (ignoredErrors.length > 0) {
					await testInfo.attach(
						"ignored-wails-v2.16.0-browser-overlay-errors.txt",
						{
							body: ignoredErrors.join("\n\n"),
							contentType: "text/plain",
						},
					);
				}

				expect(unexpectedErrors, "Unexpected browser page errors").toEqual([]);
			}
		},
		{ auto: true },
	],
});

export { expect };
