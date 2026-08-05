import type { InvokeGoOptions, InvokeHTTPOptions, InvokeToolResponse } from '@/spec/toolruntime';

import type { JSONRawString } from '@/lib/jsonschema_utils';

import type { IToolRuntimeAPI } from '@/apis/interface';
import { rawJSONToWails, requireWailsBody } from '@/apis/wailsapi/transport';
import { InvokeTool } from '@/apis/wailsjs/go/main/ToolRuntimeWrapper';
import type { spec } from '@/apis/wailsjs/go/models';

export class WailsToolRuntimeAPI implements IToolRuntimeAPI {
	async invokeTool(
		bundleID: string,
		toolSlug: string,
		version: string,
		args?: JSONRawString,
		httpOptions?: InvokeHTTPOptions,
		goOptions?: InvokeGoOptions
	): Promise<InvokeToolResponse> {
		const req = {
			BundleID: bundleID,
			ToolSlug: toolSlug,
			Version: version,
			Body: {
				args: rawJSONToWails(args ?? '{}', 'tool arguments'),
				httpOptions: httpOptions as spec.InvokeHTTPOptions | undefined,
				goOptions: goOptions as spec.InvokeGoOptions | undefined,
			} as spec.InvokeToolRequestBody,
		} as spec.InvokeToolRequest;
		const resp = await InvokeTool(req);
		return requireWailsBody(resp.Body, 'InvokeTool') as InvokeToolResponse;
	}
}
