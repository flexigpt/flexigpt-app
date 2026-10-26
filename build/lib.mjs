// @ts-check

import { spawn } from "node:child_process";
import { createHash } from "node:crypto";
import fs from "node:fs";
import { createServer } from "node:net";
import path from "node:path";
import readline from "node:readline";
import { fileURLToPath } from "node:url";
import { parseEnv } from "node:util";

/**
 * @typedef {{
 *   cwd?: string,
 *   env?: NodeJS.ProcessEnv,
 *   input?: string,
 *   capture?: boolean,
 *   echo?: boolean,
 *   echoStderr?: boolean,
 *   sensitive?: boolean,
 *   timeoutMs?: number,
 *   allowedExitCodes?: number[]
 *   devListenerPort?: number
 * }} RunOptions
 */

export const root = fileURLToPath(new URL("../", import.meta.url));
const placeholder = path.join(root, "frontend", "dist", "client", "a.tmp");

/** @param {number} port */
async function listenerFree(port) {
	return await new Promise((resolve, reject) => {
		const server = createServer();
		server.once("error", (/** @type {NodeJS.ErrnoException} */ error) => {
			if (/** @type {NodeJS.ErrnoException} */ (error).code === "EADDRINUSE") {
				resolve(false);
			} else {
				reject(error);
			}
		});
		server.listen({ host: "127.0.0.1", port, exclusive: true }, () => {
			server.close((/** @type {any} */ error) =>
				error ? reject(error) : resolve(true),
			);
		});
	});
}

/** @param {string} version */
export function validateVersion(version) {
	if (!/^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/.test(version)) {
		throw new Error("VERSION_TAG must have the form vX.Y.Z");
	}

	if (
		version
			.slice(1)
			.split(".")
			.some((part) => Number(part) > 65535)
	) {
		throw new Error("VERSION_TAG components must not exceed 65535");
	}

	return version;
}

export function prepareEmbedPlaceholder() {
	fs.mkdirSync(path.dirname(placeholder), { recursive: true });
	fs.closeSync(fs.openSync(placeholder, "a", 0o644));
	console.log(`Prepared Go embed placeholder: ${placeholder}`);
}

export function cleanEmbedPlaceholder() {
	if (!fs.existsSync(placeholder)) {
		return;
	}

	const stat = fs.statSync(placeholder);

	if (stat.isFile() && stat.size === 0) {
		fs.rmSync(placeholder, { force: true });
		console.log(`Removed Go embed placeholder: ${placeholder}`);
	}
}

const licenseNames = [
	"PROJECT_LICENSE.txt",
	"THIRD_PARTY_NOTICES.txt",
	"go-dependency-licenses.txt",
	"js-dependency-licenses.txt",
];

const secretNamePattern = /PASSWORD|TOKEN|SECRET|CERT|PRIVATE_KEY|PASSPHRASE/i;

/** @param {string} value */
function requiredSimpleName(value) {
	if (!/^[A-Za-z0-9._-]+$/.test(value)) {
		throw new Error(`Expected a simple file name, received: ${value}`);
	}

	return value;
}

export function loadBuildVars() {
	return parseEnv(
		fs.readFileSync(path.join(root, "build/buildvars.env"), "utf8"),
	);
}

/** @param {NodeJS.ProcessEnv} vars */
export function validatePinnedVersions(vars) {
	const pinned = (/** @type {string} */ name) => {
		const value = vars[name];
		if (!value) {
			throw new Error(`Required configuration is missing: ${name}`);
		}
		return value;
	};

	const nodeVersion = pinned("NODE_VERSION");
	if (process.versions.node !== nodeVersion) {
		throw new Error(
			`Node version mismatch: expected ${nodeVersion}, received ${process.versions.node}`,
		);
	}

	const packageJson = JSON.parse(
		fs.readFileSync(path.join(root, "package.json"), "utf8"),
	);
	if (packageJson.engines?.node !== nodeVersion) {
		throw new Error("package.json Node engine differs from buildvars.env");
	}

	const managerVersion = /^pnpm@([^+]+)(?:\+.*)?$/.exec(
		packageJson.packageManager ?? "",
	)?.[1];
	if (managerVersion !== pinned("PNPM_VERSION")) {
		throw new Error("package.json pnpm version differs from buildvars.env");
	}

	const goMod = fs.readFileSync(path.join(root, "go.mod"), "utf8");
	const goVersion = /^go\s+([0-9]+\.[0-9]+(?:\.[0-9]+)?)/m.exec(goMod)?.[1];
	const configuredGo = pinned("GO_VERSION");
	const configuredGoMajorMinor = configuredGo.split(".").slice(0, 2).join(".");
	if (goVersion !== configuredGoMajorMinor && goVersion !== configuredGo) {
		throw new Error(
			`Go module version mismatch: go.mod=${goVersion ?? "missing"}; configured=${configuredGo}`,
		);
	}

	const moduleWailsVersion =
		/^\s*github\.com\/wailsapp\/wails\/v3\s+(v[^\s]+)/m.exec(goMod)?.[1];
	if (moduleWailsVersion !== pinned("WAILS_VERSION")) {
		throw new Error("go.mod Wails version differs from buildvars.env");
	}

	const frontendPackage = JSON.parse(
		fs.readFileSync(path.join(root, "frontend/package.json"), "utf8"),
	);
	const runtimeVersion = frontendPackage.dependencies?.[
		"@wailsio/runtime"
	].replace(/^\^/, "");
	const expectedRuntimeVersion = pinned("WAILS_VERSION").replace(/^v/, "");
	if (runtimeVersion !== expectedRuntimeVersion) {
		throw new Error(
			`Wails runtime mismatch: frontend=${runtimeVersion}; CLI=${expectedRuntimeVersion}`,
		);
	}

	const asdfVersions = Object.fromEntries(
		fs
			.readFileSync(path.join(root, ".tool-versions"), "utf8")
			.trim()
			.split(/\r?\n/)
			.map((line) => line.split(/\s+/)),
	);
	for (const [tool, version] of Object.entries({
		golang: configuredGo,
		nodejs: `v${nodeVersion}`,
		task: pinned("TASK_VERSION"),
	})) {
		if (asdfVersions[tool] !== version) {
			throw new Error(`.tool-versions ${tool} differs from buildvars.env`);
		}
	}
}

/**
 * @param {{
 *   production: boolean,
 *   tags: readonly string[],
 *   version: string,
 *   output: string,
 *   backendDir: string,
 *   windowsGui?: boolean
 * }} options
 */
export function goBuildArgs({
	production,
	tags,
	version,
	output,
	backendDir,
	windowsGui = false,
}) {
	const linkerFlags = [
		...(production ? ["-s", "-w"] : []),
		...(production && windowsGui ? ["-H", "windowsgui"] : []),
		"-X",
		`main.Version=${validateVersion(version)}`,
	].join(" ");

	return [
		"build",
		"-mod=readonly",
		"-trimpath",
		"-buildvcs=false",
		...(tags.length > 0 ? ["-tags", tags.join(",")] : []),
		"-ldflags",
		linkerFlags,
		"-o",
		output,
		`./${backendDir}`,
	];
}

export class Project {
	/**
	 * @param {string} platform
	 * @param {string} action
	 */
	constructor(platform, action) {
		this.platform = platform;
		this.action = action;
		this.env = /** @type {NodeJS.ProcessEnv} */ ({ ...process.env });
		this.secrets = /** @type {string[]} */ ([]);
		this.commandNumber = 0;
		this.startedAt = new Date().toISOString();

		const stamp = `${this.startedAt.replaceAll(":", "-")}-${process.pid}`;

		this.logDir = this.abs("build/out/logs", platform, action, stamp);
		this.commandLogDir = path.join(this.logDir, "commands");
		this.work = this.abs("build/out/work", platform);
		this.artifacts = this.abs("build/out/artifacts", platform);

		fs.mkdirSync(this.commandLogDir, { recursive: true });

		for (const name of ["pipeline.log", "debug.log", "error.log"]) {
			fs.writeFileSync(path.join(this.logDir, name), "", { mode: 0o600 });
		}

		this.writeSummary("running");
	}

	load() {
		const defaults = loadBuildVars();

		this.env = { ...defaults, ...this.env };

		this.secrets = [
			...new Set(
				Object.entries(this.env)
					.filter(
						([key, value]) => Boolean(value) && secretNamePattern.test(key),
					)
					.map(([, value]) => /** @type {string} */ (value))
					.filter(Boolean),
			),
		].sort((left, right) => right.length - left.length);

		requiredSimpleName(this.value("COMMON_BUILD_NAME"));
	}

	/**
	 * @param {"running" | "success" | "failure"} status
	 * @param {string} [message]
	 */
	writeSummary(status, message = "") {
		const summary = {
			status,
			platform: this.platform,
			action: this.action,
			startedAt: this.startedAt,
			finishedAt: status === "running" ? null : new Date().toISOString(),
			logDirectory: this.logDir,
			message: this.redact(message),
		};

		fs.writeFileSync(
			path.join(this.logDir, "summary.json"),
			`${JSON.stringify(summary, null, 2)}\n`,
			{ mode: 0o600 },
		);
	}

	/** @param {...string} parts */
	abs(...parts) {
		return path.resolve(root, ...parts);
	}

	/** @param {string} key */
	value(key) {
		const value = this.env[key];

		if (!value) {
			throw new Error(`Required configuration is missing: ${key}`);
		}

		return value;
	}

	/** @param {string} key */
	flag(key) {
		const value = this.env[key] ?? "false";

		if (value !== "true" && value !== "false") {
			throw new Error(`${key} must be true or false`);
		}

		return value === "true";
	}

	/** @param {Record<string, string>} [overrides] */
	values(overrides = {}) {
		/** @type {Record<string, string>} */
		const values = {};

		for (const [key, value] of Object.entries(this.env)) {
			if (typeof value === "string") {
				values[key] = value;
			}
		}

		return { ...values, ...overrides };
	}

	/** @param {string} value */
	redact(value) {
		for (const secret of this.secrets) {
			value = value.replaceAll(secret, "[REDACTED]");
		}

		return value;
	}

	/**
	 * @param {"INFO" | "DEBUG" | "WARN" | "ERROR"} level
	 * @param {string} message
	 */
	log(level, message) {
		const line = `${new Date().toISOString()} ${level} ${this.redact(message)}\n`;

		fs.appendFileSync(path.join(this.logDir, "debug.log"), line);

		if (level !== "DEBUG") {
			fs.appendFileSync(path.join(this.logDir, "pipeline.log"), line);
		}

		if (level === "WARN" || level === "ERROR") {
			fs.appendFileSync(path.join(this.logDir, "error.log"), line);
		}

		if (level === "ERROR") {
			process.stderr.write(line);
		} else if (level !== "DEBUG") {
			process.stdout.write(line);
		}
	}

	/** @param {string} message */
	step(message) {
		this.log("INFO", `STEP: ${message}`);
	}

	/**
	 * @param {string} command
	 * @param {string[]} args
	 * @param {RunOptions} [options]
	 * @returns {Promise<string>}
	 */
	async run(command, args, options = {}) {
		if (!command || command.includes("\0")) {
			throw new Error("Invalid command");
		}

		if (args.some((arg) => arg.includes("\0"))) {
			throw new Error(`Command ${command} contains an invalid argument`);
		}

		const number = String(++this.commandNumber).padStart(3, "0");
		const description = options.sensitive
			? `${command} [sensitive arguments hidden]`
			: [command, ...args.map((arg) => JSON.stringify(arg))].join(" ");

		const stdoutLog = path.join(this.commandLogDir, `${number}.stdout.log`);
		const stderrLog = path.join(this.commandLogDir, `${number}.stderr.log`);

		fs.writeFileSync(stdoutLog, "", { mode: 0o600 });
		fs.writeFileSync(stderrLog, "", { mode: 0o600 });

		this.log("INFO", `COMMAND ${number}: ${description}`);

		const started = Date.now();
		const timeoutMs = options.timeoutMs ?? 30 * 60 * 1000;
		const allowedExitCodes = new Set([0, ...(options.allowedExitCodes ?? [])]);
		const windows = process.platform === "win32";
		const echoStdout = options.echo ?? !options.capture;
		const echoStderr = options.echoStderr ?? true;

		const devPort = options.devListenerPort;
		if (devPort !== undefined) {
			if (!Number.isInteger(devPort) || devPort < 1 || devPort > 65535) {
				throw new Error("WAILS_VITE_PORT must be an integer from 1 to 65535");
			}
			if (!(await listenerFree(devPort))) {
				throw new Error(
					`127.0.0.1:${devPort} is already in use; not killing its owner`,
				);
			}
		}
		const environment = { ...this.env, ...options.env };
		const ownsGroup = !windows && !environment.FLEXIGPT_DEV_PROCESS_GROUP;

		return await new Promise((resolve, reject) => {
			const child = spawn(command, args, {
				cwd: options.cwd ?? root,
				env: {
					...environment,
					...(devPort !== undefined && !windows
						? { FLEXIGPT_DEV_PROCESS_GROUP: "1" }
						: {}),
				},
				stdio: ["pipe", "pipe", "pipe"],
				shell: false,
				detached: ownsGroup,
				windowsHide: true,
			});

			if (!child.stdin || !child.stdout || !child.stderr) {
				reject(new Error(`Could not create stdio pipes for ${command}`));
				return;
			}

			let captured = "";
			let stopReason = "";
			let settled = false;
			/**
			 * @type {string | number | NodeJS.Timeout | undefined}
			 */
			let forceKillTimer;

			/** @param {string} reason */
			const stop = (reason) => {
				if (stopReason) {
					return;
				}

				stopReason = reason;

				if (!child.pid) {
					return;
				}

				if (windows) {
					const killer = spawn(
						"taskkill.exe",
						["/pid", String(child.pid), "/T", "/F"],
						{
							stdio: "ignore",
							windowsHide: true,
						},
					);

					killer.on("error", () => child.kill());
					return;
				}

				try {
					if (ownsGroup) process.kill(-child.pid, "SIGTERM");
					else child.kill("SIGTERM");
				} catch {
					child.kill("SIGTERM");
				}

				forceKillTimer = setTimeout(() => {
					if (!child.pid) {
						return;
					}

					try {
						if (ownsGroup) process.kill(-child.pid, "SIGKILL");
						else child.kill("SIGKILL");
					} catch {
						child.kill("SIGKILL");
					}
				}, 10_000);
			};

			const onInterrupt = () => stop("interrupted by SIGINT");
			const onTerminate = () => stop("interrupted by SIGTERM");

			process.on("SIGINT", onInterrupt);
			process.on("SIGTERM", onTerminate);

			const timeout =
				timeoutMs > 0
					? setTimeout(
							() => stop(`timed out after ${Math.round(timeoutMs / 1000)}s`),
							timeoutMs,
						)
					: undefined;

			const cleanup = () => {
				if (timeout) {
					clearTimeout(timeout);
				}

				if (forceKillTimer) {
					clearTimeout(forceKillTimer);
				}

				process.off("SIGINT", onInterrupt);
				process.off("SIGTERM", onTerminate);
			};

			const stdout = readline.createInterface({ input: child.stdout });
			const stderr = readline.createInterface({ input: child.stderr });

			stdout.on("line", (line) => {
				const safe = this.redact(line);

				fs.appendFileSync(stdoutLog, `${safe}\n`);

				if (options.capture) {
					captured += `${line}\n`;
				}

				if (echoStdout) {
					process.stdout.write(`${safe}\n`);
				}

				this.log("DEBUG", `COMMAND ${number} STDOUT: ${safe}`);
			});

			stderr.on("line", (line) => {
				const safe = this.redact(line);

				fs.appendFileSync(stderrLog, `${safe}\n`);
				fs.appendFileSync(
					path.join(this.logDir, "error.log"),
					`${new Date().toISOString()} COMMAND ${number} STDERR: ${safe}\n`,
				);

				if (echoStderr) {
					process.stderr.write(`${safe}\n`);
				}

				this.log("DEBUG", `COMMAND ${number} STDERR: ${safe}`);
			});

			/** @type {Promise<void> | undefined} */
			let devCleanup;

			const groupAlive = () => {
				if (!ownsGroup || !child.pid) return false;
				try {
					process.kill(-child.pid, 0);
					return true;
				} catch (error) {
					if (/** @type {NodeJS.ErrnoException} */ (error).code === "ESRCH") {
						return false;
					}
					throw error;
				}
			};

			const waitForRelease = async () => {
				for (let attempt = 0; attempt < 20; attempt++) {
					if (
						(await listenerFree(/** @type {number} */ (devPort))) &&
						!groupAlive()
					) {
						return true;
					}
					await new Promise((resolve) => setTimeout(resolve, 100));
				}
				return false;
			};

			const finishDev = async () => {
				// Give Wails and Vite two seconds to exit normally.
				if (!(await waitForRelease()) && groupAlive()) {
					this.log("WARN", "Wails dev processes remain; sending SIGTERM");
					try {
						process.kill(-(/** @type {number} */ (child.pid)), "SIGTERM");
					} catch (error) {
						if (/** @type {NodeJS.ErrnoException} */ (error).code !== "ESRCH")
							throw error;
					}
				}

				if (!(await waitForRelease()) && groupAlive()) {
					this.log("WARN", "Wails dev processes remain; sending SIGKILL");
					try {
						process.kill(-(/** @type {number} */ (child.pid)), "SIGKILL");
					} catch (error) {
						if (/** @type {NodeJS.ErrnoException} */ (error).code !== "ESRCH")
							throw error;
					}
				}

				if (!(await waitForRelease())) {
					throw new Error(
						`Dev cleanup failed: 127.0.0.1:${devPort} is still occupied ` +
							"or Wails dev processes remain; no unrelated port owner was killed",
					);
				}
				this.log("INFO", `Dev cleanup verified: 127.0.0.1:${devPort} is free`);
			};

			// Start on exit, not close: an orphan can keep stdout/stderr open.
			child.once("exit", () => {
				if (devPort === undefined) return;
				devCleanup = finishDev();
				void devCleanup.then(
					() => {
						child.stdout?.destroy();
						child.stderr?.destroy();
					},
					() => {
						child.stdout?.destroy();
						child.stderr?.destroy();
					},
				);
			});

			child.stdin.on("error", () => {
				// A tool may exit before consuming optional standard input.
			});

			child.stdin.end(options.input);

			child.on("error", (error) => {
				if (settled) {
					return;
				}

				settled = true;
				cleanup();

				reject(
					new Error(
						`Cannot start ${command}: ${error.message}. See ${this.logDir}`,
					),
				);
			});

			child.on("close", async (code, signal) => {
				if (settled) {
					return;
				}

				settled = true;
				try {
					await devCleanup;
				} catch (error) {
					cleanup();
					reject(error);
					return;
				}
				cleanup();

				const duration = ((Date.now() - started) / 1000).toFixed(1);
				const exitCode = code ?? -1;

				if (stopReason || !allowedExitCodes.has(exitCode)) {
					const reason =
						stopReason ||
						(signal ? `terminated by ${signal}` : `exit ${exitCode}`);

					this.log(
						"ERROR",
						`COMMAND ${number} failed after ${duration}s: ${reason}. ` +
							`stdout=${stdoutLog}; stderr=${stderrLog}`,
					);

					reject(
						new Error(
							`${command} failed after ${duration}s: ${reason}. ` +
								`See ${stdoutLog}, ${stderrLog}, and ${this.logDir}`,
						),
					);

					return;
				}

				if (exitCode !== 0) {
					this.log(
						"WARN",
						`COMMAND ${number} returned accepted exit code ${exitCode}`,
					);
				} else {
					this.log("INFO", `COMMAND ${number} completed in ${duration}s`);
				}

				resolve(captured);
			});
		});
	}

	/**
	 * pnpm arguments are intentionally restricted because Windows needs cmd.exe
	 * to launch pnpm.cmd. No user-controlled shell syntax is accepted.
	 *
	 * @param {string[]} args
	 * @param {RunOptions} [options]
	 */
	async pnpm(args, options = {}) {
		if (args.some((arg) => !/^[A-Za-z0-9_@./:=,+-]+$/.test(arg))) {
			throw new Error("Unexpected pnpm argument");
		}

		if (process.platform === "win32") {
			return await this.run(
				"cmd.exe",
				["/d", "/s", "/c", ["pnpm.cmd", ...args].join(" ")],
				options,
			);
		}

		return await this.run("pnpm", args, options);
	}

	/**
	 * The Flatpak SDK supplies Go at an explicit path. Native builds continue
	 * to use the Go executable available on the host PATH.
	 *
	 * @param {string[]} args
	 * @param {RunOptions} [options]
	 */
	async go(args, options = {}) {
		return await this.run(
			this.env.FLEXIGPT_GO_EXECUTABLE ?? "go",
			args,
			options,
		);
	}

	validateBuildConfiguration() {
		this.step("Validate pinned application toolchain versions");
		validatePinnedVersions(this.env);
	}

	async installDependencies() {
		this.step("Prepare locked JavaScript and Go dependencies");

		const pnpmVersion = (
			await this.pnpm(["--version"], {
				capture: true,
				echo: false,
			})
		).trim();

		if (pnpmVersion !== this.value("PNPM_VERSION")) {
			throw new Error(
				`pnpm version mismatch: expected ${this.value("PNPM_VERSION")}; received ${pnpmVersion}`,
			);
		}

		const dependencyFiles = [
			"package.json",
			"pnpm-lock.yaml",
			"pnpm-workspace.yaml",
			"frontend/package.json",
			"go.mod",
			"go.sum",
		]
			.map((filename) => this.abs(filename))
			.filter((filename) => fs.existsSync(filename));

		const fingerprint = createHash("sha256");
		fingerprint.update(process.versions.node);
		fingerprint.update(pnpmVersion);

		for (const filename of dependencyFiles) {
			fingerprint.update(filename);
			fingerprint.update(fs.readFileSync(filename));
		}

		const dependencyFingerprint = fingerprint.digest("hex");

		const statePath = this.abs(
			"build/out/cache/dependencies",
			`${this.platform}-${process.arch}.json`,
		);

		const installed =
			fs.existsSync(this.abs("node_modules")) &&
			fs.existsSync(this.abs("frontend/node_modules"));

		let reusable = false;

		if (installed && fs.existsSync(statePath)) {
			try {
				const state = JSON.parse(fs.readFileSync(statePath, "utf8"));
				reusable = state.fingerprint === dependencyFingerprint;
			} catch {
				reusable = false;
			}
		}

		if (reusable) {
			this.log("INFO", "Reusing locked pnpm installation");
		} else {
			await this.pnpm(["install", "--frozen-lockfile", "--prefer-offline"]);

			fs.mkdirSync(path.dirname(statePath), { recursive: true });
			fs.writeFileSync(
				statePath,
				`${JSON.stringify(
					{
						fingerprint: dependencyFingerprint,
					},
					null,
					2,
				)}\n`,
			);
		}

		await this.go(["mod", "download"]);
	}

	/**
	 * @param {string} name
	 * @param {string} version
	 * @param {boolean} [includePath]
	 */
	findTool(name, version, includePath = true) {
		const extension = process.platform === "win32" ? ".exe" : "";
		const directories = [
			this.env.FLEXIGPT_TOOLS_DIR ?? "",
			this.abs(
				"build/out/tools",
				`${process.platform}-${process.arch}`,
				`${name}-${version}`,
			),
			...(includePath
				? (this.env.PATH ?? this.env.Path ?? "").split(path.delimiter)
				: []),
		].filter(Boolean);

		for (const directory of directories) {
			const executable = path.resolve(directory, `${name}${extension}`);

			if (fs.existsSync(executable)) {
				return this.file(executable);
			}
		}

		return undefined;
	}

	/**
	 * @param {string} name
	 * @param {string} modulePath
	 * @param {string} version
	 */
	async tool(name, modulePath, version) {
		const installed = this.findTool(name, version);
		if (!installed) {
			throw new Error(
				`${name}@${version} is not installed. ` +
					`Run task install-go-tools or go install ${modulePath}@${version} explicitly.`,
			);
		}
		return installed;
	}

	/**
	 * @param {string} name
	 * @param {string} modulePath
	 * @param {string} version
	 */
	async installTool(name, modulePath, version) {
		const installed = this.findTool(name, version, false);
		if (installed) {
			return installed;
		}

		const extension = process.platform === "win32" ? ".exe" : "";
		const directory = this.abs(
			"build/out/tools",
			`${process.platform}-${process.arch}`,
			`${name}-${version}`,
		);
		const executable = path.join(directory, `${name}${extension}`);

		this.step(`Install ${name}@${version}`);
		fs.mkdirSync(directory, { recursive: true });

		await this.go(["install", `${modulePath}@${version}`], {
			env: {
				GOBIN: directory,
				GOFLAGS: "",
				GOWORK: "off",
			},
		});

		return this.file(executable);
	}

	async wailsTool() {
		const expected = this.value("WAILS_VERSION");

		const actual = (
			await this.go(
				[
					"list",
					"-m",
					"-mod=readonly",
					"-f",
					"{{.Version}}",
					"github.com/wailsapp/wails/v3",
				],
				{
					capture: true,
					echo: false,
				},
			)
		).trim();

		if (actual !== expected) {
			throw new Error(
				`Wails module mismatch: go.mod=${actual}; buildvars.env=${expected}`,
			);
		}

		return await this.tool(
			"wails3",
			"github.com/wailsapp/wails/v3/cmd/wails3",
			expected,
		);
	}

	/**
	 * @param {boolean} production
	 * @param {string} [platformTags]
	 */
	tags(production, platformTags = "") {
		const customTags = `${this.env.FLEXIGPT_EXTRA_TAGS ?? ""},${platformTags}`
			.split(/[,\s]+/)
			.filter(Boolean);

		for (const tag of customTags) {
			if (!/^[A-Za-z0-9_.]+$/.test(tag)) {
				throw new Error(`Invalid Go build tag: ${tag}`);
			}

			if (
				tag === "server" ||
				(production && ["mcp", "pprof", "devtools", "debug"].includes(tag))
			) {
				throw new Error(`Forbidden desktop build tag: ${tag}`);
			}
		}

		return [...new Set([...(production ? ["production"] : []), ...customTags])];
	}

	/** @param {boolean} [development] */
	version(development = false) {
		const version =
			this.env.VERSION_TAG ||
			(development ? this.value("COMMON_APP_DEV_VERSION") : "");

		return validateVersion(version);
	}

	/** @param {string[]} tags */
	async bindings(tags) {
		this.step("Generate Wails TypeScript bindings");

		const placeholder = this.abs("frontend/dist/client/a.tmp");
		const existed = fs.existsSync(placeholder);

		fs.mkdirSync(path.dirname(placeholder), { recursive: true });

		if (!existed) {
			fs.closeSync(fs.openSync(placeholder, "a", 0o644));
		}

		try {
			const wails = await this.wailsTool();
			const args = [
				"generate",
				"bindings",
				"-ts",
				"-i",
				"-noindex",
				"-names",
				"-d",
				this.abs("frontend/app/apis/wailsbinding"),
			];

			if (tags.length > 0) {
				args.push("-f", `-tags=${tags.join(",")}`);
			}

			await this.run(wails, args, {
				cwd: this.abs(this.value("GO_BACKEND_DIR")),
			});
		} finally {
			if (!existed) {
				fs.rmSync(placeholder, { force: true });
			}
		}
	}

	/**
	 * @param {string[]} tags
	 * @param {boolean} production
	 * @param {boolean} [devServer]
	 */
	async frontend(tags, production, devServer = false) {
		await this.bindings(tags);

		if (devServer) {
			return;
		}

		this.step(
			production
				? "Build frontend and generate JavaScript dependency licenses"
				: "Build frontend",
		);

		if (production) {
			const output = this.abs(this.value("LICENSE_JS_FILENAME"));
			fs.rmSync(output, { force: true });

			await this.pnpm(["--filter", "frontend", "run", "build:wails:licenses"], {
				env: {
					GEN_LICENSES: "true",
					LICENSE_JS_OUT: output,
				},
			});

			this.file(output);
		} else {
			await this.pnpm(["--filter", "frontend", "run", "build:wails"]);
		}

		this.file(this.abs("frontend/dist/client/index.html"));
	}

	/**
	 * @param {string[]} tags
	 * @param {NodeJS.ProcessEnv[]} nativeEnvironments
	 */
	async licenses(tags, nativeEnvironments) {
		this.step("Generate project, JavaScript, Go, and combined license notices");

		const directory = this.abs(this.value("LICENSE_OUTPUT_DIR"));

		const jsOutput = this.file(this.abs(this.value("LICENSE_JS_FILENAME")));

		for (const name of licenseNames) {
			const filename = path.join(directory, name);
			if (filename !== jsOutput) {
				fs.rmSync(filename, { force: true });
			}
		}

		fs.mkdirSync(directory, { recursive: true });
		this.copy(this.abs("LICENSE"), path.join(directory, "PROJECT_LICENSE.txt"));

		const tool = await this.tool(
			"go-licenses",
			"github.com/google/go-licenses/v2",
			this.value("GO_LICENSES_VERSION"),
		);

		const reports = [];

		for (const nativeEnvironment of nativeEnvironments) {
			const target = `${nativeEnvironment.GOOS}/${nativeEnvironment.GOARCH}`;
			this.step(`Generate Go dependency license report for ${target}`);

			const flags = [
				this.env.GOFLAGS ?? "",
				"-mod=readonly",
				"-buildvcs=false",
				...(tags.length > 0 ? [`-tags=${tags.join(",")}`] : []),
			]
				.filter(Boolean)
				.join(" ");

			const report = await this.run(
				tool,
				[
					"report",
					"./...",
					"--template",
					this.abs(this.value("LICENSE_TEMPLATE_PATH")),
					"--ignore",
					this.value("COMMON_PRODUCT_GITPATH"),
				],
				{
					capture: true,
					echo: false,
					echoStderr: false,
					env: {
						...nativeEnvironment,
						GOFLAGS: flags,
					},
				},
			);

			if (!report.trim()) {
				throw new Error(`Go license report is empty for ${target}`);
			}

			// Non-fatal go-licenses warnings remain in the command stderr log.
			// A non-zero go-licenses exit still fails the package build.
			reports.push(`Target: ${target}\n${report}`);
		}

		const goOutput = this.abs(this.value("LICENSE_GO_FILENAME"));

		fs.writeFileSync(goOutput, reports.join("\n"));

		const notices = [
			"FlexiGPT - Third Party Notices",
			`Version: ${this.version()}`,
			"",
			"Go dependencies",
			"===============",
			fs.readFileSync(goOutput, "utf8"),
			"",
			"JavaScript dependencies",
			"=======================",
			fs.readFileSync(jsOutput, "utf8"),
			"",
		].join("\n");

		fs.writeFileSync(path.join(directory, "THIRD_PARTY_NOTICES.txt"), notices);

		for (const name of licenseNames) {
			this.file(path.join(directory, name));
		}
	}

	/** @param {string} destination */
	copyLicenses(destination) {
		for (const name of licenseNames) {
			this.copy(
				this.abs(this.value("LICENSE_OUTPUT_DIR"), name),
				path.join(destination, name),
			);
		}
	}

	/** @param {string} filename */
	file(filename) {
		if (!fs.existsSync(filename)) {
			throw new Error(`Required file is missing: ${filename}`);
		}

		const stat = fs.statSync(filename);

		if (!stat.isFile() || stat.size === 0) {
			throw new Error(`Required file is empty or invalid: ${filename}`);
		}

		return filename;
	}

	/**
	 * @param {string} source
	 * @param {string} destination
	 * @param {number} [mode]
	 */
	copy(source, destination, mode = 0o644) {
		this.file(source);
		fs.mkdirSync(path.dirname(destination), { recursive: true });
		fs.copyFileSync(source, destination);
		fs.chmodSync(destination, mode);
	}

	/**
	 * @param {string} template
	 * @param {string} destination
	 * @param {Record<string, string>} values
	 * @param {boolean} [xml]
	 */
	render(template, destination, values, xml = true) {
		const escapeXML = (/** @type {string} */ value) =>
			value
				.replaceAll("&", "&amp;")
				.replaceAll("<", "&lt;")
				.replaceAll(">", "&gt;")
				.replaceAll('"', "&quot;")
				.replaceAll("'", "&apos;");

		const rendered = fs
			.readFileSync(template, "utf8")
			.replace(/\{\{([A-Z][A-Z0-9_]*)\}\}/g, (_match, key) => {
				const value = values[key];

				if (value === undefined) {
					throw new Error(`Template variable is missing: ${key}`);
				}

				if (!xml && /[\r\n]/.test(value)) {
					throw new Error(`Desktop entry template value is multiline: ${key}`);
				}

				return xml ? escapeXML(value) : value;
			});

		if (/\{\{[A-Z][A-Z0-9_]*\}\}/.test(rendered)) {
			throw new Error(`Unresolved template variable in ${template}`);
		}

		fs.mkdirSync(path.dirname(destination), { recursive: true });
		fs.writeFileSync(destination, rendered);
	}

	resetWork() {
		fs.rmSync(this.work, { recursive: true, force: true });
		fs.mkdirSync(this.work, { recursive: true });
	}

	resetArtifacts() {
		fs.rmSync(this.artifacts, { recursive: true, force: true });
		fs.mkdirSync(this.artifacts, { recursive: true });
	}

	/**
	 * @param {string} source
	 * @param {string} name
	 */
	artifact(source, name) {
		requiredSimpleName(name);

		const destination = path.join(this.artifacts, name);
		const temporary = `${destination}.tmp`;

		this.copy(source, temporary);
		fs.renameSync(temporary, destination);

		const size = fs.statSync(destination).size;
		const hash = createHash("sha256")
			.update(fs.readFileSync(destination))
			.digest("hex");

		fs.writeFileSync(`${destination}.sha256`, `${hash}  ${name}\n`);

		this.log(
			"INFO",
			`Artifact ready: ${destination} ` +
				`(${(size / 1024 ** 2).toFixed(2)} MiB; ${size} bytes)`,
		);
	}

	async epoch() {
		const epoch = (
			await this.run(
				"git",
				["-c", `safe.directory=${root}`, "log", "-1", "--format=%ct"],
				{
					capture: true,
					echo: false,
				},
			)
		).trim();

		if (!/^\d+$/.test(epoch)) {
			throw new Error("Unable to determine SOURCE_DATE_EPOCH from Git");
		}

		return epoch;
	}
}

/**
 * Install only in production or CI. Local development uses installed tools.
 *
 * @param {Project} project
 * @param {{production?: boolean, dependencies?: boolean, wails?: boolean}} [options]
 */
export async function prepareBuildDependencies(
	project,
	{ production = false, dependencies = true, wails = true } = {},
) {
	if (
		!production &&
		project.env.CI !== "true" &&
		project.env.GITHUB_ACTIONS !== "true"
	) {
		return;
	}

	if (dependencies) {
		await project.installDependencies();
	}

	if (wails) {
		await project.installTool(
			"wails3",
			"github.com/wailsapp/wails/v3/cmd/wails3",
			project.value("WAILS_VERSION"),
		);
	}

	if (production) {
		await project.installTool(
			"go-licenses",
			"github.com/google/go-licenses/v2",
			project.value("GO_LICENSES_VERSION"),
		);
	}
}

/**
 * @template T
 * @param {Project} project
 * @param {(project: Project) => T | Promise<T>} operation
 */
export async function runProject(project, operation) {
	const { platform, action } = project;

	try {
		project.load();
		project.log("INFO", `Starting ${platform}/${action}`);
		const result = await operation(project);
		project.log("INFO", `Completed ${platform}/${action}`);
		project.writeSummary("success");
		return result;
	} catch (error) {
		const message =
			error instanceof Error ? (error.stack ?? error.message) : String(error);

		project.log("ERROR", message);
		project.log("ERROR", `Logs are available in ${project.logDir}`);
		project.writeSummary("failure", message);
		throw error;
	}
}
