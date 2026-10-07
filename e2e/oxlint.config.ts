import { defineConfig } from "oxlint";

export default defineConfig({
	categories: {
		correctness: "error",
		suspicious: "error",
	},

	plugins: ["typescript", "import", "promise"],

	env: {
		node: true,
		browser: true,
	},

	ignorePatterns: ["test-results/**", "playwright-report/**"],

	rules: {
		"no-console": "off",

		"no-unused-vars": [
			"error",
			{
				argsIgnorePattern: "^_",
				caughtErrorsIgnorePattern: "^_",
				varsIgnorePattern: "^_",
			},
		],

		"typescript/consistent-type-imports": "error",
		"typescript/no-floating-promises": "error",

		"import/no-nodejs-modules": "off",
		"import/no-default-export": "off",
	},
});
