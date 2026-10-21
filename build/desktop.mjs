import { spawnSync } from "node:child_process";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { parse, stringify } from "yaml";

const root = fileURLToPath(new URL("../", import.meta.url));
process.chdir(root);

function readBuildVariables() {
	const result = {};
	const contents = fs.readFileSync(
		path.join(root, "build/buildvars.env"),
		"utf8",
	);

	for (const input of contents.split(/\r?\n/)) {
		const line = input.trim();
		if (!line || line.startsWith("#")) {
			continue;
		}

		// Parse scalar assignments without evaluating shell code.
		const match = /^([A-Z][A-Z0-9_]*)=(.*)$/.exec(line);
		if (!match) {
			throw new Error(
				`Unsupported buildvars.env assignment: ${line.split("=")[0]}`,
			);
		}
		let value = match[2].trim();
		if (value.startsWith("'") || value.startsWith('"')) {
			if (value.length < 2 || !value.endsWith(value[0])) {
				throw new Error(`Unterminated quoted build variable: ${match[1]}`);
			}
			value = value.slice(1, -1);
		} else if (!/^[A-Za-z0-9_./:@+,-]*$/.test(value)) {
			throw new Error(`Unsupported unquoted build variable: ${match[1]}`);
		}
		result[match[1]] = value;
	}
	return result;
}

const env = { ...readBuildVariables(), ...process.env };
const hostPlatform = {
	win32: "windows",
	darwin: "darwin",
	linux: "linux",
}[process.platform];
const hostArch = { x64: "amd64", arm64: "arm64" }[process.arch];

const buildName = env.COMMON_BUILD_NAME;
if (!buildName || !/^[A-Za-z0-9._-]+$/.test(buildName)) {
	throw new Error("COMMON_BUILD_NAME must be a simple executable name");
}

const backend = path.resolve(root, env.GO_BACKEND_DIR);
const bindingsDirectory = path.join(root, "frontend/app/apis/wailsbinding");
const licenseNames = [
	"PROJECT_LICENSE.txt",
	"THIRD_PARTY_NOTICES.txt",
	"go-dependency-licenses.txt",
	"js-dependency-licenses.txt",
];

function execute(command, args, options = {}) {
	const result = spawnSync(command, args, {
		cwd: options.cwd ?? root,
		env: { ...env, ...options.env },
		stdio: "inherit",
	});

	if (result.error) {
		throw result.error;
	}
	if (result.status !== 0) {
		throw new Error(
			`${command} failed with status ${result.status ?? result.signal}`,
		);
	}
}

function frontendScript(script) {
	const args = ["--filter", "frontend", "run", script];

	// pnpm supplies its JS entry point when this process originates from pnpm.
	if (env.npm_execpath && fs.existsSync(env.npm_execpath)) {
		execute(process.execPath, [env.npm_execpath, ...args]);
	} else if (process.platform === "win32") {
		// These arguments are fixed project script names.
		execute("cmd.exe", ["/d", "/s", "/c", `pnpm ${args.join(" ")}`]);
	} else {
		execute("pnpm", args);
	}
}

function requireFile(filename) {
	if (
		!fs.existsSync(filename) ||
		!fs.statSync(filename).isFile() ||
		fs.statSync(filename).size === 0
	) {
		throw new Error(`Required file is missing or empty: ${filename}`);
	}
	return filename;
}

function copyFile(source, destination) {
	requireFile(source);
	fs.mkdirSync(path.dirname(destination), { recursive: true });
	fs.copyFileSync(source, destination);
}

function prepareEmbed() {
	const baseDirectory = path.join(root, "frontend/dist/client");
	fs.mkdirSync(baseDirectory, { recursive: true });
	fs.closeSync(fs.openSync(path.join(baseDirectory, "a.tmp"), "a"));
}

function versionFor(config) {
	const version =
		env.VERSION_TAG || (config === "debug" ? env.COMMON_APP_DEV_VERSION : "");
	if (!/^v\d+\.\d+\.\d+$/.test(version ?? "")) {
		throw new Error("VERSION_TAG must match vX.Y.Z");
	}
	return version;
}

function extraTags(value = "") {
	const tags = value.split(/[,\s]+/).filter(Boolean);
	for (const tag of tags) {
		if (!/^[A-Za-z0-9_.]+$/.test(tag)) {
			throw new Error(`Invalid Go build tag: ${tag}`);
		}
	}
	return tags;
}

function customTagsFor(platform, value = "") {
	const platformTags = platform === "darwin" ? env.MACOS_EXTRA_TAGS : "";
	return extraTags([value, platformTags].filter(Boolean).join(","));
}

function generateBindings(tags = []) {
	prepareEmbed();

	const args = [
		"generate",
		"bindings",
		"-ts",
		"-i",
		"-noindex",
		"-names",
		"-d",
		bindingsDirectory,
	];
	if (tags.length > 0) {
		args.push("-f", `-tags=${tags.join(",")}`);
	}

	execute("wails3", args, { cwd: backend });
}

function checkNativeTarget(platform, arch) {
	if (platform !== hostPlatform) {
		throw new Error(`The ${platform} build requires a ${platform} build host`);
	}
	if (platform !== "darwin" && arch !== hostArch) {
		throw new Error(
			`The ${platform}/${arch} build requires a matching native compiler host`,
		);
	}
	if (
		platform === "darwin" &&
		!["amd64", "arm64", "universal"].includes(arch)
	) {
		throw new Error(`Unsupported macOS architecture: ${arch}`);
	}
}

function generateIcons(directory, platform) {
	const input = requireFile(path.resolve(root, env.COMMON_ICON_PNG));
	const output = path.join(
		directory,
		platform === "windows" ? "icon.ico" : "iconfile.icns",
	);

	execute(
		"wails3",
		[
			"generate",
			"icons",
			"-input",
			input,
			platform === "windows" ? "-windowsfilename" : "-macfilename",
			output,
		],
		{ cwd: directory },
	);

	return requireFile(output);
}

function generateWindowsAssets(directory, version) {
	const args = [
		"generate",
		"build-assets",
		"-dir",
		directory,
		"-name",
		buildName,
		"-company",
		env.COMMON_DEVELOPER_NAME,
		"-productname",
		env.COMMON_PRODUCT_NAME,
		"-description",
		env.COMMON_PRODUCT_DESCRIPTION,
		"-version",
		version.slice(1),
		"-identifier",
		env.MACOS_BUNDLE_ID,
		"-copyright",
		env.COMMON_PRODUCT_COPYRIGHT,
		"-comments",
		env.COMMON_PRODUCT_DESCRIPTION,
	];
	execute("wails3", args);

	// Retain the schema and metadata emitted by the installed Wails CLI.
	// Apply the project's associations before rendering the native templates.
	const generatedConfigPath = requireFile(path.join(directory, "config.yml"));
	const generatedConfig = parse(fs.readFileSync(generatedConfigPath, "utf8"));
	const projectConfig = parse(
		fs.readFileSync(path.join(root, "build/config.yml"), "utf8"),
	);

	generatedConfig.fileAssociations = projectConfig.fileAssociations ?? [];
	generatedConfig.protocols = projectConfig.protocols ?? [];
	fs.writeFileSync(generatedConfigPath, stringify(generatedConfig));

	execute("wails3", [
		"update",
		"build-assets",
		"-config",
		generatedConfigPath,
		"-dir",
		directory,
	]);

	const windows = path.join(directory, "windows");
	const icon = generateIcons(directory, "windows");
	copyFile(icon, path.join(windows, "icon.ico"));

	const installer = ["nsis", "installer"]
		.map((name) => path.join(windows, name))
		.find((candidate) =>
			fs.existsSync(path.join(candidate, "wails_tools.nsh")),
		);
	if (!installer) {
		throw new Error(
			`Generated Windows NSIS support was not found in ${windows}`,
		);
	}

	return {
		windows,
		icon,
		manifest: requireFile(path.join(windows, "wails.exe.manifest")),
		info: requireFile(path.join(windows, "info.json")),
		installer,
	};
}

function compile(platform, arch, config, tags, version, destination) {
	fs.mkdirSync(path.dirname(destination), { recursive: true });

	const ldflags = ["-X", `main.Version=${version}`];
	if (config === "production") {
		ldflags.push("-s", "-w");
		if (platform === "windows") {
			ldflags.push("-H", "windowsgui");
		}
	}

	const args = ["build", "-trimpath", "-buildvcs=false"];
	if (tags.length > 0) {
		args.push("-tags", tags.join(","));
	}
	args.push("-ldflags", ldflags.join(" "), "-o", destination, "./cmd/agentgo");

	const compilerEnv = {
		GOOS: platform,
		GOARCH: arch,
		CGO_ENABLED: platform === "windows" ? "0" : "1",
	};

	if (platform === "darwin") {
		const minimum = env.MACOS_MIN_VERSION;
		compilerEnv.MACOSX_DEPLOYMENT_TARGET = minimum;
		compilerEnv.CGO_CFLAGS =
			`${env.CGO_CFLAGS ?? ""} -mmacosx-version-min=${minimum}`.trim();
		compilerEnv.CGO_LDFLAGS =
			`${env.CGO_LDFLAGS ?? ""} -mmacosx-version-min=${minimum}`.trim();
	}

	execute("go", args, { env: compilerEnv });
	requireFile(destination);
}

function applyMacRegistrations(plist) {
	const config = parse(
		fs.readFileSync(path.join(root, "build/config.yml"), "utf8"),
	);
	const documents = (config.fileAssociations ?? []).map((association) => ({
		CFBundleTypeExtensions: [association.ext],
		CFBundleTypeName:
			association.name || association.description || association.ext,
		CFBundleTypeRole: association.role || "Editor",
		CFBundleTypeIconFile: `${association.iconName || "appicon"}.icns`,
	}));
	const protocols = (config.protocols ?? []).map((protocol) => ({
		CFBundleURLName: protocol.description || protocol.scheme,
		CFBundleURLSchemes: [protocol.scheme],
		CFBundleTypeRole: "Editor",
	}));

	for (const [key, value] of [
		["CFBundleDocumentTypes", documents],
		["CFBundleURLTypes", protocols],
	]) {
		execute("plutil", ["-replace", key, "-json", JSON.stringify(value), plist]);
	}
}

function assembleMacBundle(
	directory,
	outputDirectory,
	arch,
	config,
	tags,
	version,
) {
	const bundle =
		config === "production"
			? path.resolve(root, env.MACOS_APP_BUNDLE_PATH)
			: path.join(outputDirectory, `${buildName}.app`);
	const executable = path.join(bundle, "Contents/MacOS", buildName);
	const resources = path.join(bundle, "Contents/Resources");

	fs.rmSync(bundle, { recursive: true, force: true });
	fs.mkdirSync(resources, { recursive: true });
	fs.mkdirSync(path.dirname(executable), { recursive: true });

	if (arch === "universal") {
		const amd64 = path.join(directory, `${buildName}-amd64`);
		const arm64 = path.join(directory, `${buildName}-arm64`);
		compile("darwin", "amd64", config, tags, version, amd64);
		compile("darwin", "arm64", config, tags, version, arm64);
		execute("xcrun", ["lipo", "-create", amd64, arm64, "-output", executable]);
		execute("xcrun", ["lipo", "-verify_arch", "x86_64", "arm64", executable]);
	} else {
		compile("darwin", arch, config, tags, version, executable);
	}
	fs.chmodSync(executable, 0o755);

	const plist = path.join(directory, "Info.plist");
	execute(
		"bash",
		["build/darwin/render_templates.sh", "--version", version, "--info-plist"],
		{ env: { MACOS_INFO_PLIST_PATH: plist } },
	);

	copyFile(plist, path.join(bundle, "Contents/Info.plist"));
	applyMacRegistrations(path.join(bundle, "Contents/Info.plist"));
	const icon = generateIcons(directory, "darwin");
	copyFile(icon, path.join(resources, "iconfile.icns"));
	copyFile(icon, path.join(resources, "appicon.icns"));

	if (config === "debug") {
		execute("plutil", [
			"-insert",
			"NSAppTransportSecurity",
			"-json",
			'{"NSAllowsLocalNetworking":true}',
			path.join(bundle, "Contents/Info.plist"),
		]);
	}

	execute("plutil", ["-lint", path.join(bundle, "Contents/Info.plist")]);

	// Local validity signature. The release wrapper applies the requested
	// Developer ID signature after adding its final resources.
	execute("codesign", ["--force", "--deep", "--sign", "-", bundle]);
}

function build(platform, arch, config, additionalTags) {
	if (!["debug", "production"].includes(config)) {
		throw new Error("CONFIG must be debug or production");
	}
	checkNativeTarget(platform, arch);

	const version = versionFor(config);
	const customTags = customTagsFor(platform, additionalTags);
	if (customTags.includes("server")) {
		throw new Error("Desktop build tasks require the native desktop runtime");
	}
	if (
		config === "production" &&
		customTags.some((tag) =>
			["mcp", "pprof", "devtools", "debug"].includes(tag),
		)
	) {
		throw new Error(
			"Production build tags must exclude development control and profiling features",
		);
	}
	const tags = [
		...new Set([
			...(config === "production" ? ["production"] : []),
			...customTags,
		]),
	];

	if (platform === "linux") {
		execute("pkg-config", [
			"--exists",
			...(tags.includes("gtk3")
				? ["gtk+-3.0", "webkit2gtk-4.1"]
				: ["gtk4", "webkitgtk-6.0"]),
		]);
	}

	generateBindings(tags);
	if (env.FLEXIGPT_DEV_SERVER !== "1") {
		frontendScript("build:wails");
		requireFile(path.join(root, "frontend/dist/client/index.html"));
	}

	const outputDirectory = path.join(
		root,
		config === "production" ? "build/bin" : "out/desktop-dev",
	);
	fs.mkdirSync(outputDirectory, { recursive: true });

	const directory = fs.mkdtempSync(path.join(os.tmpdir(), "flexigpt-build-"));
	let resourceFile;
	try {
		if (platform === "darwin") {
			assembleMacBundle(
				directory,
				outputDirectory,
				arch,
				config,
				tags,
				version,
			);
		} else {
			if (platform === "windows") {
				const assets = generateWindowsAssets(directory, version);
				resourceFile = path.join(backend, `flexigpt_windows_${arch}.syso`);
				execute("wails3", [
					"generate",
					"syso",
					"-manifest",
					assets.manifest,
					"-icon",
					assets.icon,
					"-info",
					assets.info,
					"-arch",
					arch,
					"-out",
					resourceFile,
				]);
			}
			const executable = path.join(
				outputDirectory,
				`${buildName}${platform === "windows" ? ".exe" : ""}`,
			);
			compile(platform, arch, config, tags, version, executable);
		}
	} finally {
		if (resourceFile) {
			fs.rmSync(resourceFile, { force: true });
		}
		fs.rmSync(directory, { recursive: true, force: true });
	}
}

function installer() {
	checkNativeTarget("windows", "amd64");
	const version = versionFor("production");
	const binary = requireFile(path.join(root, "build/bin", `${buildName}.exe`));
	const output = path.resolve(root, env.WIN_INSTALLER_PATH);
	const directory = fs.mkdtempSync(
		path.join(os.tmpdir(), "flexigpt-installer-"),
	);

	try {
		const assets = generateWindowsAssets(directory, version);
		requireFile(path.join(assets.installer, "wails_tools.nsh"));

		copyFile(
			path.join(root, "build/windows/installer/project.nsi"),
			path.join(assets.installer, "project.nsi"),
		);
		for (const name of licenseNames) {
			copyFile(
				path.join(root, "build/licenses", name),
				path.join(directory, "licenses", name),
			);
		}

		execute("wails3", ["generate", "webview2bootstrapper"], {
			cwd: assets.installer,
		});
		fs.mkdirSync(path.dirname(output), { recursive: true });
		execute("makensis", [
			"/V3",
			`/DARG_WAILS_AMD64_BINARY=${binary}`,
			`/DARG_FLEXIGPT_INSTALLER=${output}`,
			path.join(assets.installer, "project.nsi"),
		]);
		requireFile(output);
	} finally {
		fs.rmSync(directory, { recursive: true, force: true });
	}
}

function renderTemplate(template, destination, numericVersion) {
	const values = { ...env, VERSION_TAG: numericVersion };
	let text = fs.readFileSync(template, "utf8");

	const valueFor = (key) => {
		if (values[key] === undefined) {
			throw new Error(`Template variable is not configured: ${key}`);
		}
		return String(values[key]);
	};

	if (template.includes(".json")) {
		text = text.replace(/"\{\{([A-Z][A-Z0-9_]*)\}\}"/g, (_, key) =>
			JSON.stringify(valueFor(key)),
		);
		JSON.parse(text);
	} else {
		const escapeXML = (value) =>
			value
				.replaceAll("&", "&amp;")
				.replaceAll("<", "&lt;")
				.replaceAll(">", "&gt;")
				.replaceAll('"', "&quot;")
				.replaceAll("'", "&apos;");
		text = text.replace(/\{\{([A-Z][A-Z0-9_]*)\}\}/g, (_, key) =>
			escapeXML(valueFor(key)),
		);
	}

	if (/\{\{[A-Z][A-Z0-9_]*\}\}/.test(text)) {
		throw new Error(`Unresolved template variable in ${template}`);
	}
	fs.mkdirSync(path.dirname(path.resolve(destination)), { recursive: true });
	fs.writeFileSync(destination, text);
}

try {
	const [action, ...args] = process.argv.slice(2);
	switch (action) {
		case "bindings":
			generateBindings(customTagsFor(hostPlatform, env.FLEXIGPT_EXTRA_TAGS));
			break;
		case "frontend":
			frontendScript("run:wailsdev");
			break;
		case "build":
			build(
				args[0] || hostPlatform,
				args[1] || hostArch,
				args[2] || "debug",
				args[3],
			);
			break;
		case "installer":
			installer();
			break;
		case "run": {
			const executable =
				hostPlatform === "darwin"
					? path.join(
							root,
							"out/desktop-dev",
							`${buildName}.app/Contents/MacOS`,
							buildName,
						)
					: path.join(
							root,
							"out/desktop-dev",
							`${buildName}${hostPlatform === "windows" ? ".exe" : ""}`,
						);
			execute(requireFile(executable), []);
			break;
		}
		case "render":
			renderTemplate(args[0], args[1], args[2]);
			break;
		default:
			throw new Error(`Unknown desktop build action: ${action}`);
	}
} catch (error) {
	console.error(error instanceof Error ? error.message : error);
	process.exitCode = 1;
}
