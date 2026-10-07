import { spawn } from "node:child_process";
import { mkdir, mkdtemp, rm, writeFile } from "node:fs/promises";
import { createRequire } from "node:module";
import { createServer } from "node:net";
import { dirname, join, resolve } from "node:path";

const require = createRequire(import.meta.url);
const repoRoot = resolve(import.meta.dirname, "..");

if (process.platform !== "linux") {
	throw new Error("This E2E launcher currently targets Linux.");
}

// Do not accidentally connect to a normal development session.
// Check both loopback address families.
async function assertPortAvailable(port: number): Promise<void> {
	for (const host of ["127.0.0.1", "::1"]) {
		await new Promise<void>((resolvePromise, rejectPromise) => {
			const server = createServer();

			server.once("error", (error: NodeJS.ErrnoException) => {
				if (error.code === "EADDRNOTAVAIL" || error.code === "EAFNOSUPPORT") {
					resolvePromise();
					return;
				}

				rejectPromise(
					new Error(
						`Cannot use ${host}:${port}. Stop any existing development server.`,
						{ cause: error },
					),
				);
			});

			server.listen({ host, port, exclusive: true }, () => {
				server.close((error) => {
					if (error) {
						rejectPromise(error);
						return;
					}

					resolvePromise();
				});
			});
		});
	}
}

await assertPortAvailable(5173);
await assertPortAvailable(34115);

const artifactsRoot = join(repoRoot, ".e2e-artifacts");
const runsRoot = join(artifactsRoot, "runs");

await mkdir(runsRoot, { recursive: true, mode: 0o700 });

const runRoot = await mkdtemp(join(runsRoot, "run-"));

const dataHome = join(runRoot, "data");

let exitCode = 1;

console.log(`E2E run directory: ${runRoot}`);
console.log(`Application XDG_DATA_HOME: ${dataHome}`);

try {
	await mkdir(dataHome, { recursive: true, mode: 0o700 });

	// Equivalent to your existing touch:tmp prerequisite.
	// This supports Go embedding before a frontend production build exists.
	const placeholder = join(repoRoot, "frontend", "dist", "client", "a.tmp");
	await mkdir(dirname(placeholder), { recursive: true });
	await writeFile(placeholder, "", { flag: "a" });
	const childEnv: NodeJS.ProcessEnv = {
		...process.env,
		FLEXIGPT_E2E_RUN_ROOT: runRoot,
	};

	// These credentials are not needed for this local startup test.
	delete childEnv.GH_TOKEN;
	delete childEnv.GITHUB_TOKEN;

	const child = spawn(
		process.execPath,
		[
			require.resolve("@playwright/test/cli"),
			"test",
			"--config",
			join(repoRoot, "e2e", "playwright.config.ts"),
			...process.argv.slice(2),
		],
		{
			cwd: repoRoot,
			stdio: "inherit",
			env: childEnv,
		},
	);

	const onInterrupt = () => {
		child.kill("SIGINT");
	};

	const onTerminate = () => {
		child.kill("SIGTERM");
	};

	process.once("SIGINT", onInterrupt);
	process.once("SIGTERM", onTerminate);

	try {
		exitCode = await new Promise<number>((resolvePromise, rejectPromise) => {
			child.once("error", rejectPromise);
			child.once("exit", (code) => {
				resolvePromise(code ?? 1);
			});
		});
	} finally {
		process.removeListener("SIGINT", onInterrupt);
		process.removeListener("SIGTERM", onTerminate);
	}
} finally {
	const keepData = exitCode !== 0 || process.env.FLEXIGPT_E2E_KEEP_DATA === "1";

	if (keepData) {
		console.log(`Retained E2E data and application logs: ${runRoot}`);
	} else {
		await rm(runRoot, {
			recursive: true,
			force: true,
			maxRetries: 3,
		});
	}
}

process.exitCode = exitCode;
