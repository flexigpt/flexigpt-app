import type { DummyRuleMap, OxlintConfig } from 'oxlint';
import { defineConfig } from 'oxlint';

// oxlint-disable-next-line no-restricted-imports import/no-relative-parent-imports
import baseConfig from '../.oxlintrc.json' with { type: 'json' };

const baseIgnorePatterns = baseConfig.ignorePatterns ?? [];
const baseRules = baseConfig.rules ?? {};

const reactHooksRules: DummyRuleMap = {
	// Done by OxLint
	'jsreact-hooks/exhaustive-deps': 'off',
	'jsreact-hooks/rules-of-hooks': 'off',

	// Compiler rules
	'jsreact-hooks/config': 'error',
	'jsreact-hooks/error-boundaries': 'error',
	'jsreact-hooks/gating': 'error',
	'jsreact-hooks/globals': 'error',
	'jsreact-hooks/immutability': 'error',
	'jsreact-hooks/preserve-manual-memoization': 'error',
	'jsreact-hooks/purity': 'error',
	'jsreact-hooks/refs': 'error',
	'jsreact-hooks/set-state-in-effect': 'error',
	'jsreact-hooks/set-state-in-render': 'error',
	'jsreact-hooks/static-components': 'error',
	'jsreact-hooks/unsupported-syntax': 'error',
	'jsreact-hooks/use-memo': 'error',
	'jsreact-hooks/incompatible-library': 'error',
};

const reactYouMightNotNeedAnEffectRules: DummyRuleMap = {
	'react-you-might-not-need-an-effect/no-derived-state': 'error',
	'react-you-might-not-need-an-effect/no-chain-state-updates': 'error',
	'react-you-might-not-need-an-effect/no-adjust-state-on-prop-change': 'error',
	'react-you-might-not-need-an-effect/no-reset-all-state-on-prop-change': 'error',
	'react-you-might-not-need-an-effect/no-pass-live-state-to-parent': 'error',
	'react-you-might-not-need-an-effect/no-pass-data-to-parent': 'error',
	'react-you-might-not-need-an-effect/no-external-store-subscription': 'error',
	'react-you-might-not-need-an-effect/no-initialize-state': 'error',

	'react-you-might-not-need-an-effect/no-event-handler': 'off',
};

const betterTailwindCSSRules: DummyRuleMap = {
	'better-tailwindcss/enforce-canonical-classes': 'error',
	'better-tailwindcss/enforce-consistent-class-order': 'off',
	'better-tailwindcss/enforce-consistent-important-position': 'error',
	'better-tailwindcss/enforce-consistent-line-wrapping': 'off',
	'better-tailwindcss/enforce-consistent-variable-syntax': 'error',
	'better-tailwindcss/enforce-consistent-variant-order': 'error',
	'better-tailwindcss/enforce-logical-properties': 'off',
	'better-tailwindcss/enforce-shorthand-classes': 'error',
	'better-tailwindcss/no-conflicting-classes': 'error',
	'better-tailwindcss/no-deprecated-classes': 'error',
	'better-tailwindcss/no-duplicate-classes': 'error',
	'better-tailwindcss/no-restricted-classes': 'error',
	'better-tailwindcss/no-unknown-classes': [
		'error',
		{
			ignore: ['app-*'],
		},
	],
	'better-tailwindcss/no-unnecessary-whitespace': 'error',
};

const eslintRules: DummyRuleMap = {
	// Correctness. Default On.
	'no-unused-vars': [
		'error',
		{
			argsIgnorePattern: '^_',
			caughtErrorsIgnorePattern: '^_',
			varsIgnorePattern: '^_',
		},
	],

	// Restriction. Default On.
	complexity: 'off',
	'class-methods-use-this': 'off',
	'default-case': 'off',
	'no-bitwise': 'off',
	'no-console': 'off',
	'no-div-regex': 'off',
	'no-empty-function': 'off',
	'no-plusplus': 'off',

	'no-restricted-imports': [
		'error',
		{
			paths: [
				{
					importNames: ['default', 'React'],
					message: 'Avoid React import directly. Prefer explicitly importing.',
					name: 'react',
				},
				{
					importNames: ['FC', 'FunctionComponent'],
					message: 'Avoid FC imports. Prefer explicitly importing or explicitly typing your component props.',
					name: 'react',
				},
			],
			patterns: [
				{
					group: ['./*', '../*'],
					message: 'Relative imports are not allowed.',
				},
			],
		},
	],

	'no-undefined': 'off',
	'no-use-before-define': 'off',
	'no-void': 'off',

	// Suspicious. Default on.
	'no-underscore-dangle': 'off',

	// Pedantic. Default On.
	'max-lines-per-function': 'off',
	'max-lines': 'off',
	'no-inline-comments': 'off',
	'no-negated-condition': 'off',
	'require-unicode-regexp': 'off',
	'require-await': 'off',
	'max-depth': 'off',

	// Perf. Default on.
	'no-useless-call': 'error',
	'no-await-in-loop': 'off',

	// Style. Default on.
	curly: 'error',
	'default-case-last': 'error',
	'default-param-last': 'error',
	'func-name-matching': 'error',
	'func-names': 'error',
	'grouped-accessor-pairs': 'error',
	'guard-for-in': 'error',
	'no-duplicate-imports': ['error', { allowSeparateTypeImports: true }],
	'no-extra-label': 'error',
	'no-label-var': 'error',
	'no-labels': 'error',
	'no-lone-blocks': 'error',
	'no-multi-assign': 'error',
	'no-new-func': 'error',
	'no-return-assign': 'error',
	'no-script-url': 'error',
	'no-template-curly-in-string': 'error',
	'no-useless-computed-key': 'error',
	'prefer-const': 'error',
	'prefer-numeric-literals': 'error',
	'prefer-object-has-own': 'error',
	'prefer-promise-reject-errors': 'error',
	'prefer-regex-literals': 'error',
	'prefer-rest-params': 'error',
	'prefer-spread': 'error',
	'vars-on-top': 'error',
	yoda: 'error',

	'arrow-body-style': 'off',
	'func-style': 'off',
	'capitalized-comments': 'off',
	'id-length': 'off',
	'id-match': 'off',
	'init-declarations': 'off',
	'logical-assignment-operators': 'off',
	'max-params': 'off',
	'max-statements': 'off',
	'new-cap': 'off',
	'no-continue': 'off',
	'no-implicit-coercion': 'off',
	'no-magic-numbers': 'off',
	'no-multi-str': 'off',
	'no-nested-ternary': 'off',
	'no-ternary': 'off',
	'object-shorthand': 'off',
	'operator-assignment': 'off',
	'prefer-arrow-callback': 'off',
	'prefer-destructuring': 'off',
	'prefer-exponentiation-operator': 'off',
	'prefer-named-capture-group': 'off',
	'prefer-object-spread': 'off',
	'prefer-template': 'off',
	'sort-imports': 'off',
	'sort-keys': 'off',

	// Nursery. Default Off.
	'no-useless-assignment': 'error',
	'no-undef': 'off',
	'no-restricted-exports': [
		'error',
		{
			restrictDefaultExports: {
				defaultFrom: true,
				direct: true,
				named: true,
				namedFrom: true,
				namespaceFrom: true,
			},
		},
	],
};

const tsRules: DummyRuleMap = {
	// Restriction. Default On.
	'typescript/explicit-function-return-type': 'off',
	'typescript/explicit-member-accessibility': 'off',
	'typescript/explicit-module-boundary-types': 'off',
	'typescript/non-nullable-type-assertion-style': 'off',
	'typescript/no-explicit-any': 'off',
	'typescript/promise-function-async': 'off',

	// Suspicious. Default on.
	'typescript/no-unsafe-type-assertion': 'off',
	'typescript/no-unnecessary-type-assertion': 'off',
	'typescript/consistent-return': 'off',

	// Pedantic. Default On.
	'typescript/ban-ts-comment': [
		'error',
		{
			minimumDescriptionLength: 10,
		},
	],
	'typescript/restrict-plus-operands': [
		'error',
		{
			allowAny: false,
			allowBoolean: false,
			allowNullish: false,
			allowNumberAndString: false,
			allowRegExp: false,
		},
	],
	'typescript/return-await': ['error', 'error-handling-correctness-only'],
	'typescript/switch-exhaustiveness-check': ['error', { considerDefaultExhaustiveForUnions: true }],

	'typescript/no-misused-promises': 'off',
	'typescript/no-unsafe-argument': 'off',
	'typescript/no-unsafe-assignment': 'off',
	'typescript/no-unsafe-call': 'off',
	'typescript/prefer-readonly-parameter-types': 'off',
	'typescript/prefer-nullish-coalescing': 'off',
	'typescript/prefer-promise-reject-errors': 'off',
	'typescript/require-await': 'off',
	'typescript/strict-void-return': 'off',
	'typescript/strict-boolean-expressions': 'off',

	// Style. Default on.
	'typescript/adjacent-overload-signatures': 'error',
	'typescript/ban-tslint-comment': 'error',
	'typescript/class-literal-property-style': 'error',
	'typescript/consistent-generic-constructors': 'error',
	'typescript/consistent-indexed-object-style': 'error',
	'typescript/consistent-type-assertions': 'error',
	'typescript/consistent-type-definitions': 'error',
	'typescript/consistent-type-exports': 'error',
	'typescript/consistent-type-imports': 'error',
	'typescript/no-empty-interface': 'error',
	'typescript/no-inferrable-types': 'error',
	'typescript/no-unnecessary-qualifier': 'error',
	'typescript/parameter-properties': 'error',
	'typescript/prefer-find': 'error',
	'typescript/prefer-function-type': 'error',
	'typescript/prefer-reduce-type-parameter': 'error',
	'typescript/prefer-return-this-type': 'error',
	'typescript/prefer-string-starts-ends-with': 'error',
	'typescript/unified-signatures': 'error',

	'typescript/array-type': 'off',
	'typescript/dot-notation': 'off',
	'typescript/method-signature-style': 'off',
	'typescript/prefer-for-of': 'off',
	'typescript/prefer-readonly': 'off',
	'typescript/prefer-regexp-exec': 'off',

	// Nursery. Default Off.
	'typescript/no-unnecessary-condition': 'off',
	'typescript/prefer-optional-chain': 'off',
};

const importRules: DummyRuleMap = {
	// Restriction. Default on.
	'import/no-default-export': 'off',
	'import/unambiguous': 'off',

	// Pedantic. Default on.
	'import/max-dependencies': 'off',

	// Style. Default on.
	'import/first': 'error',
	'import/newline-after-import': 'error',
	'import/no-mutable-exports': 'error',
	'import/no-namespace': 'error',
	'import/no-nodejs-modules': ['error', { allow: ['path', 'node:path'] }],
	'import/consistent-type-specifier-style': ['error', 'prefer-top-level'],

	'import/exports-last': 'off',
	'import/group-exports': 'off',
	'import/no-anonymous-default-export': 'off',
	'import/no-duplicates': 'off',
	'import/no-named-default': 'off',
	'import/no-named-export': 'off',
	'import/prefer-default-export': 'off',

	// Nursery. Default Off.
	'import/export': 'error',
	'import/named': 'error',
};

const reactRules: DummyRuleMap = {
	// Restriction. Default on.
	'react/only-export-components': ['error', { allowConstantExport: true }],

	'react/no-multi-comp': 'off',
	'react/jsx-filename-extension': 'off',
	'react/jsx-no-literals': 'off',
	'react/forbid-component-props': 'off',

	// Suspicious. Default on.
	'react/react-in-jsx-scope': 'off',

	// Perf. Default on.
	'react/jsx-no-constructed-context-values': 'error',
	'react/no-array-index-key': 'off',
	'react/no-object-type-as-default-prop': 'off',

	// Style. Default on.
	'react/jsx-curly-brace-presence': 'error',
	'react/jsx-fragments': 'error',
	'react/jsx-pascal-case': 'error',
	'react/no-redundant-should-component-update': 'error',
	'react/prefer-es6-class': 'error',
	'react/self-closing-comp': 'error',

	'react/hook-use-state': 'off',
	'react/jsx-boolean-value': 'off',
	'react/jsx-handler-names': 'off',
	'react/jsx-max-depth': 'off',
	'react/jsx-props-no-spreading': 'off',
	'react/no-set-state': 'off',
	'react/state-in-constructor': 'off',

	// Nursery. Default Off.
	'react/require-render-return': 'error',
	'react/react-compiler': 'off',
};

const reactPerfRules: DummyRuleMap = {
	// Perf. Default on.
	'react-perf/jsx-no-jsx-as-prop': 'off',
	'react-perf/jsx-no-new-array-as-prop': 'off',
	'react-perf/jsx-no-new-function-as-prop': 'off',
	'react-perf/jsx-no-new-object-as-prop': 'off',
};

const promiseRules: DummyRuleMap = {
	// Restriction. Default on.
	'promise/catch-or-return': 'off',

	// Suspicious. Default on.
	'promise/always-return': 'off',

	// Style. Default on.
	'promise/no-nesting': 'error',
	'promise/no-return-wrap': 'error',
	'promise/param-names': 'error',
	'promise/prefer-catch': 'error',

	'promise/avoid-new': 'off',
	'promise/prefer-await-to-callbacks': 'off',
	'promise/prefer-await-to-then': 'off',

	// Nursery. Default Off.
	'promise/no-return-in-finally': 'error',
};

const oxcRules: DummyRuleMap = {
	// Restriction. Default on.
	'oxc/no-rest-spread-properties': 'off',
	'oxc/no-optional-chaining': 'off',
	'oxc/no-async-await': 'off',

	// Perf. Default on.
	'oxc/no-accumulating-spread': 'off',
	'oxc/no-map-spread': 'error',
};

const unicornRules: DummyRuleMap = {
	// Restriction. Default on.
	'unicorn/prefer-node-protocol': 'off',
	'unicorn/no-array-for-each': 'off',
	'unicorn/no-array-reduce': 'off',

	// Suspicious. Default On.
	'unicorn/prefer-add-event-listener': 'off',

	// Pedantic. Default On.
	'unicorn/no-negated-condition': 'off',
	'unicorn/no-useless-undefined': 'off',

	// Perf. Default on.
	'unicorn/prefer-set-has': 'error',
	'unicorn/prefer-array-flat-map': 'error',
	'unicorn/prefer-array-find': 'error',

	// Style. Default on.
	'unicorn/consistent-date-clone': 'error',
	'unicorn/consistent-template-literal-escape': 'error',
	'unicorn/error-message': 'error',
	'unicorn/filename-case': [
		'error',
		{
			case: 'snakeCase',
			ignore: ['react-router.config.ts'],
		},
	],
	'unicorn/no-unreadable-array-destructuring': 'error',
	'unicorn/no-useless-collection-argument': 'error',
	'unicorn/prefer-array-index-of': 'error',
	'unicorn/prefer-class-fields': 'error',
	'unicorn/prefer-classlist-toggle': 'error',
	'unicorn/prefer-dom-node-text-content': 'error',
	'unicorn/prefer-includes': 'error',
	'unicorn/prefer-keyboard-event-key': 'error',
	'unicorn/prefer-modern-dom-apis': 'error',
	'unicorn/prefer-object-from-entries': 'error',
	'unicorn/prefer-optional-catch-binding': 'error',
	'unicorn/prefer-spread': 'error',
	'unicorn/prefer-string-trim-start-end': 'error',
	'unicorn/require-array-join-separator': 'error',
	'unicorn/text-encoding-identifier-case': 'error',
	'unicorn/throw-new-error': 'error',

	'unicorn/catch-error-name': 'off',
	'unicorn/consistent-existence-index-check': 'off',
	'unicorn/custom-error-definition': 'off',
	'unicorn/empty-brace-spaces': 'off',

	'unicorn/max-nested-calls': 'off',
	'unicorn/no-array-method-this-argument': 'off',
	'unicorn/no-await-expression-member': 'off',
	'unicorn/no-console-spaces': 'off',
	'unicorn/no-nested-ternary': 'off',
	'unicorn/no-null': 'off',
	'unicorn/no-zero-fractions': 'off',
	'unicorn/number-literal-case': 'off',
	'unicorn/numeric-separators-style': 'off',
	'unicorn/prefer-bigint-literals': 'off',
	'unicorn/prefer-default-parameters': 'off',
	'unicorn/prefer-export-from': 'off',
	'unicorn/prefer-global-this': 'off',
	'unicorn/prefer-logical-operator-over-ternary': 'off',
	'unicorn/prefer-negative-index': 'off',
	'unicorn/prefer-reflect-apply': 'off',
	'unicorn/prefer-response-static-json': 'off',
	'unicorn/prefer-string-raw': 'off',
	'unicorn/prefer-structured-clone': 'off',
	'unicorn/prefer-ternary': 'off',
	'unicorn/relative-url-style': 'off',
	'unicorn/require-module-attributes': 'off',
	'unicorn/switch-case-braces': 'off',
	'unicorn/switch-case-break-position': 'off',

	// Nursery. Default Off.
	'unicorn/no-useless-iterator-to-array': 'error',
};

const nodeRules: DummyRuleMap = {
	// Restriction. Default on.
	'node/no-process-env': 'off',

	// Style. Default on.
	'node/global-require': 'error',
	'node/no-exports-assign': 'error',
	'node/no-mixed-requires': 'error',

	'node/callback-return': 'off',
	'node/no-sync': 'off',
};

// oxlint-disable-next-line no-restricted-exports
export default defineConfig({
	categories: {
		correctness: 'error',
		suspicious: 'error',
		restriction: 'error',
		pedantic: 'error',
		style: 'error',
		perf: 'error',
		nursery: 'off',
	},

	extends: [baseConfig as OxlintConfig],

	ignorePatterns: [...new Set([...baseIgnorePatterns, 'dist/**', 'app/apis/wailsjs/**', '.react-router/**'])],

	jsPlugins: [
		{
			name: 'jsreact-hooks',
			specifier: 'eslint-plugin-react-hooks',
		},
		'eslint-plugin-react-you-might-not-need-an-effect',
		'eslint-plugin-better-tailwindcss',
	],

	rules: {
		...baseRules,
		...nodeRules,
		...promiseRules,
		...eslintRules,
		...importRules,
		...tsRules,
		...unicornRules,
		...reactRules,
		...reactPerfRules,
		...oxcRules,
		...reactHooksRules,
		...reactYouMightNotNeedAnEffectRules,
		...betterTailwindCSSRules,
	},

	settings: {
		'eslint-plugin-better-tailwindcss': {
			detectComponentClasses: true,
			entryPoint: './app/globals.css',
		},
	},
});
