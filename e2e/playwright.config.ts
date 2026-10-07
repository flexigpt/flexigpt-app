import { statSync } from "node:fs";
import { isAbsolute, join, resolve } from "node:path";

import { defineConfig } from "@playwright/test";

const e2eDir = import.meta.dirname;
const repoRoot = resolve(e2eDir, "..");
const artifactsRoot = join(repoRoot, ".e2e-artifacts");
const runRoot = process.env.FLEXIGPT_E2E_RUN_ROOT;

if (!runRoot || !isAbsolute(runRoot)) {
	throw new Error(
		'Run E2E tests through "pnpm run test:e2e" so application data is isolated.',
	);
}

const dataHome = join(runRoot, "data");

if (!statSync(dataHome).isDirectory()) {
	throw new Error(`Missing E2E data directory: ${dataHome}`);
}

export default defineConfig({
	testDir: join(e2eDir, "tests"),
	outputDir: join(artifactsRoot, "test-results"),

	// One Wails application and one database per run.
	workers: 1,
	fullyParallel: false,
	retries: 0,

	timeout: 90_000,

	expect: {
		timeout: 15_000,
	},

	reporter: [
		["list"],
		[
			"html",
			{
				outputFolder: join(artifactsRoot, "playwright-report"),
				open: "never",
			},
		],
	],

	use: {
		browserName: "chromium",
		baseURL: "http://localhost:34115",
		actionTimeout: 15_000,
		navigationTimeout: 60_000,
		trace: "retain-on-failure",
		screenshot: "only-on-failure",
	},

	webServer: {
		command:
			'wails dev -v 0 -noreload -nosyncgomod -tags webkit2_41 -ldflags="-X main.Version=0.0.1"',

		cwd: join(repoRoot, "cmd", "agentgo"),
		url: "http://localhost:34115",
		timeout: 240_000,
		reuseExistingServer: false,

		env: {
			// Inherited by the Go application before xdg package initialization.
			XDG_DATA_HOME: dataHome,
			VITE_WAILS_RUNTIME_INJECTION: "auto",
			// Browser endpoint is not a native desktop webview, so there is no
			// meaningful native OS file-drop event to subscribe to.
			VITE_WAILS_NATIVE_FILE_DROP: "disabled",
		},

		stdout: "pipe",
		stderr: "pipe",

		gracefulShutdown: {
			signal: "SIGINT",
			timeout: 15_000,
		},
	},
});
