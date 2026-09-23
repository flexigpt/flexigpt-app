import type { InvokeGoOptions, InvokeToolResponse } from '@/spec/toolruntime';

import type { JSONRawString } from '@/lib/jsonschema_utils';

import type { IToolRuntimeAPI } from '@/apis/interface';
import type { spec } from '@/apis/wailsjs/go/models';
import { rawJSONToWails, requireWailsBody } from '@/apis/wailsapi/transport';
import { InvokeTool } from '@/apis/wailsjs/go/main/ToolRuntimeWrapper';

export class WailsToolRuntimeAPI implements IToolRuntimeAPI {
	async invokeTool(
		bundleID: string,
		toolSlug: string,
		version: string,
		args?: JSONRawString,
		goOptions?: InvokeGoOptions
	): Promise<InvokeToolResponse> {
		const req = {
			BundleID: bundleID,
			ToolSlug: toolSlug,
			Version: version,
			Body: {
				args: rawJSONToWails(args ?? '{}', 'tool arguments'),
				goOptions: goOptions as spec.InvokeGoOptions | undefined,
			} as spec.InvokeToolRequestBody,
		} as spec.InvokeToolRequest;
		const resp = await InvokeTool(req);
		return requireWailsBody(resp.Body, 'InvokeTool') as InvokeToolResponse;
	}
}
