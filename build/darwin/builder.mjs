// @ts-check

import { randomBytes } from "node:crypto";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

import { main } from "../cli.mjs";
import { goBuildArgs } from "../lib.mjs";

/** @typedef {import("../lib.mjs").Project} Project */
/** @typedef {"amd64" | "arm64"} MacArchitecture */

function nativeHost() {
	if (process.platform !== "darwin") {
		throw new Error("macOS build tasks require a macOS host");
	}
}

/**
 * @param {Project} project
 * @param {MacArchitecture} architecture
 */
function nativeEnvironment(project, architecture) {
	const cpu = architecture === "amd64" ? "x86_64" : "arm64";
	const minimum = project.value("MACOS_MIN_VERSION");

	return {
		GOOS: "darwin",
		GOARCH: architecture,
		CGO_ENABLED: "1",
		MACOSX_DEPLOYMENT_TARGET: minimum,
		CGO_CFLAGS: `-arch ${cpu} -mmacosx-version-min=${minimum}`,
		CGO_LDFLAGS: `-arch ${cpu} -mmacosx-version-min=${minimum}`,
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
	const tags = project.tags(production, project.env.MACOS_EXTRA_TAGS ?? "");

	await project.dependencies();
	await project.frontend(tags, production, devServer);

	/** @type {MacArchitecture[]} */
	const architectures = production
		? ["amd64", "arm64"]
		: [process.arch === "arm64" ? "arm64" : "amd64"];

	return {
		version,
		tags,
		architectures,
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
		await project.licenses(
			prepared.tags,
			prepared.architectures.map((architecture) => ({
				...nativeEnvironment(project, architecture),
				FLEXIGPT_DEV_SERVER: "0",
			})),
		);
	}

	const name = project.value("COMMON_BUILD_NAME");

	const bundle = project.abs(
		production
			? project.value("MACOS_APP_BUNDLE_PATH")
			: `build/out/dev/mac/${name}.app`,
	);

	fs.rmSync(bundle, { recursive: true, force: true });

	const executable = path.join(bundle, "Contents/MacOS", name);
	const resources = path.join(bundle, "Contents/Resources");

	fs.mkdirSync(path.dirname(executable), { recursive: true });
	fs.mkdirSync(resources, { recursive: true });

	/** @type {string[]} */
	const binaries = [];

	for (const architecture of prepared.architectures) {
		const binary = path.join(project.work, `${name}-${architecture}`);

		project.step(`Compile macOS ${architecture} executable`);

		await project.run(
			"go",
			goBuildArgs({
				production,
				tags: prepared.tags,
				version: prepared.version,
				output: binary,
				backendDir: project.value("GO_BACKEND_DIR"),
			}),
			{
				env: {
					...nativeEnvironment(project, architecture),
					FLEXIGPT_DEV_SERVER: devServer ? "1" : "0",
				},
			},
		);

		binaries.push(project.file(binary));
	}

	if (production) {
		project.step("Create universal macOS executable");

		await project.run("xcrun", [
			"lipo",
			"-create",
			...binaries,
			"-output",
			executable,
		]);

		await project.run("xcrun", [
			"lipo",
			"-verify_arch",
			"x86_64",
			"arm64",
			executable,
		]);
	} else {
		project.copy(binaries[0], executable, 0o755);
	}

	fs.chmodSync(executable, 0o755);

	const plist = path.join(bundle, "Contents/Info.plist");

	project.render(
		project.abs(project.value("MACOS_INFO_PLIST_TEMPLATE_PATH")),
		plist,
		project.values({
			VERSION_TAG: prepared.version.slice(1),
		}),
	);

	if (!production) {
		await project.run("plutil", [
			"-insert",
			"NSAppTransportSecurity",
			"-json",
			'{"NSAllowsLocalNetworking":true}',
			plist,
		]);
	}

	await project.run("plutil", ["-lint", plist]);

	const wails = await project.wailsTool();
	const icon = path.join(project.work, "iconfile.icns");

	project.step("Generate macOS application icon");

	await project.run(
		wails,
		[
			"generate",
			"icons",
			"-input",
			project.abs(project.value("COMMON_ICON_PNG")),
			"-macfilename",
			icon,
		],
		{
			cwd: project.work,
		},
	);

	project.copy(icon, path.join(resources, "iconfile.icns"));
	project.copy(icon, path.join(resources, "appicon.icns"));

	if (production) {
		project.copyLicenses(path.join(resources, "licenses"));
	}

	project.step("Apply local ad-hoc signature");

	await project.run("codesign", ["--force", "--sign", "-", bundle]);
	await project.run("codesign", ["--verify", "--deep", "--strict", bundle]);

	return bundle;
}

/** @param {Project} project */
async function generateLicenses(project) {
	const prepared = await prepare(project, true, false);

	await project.licenses(
		prepared.tags,
		prepared.architectures.map((architecture) => ({
			...nativeEnvironment(project, architecture),
			FLEXIGPT_DEV_SERVER: "0",
		})),
	);
}

/**
 * @param {Project} project
 * @param {(keychain?: string) => Promise<void>} operation
 */
async function withSigningKeychain(project, operation) {
	if (!project.flag("SIGN")) {
		await operation();
		return;
	}

	const applicationCertificate = project.env.MACOS_SIGN_APP_CERT;
	const installerCertificate = project.env.MACOS_SIGN_INSTALLER_CERT;

	if (!applicationCertificate && !installerCertificate) {
		project.log(
			"INFO",
			"Using an existing local macOS signing identity from the login keychain",
		);

		await operation();
		return;
	}

	if (!applicationCertificate || !installerCertificate) {
		throw new Error(
			"MACOS_SIGN_APP_CERT and MACOS_SIGN_INSTALLER_CERT must be supplied together",
		);
	}

	const directory = fs.mkdtempSync(
		path.join(os.tmpdir(), "flexigpt-signing-keychain-"),
	);
	const keychain = path.join(directory, "signing.keychain-db");
	const keychainPassword = randomBytes(32).toString("hex");

	project.secrets.push(keychainPassword);

	fs.chmodSync(directory, 0o700);

	const previousText = await project.run(
		"security",
		["list-keychains", "-d", "user"],
		{
			capture: true,
			echo: false,
		},
	);

	const previousKeychains = [...previousText.matchAll(/"([^"]+)"/g)].map(
		(match) => match[1],
	);

	let created = false;
	let changedSearchList = false;

	try {
		project.step("Create disposable macOS signing keychain");

		await project.run(
			"security",
			["create-keychain", "-p", keychainPassword, keychain],
			{
				sensitive: true,
			},
		);

		created = true;

		await project.run(
			"security",
			["unlock-keychain", "-p", keychainPassword, keychain],
			{
				sensitive: true,
			},
		);

		await project.run("security", [
			"set-keychain-settings",
			"-lut",
			"21600",
			keychain,
		]);

		const certificates = [
			{
				data: applicationCertificate,
				password: project.value("MACOS_SIGN_APP_CERT_PASSWORD"),
				name: "application.p12",
			},
			{
				data: installerCertificate,
				password: project.value("MACOS_SIGN_INSTALLER_CERT_PASSWORD"),
				name: "installer.p12",
			},
		];

		for (const certificate of certificates) {
			const filename = path.join(directory, certificate.name);

			fs.writeFileSync(filename, Buffer.from(certificate.data, "base64"), {
				mode: 0o600,
			});

			await project.run(
				"security",
				[
					"import",
					filename,
					"-k",
					keychain,
					"-P",
					certificate.password,
					"-T",
					"/usr/bin/codesign",
					"-T",
					"/usr/bin/productbuild",
				],
				{
					sensitive: true,
				},
			);
		}

		await project.run("security", [
			"list-keychains",
			"-d",
			"user",
			"-s",
			...previousKeychains,
			keychain,
		]);

		changedSearchList = true;

		await project.run(
			"security",
			[
				"set-key-partition-list",
				"-S",
				"apple-tool:,apple:,codesign:",
				"-s",
				"-k",
				keychainPassword,
				keychain,
			],
			{
				sensitive: true,
			},
		);

		await operation(keychain);
	} finally {
		if (changedSearchList) {
			try {
				await project.run("security", [
					"list-keychains",
					"-d",
					"user",
					"-s",
					...previousKeychains,
				]);
			} catch (error) {
				project.log(
					"WARN",
					`Could not restore keychain search list: ${
						error instanceof Error ? error.message : String(error)
					}`,
				);
			}
		}

		if (created) {
			try {
				await project.run("security", ["delete-keychain", keychain]);
			} catch (error) {
				project.log(
					"WARN",
					`Could not delete temporary keychain: ${
						error instanceof Error ? error.message : String(error)
					}`,
				);
			}
		}

		fs.rmSync(directory, { recursive: true, force: true });
	}
}

/** @param {Project} project */
async function packageMac(project) {
	nativeHost();
	project.resetArtifacts();

	const version = project.version();
	const signing = project.flag("SIGN");
	const notarize = project.flag("NOTARIZE");

	if (notarize && !signing) {
		throw new Error("NOTARIZE=true requires SIGN=true");
	}

	if (signing) {
		if (
			!project
				.value("MACOS_SIGN_APPLE_DEVELOPER_IDENTITY")
				.startsWith("Developer ID Application:")
		) {
			throw new Error(
				"MACOS_SIGN_APPLE_DEVELOPER_IDENTITY must be a Developer ID Application identity",
			);
		}

		if (
			!project
				.value("MACOS_SIGN_INSTALLER_ID")
				.startsWith("Developer ID Installer:")
		) {
			throw new Error(
				"MACOS_SIGN_INSTALLER_ID must be a Developer ID Installer identity",
			);
		}
	}

	if (notarize) {
		project.value("MACOS_SIGN_APPLE_USERNAME");
		project.value("MACOS_SIGN_APPLE_APP_PASSWORD");
		project.value("MACOS_SIGN_APPLE_TEAM_ID");
	}

	const bundle = await build(project, true);
	const packageFile = path.join(project.work, "FlexiGPT.pkg");

	await withSigningKeychain(project, async (keychain) => {
		if (signing) {
			const entitlements = path.join(project.work, "entitlements.plist");

			project.copy(
				project.abs(project.value("MACOS_ENTITLEMENTS_TEMPLATE_PATH")),
				entitlements,
			);

			const signingArguments = [
				"--force",
				"--options",
				"runtime",
				"--timestamp",
				"--entitlements",
				entitlements,
				...(keychain ? ["--keychain", keychain] : []),
				"--sign",
				project.value("MACOS_SIGN_APPLE_DEVELOPER_IDENTITY"),
			];

			project.step("Sign macOS application executable");

			await project.run("codesign", [
				...signingArguments,
				path.join(bundle, "Contents/MacOS", project.value("COMMON_BUILD_NAME")),
			]);

			project.step("Sign macOS application bundle");

			await project.run("codesign", [...signingArguments, bundle]);

			await project.run("codesign", [
				"--verify",
				"--deep",
				"--strict",
				"--verbose=2",
				bundle,
			]);
		}

		project.step("Create macOS installer package");

		await project.run("productbuild", [
			...(signing
				? [
						"--sign",
						project.value("MACOS_SIGN_INSTALLER_ID"),
						...(keychain ? ["--keychain", keychain] : []),
					]
				: []),
			"--component",
			bundle,
			"/Applications",
			packageFile,
		]);

		if (signing) {
			await project.run("pkgutil", ["--check-signature", packageFile]);
		}

		if (notarize) {
			project.step("Submit macOS installer package for notarization");

			const authentication = [
				"--apple-id",
				project.value("MACOS_SIGN_APPLE_USERNAME"),
				"--password",
				project.value("MACOS_SIGN_APPLE_APP_PASSWORD"),
				"--team-id",
				project.value("MACOS_SIGN_APPLE_TEAM_ID"),
			];

			const response = await project.run(
				"xcrun",
				[
					"notarytool",
					"submit",
					packageFile,
					...authentication,
					"--wait",
					"--output-format",
					"json",
				],
				{
					capture: true,
					echo: false,
					sensitive: true,
					timeoutMs: 60 * 60 * 1000,
				},
			);

			/** @type {{status?: string, id?: string}} */
			const result = JSON.parse(response);

			if (result.status !== "Accepted") {
				if (result.id) {
					await project.run(
						"xcrun",
						["notarytool", "log", result.id, ...authentication],
						{
							sensitive: true,
						},
					);
				}

				throw new Error(
					`Apple notarization was not accepted: ${result.status ?? "unknown status"}`,
				);
			}

			project.step("Staple and validate notarization ticket");

			await project.run("xcrun", ["stapler", "staple", packageFile]);
			await project.run("xcrun", ["stapler", "validate", packageFile]);
			await project.run("spctl", [
				"--assess",
				"--type",
				"install",
				"--verbose=4",
				packageFile,
			]);
		}
	});

	const localPackage = project.abs(project.value("MACOS_PKG_BUNDLE_PATH"));

	project.copy(packageFile, localPackage);

	project.artifact(
		localPackage,
		`${project.value("COMMON_BUILD_NAME")}-macos-universal-${version}.pkg`,
	);
}

/**
 * @param {Project} project
 * @param {string} action
 */
export async function runAction(project, action) {
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

			await project.bindings(
				project.tags(false, project.env.MACOS_EXTRA_TAGS ?? ""),
			);
			break;

		case "licenses":
			await generateLicenses(project);
			break;

		case "package":
			await packageMac(project);
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
					"build/darwin/dev.yml",
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
						"build/out/dev/mac",
						`${project.value("COMMON_BUILD_NAME")}.app/Contents/MacOS`,
						project.value("COMMON_BUILD_NAME"),
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

		default:
			throw new Error(`Unknown macOS action: ${action}`);
	}
}

await main("mac", runAction);
