#!/usr/bin/env node

import { spawn } from "node:child_process";
import fs from "node:fs";
import fsp from "node:fs/promises";
import os from "node:os";
import path from "node:path";

function input(name, fallback = "") {
	const nativeKey = `INPUT_${name.replace(/ /g, "_").toUpperCase()}`;
	const compatKey = `INPUT_${name.replace(/ /g, "_").replace(/-/g, "_").toUpperCase()}`;

	if (process.env[nativeKey] !== undefined) {
		return process.env[nativeKey];
	}

	if (process.env[compatKey] !== undefined) {
		return process.env[compatKey];
	}

	return fallback;
}

function env(name, fallback = "") {
	return process.env[name] ?? fallback;
}

function parseBool(value, fallback = false) {
	if (value === undefined || value === null || String(value).trim() === "")
		return fallback;
	const v = String(value).trim().toLowerCase();
	if (["1", "true", "yes", "y", "on"].includes(v)) return true;
	if (["0", "false", "no", "n", "off"].includes(v)) return false;
	throw new Error(`Invalid boolean value: ${value}`);
}

function parseJsonInput(name, raw, fallback, { required = false } = {}) {
	const text = String(raw ?? "").trim();
	if (!text) {
		if (required) throw new Error(`Input "${name}" is required.`);
		return fallback;
	}

	try {
		return JSON.parse(text);
	} catch (err) {
		throw new Error(`Input "${name}" is not valid JSON: ${err.message}`);
	}
}

function escapeCommandValue(value) {
	return String(value)
		.replace(/%/g, "%25")
		.replace(/\r/g, "%0D")
		.replace(/\n/g, "%0A");
}

function notice(message) {
	console.log(`::notice::${escapeCommandValue(message)}`);
}

function warn(message) {
	console.log(`::warning::${escapeCommandValue(message)}`);
}

function errorAnnotation(message) {
	console.log(`::error::${escapeCommandValue(message)}`);
}

function maskSecret(secret) {
	if (secret) {
		console.log(`::add-mask::${secret}`);
	}
}

function setOutput(name, value) {
	const file = env("GITHUB_OUTPUT");
	if (!file) return;
	fs.appendFileSync(file, `${name}<<__EOF__\n${String(value)}\n__EOF__\n`);
}

async function appendStepSummary(text) {
	const file = env("GITHUB_STEP_SUMMARY");
	if (!file) return;
	await fsp.appendFile(file, text);
}

function exists(p) {
	try {
		return fs.existsSync(p);
	} catch {
		return false;
	}
}

async function mkdirp(p) {
	await fsp.mkdir(p, { recursive: true });
}

async function rmrf(p) {
	await fsp.rm(p, { recursive: true, force: true });
}

function sanitizeSegment(value) {
	return (
		String(value)
			.replace(/[^A-Za-z0-9._-]+/g, "_")
			.replace(/^_+/, "")
			.replace(/_+$/, "")
			.slice(0, 120) || "item"
	);
}

function formatInt(value) {
	return Number(value || 0).toLocaleString("en-US");
}

function mdEscape(value) {
	return String(value ?? "")
		.replace(/\|/g, "\\|")
		.replace(/\r?\n/g, " ");
}

function shortText(value, max = 180) {
	const text = String(value ?? "")
		.replace(/\s+/g, " ")
		.trim();
	return text.length <= max ? text : `${text.slice(0, max - 3)}...`;
}

function excerpt(value, max = 1200) {
	const text = String(value ?? "")
		.replace(/\s+/g, " ")
		.trim();
	return text.length <= max ? text : `${text.slice(0, max - 3)}...`;
}

function ensureStringArray(name, value) {
	if (!Array.isArray(value)) {
		throw new Error(`"${name}" must be a JSON array.`);
	}

	return value
		.map((item, idx) => {
			if (item === null || item === undefined) {
				throw new Error(`"${name}" contains null/undefined at index ${idx}.`);
			}
			return String(item);
		})
		.filter(Boolean);
}

function normalizeExcludeDir(value, fallback) {
	if (value === undefined) return [...fallback];

	if (Array.isArray(value)) {
		return value
			.map(String)
			.map((v) => v.trim())
			.filter(Boolean);
	}

	if (typeof value === "string") {
		return value
			.split(",")
			.map((v) => v.trim())
			.filter(Boolean);
	}

	throw new Error(`excludeDir must be an array or comma-separated string.`);
}

function validateClocArgs(name, args) {
	for (const arg of args) {
		if (arg === "--json" || arg === "--out" || arg.startsWith("--out=")) {
			throw new Error(`${name} must not include --json or --out.`);
		}
	}
}

function normalizeClocArgs(value, fallback) {
	if (value === undefined) return [...fallback];

	if (!Array.isArray(value)) {
		throw new Error(`clocArgs must be a JSON array of strings.`);
	}

	const args = value.map(String);
	validateClocArgs("clocArgs", args);
	return args;
}

function normalizeRelativePath(name, value) {
	if (value === undefined || value === null || String(value).trim() === "") {
		return "";
	}

	const raw = String(value).replace(/\\/g, "/").trim();
	const normalized = path.posix.normalize(raw);

	if (
		path.posix.isAbsolute(normalized) ||
		normalized === ".." ||
		normalized.startsWith("../")
	) {
		throw new Error(
			`Invalid ${name} "${value}". It must be a relative path within the repo.`,
		);
	}

	return normalized === "." ? "" : normalized.replace(/\/+$/, "");
}

function normalizeRepoIdentifier(rawValue, { currentRepo, githubServerUrl }) {
	let value = String(rawValue ?? "").trim();
	if (!value) {
		throw new Error(`repo is empty`);
	}

	if (value === "self" || value === "." || value === "current") {
		if (!currentRepo) {
			throw new Error(
				`repo "${value}" cannot be resolved because GITHUB_REPOSITORY is empty`,
			);
		}
		return currentRepo;
	}

	const serverUrl = githubServerUrl || "https://github.com";
	let serverHost = "github.com";
	try {
		serverHost = new URL(serverUrl).host;
	} catch {
		// ignore
	}

	value = value.replace(/^https?:\/\//i, "");
	value = value.replace(
		new RegExp(`^${serverHost.replace(/\./g, "\\.")}\\/`, "i"),
		"",
	);
	value = value.replace(/^github\.com\//i, "");
	value = value.replace(/\.git$/i, "");
	value = value.replace(/^\/+/, "").replace(/\/+$/, "");

	const parts = value.split("/");
	if (parts.length !== 2 || !parts[0] || !parts[1]) {
		throw new Error(
			`repo "${rawValue}" is not a valid owner/repo or supported GitHub URL`,
		);
	}

	return `${parts[0]}/${parts[1]}`;
}

function emptyStats() {
	return {
		nFiles: 0,
		blank: 0,
		comment: 0,
		code: 0,
	};
}

function toStats(value = {}) {
	return {
		nFiles: Number(value.nFiles) || 0,
		blank: Number(value.blank) || 0,
		comment: Number(value.comment) || 0,
		code: Number(value.code) || 0,
	};
}

function addStats(target, source) {
	target.nFiles += Number(source.nFiles) || 0;
	target.blank += Number(source.blank) || 0;
	target.comment += Number(source.comment) || 0;
	target.code += Number(source.code) || 0;
	return target;
}

class CommandError extends Error {
	constructor(message, result) {
		super(message);
		this.name = "CommandError";
		this.result = result;
	}
}

async function runCommand(command, args, options = {}) {
	const { cwd, env: extraEnv = {}, stdoutPath, stderrPath } = options;

	if (stdoutPath) await mkdirp(path.dirname(stdoutPath));
	if (stderrPath) await mkdirp(path.dirname(stderrPath));

	return await new Promise((resolve, reject) => {
		let stdout = "";
		let stderr = "";

		const stdoutStream = stdoutPath
			? fs.createWriteStream(stdoutPath, { flags: "a" })
			: null;
		const stderrStream = stderrPath
			? fs.createWriteStream(stderrPath, { flags: "a" })
			: null;

		const child = spawn(command, args, {
			cwd,
			env: { ...process.env, ...extraEnv },
			stdio: ["ignore", "pipe", "pipe"],
		});

		child.on("error", (err) => {
			stdoutStream?.end();
			stderrStream?.end();
			reject(err);
		});

		child.stdout.on("data", (chunk) => {
			const text = chunk.toString();
			stdout += text;
			stdoutStream?.write(chunk);
		});

		child.stderr.on("data", (chunk) => {
			const text = chunk.toString();
			stderr += text;
			stderrStream?.write(chunk);
		});

		child.on("close", (code) => {
			stdoutStream?.end();
			stderrStream?.end();
			resolve({
				code: code ?? 1,
				stdout,
				stderr,
			});
		});
	});
}

function assertZero(result, context) {
	if (result.code !== 0) {
		throw new CommandError(
			`${context} failed with exit code ${result.code}`,
			result,
		);
	}
	return result;
}

async function hasCommand(command, args = ["--version"]) {
	try {
		const result = await runCommand(command, args);
		return result.code === 0;
	} catch {
		return false;
	}
}

async function ensureClocAvailable({ installCloc }) {
	if (await hasCommand("cloc")) {
		return;
	}

	if (!installCloc) {
		throw new Error(`cloc is not installed and install-cloc=false.`);
	}

	if (env("RUNNER_OS") !== "Linux") {
		throw new Error(
			`cloc is not installed. Auto-install is only implemented for Linux runners.`,
		);
	}

	notice("cloc not found, installing via apt-get");
	assertZero(await runCommand("sudo", ["apt-get", "update"]), "apt-get update");
	assertZero(
		await runCommand("sudo", ["apt-get", "install", "-y", "cloc"]),
		"apt-get install cloc",
	);

	if (!(await hasCommand("cloc"))) {
		throw new Error(`cloc still not available after installation.`);
	}
}

function gitAuthArgs(token, serverUrl) {
	if (!token) return [];
	const url = serverUrl.replace(/\/+$/, "");
	const basic = Buffer.from(`x-access-token:${token}`, "utf8").toString(
		"base64",
	);
	return ["-c", `http.${url}/.extraheader=AUTHORIZATION: basic ${basic}`];
}

async function gitRevParseHead(repoPath) {
	const result = await runCommand("git", ["-C", repoPath, "rev-parse", "HEAD"]);
	if (result.code !== 0) return "";
	return result.stdout.trim();
}

async function cloneRepo({
	repo,
	ref,
	token,
	serverUrl,
	dest,
	stdoutPath,
	stderrPath,
}) {
	const repoUrl = `${serverUrl.replace(/\/+$/, "")}/${repo}.git`;
	const auth = gitAuthArgs(token, serverUrl);

	await rmrf(dest);

	if (!ref) {
		const result = await runCommand(
			"git",
			[...auth, "clone", "--depth", "1", repoUrl, dest],
			{ stdoutPath, stderrPath },
		);
		assertZero(result, `git clone ${repo}`);
	} else {
		await mkdirp(dest);

		assertZero(
			await runCommand("git", ["init", dest], { stdoutPath, stderrPath }),
			`git init ${repo}`,
		);

		assertZero(
			await runCommand(
				"git",
				["-C", dest, "remote", "add", "origin", repoUrl],
				{ stdoutPath, stderrPath },
			),
			`git remote add origin ${repo}`,
		);

		assertZero(
			await runCommand(
				"git",
				[...auth, "-C", dest, "fetch", "--depth", "1", "origin", ref],
				{ stdoutPath, stderrPath },
			),
			`git fetch ${repo}@${ref}`,
		);

		assertZero(
			await runCommand(
				"git",
				["-C", dest, "checkout", "--detach", "FETCH_HEAD"],
				{ stdoutPath, stderrPath },
			),
			`git checkout ${repo}@${ref}`,
		);
	}

	return await gitRevParseHead(dest);
}

function parseRepoSpecs({
	reposJsonRaw,
	githubServerUrl,
	currentRepo,
	defaultToken,
	defaultRef,
	defaultExcludeDir,
	defaultClocArgs,
	defaultUseCurrentWorkspace,
}) {
	const parsed = parseJsonInput("repos-json", reposJsonRaw, null, {
		required: true,
	});

	if (!Array.isArray(parsed)) {
		throw new Error(`Input "repos-json" must be a JSON array.`);
	}

	return parsed.map((item, index) => {
		let obj;

		if (typeof item === "string") {
			obj = { repo: item };
		} else if (item && typeof item === "object" && !Array.isArray(item)) {
			obj = item;
		} else {
			throw new Error(
				`repos-json item at index ${index} must be a string or object.`,
			);
		}

		const repo = normalizeRepoIdentifier(obj.repo, {
			currentRepo,
			githubServerUrl,
		});
		const ref =
			obj.ref !== undefined && obj.ref !== null
				? String(obj.ref).trim()
				: String(defaultRef || "").trim();
		const token =
			obj.token !== undefined && obj.token !== null
				? String(obj.token)
				: defaultToken;
		const subdir = normalizeRelativePath("subdir", obj.subdir);
		const ignoredFile = normalizeRelativePath("ignoredFile", obj.ignoredFile);

		const useWorkspace =
			obj.useWorkspace !== undefined
				? Boolean(obj.useWorkspace)
				: defaultUseCurrentWorkspace;

		const excludeDir = normalizeExcludeDir(obj.excludeDir, defaultExcludeDir);

		const inheritDefaultClocArgs =
			obj.inheritDefaultClocArgs !== undefined
				? Boolean(obj.inheritDefaultClocArgs)
				: true;

		const repoClocArgs =
			obj.clocArgs !== undefined ? normalizeClocArgs(obj.clocArgs, []) : [];

		const clocArgs = inheritDefaultClocArgs
			? [...defaultClocArgs, ...repoClocArgs]
			: [...repoClocArgs];

		validateClocArgs("final cloc args", clocArgs);

		return {
			repo,
			ref,
			token,
			subdir,
			ignoredFile,
			useWorkspace,
			excludeDir,
			clocArgs,
		};
	});
}

function resolveOutputDir(rawOutputDir) {
	const workspace = env("GITHUB_WORKSPACE", process.cwd());
	const runnerTemp = env("RUNNER_TEMP", os.tmpdir());

	if (!rawOutputDir || !String(rawOutputDir).trim()) {
		return path.join(runnerTemp, "multi-repo-cloc");
	}

	if (path.isAbsolute(rawOutputDir)) {
		return rawOutputDir;
	}

	return path.join(workspace, rawOutputDir);
}

function repoDisplayName(repo, subdir) {
	return subdir ? `${repo}:${subdir}` : repo;
}

function compareLanguage(a, b) {
	return b.code - a.code || a.language.localeCompare(b.language);
}

function compareRepo(a, b) {
	const aRank = a.status === "ok" ? 0 : 1;
	const bRank = b.status === "ok" ? 0 : 1;
	if (aRank !== bRank) return aRank - bRank;
	return (
		b.sum.code - a.sum.code || a.display_name.localeCompare(b.display_name)
	);
}

function padCell(value, width, align = "left") {
	const text = String(value ?? "");
	if (text.length >= width) return text;
	const pad = " ".repeat(width - text.length);
	return align === "right" ? `${pad}${text}` : `${text}${pad}`;
}

function renderMarkdownTable(headers, rows, aligns = []) {
	const normalizedHeaders = headers.map((v) => String(v ?? ""));
	const normalizedRows = rows.map((row) => row.map((v) => String(v ?? "")));

	const widths = normalizedHeaders.map((header, colIndex) => {
		let width = header.length;
		for (const row of normalizedRows) {
			width = Math.max(width, (row[colIndex] ?? "").length);
		}
		return width;
	});

	const headerLine = `| ${normalizedHeaders
		.map((value, i) => padCell(value, widths[i], "left"))
		.join(" | ")} |`;

	const separatorLine = `| ${widths
		.map((width, i) => {
			const dashCount = Math.max(width - 1, 3);
			const dashes = "-".repeat(dashCount);
			const align = aligns[i] || "left";
			if (align === "right") return `${dashes}:`;
			if (align === "center") return `:${dashes}:`;
			return `:${dashes}`;
		})
		.join(" | ")} |`;

	const bodyLines = normalizedRows.map((row) => {
		return `| ${row
			.map((value, i) => padCell(value, widths[i], aligns[i] || "left"))
			.join(" | ")} |`;
	});

	return [headerLine, separatorLine, ...bodyLines].join("\n");
}

function refDisplay(item) {
	const sha = item.resolved_sha ? item.resolved_sha.slice(0, 12) : "";
	if (item.requested_ref && sha) return `${item.requested_ref} / ${sha}`;
	if (sha) return sha;
	if (item.requested_ref) return item.requested_ref;
	return "";
}

function deriveRepoSum(raw, repoLanguages) {
	const fromSum = toStats(raw.SUM || {});
	if (fromSum.nFiles || fromSum.blank || fromSum.comment || fromSum.code) {
		return fromSum;
	}

	return Object.values(repoLanguages).reduce(
		(acc, stats) => addStats(acc, stats),
		emptyStats(),
	);
}

function isEmptyClocResult(raw) {
	const sum = toStats(raw.SUM || {});
	const languageKeys = Object.keys(raw).filter(
		(key) => key !== "header" && key !== "SUM",
	);

	return (
		languageKeys.length === 0 &&
		sum.nFiles === 0 &&
		sum.blank === 0 &&
		sum.comment === 0 &&
		sum.code === 0
	);
}

function aggregateResults(results) {
	const aggregate = {
		schema_version: 1,
		generated_at: new Date().toISOString(),
		repo_count: results.length,
		scanned_repo_count: 0,
		failed_repo_count: 0,
		totals: emptyStats(),
		languages: {},
		repos: [],
	};

	for (const result of results) {
		if (result.status !== "ok") {
			aggregate.failed_repo_count += 1;
			aggregate.repos.push({
				repo: result.repo,
				subdir: result.subdir,
				display_name: repoDisplayName(result.repo, result.subdir),
				status: "error",
				source: result.source,
				requested_ref: result.requested_ref,
				resolved_sha: result.resolved_sha,
				raw_file: result.raw_file,
				stdout_log: result.stdout_log,
				stderr_log: result.stderr_log,
				error: result.error,
				sum: emptyStats(),
				languages: {},
			});
			continue;
		}

		const raw = result.clocData;
		const repoLanguages = {};

		for (const [language, stats] of Object.entries(raw)) {
			if (language === "header" || language === "SUM") continue;

			const parsed = toStats(stats);
			repoLanguages[language] = parsed;

			if (!aggregate.languages[language]) {
				aggregate.languages[language] = { repos: 0, ...emptyStats() };
			}

			addStats(aggregate.languages[language], parsed);
			aggregate.languages[language].repos += 1;
		}

		const repoSum = deriveRepoSum(raw, repoLanguages);

		addStats(aggregate.totals, repoSum);
		aggregate.scanned_repo_count += 1;

		aggregate.repos.push({
			repo: result.repo,
			subdir: result.subdir,
			display_name: repoDisplayName(result.repo, result.subdir),
			status: "ok",
			source: result.source,
			requested_ref: result.requested_ref,
			resolved_sha: result.resolved_sha,
			raw_file: result.raw_file,
			stdout_log: result.stdout_log,
			stderr_log: result.stderr_log,
			error: "",
			header: raw.header || {},
			sum: repoSum,
			languages: repoLanguages,
		});
	}

	aggregate.repos.sort(compareRepo);

	aggregate.language_list = Object.entries(aggregate.languages)
		.map(([language, stats]) => ({
			language,
			repos: Number(stats.repos) || 0,
			nFiles: Number(stats.nFiles) || 0,
			blank: Number(stats.blank) || 0,
			comment: Number(stats.comment) || 0,
			code: Number(stats.code) || 0,
		}))
		.sort(compareLanguage);

	aggregate.repo_list = aggregate.repos.map((repo) => ({
		repo: repo.repo,
		subdir: repo.subdir,
		display_name: repo.display_name,
		status: repo.status,
		source: repo.source,
		requested_ref: repo.requested_ref,
		resolved_sha: repo.resolved_sha,
		nFiles: repo.sum.nFiles,
		blank: repo.sum.blank,
		comment: repo.sum.comment,
		code: repo.sum.code,
		error: repo.error,
	}));

	return aggregate;
}

function renderMarkdownSummary(aggregate) {
	const lines = [];

	lines.push("# CLOC Summary");
	lines.push("");
	lines.push("## Aggregate overview");
	lines.push("");
	lines.push(
		renderMarkdownTable(
			["Metric", "Value"],
			[
				["Generated at", aggregate.generated_at],
				["Repos requested", formatInt(aggregate.repo_count)],
				["Repos scanned", formatInt(aggregate.scanned_repo_count)],
				["Repos failed", formatInt(aggregate.failed_repo_count)],
				["Total files", formatInt(aggregate.totals.nFiles)],
				["Total code", formatInt(aggregate.totals.code)],
				["Total comment", formatInt(aggregate.totals.comment)],
				["Total blank", formatInt(aggregate.totals.blank)],
			],
			["left", "right"],
		),
	);
	lines.push("");

	if (aggregate.language_list.length > 0) {
		lines.push("## By language");
		lines.push("");
		lines.push(
			renderMarkdownTable(
				["Language", "Repos", "Files", "Code", "Comment", "Blank"],
				aggregate.language_list.map((lang) => [
					mdEscape(lang.language),
					formatInt(lang.repos),
					formatInt(lang.nFiles),
					formatInt(lang.code),
					formatInt(lang.comment),
					formatInt(lang.blank),
				]),
				["left", "right", "right", "right", "right", "right"],
			),
		);

		lines.push("");
	}

	if (aggregate.repos.length > 0) {
		lines.push("## By repo");
		lines.push("");
		lines.push(
			renderMarkdownTable(
				["Repo", "Status", "Files", "Code", "Comment", "Blank", "Ref / SHA"],
				aggregate.repos.map((repo) => [
					`\`${mdEscape(repo.display_name)}\``,
					mdEscape(repo.status),
					formatInt(repo.sum.nFiles),
					formatInt(repo.sum.code),
					formatInt(repo.sum.comment),
					formatInt(repo.sum.blank),
					mdEscape(refDisplay(repo)),
				]),
				["left", "left", "right", "right", "right", "right", "left"],
			),
		);

		lines.push("");
	}

	const okRepos = aggregate.repos.filter((r) => r.status === "ok");
	if (okRepos.length > 0) {
		lines.push("## Repo language breakdown");
		lines.push("");

		for (const repo of okRepos) {
			const repoLanguages = Object.entries(repo.languages || {})
				.map(([language, stats]) => ({
					language,
					nFiles: Number(stats.nFiles) || 0,
					blank: Number(stats.blank) || 0,
					comment: Number(stats.comment) || 0,
					code: Number(stats.code) || 0,
				}))
				.sort(compareLanguage);

			lines.push(`### \`${mdEscape(repo.display_name)}\``);
			lines.push("");
			lines.push(
				renderMarkdownTable(
					["Metric", "Value"],
					[
						["Status", mdEscape(repo.status)],
						["Files", formatInt(repo.sum.nFiles)],
						["Code", formatInt(repo.sum.code)],
						["Comment", formatInt(repo.sum.comment)],
						["Blank", formatInt(repo.sum.blank)],
						["Ref / SHA", mdEscape(refDisplay(repo))],
					],
					["left", "right"],
				),
			);
			lines.push("");

			if (repoLanguages.length > 0) {
				lines.push(
					renderMarkdownTable(
						["Language", "Files", "Code", "Comment", "Blank"],
						repoLanguages.map((lang) => [
							mdEscape(lang.language),
							formatInt(lang.nFiles),
							formatInt(lang.code),
							formatInt(lang.comment),
							formatInt(lang.blank),
						]),
						["left", "right", "right", "right", "right"],
					),
				);
				lines.push("");
			} else {
				lines.push("_No language rows returned by cloc._");
				lines.push("");
			}
		}
	}

	const failed = aggregate.repos.filter((r) => r.status !== "ok");
	if (failed.length > 0) {
		lines.push("## Errors");
		lines.push("");
		lines.push(
			renderMarkdownTable(
				["Repo", "Error"],
				failed.map((repo) => [
					`\`${mdEscape(repo.display_name)}\``,
					mdEscape(shortText(repo.error)),
				]),
				["left", "left"],
			),
		);
		lines.push("");
	}

	return `${lines.join("\n")}\n`;
}

async function main() {
	const defaultToken = input("token", "");
	const defaultRef = input("default-ref", "");
	const githubServerUrl =
		input("github-server-url", "").trim() ||
		env("GITHUB_SERVER_URL", "https://github.com");
	const outputDir = resolveOutputDir(input("output-dir", ".cloc-report"));
	const useCurrentWorkspace = parseBool(
		input("use-current-workspace", "true"),
		true,
	);
	const installCloc = parseBool(input("install-cloc", "true"), true);
	const writeJobSummary = parseBool(input("write-job-summary", "true"), true);
	const printSummary = parseBool(input("print-summary", "false"), false);
	const failOnRepoError = parseBool(
		input("fail-on-repo-error", "false"),
		false,
	);

	const defaultExcludeDir = ensureStringArray(
		"default-exclude-dir-json",
		parseJsonInput(
			"default-exclude-dir-json",
			input("default-exclude-dir-json", "[]"),
			[],
		),
	);

	const defaultClocArgs = ensureStringArray(
		"default-cloc-args-json",
		parseJsonInput(
			"default-cloc-args-json",
			input("default-cloc-args-json", "[]"),
			[],
		),
	);

	validateClocArgs("default-cloc-args-json", defaultClocArgs);

	maskSecret(defaultToken);

	const currentRepo = env("GITHUB_REPOSITORY", "");

	const specs = parseRepoSpecs({
		reposJsonRaw: input("repos-json"),
		githubServerUrl,
		currentRepo,
		defaultToken,
		defaultRef,
		defaultExcludeDir,
		defaultClocArgs,
		defaultUseCurrentWorkspace: useCurrentWorkspace,
	});

	for (const spec of specs) {
		if (spec.token) maskSecret(spec.token);
	}

	await ensureClocAvailable({ installCloc });

	const rawDir = path.join(outputDir, "raw");
	const logDir = path.join(outputDir, "logs");
	const workDir = path.join(outputDir, "work");
	const aggregatePath = path.join(outputDir, "aggregate-cloc.json");
	const summaryPath = path.join(outputDir, "summary.md");
	const manifestPath = path.join(outputDir, "manifest.json");

	await rmrf(outputDir);
	await mkdirp(rawDir);
	await mkdirp(logDir);
	await mkdirp(workDir);

	const workspace = env("GITHUB_WORKSPACE", process.cwd());
	const currentWorkspaceGit = exists(path.join(workspace, ".git"));

	const results = [];

	for (let i = 0; i < specs.length; i += 1) {
		const spec = specs[i];
		const id = `${String(i + 1).padStart(3, "0")}__${sanitizeSegment(spec.repo)}${spec.subdir ? `__${sanitizeSegment(spec.subdir)}` : ""}${spec.ref ? `__${sanitizeSegment(spec.ref)}` : ""}`;

		const stdoutRel = path.join("logs", `${id}.stdout.log`);
		const stderrRel = path.join("logs", `${id}.stderr.log`);
		const rawRel = path.join("raw", `${id}.json`);

		const stdoutPath = path.join(outputDir, stdoutRel);
		const stderrPath = path.join(outputDir, stderrRel);
		const rawPath = path.join(outputDir, rawRel);

		await fsp.writeFile(stdoutPath, "");
		await fsp.writeFile(stderrPath, "");

		let source = "clone";
		let repoPath = "";
		let resolvedSha = "";

		console.log(`Processing ${repoDisplayName(spec.repo, spec.subdir)}`);

		try {
			const canUseWorkspace =
				spec.useWorkspace &&
				spec.repo === currentRepo &&
				!spec.ref &&
				currentWorkspaceGit;

			if (canUseWorkspace) {
				source = "workspace";
				repoPath = workspace;
				resolvedSha = await gitRevParseHead(workspace);
			} else {
				if (
					spec.useWorkspace &&
					spec.repo === currentRepo &&
					!spec.ref &&
					!currentWorkspaceGit
				) {
					warn(
						`Current repo ${spec.repo} requested useWorkspace but workspace is not checked out. Falling back to clone.`,
					);
				}

				repoPath = path.join(workDir, id);
				resolvedSha = await cloneRepo({
					repo: spec.repo,
					ref: spec.ref,
					token: spec.token,
					serverUrl: githubServerUrl,
					dest: repoPath,
					stdoutPath,
					stderrPath,
				});
			}

			const scanTarget = spec.subdir
				? path.join(repoPath, spec.subdir)
				: repoPath;

			if (!exists(scanTarget)) {
				throw new Error(`Scan target does not exist: ${spec.subdir || "."}`);
			}

			const clocCwd = repoPath;
			const clocTargetArg = spec.subdir || ".";

			const clocArgs = ["--json", `--out=${rawPath}`];

			if (spec.excludeDir.length > 0) {
				clocArgs.push(`--exclude-dir=${spec.excludeDir.join(",")}`);
			}

			if (spec.ignoredFile) {
				clocArgs.push(`--ignored=${spec.ignoredFile}`);
			}

			clocArgs.push(...spec.clocArgs);
			clocArgs.push(clocTargetArg);

			const clocResult = await runCommand("cloc", clocArgs, {
				cwd: clocCwd,
				stdoutPath,
				stderrPath,
			});
			assertZero(clocResult, `cloc ${spec.repo}`);

			if (!exists(rawPath)) {
				throw new Error(`cloc completed but did not produce JSON output.`);
			}

			let clocData;
			try {
				clocData = JSON.parse(await fsp.readFile(rawPath, "utf8"));
			} catch (err) {
				throw new Error(`Failed to parse cloc JSON output: ${err.message}`);
			}

			if (isEmptyClocResult(clocData)) {
				throw new Error(
					`cloc returned an empty result. This usually means all files were excluded or --vcs=git was not evaluated from the repo root.`,
				);
			}

			results.push({
				repo: spec.repo,
				subdir: spec.subdir,
				status: "ok",
				source,
				requested_ref: spec.ref,
				resolved_sha: resolvedSha,
				raw_file: rawRel,
				stdout_log: stdoutRel,
				stderr_log: stderrRel,
				error: "",
				clocData,
			});
		} catch (err) {
			let message = err.message || "unknown error";

			if (err instanceof CommandError) {
				const tail = excerpt(err.result?.stderr || err.result?.stdout || "");
				if (tail) {
					message = `${message}: ${tail}`;
				}
			}

			errorAnnotation(`${repoDisplayName(spec.repo, spec.subdir)}: ${message}`);

			results.push({
				repo: spec.repo,
				subdir: spec.subdir,
				status: "error",
				source,
				requested_ref: spec.ref,
				resolved_sha: resolvedSha,
				raw_file: "",
				stdout_log: stdoutRel,
				stderr_log: stderrRel,
				error: message,
			});
		}
	}

	const aggregate = aggregateResults(results);
	const summaryMarkdown = renderMarkdownSummary(aggregate);
	const manifest = results.map((result) => {
		const copy = { ...result };
		delete copy.clocData;
		return copy;
	});

	await fsp.writeFile(manifestPath, `${JSON.stringify(manifest, null, 2)}\n`);
	await fsp.writeFile(aggregatePath, `${JSON.stringify(aggregate, null, 2)}\n`);
	await fsp.writeFile(summaryPath, summaryMarkdown);

	if (printSummary) {
		console.log("----- summary.md -----");
		console.log(summaryMarkdown);
	}

	if (writeJobSummary) {
		await appendStepSummary(summaryMarkdown);
	}

	setOutput("output_dir", outputDir);
	setOutput("aggregate_json", aggregatePath);
	setOutput("summary_markdown", summaryPath);
	setOutput("manifest_json", manifestPath);
	setOutput("repo_count", aggregate.repo_count);
	setOutput("scanned_repo_count", aggregate.scanned_repo_count);
	setOutput("failed_repo_count", aggregate.failed_repo_count);
	setOutput("total_files", aggregate.totals.nFiles);
	setOutput("total_blank", aggregate.totals.blank);
	setOutput("total_comment", aggregate.totals.comment);
	setOutput("total_code", aggregate.totals.code);
	setOutput("has_errors", aggregate.failed_repo_count > 0 ? "true" : "false");

	console.log(`Output directory: ${outputDir}`);
	console.log(`Aggregate JSON: ${aggregatePath}`);
	console.log(`Summary Markdown: ${summaryPath}`);

	if (failOnRepoError && aggregate.failed_repo_count > 0) {
		process.exitCode = 1;
	}
}

main().catch((err) => {
	errorAnnotation(err.stack || err.message || String(err));
	process.exit(1);
});
