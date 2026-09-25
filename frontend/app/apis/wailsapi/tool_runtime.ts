import type { InvokeToolResponse } from '@/spec/toolruntime';

import type { JSONRawString } from '@/lib/jsonschema_utils';

import type { IToolRuntimeAPI } from '@/apis/interface';
import { rawJSONToWails, requiredObject } from '@/apis/wailsapi/transport';
import { InvokeTool } from '@/apis/wailsjs/go/main/ToolRuntimeWrapper';

export class WailsToolRuntimeAPI implements IToolRuntimeAPI {
	async invokeTool(functionName: string, args?: JSONRawString, timeoutMS?: number): Promise<InvokeToolResponse> {
		return requiredObject<InvokeToolResponse>(
			await InvokeTool({
				function: functionName,
				args: args === undefined ? undefined : rawJSONToWails(args, 'tool arguments'),
				timeoutMS,
			}),
			'InvokeTool'
		);
	}
}
