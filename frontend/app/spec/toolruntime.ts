import type { ToolOutputUnion } from '@/spec/tool';

export interface InvokeToolResponse {
	outputs?: ToolOutputUnion[];
	meta?: Record<string, unknown>;
	isError: boolean;
	errorMessage?: string;
}
