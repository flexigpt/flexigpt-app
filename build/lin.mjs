// @ts-check

import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { main, root } from "./lib.mjs";

/** @typedef {import("./lib.mjs").Project} Project */

function nativeHost() {
	if (process.platform !== "linux" || process.arch !== "x64") {
		throw new Error("Linux build tasks require a Linux amd64 host");
	}
}

/** @param {boolean} devServer */
function linuxEnvironment(devServer) {
	return {
		GOOS: "linux",
		GOARCH: "amd64",
		CGO_ENABLED: "1",
		FLEXIGPT_DEV_SERVER: devServer ? "1" : "0",
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

	project.step("Validate Linux GTK and WebKit development packages");

	await project.run("pkg-config", [
		"--exists",
		...(tags.includes("gtk3")
			? ["gtk+-3.0", "webkit2gtk-4.1"]
			: ["gtk4", "webkitgtk-6.0"]),
	]);

	await project.dependencies();
	await project.frontend(tags, production, devServer);

	return {
		version,
		tags,
		nativeEnvironment: linuxEnvironment(devServer),
	};
}

/**
 * @param {Project} project
 * @param {boolean} production
 * @param {boolean} [devServer]
 */
async function build(project, production, devServer = false) {
	const prepared = await prepare(project, production, devServer);

	if (production) {
		await project.licenses(prepared.tags, [prepared.nativeEnvironment]);
	}

	const executable = project.abs(
		production ? "build/bin" : "build/out/dev/lin",
		project.value("COMMON_BUILD_NAME"),
	);

	fs.mkdirSync(path.dirname(executable), { recursive: true });

	project.step("Compile Linux application");

	await project.run(
		"go",
		[
			"build",
			"-mod=readonly",
			"-trimpath",
			"-buildvcs=false",
			...(prepared.tags.length > 0 ? ["-tags", prepared.tags.join(",")] : []),
			"-ldflags",
			`${production ? "-s -w " : ""}-X main.Version=${prepared.version}`,
			"-o",
			executable,
			`./${project.value("GO_BACKEND_DIR")}`,
		],
		{
			env: prepared.nativeEnvironment,
		},
	);

	fs.chmodSync(project.file(executable), 0o755);
	return executable;
}

/** @param {Project} project */
async function generateLicenses(project) {
	const prepared = await prepare(project, true, false);
	await project.licenses(prepared.tags, [prepared.nativeEnvironment]);
}

/**
 * @param {import("./lib.mjs").Project} project
 * @param {fs.PathOrFileDescriptor} destination
 * @param {string} version
 * @param {string} sourceDateEpoch
 */
function renderFlatpakManifest(project, destination, version, sourceDateEpoch) {
	const template = fs.readFileSync(
		project.abs(project.value("FLATPAK_MANIFEST_PATH")),
		"utf8",
	);

	const replacements = new Map([
		["__FLEXIGPT_VERSION_TAG__", JSON.stringify(version)],
		["__FLEXIGPT_SOURCE_DATE_EPOCH__", JSON.stringify(sourceDateEpoch)],
		["__FLEXIGPT_SOURCE_DIR__", JSON.stringify(root)],
	]);

	let rendered = template;

	for (const [marker, value] of replacements) {
		rendered = rendered.replaceAll(marker, value);
	}

	if (/__FLEXIGPT_[A-Z_]+__/.test(rendered)) {
		throw new Error("Flatpak manifest contains an unresolved build marker");
	}

	fs.writeFileSync(destination, rendered);
}

/** @param {Project} project */
async function flatpakInstall(project) {
	nativeHost();

	if (project.value("FLATPAK_DEST") !== "/app") {
		throw new Error("flatpak-install must run inside Flatpak Builder");
	}

	project.step("Build application inside the Flatpak SDK");

	const executable = await build(project, true);
	const id = project.value("FLATPAK_ID");
	const epoch = Number(project.value("SOURCE_DATE_EPOCH"));

	if (!Number.isSafeInteger(epoch) || epoch <= 0) {
		throw new Error("SOURCE_DATE_EPOCH must be a positive integer");
	}

	const values = project.values({
		VERSION_TAG: project.version().slice(1),
		APPDATA_DATE: new Date(epoch * 1000).toISOString().slice(0, 10),
	});

	const desktopFile = path.join(project.work, `${id}.desktop`);
	const metadataFile = path.join(project.work, `${id}.metainfo.xml`);

	project.render(
		project.abs(project.value("FLATPAK_DESKTOP_TEMPLATE_PATH")),
		desktopFile,
		values,
		false,
	);

	project.render(
		project.abs(project.value("FLATPAK_APPDATA_TEMPLATE_PATH")),
		metadataFile,
		values,
	);

	project.step("Install Flatpak application files");

	project.copy(
		executable,
		`/app/bin/${project.value("COMMON_BUILD_NAME")}`,
		0o755,
	);

	project.copy(
		project.abs(project.value("COMMON_ICON_PNG")),
		`/app/share/icons/hicolor/256x256/apps/${id}.png`,
	);

	project.copy(desktopFile, `/app/share/applications/${id}.desktop`);
	project.copy(metadataFile, `/app/share/metainfo/${id}.metainfo.xml`);

	project.copy(
		project.abs(project.value("FLATPAK_MIME_PATH")),
		`/app/share/mime/packages/${id}.xml`,
	);

	project.copyLicenses(`/app/share/licenses/${id}`);

	await project.run("desktop-file-validate", [
		`/app/share/applications/${id}.desktop`,
	]);

	await project.run("update-mime-database", ["/app/share/mime"]);
}

/**
 * @typedef {{
 *   args: string[],
 *   cleanup: () => void
 * }} FlatpakSigning
 */

/**
 * @param {string} value
 */
function decodeArmoredPrivateKey(value) {
	if (value.includes("-----BEGIN PGP PRIVATE KEY BLOCK-----")) {
		return value;
	}

	const decoded = Buffer.from(value.replace(/\s+/g, ""), "base64").toString(
		"utf8",
	);

	if (!decoded.includes("-----BEGIN PGP PRIVATE KEY BLOCK-----")) {
		throw new Error(
			"FLATPAK_GPG_PRIVATE_KEY must be ASCII-armored or base64-encoded ASCII-armored PGP private key data",
		);
	}

	return decoded;
}

/** @param {Project} project */
async function prepareFlatpakSigning(project) {
	/** @type {FlatpakSigning} */
	const unsigned = {
		args: [],
		cleanup: () => {},
	};

	if (!project.flag("SIGN")) {
		return unsigned;
	}

	project.step("Prepare Flatpak GPG signing identity");

	const key = project.value("FLATPAK_GPG_KEY_ID");

	if (!/^[A-Fa-f0-9]{40,64}$/.test(key)) {
		throw new Error("FLATPAK_GPG_KEY_ID must be a full GPG fingerprint");
	}

	const suppliedHome = project.env.FLATPAK_GPG_HOME;
	const privateKey = project.env.FLATPAK_GPG_PRIVATE_KEY;

	if (!privateKey) {
		if (!suppliedHome) {
			throw new Error(
				"Signed Flatpak packaging requires FLATPAK_GPG_HOME or FLATPAK_GPG_PRIVATE_KEY",
			);
		}

		const home = path.resolve(suppliedHome);

		await project.run("gpg", [
			"--homedir",
			home,
			"--batch",
			"--list-secret-keys",
			key,
		]);

		return {
			args: [`--gpg-homedir=${home}`, `--gpg-sign=${key}`],
			cleanup: () => {},
		};
	}

	const temporaryDirectory = fs.mkdtempSync(
		path.join(os.tmpdir(), "flexigpt-flatpak-gpg-"),
	);
	const home = path.join(temporaryDirectory, "gnupg");
	const keyFile = path.join(temporaryDirectory, "signing-key.asc");

	fs.mkdirSync(home, { recursive: true, mode: 0o700 });
	fs.writeFileSync(keyFile, decodeArmoredPrivateKey(privateKey), {
		mode: 0o600,
	});

	try {
		const passphrase = project.env.FLATPAK_GPG_PASSPHRASE ?? "";
		const importArguments = [
			"--homedir",
			home,
			"--batch",
			...(passphrase
				? ["--pinentry-mode", "loopback", "--passphrase-fd", "0"]
				: []),
			"--import",
			keyFile,
		];

		await project.run("gpg", importArguments, {
			input: passphrase,
			sensitive: true,
		});

		await project.run("gpg", [
			"--homedir",
			home,
			"--batch",
			"--list-secret-keys",
			key,
		]);

		return {
			args: [`--gpg-homedir=${home}`, `--gpg-sign=${key}`],
			cleanup: () => {
				fs.rmSync(temporaryDirectory, {
					recursive: true,
					force: true,
				});
			},
		};
	} catch (error) {
		fs.rmSync(temporaryDirectory, {
			recursive: true,
			force: true,
		});
		throw error;
	}
}

/** @param {Project} project */
async function packageFlatpak(project) {
	nativeHost();
	project.resetWork();
	project.resetArtifacts();
	project.validateBuildConfiguration();

	const version = project.version();
	const sourceDateEpoch = await project.epoch();
	const manifest = path.join(project.work, "manifest.yml");

	project.step("Render isolated Flatpak manifest");

	renderFlatpakManifest(project, manifest, version, sourceDateEpoch);

	const signing = await prepareFlatpakSigning(project);

	try {
		const repository = project.abs("build/out/cache/flatpak-repo");
		const state = project.abs("build/out/cache/flatpak-builder");
		const applicationDirectory = path.join(project.work, "application");
		const bundle = path.join(project.work, "application.flatpak");

		project.step("Ensure Flathub runtime remote is configured");

		await project.run("flatpak", [
			"remote-add",
			"--user",
			"--if-not-exists",
			"flathub",
			"https://flathub.org/repo/flathub.flatpakrepo",
		]);

		project.step("Build Flatpak application");

		await project.run(
			"flatpak-builder",
			[
				"--user",
				"--force-clean",
				"--disable-rofiles-fuse",
				"--arch=x86_64",
				"--install-deps-from=flathub",
				`--state-dir=${state}`,
				`--repo=${repository}`,
				...signing.args,
				applicationDirectory,
				manifest,
			],
			{
				timeoutMs: 60 * 60 * 1000,
			},
		);

		project.step("Create Flatpak bundle");

		await project.run("flatpak", [
			"build-bundle",
			"--arch=x86_64",
			...signing.args,
			repository,
			bundle,
			project.value("FLATPAK_ID"),
			"master",
		]);

		const artifactName = `${project.value("COMMON_BUILD_NAME")}-linux-amd64-${version}.flatpak`;

		const localBundle = project.abs("build/bin", artifactName);
		project.copy(bundle, localBundle);
		project.artifact(localBundle, artifactName);
	} finally {
		signing.cleanup();
	}
}

/**
 * @param {Project} project
 * @param {"ci"} target
 * @param {string} imageSuffix
 * @param {string} tag
 */
async function publishContainerImage(project, target, imageSuffix, tag) {
	nativeHost();

	const repository = (
		project.env.GITHUB_REPOSITORY ??
		project.env.CI_IMAGE_REPOSITORY ??
		""
	).toLowerCase();

	if (!repository) {
		throw new Error(
			"GITHUB_REPOSITORY or CI_IMAGE_REPOSITORY is required to publish a CI image",
		);
	}

	const image = `ghcr.io/${repository}-${imageSuffix}`;
	const cache = `ghcr.io/${repository}-ci-cache:buildcache`;

	const buildArguments = [
		"GO_VERSION",
		"GO_LINUX_AMD64_SHA256",
		"NODE_VERSION",
		"NODE_LINUX_X64_SHA256",
		"PNPM_VERSION",
		"TASK_VERSION",
		"WAILS_VERSION",
		"GO_LICENSES_VERSION",
		"GOLANGCI_LINT_VERSION",
	].flatMap((key) => ["--build-arg", `${key}=${project.value(key)}`]);

	const tags = ["--tag", `${image}:${tag}`];

	if (project.env.GITHUB_SHA) {
		tags.push("--tag", `${image}:sha-${project.env.GITHUB_SHA}`);
	}

	project.step(`Publish ${target} CI image ${image}:${tag}`);

	await project.run(
		"docker",
		[
			"buildx",
			"build",
			"--target",
			target,
			"--push",
			...tags,
			"--cache-from",
			`type=registry,ref=${cache}`,
			"--cache-to",
			`type=registry,ref=${cache},mode=max`,
			...buildArguments,
			"--file",
			"build/ci/linux.Dockerfile",
			".",
		],
		{
			timeoutMs: 90 * 60 * 1000,
		},
	);
}

await main("lin", async (project, action) => {
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
			await packageFlatpak(project);
			break;

		case "flatpak-install":
			await flatpakInstall(project);
			break;

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
					"build/flatpak/dev.yml",
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
					project.abs("build/out/dev/lin", project.value("COMMON_BUILD_NAME")),
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

		case "setup":
			nativeHost();
			await project.run("pkg-config", ["--exists", "gtk4", "webkitgtk-6.0"]);
			project.log(
				"INFO",
				"Linux GTK4 and WebKitGTK 6.0 dependencies are available",
			);
			break;

		case "ci-image-lin":
			await publishContainerImage(
				project,
				"ci",
				"ci-linux",
				project.value("CI_LINUX_IMAGE_TAG"),
			);
			break;

		default:
			throw new Error(`Unknown Linux action: ${action}`);
	}
});
