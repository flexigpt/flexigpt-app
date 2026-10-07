import { expect, test } from "../wails_browser_errors.ts";

test("browser reaches the real initialized Wails backend", async ({ page }) => {
	await page.goto("/");

	await page.waitForFunction(
		() => typeof window.go?.main?.App?.Ping === "function",
		undefined,
		{ timeout: 60_000 },
	);

	const health = await page.evaluate(async () => {
		const app = window.go?.main?.App;

		if (!app) {
			throw new Error("Wails App bindings are unavailable.");
		}

		return {
			ping: await app.Ping(),
			version: await app.GetAppVersion(),
			artifactInitializationError: await app.GetArtifactInitializationError(),
		};
	});

	expect(health).toEqual({
		ping: "pong",
		version: "0.0.1",
		artifactInitializationError: "",
	});

	await expect(
		page.getByRole("heading", {
			name: "Open Chat Workspace",
			exact: true,
		}),
	).toBeVisible();

	// This card initially links to /agents/ while its Agent is unavailable.
	// A chat link containing the Agent reference verifies that the real
	// catalog request completed and produced a selectable built-in Agent.
	const developStarter = page.getByRole("link", {
		name: /^Develop a Feature/,
	});

	await expect(developStarter).toBeVisible();

	await expect(developStarter).toHaveAttribute(
		"href",
		/^\/chats\?workflow=develop-feature&agentRootID=[^&]+&agentArtifactID=[^&]+$/,
	);

	const wailsScriptPaths = await page.evaluate(() =>
		Array.from(document.scripts)
			.map((script) => script.src)
			.filter(Boolean)
			.map((src) => new URL(src).pathname)
			.filter(
				(pathname) =>
					pathname === "/wails/ipc.js" || pathname === "/wails/runtime.js",
			),
	);

	console.log("Wails runtime script paths:", wailsScriptPaths);

	expect(
		wailsScriptPaths.filter((pathname) => pathname === "/wails/ipc.js"),
	).toHaveLength(1);

	expect(
		wailsScriptPaths.filter((pathname) => pathname === "/wails/runtime.js"),
	).toHaveLength(1);
});
