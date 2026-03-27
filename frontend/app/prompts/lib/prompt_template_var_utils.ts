import { type PromptVariable, VarSource, VarType } from '@/spec/prompt';

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

export function computeTemplateVarRequirements(
	variablesSchema: PromptVariable[],
	variableValues: Record<string, unknown>
) {
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
