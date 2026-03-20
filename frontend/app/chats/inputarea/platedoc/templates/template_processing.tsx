import {
	type MessageBlock,
	PromptRoleEnum,
	type PromptTemplate,
	PromptTemplateKind,
	type PromptVariable,
	VarSource,
	VarType,
} from '@/spec/prompt';

import { KEY_TEMPLATE_SELECTION, type TemplateSelectionElementNode } from '@/chats/inputarea/platedoc/nodes';

const TEMPLATE_PLACEHOLDER_RE = /\{\{([a-zA-Z_][a-zA-Z0-9_-]*)\}\}/g;

/**
 * Execution-ready derived representation of a selected template.
 */
export interface SelectedTemplateForRun {
	type: typeof KEY_TEMPLATE_SELECTION;
	bundleID: string;
	templateSlug: string;
	templateVersion: string;
	selectionID: string;

	// Final structures after applying local overrides
	template: PromptTemplate;
	blocks: MessageBlock[];
	variablesSchema: PromptVariable[];

	// Effective variable values for execution
	variableValues: Record<string, unknown>;

	// Requirements state
	requiredVariables: string[];
	requiredCount: number;

	// Convenience
	isReady: boolean;
}

function renderTemplateTextWithVariableValues(text: string, variableValues: Record<string, unknown>): string {
	return text.replace(TEMPLATE_PLACEHOLDER_RE, (_match, name: string) => {
		const value = variableValues[name];
		return value === undefined || value === null ? '' : (value as string);
	});
}

function getInstructionPromptPartFromSelection(selection: SelectedTemplateForRun): string {
	return selection.blocks
		.filter(block => block.role === PromptRoleEnum.System || block.role === PromptRoleEnum.Developer)
		.map(block => renderTemplateTextWithVariableValues(block.content, selection.variableValues).trim())
		.filter(Boolean)
		.join('\n\n');
}

export function getInstructionPromptPartsFromSelections(selections: SelectedTemplateForRun[]): string[] {
	return selections.map(getInstructionPromptPartFromSelection).filter(Boolean);
}

/**
 * Merge templateSnapshot with local overrides to produce effective template structures.
 */
export function computeEffectiveTemplate(el: TemplateSelectionElementNode): {
	template: PromptTemplate | undefined;
	blocks: MessageBlock[];
	variablesSchema: PromptVariable[];
} {
	const base = el.templateSnapshot;
	const blocks = el.overrides?.blocks ?? base?.blocks ?? [];
	const variablesSchema = el.overrides?.variables ?? base?.variables ?? [];

	return { template: base, blocks, variablesSchema };
}

export function effectiveVarValueLocal(varDef: PromptVariable, userValues: Record<string, unknown>): unknown {
	if (userValues[varDef.name] !== undefined && userValues[varDef.name] !== null) {
		return userValues[varDef.name];
	}
	if (varDef.source === VarSource.Static && varDef.staticVal !== undefined) {
		return varDef.staticVal;
	}
	if (varDef.default !== undefined) {
		return varDef.default;
	}

	if (varDef.type === VarType.String && !varDef.required) {
		return '';
	}

	return undefined;
}

function effectiveVarValue(varDef: PromptVariable, userValues: Record<string, unknown>): unknown {
	// Local override always wins if present
	if (userValues[varDef.name] !== undefined && userValues[varDef.name] !== null) {
		return userValues[varDef.name];
	}

	// Source & defaults
	switch (varDef.source) {
		case VarSource.Static:
			if (varDef.staticVal !== undefined && varDef.staticVal !== '') {
				return varDef.staticVal;
			}
			break;

		case VarSource.User:
		default:
			break;
	}

	// Fallback declared default
	if (varDef.default !== undefined && varDef.default !== '') return varDef.default;

	return undefined;
}

export function computeRequirements(variablesSchema: PromptVariable[], variableValues: Record<string, unknown>) {
	const requiredNames: string[] = [];
	const values: Record<string, unknown> = { ...variableValues };

	// Fill effective values
	for (const v of variablesSchema) {
		let val = values[v.name];
		if (val === undefined) {
			val = effectiveVarValue(v, variableValues);
		}
		values[v.name] = val;
	}

	// Identify required ones that remain empty
	for (const v of variablesSchema) {
		const val = values[v.name];
		if (v.required && (val === undefined || val === null || val === '')) {
			requiredNames.push(v.name);
		}
	}

	return {
		variableValues: values,
		requiredVariables: requiredNames,
		requiredCount: requiredNames.length,
	};
}

export function makeSelectedTemplateForRun(tsenode: TemplateSelectionElementNode): SelectedTemplateForRun {
	const { template, blocks, variablesSchema } = computeEffectiveTemplate(tsenode);

	const effTemplate: PromptTemplate =
		template ??
		({
			kind: PromptTemplateKind.Generic,
			id: '',
			displayName: tsenode.templateSlug,
			slug: tsenode.templateSlug,
			isEnabled: true,
			description: '',
			tags: [],
			blocks,
			variables: variablesSchema,
			isResolved: true,
			version: tsenode.templateVersion,
			createdAt: new Date().toISOString(),
			modifiedAt: new Date().toISOString(),
			isBuiltIn: false,
		} as PromptTemplate);

	const req = computeRequirements(variablesSchema, tsenode.variables);

	return {
		type: KEY_TEMPLATE_SELECTION,
		bundleID: tsenode.bundleID,
		templateSlug: tsenode.templateSlug,
		templateVersion: tsenode.templateVersion,
		selectionID: tsenode.selectionID,
		template: effTemplate,
		blocks,
		variablesSchema,

		variableValues: req.variableValues,
		requiredVariables: req.requiredVariables,
		requiredCount: req.requiredCount,

		isReady: req.requiredCount === 0,
	};
}

// Returns all user-facing blocks concatenated in template order.
// System/developer blocks are sent via systemPrompt, so only user blocks belong
// in the editor/message body.
export function getUserBlocksContent(el: TemplateSelectionElementNode): string {
	const { blocks } = computeEffectiveTemplate(el);
	return blocks
		.filter(block => block.role === PromptRoleEnum.User)
		.map(block => block.content)
		.filter(content => content.trim().length > 0)
		.join('\n\n');
}
