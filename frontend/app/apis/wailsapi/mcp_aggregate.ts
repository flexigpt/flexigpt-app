import type { ArtifactRef } from '@/spec/artifact';
import type {
	ManagedMCPCreateRequest,
	ManagedMCPCreateResult,
	ManagedMCPPolicyUpsertRequest,
	ManagedMCPPolicyUpsertResult,
	ManagedMCPReplaceRequest,
	ManagedMCPReplaceResult,
	MCPRuntimeServerID,
	MCPServerAggregateDetails,
	MCPServerData,
} from '@/spec/mcp';

import type { IMCPAggregateAPI } from '@/apis/interface';
import { requiredObject } from '@/apis/wailsapi/transport';
import {
	ClearMCPServerSecret,
	CreateMCPServer,
	DeleteMCPPolicy,
	DeleteMCPServer,
	GetMCPServer,
	GetMCPServerForRuntimeServer,
	SaveMCPPolicy,
	SaveMCPServerSettings,
	SetMCPServerSecret,
	UpdateMCPServer,
} from '@/apis/wailsjs/go/main/MCPAggregateWrapper';

export class WailsMCPAggregateAPI implements IMCPAggregateAPI {
	async getMCPServer(server: ArtifactRef): Promise<MCPServerAggregateDetails> {
		return requiredObject<MCPServerAggregateDetails>(
			await GetMCPServer(server as Parameters<typeof GetMCPServer>[0]),
			'GetMCPServer'
		);
	}

	async getMCPServerForRuntimeServer(server: MCPRuntimeServerID): Promise<MCPServerAggregateDetails> {
		return requiredObject<MCPServerAggregateDetails>(
			await GetMCPServerForRuntimeServer(server as Parameters<typeof GetMCPServerForRuntimeServer>[0]),
			'GetMCPServerForRuntimeServer'
		);
	}

	async createMCPServer(request: ManagedMCPCreateRequest): Promise<ManagedMCPCreateResult> {
		return requiredObject<ManagedMCPCreateResult>(
			await CreateMCPServer(request as Parameters<typeof CreateMCPServer>[0]),
			'CreateMCPServer'
		);
	}

	async updateMCPServer(request: ManagedMCPReplaceRequest): Promise<ManagedMCPReplaceResult> {
		return requiredObject<ManagedMCPReplaceResult>(
			await UpdateMCPServer(request as Parameters<typeof UpdateMCPServer>[0]),
			'UpdateMCPServer'
		);
	}

	async deleteMCPServer(server: ArtifactRef, expectedRevision: number): Promise<void> {
		await DeleteMCPServer(server as Parameters<typeof DeleteMCPServer>[0], expectedRevision);
	}

	async saveMCPPolicy(request: ManagedMCPPolicyUpsertRequest): Promise<ManagedMCPPolicyUpsertResult> {
		return requiredObject<ManagedMCPPolicyUpsertResult>(
			await SaveMCPPolicy(request as Parameters<typeof SaveMCPPolicy>[0]),
			'SaveMCPPolicy'
		);
	}

	async deleteMCPPolicy(policy: ArtifactRef, expectedRevision: number): Promise<void> {
		await DeleteMCPPolicy(policy as Parameters<typeof DeleteMCPPolicy>[0], expectedRevision);
	}

	async saveMCPServerSettings(
		server: ArtifactRef,
		expectedSettingsRevision: number,
		data: MCPServerData
	): Promise<MCPServerAggregateDetails> {
		return requiredObject<MCPServerAggregateDetails>(
			await SaveMCPServerSettings(
				server as Parameters<typeof SaveMCPServerSettings>[0],
				expectedSettingsRevision,
				data as Parameters<typeof SaveMCPServerSettings>[2]
			),
			'SaveMCPServerSettings'
		);
	}

	async setMCPServerSecret(server: ArtifactRef, input: string, secret: string): Promise<MCPServerAggregateDetails> {
		return requiredObject<MCPServerAggregateDetails>(
			await SetMCPServerSecret(server as Parameters<typeof SetMCPServerSecret>[0], input, secret),
			'SetMCPServerSecret'
		);
	}

	async clearMCPServerSecret(server: ArtifactRef, input: string): Promise<MCPServerAggregateDetails> {
		return requiredObject<MCPServerAggregateDetails>(
			await ClearMCPServerSecret(server as Parameters<typeof ClearMCPServerSecret>[0], input),
			'ClearMCPServerSecret'
		);
	}
}
