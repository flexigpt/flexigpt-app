import type { ArtifactRef, MappedTarget } from '@/spec/artifact';
import type { CollectionView } from '@/spec/collection';
import type { ToolChoice } from '@/spec/inference';
import type { ResolvedToolView, ToolSelection } from '@/spec/tool';
import type { InvokeToolResponse } from '@/spec/toolruntime';

import type { JSONRawString } from '@/lib/jsonschema_utils';

import type { IToolAggregateAPI } from '@/apis/interface';
import type { aggregate, main as wailsMain } from '@/apis/wailsjs/go/models';
import { toolViewFromWails } from '@/apis/wailsapi/tool_store';
import { rawJSONToWails, requiredObject } from '@/apis/wailsapi/transport';
import {
	HydrateInferenceToolChoice,
	InvokeMappedTool,
	MapToolTarget,
	ResolveMappedTool,
} from '@/apis/wailsjs/go/main/ToolAggregateWrapper';

function resolvedToolViewFromWails(value: unknown, operation: string): ResolvedToolView {
	const resolved = requiredObject<ResolvedToolView>(value, operation);

	return {
		tool: toolViewFromWails(resolved.tool, `${operation}.tool`),
		collection: requiredObject<CollectionView>(resolved.collection, `${operation}.collection`),
	};
}

export class WailsToolAggregateAPI implements IToolAggregateAPI {
	async mapToolTarget(tool: ArtifactRef): Promise<MappedTarget> {
		return requiredObject<MappedTarget>(await MapToolTarget(tool), 'MapToolTarget');
	}

	async resolveMappedTool(target: MappedTarget): Promise<ResolvedToolView> {
		return resolvedToolViewFromWails(await ResolveMappedTool(target), 'ResolveMappedTool');
	}

	async invokeMappedTool(target: MappedTarget, args?: JSONRawString, timeoutMS?: number): Promise<InvokeToolResponse> {
		return requiredObject<InvokeToolResponse>(
			await InvokeMappedTool({
				target,
				args: args === undefined ? undefined : rawJSONToWails(args, 'mapped tool arguments'),
				timeoutMS,
			} as wailsMain.ToolAggregateInvokeRequest),
			'InvokeMappedTool'
		);
	}

	async hydrateInferenceToolChoice(selection: ToolSelection): Promise<ToolChoice> {
		return requiredObject<ToolChoice>(
			await HydrateInferenceToolChoice(selection as aggregate.ToolSelection),
			'HydrateInferenceToolChoice'
		);
	}
}
