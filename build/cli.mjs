// oxlint-disable typescript/no-redundant-type-constituents
// @ts-check

import path from "node:path";
import { fileURLToPath } from "node:url";

import {
	cleanEmbedPlaceholder,
	loadBuildVars,
	prepareEmbedPlaceholder,
	Project,
	runProject,
	validatePinnedVersions,
	validateVersion,
} from "./lib.mjs";

/** @param {unknown} error */
function fail(error) {
	console.error(
		error instanceof Error ? (error.stack ?? error.message) : String(error),
	);
	process.exitCode = 1;
}

/**
 * @param {string} platform
 * @param {(project: Project, action: string, args: string[]) => unknown | Promise<unknown>} handler
 * @param {string[]} [argv]
 */
export async function main(platform, handler, argv = process.argv.slice(2)) {
	const [action = "", ...args] = argv;
	/** @type {Project} */
	let project;

	try {
		if (!/^[a-z][a-z-]*$/.test(action)) {
			throw new Error("A valid build action is required");
		}
		project = new Project(platform, action);
	} catch (error) {
		fail(error);
		return;
	}

	try {
		await runProject(project, (/** @type {any} */ current) =>
			handler(current, action, args),
		);
	} catch {
		// runProject already logged the error with secret redaction.
		process.exitCode = 1;
	}
}

/** @type {Record<string, () => void>} */
const utilities = {
	"validate-version": () => {
		const version = validateVersion(process.env.VERSION_TAG ?? "");
		console.log(`Validated VERSION_TAG=${version}`);
	},
	"validate-toolchain": () => {
		validatePinnedVersions(loadBuildVars());
		console.log("Validated toolchain pins in build/buildvars.env");
	},
	"embed-prepare": prepareEmbedPlaceholder,
	"embed-clean": cleanEmbedPlaceholder,
};

if (
	process.argv[1] &&
	path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)
) {
	try {
		const [action = "", ...args] = process.argv.slice(2);
		if (args.length !== 0 || !Object.hasOwn(utilities, action)) {
			throw new Error(
				`Usage: node build/cli.mjs <${Object.keys(utilities).join("|")}>`,
			);
		}
		utilities[action]();
	} catch (error) {
		fail(error);
	}
}
