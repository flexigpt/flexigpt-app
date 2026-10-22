// @ts-check

import fs from "node:fs";
import path from "node:path";

import { main } from "../cli.mjs";
import { goBuildArgs } from "../lib.mjs";

/** @typedef {import("../lib.mjs").Project} Project */

function nativeHost() {
	if (process.platform !== "win32" || process.arch !== "x64") {
		throw new Error("Windows build tasks require a Windows amd64 host");
	}
}

/** @param {Project} project */
function powerShell(project) {
	const systemRoot = project.env.SystemRoot ?? "C:\\Windows";
	const executable = path.join(
		systemRoot,
		"System32",
		"WindowsPowerShell",
		"v1.0",
		"powershell.exe",
	);

	return fs.existsSync(executable) ? executable : "powershell.exe";
}

/** @param {Project} project */
function makensisPath(project) {
	return path.join(
		project.env["ProgramFiles(x86)"] ?? "C:\\Program Files (x86)",
		"NSIS",
		"makensis.exe",
	);
}

/** @param {Project} project */
function makensis(project) {
	const executable = makensisPath(project);

	if (!fs.existsSync(executable)) {
		throw new Error(
			"NSIS is missing. Run task setup-windows from an elevated terminal.",
		);
	}

	return executable;
}

/**
 * Wails build assets are configured through build/config.yml. Keep the
 * committed configuration at the development version and render a short-lived
 * copy beside it so relative frontend paths retain their meaning.
 *
 * @param {Project} project
 * @param {string} version
 */
function versionedWailsConfig(project, version) {
	const source = project.file(project.abs("build/config.yml"));
	const developmentVersion = project.value("COMMON_APP_DEV_VERSION").slice(1);
	const expected = `productVersion: "${developmentVersion}"`;
	const contents = fs.readFileSync(source, "utf8");

	if (!contents.includes(expected)) {
		throw new Error(
			`build/config.yml must contain ${expected} to generate versioned Windows build assets`,
		);
	}

	const config = project.abs(
		"build",
		`.wails-config-${process.pid}-${Date.now()}.yml`,
	);

	fs.writeFileSync(
		config,
		contents.replace(expected, `productVersion: "${version.slice(1)}"`),
		{ mode: 0o600 },
	);

	return config;
}

/**
 * @param {Project} project
 * @param {string} version
 */
async function windowsAssets(project, version) {
	const directory = path.join(project.work, "assets");
	fs.mkdirSync(directory, { recursive: true });

	const wails = await project.wailsTool();

	const config = versionedWailsConfig(project, version);

	try {
		project.step("Generate Windows Wails build assets from build/config.yml");

		await project.run(wails, [
			"update",
			"build-assets",
			"-config",
			config,
			"-dir",
			directory,
		]);
	} finally {
		fs.rmSync(config, { force: true });
	}

	const windows = path.join(directory, "windows");
	const icon = path.join(directory, "icon.ico");

	await project.run(
		wails,
		[
			"generate",
			"icons",
			"-input",
			project.abs(project.value("COMMON_ICON_PNG")),
			"-windowsfilename",
			icon,
		],
		{
			cwd: directory,
		},
	);

	project.copy(icon, path.join(windows, "icon.ico"));

	const installer = ["nsis", "installer"]
		.map((name) => path.join(windows, name))
		.find((candidate) =>
			fs.existsSync(path.join(candidate, "wails_tools.nsh")),
		);

	if (!installer) {
		throw new Error(`Generated Wails NSIS support was not found in ${windows}`);
	}

	return {
		wails,
		directory,
		windows,
		installer,
		icon: project.file(icon),
		manifest: project.file(path.join(windows, "wails.exe.manifest")),
		info: project.file(path.join(windows, "info.json")),
	};
}

/**
 * @param {Project} project
 * @param {boolean} production
 * @param {boolean} devServer
 */
async function prepare(project, production, devServer) {
	nativeHost();
	project.resetWork();
	project.validateBuildConfiguration();

	const version = project.version(!production);
	const tags = project.tags(production);

	await project.dependencies();
	await project.frontend(tags, production, devServer);

	return {
		version,
		tags,
		nativeEnvironment: {
			GOOS: "windows",
			GOARCH: "amd64",
			CGO_ENABLED: "0",
			FLEXIGPT_DEV_SERVER: devServer ? "1" : "0",
		},
	};
}

/**
 * @param {Project} project
 * @param {boolean} production
 * @param {boolean} [devServer]
 */
export async function build(project, production, devServer = false) {
	const prepared = await prepare(project, production, devServer);

	if (production) {
		await project.licenses(prepared.tags, [prepared.nativeEnvironment]);
	}

	const assets = await windowsAssets(project, prepared.version);
	const resourceFile = project.abs(
		project.value("GO_BACKEND_DIR"),
		"flexigpt_windows_amd64.syso",
	);

	if (fs.existsSync(resourceFile)) {
		throw new Error(
			`Refusing to overwrite an existing Windows resource file: ${resourceFile}`,
		);
	}

	const executable = project.abs(
		production
			? project.value("WIN_APP_PATH")
			: `build/out/dev/win/${project.value("COMMON_BUILD_NAME")}.exe`,
	);

	fs.mkdirSync(path.dirname(executable), { recursive: true });

	try {
		project.step("Generate Windows Go resource file");

		await project.run(assets.wails, [
			"generate",
			"syso",
			"-manifest",
			assets.manifest,
			"-icon",
			assets.icon,
			"-info",
			assets.info,
			"-arch",
			"amd64",
			"-out",
			resourceFile,
		]);

		project.step("Compile Windows application");

		await project.go(
			goBuildArgs({
				production,
				tags: prepared.tags,
				version: prepared.version,
				output: executable,
				backendDir: project.value("GO_BACKEND_DIR"),
				windowsGui: true,
			}),
			{
				env: prepared.nativeEnvironment,
			},
		);
	} finally {
		fs.rmSync(resourceFile, { force: true });
	}

	return {
		executable: project.file(executable),
		assets,
	};
}

/** @param {Project} project */
async function generateLicenses(project) {
	const prepared = await prepare(project, true, false);
	await project.licenses(prepared.tags, [prepared.nativeEnvironment]);
}

/**
 * @param {Project} project
 * @param {string} filename
 */
async function signFile(project, filename) {
	nativeHost();

	project.value("SIGN_WINDOWS_CERT");
	project.value("SIGN_WINDOWS_CERT_PASSWORD");

	const target = project.file(path.resolve(filename));
	const logPath = path.join(
		project.logDir,
		`windows-sign-${path.basename(target)}.log`,
	);

	project.step(`Sign Windows file ${path.basename(target)}`);

	await project.run(powerShell(project), [
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy",
		"Bypass",
		"-File",
		project.abs("build/windows/sign.ps1"),
		"-FilePath",
		target,
		"-LogPath",
		logPath,
	]);
}

/**
 * @param {Project} project
 * @param {string} output
 */
async function archiveLicenses(project, output) {
	const logPath = path.join(project.logDir, "windows-license-archive.log");

	project.step("Create Windows license ZIP artifact");

	await project.run(powerShell(project), [
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy",
		"Bypass",
		"-File",
		project.abs("build/windows/archive-licenses.ps1"),
		"-LicenseDirectory",
		project.abs(project.value("LICENSE_OUTPUT_DIR")),
		"-OutputPath",
		output,
		"-LogPath",
		logPath,
	]);

	return project.file(output);
}

/** @param {Project} project */
async function packageWindows(project) {
	nativeHost();
	project.resetArtifacts();

	const version = project.version();
	const signing = project.flag("SIGN");

	if (signing) {
		project.value("SIGN_WINDOWS_CERT");
		project.value("SIGN_WINDOWS_CERT_PASSWORD");
	}

	const nsis = makensis(project);
	const { executable, assets } = await build(project, true);

	if (signing) {
		await signFile(project, executable);
	}

	project.step("Prepare NSIS installer project");

	project.copy(
		project.abs(project.value("WIN_NSIS_PROJECT_PATH")),
		path.join(assets.installer, "project.nsi"),
	);

	project.copyLicenses(path.join(assets.directory, "licenses"));

	await project.run(assets.wails, ["generate", "webview2bootstrapper"], {
		cwd: assets.installer,
	});

	const temporaryInstaller = path.join(project.work, "installer.exe");
	const temporaryLicenses = path.join(project.work, "licenses.zip");

	const signingDefinitions = signing
		? [
				"/DARG_FLEXIGPT_SIGN_UNINSTALLER=1",
				`/DARG_FLEXIGPT_POWERSHELL=${powerShell(project)}`,
				`/DARG_FLEXIGPT_SIGN_SCRIPT=${project.abs("build/windows/sign.ps1")}`,
				`/DARG_FLEXIGPT_SIGN_LOG=${path.join(project.logDir, "windows-uninstaller-sign.log")}`,
			]
		: [];

	project.step("Build NSIS installer");

	await project.run(nsis, [
		"/V3",
		`/DARG_WAILS_AMD64_BINARY=${executable}`,
		`/DARG_FLEXIGPT_INSTALLER=${temporaryInstaller}`,
		...signingDefinitions,
		path.join(assets.installer, "project.nsi"),
	]);

	project.file(temporaryInstaller);

	if (signing) {
		await signFile(project, temporaryInstaller);
	}

	await archiveLicenses(project, temporaryLicenses);

	const localInstaller = project.abs(project.value("WIN_INSTALLER_PATH"));
	const localLicenses = project.abs(project.value("WIN_LICENSES_ZIP_PATH"));

	project.copy(temporaryInstaller, localInstaller);
	project.copy(temporaryLicenses, localLicenses);

	project.artifact(
		localInstaller,
		`${project.value("COMMON_BUILD_NAME")}-win-amd64-${version}-installer.exe`,
	);

	project.artifact(
		localLicenses,
		`${project.value("COMMON_BUILD_NAME")}-win-amd64-${version}-licenses.zip`,
	);
}

/**
 * @param {Project} project
 * @param {string} action
 * @param {string[]} [args]
 */
export async function runAction(project, action, args = []) {
	switch (action) {
		case "build":
			await build(project, true);
			break;

		case "build-dev":
			await build(project, false, false);
			break;

		case "dev-build":
			await build(project, false, true);
			break;

		case "bindings":
			nativeHost();
			project.validateBuildConfiguration();
			await project.bindings(project.tags(false));
			break;

		case "licenses":
			await generateLicenses(project);
			break;

		case "package":
			await packageWindows(project);
			break;

		case "sign-file": {
			const filename = args[0];

			if (!filename || args.length !== 1) {
				throw new Error("sign-file requires exactly one target file");
			}

			await signFile(project, filename);
			break;
		}

		case "dev": {
			nativeHost();
			project.validateBuildConfiguration();
			await project.dependencies();

			const wails = await project.wailsTool();

			await project.run(
				wails,
				[
					"dev",
					"-config",
					"build/windows/dev.yml",
					"-port",
					project.env.WAILS_VITE_PORT ?? "5173",
				],
				{
					timeoutMs: 0,
				},
			);
			break;
		}

		case "frontend":
			nativeHost();
			await project.dependencies();

			await project.pnpm(["--filter", "frontend", "run", "run:wailsdev"], {
				timeoutMs: 0,
			});
			break;

		case "dev-run":
			await project.run(
				project.file(
					project.abs(
						"build/out/dev/win",
						`${project.value("COMMON_BUILD_NAME")}.exe`,
					),
				),
				[],
				{
					env: {
						FLEXIGPT_DEV_SERVER: "1",
					},
					timeoutMs: 0,
				},
			);
			break;

		case "setup": {
			nativeHost();

			if (!fs.existsSync(makensisPath(project))) {
				await project.run("choco", [
					"install",
					"nsis",
					`--version=${project.value("NSIS_VERSION")}`,
					"--yes",
					"--no-progress",
				]);
			}

			makensis(project);
			project.log("INFO", "Windows NSIS tooling is available");
			break;
		}

		default:
			throw new Error(`Unknown Windows action: ${action}`);
	}
}

await main("win", runAction);
