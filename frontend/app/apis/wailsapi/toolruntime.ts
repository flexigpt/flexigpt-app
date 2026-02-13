import type { InvokeGoOptions, InvokeHTTPOptions, InvokeToolResponse } from '@/spec/toolruntime';

import type { JSONRawString } from '@/lib/jsonschema_utils';

import type { IToolRuntimeAPI } from '@/apis/interface';
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
	): Promise<any> {
		const req = {
			BundleID: bundleID,
			ToolSlug: toolSlug,
			Version: version,
			Body: {
				args: args ?? {},
				httpOptions: httpOptions as spec.InvokeHTTPOptions,
				goOptions: goOptions as spec.InvokeGoOptions,
			} as spec.InvokeToolRequestBody,
		} as spec.InvokeToolRequest;
		const resp = await InvokeTool(req);
		return resp.Body as InvokeToolResponse;
	}
}
