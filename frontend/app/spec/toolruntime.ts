import type { ToolOutputUnion } from '@/spec/tool';

export interface InvokeHTTPOptions {
	timeoutMS?: number;
	extraHeaders?: Record<string, string>;
	secrets?: Record<string, string>;
}

export interface InvokeGoOptions {
	timeoutMS?: number;
}

export interface InvokeToolResponse {
	outputs?: ToolOutputUnion[];
	meta?: Record<string, any>;
	isBuiltIn: boolean;
	isError?: boolean;
	errorMessage?: string;
}
