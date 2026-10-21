// @ts-check

import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("../", import.meta.url));
const placeholder = path.join(
	root,
	"frontend",
	"dist",
	"client",
	"build-placeholder.txt",
);

const placeholderContents =
	"FlexiGPT temporary Go embed placeholder. Safe to remove after quality checks.\n";

function validateVersion() {
	const version = process.env.VERSION_TAG ?? "";

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

	console.log(`Validated VERSION_TAG=${version}`);
}

function prepareEmbedPlaceholder() {
	fs.mkdirSync(path.dirname(placeholder), { recursive: true });
	fs.writeFileSync(placeholder, placeholderContents, { mode: 0o644 });
	console.log(`Prepared Go embed placeholder: ${placeholder}`);
}

function cleanEmbedPlaceholder() {
	if (!fs.existsSync(placeholder)) {
		return;
	}

	if (fs.readFileSync(placeholder, "utf8") === placeholderContents) {
		fs.rmSync(placeholder, { force: true });
		console.log(`Removed Go embed placeholder: ${placeholder}`);
	}
}

try {
	const action = process.argv[2];

	switch (action) {
		case "validate-version":
			validateVersion();
			break;

		case "embed-prepare":
			prepareEmbedPlaceholder();
			break;

		case "embed-clean":
			cleanEmbedPlaceholder();
			break;

		default:
			throw new Error(
				"Usage: node build/support.mjs <validate-version|embed-prepare|embed-clean>",
			);
	}
} catch (error) {
	console.error(
		error instanceof Error ? (error.stack ?? error.message) : error,
	);
	process.exitCode = 1;
}
