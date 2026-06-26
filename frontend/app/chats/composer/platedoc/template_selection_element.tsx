import type { PlateElementProps } from 'platejs/react';

import type { TemplateSelectionElementNode } from '@/chats/composer/platedoc/nodes';
import { computeEffectiveTemplate } from '@/chats/composer/platedoc/template_document_ops';
import { computeTemplateVarRequirements } from '@/prompts/lib/prompt_template_var_utils';

/**
 * Template selection element (data carrier).
 * We render it as a hidden inline element so it doesn't affect the text layout.
 * The toolbar is responsible for user-facing controls.
 */
export function TemplateSelectionElement(props: PlateElementProps<any>) {
	const { element, attributes, children } = props as any;
	const el = element as TemplateSelectionElementNode;

	// We still compute badges for accessibility/title, but hide it from visual flow.
	const { template, variablesSchema } = computeEffectiveTemplate(el);
	const req = computeTemplateVarRequirements(variablesSchema, el.variables);

	return (
		<span
			{...attributes}
			contentEditable={false}
			className="pointer-events-none sr-only"
			data-template-chip
			title={`Template: ${el.overrides?.displayName ?? template?.displayName ?? el.templateSlug} • pending vars: ${req.requiredCount}`}
			aria-hidden="true"
		>
			{/* Invisible info holder */}
			<span className="sr-only">{el.overrides?.displayName ?? template?.displayName ?? el.templateSlug}</span>
			{children}
		</span>
	);
}
