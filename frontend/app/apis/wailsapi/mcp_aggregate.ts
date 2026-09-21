import type { ArtifactRef, ArtifactRootID, StoreArtifact } from '@/spec/artifact';
import type { MCPAuthHealth, MCPRuntimeServerID, MCPSecretKind, MCPSecretWriteResult, MCPServerData } from '@/spec/mcp';

import type { IMCPAggregateAPI } from '@/apis/interface';
import { requiredObject, requireNonBlankString } from '@/apis/wailsapi/transport';
import {
	ArtifactRefForRuntimeServerID,
	DeleteMCPServerSecret,
	GetMCPServerAuthHealth,
	PutMCPServerSecret,
	RootIDForRuntimeCatalogID,
	RuntimeServerIDForArtifact,
	UpdateMCPServerInstallation,
	UpdateProtectedMCPServerInstallation,
} from '@/apis/wailsjs/go/main/MCPAggregateWrapper';

export class WailsMCPAggregateAPI implements IMCPAggregateAPI {
	async artifactRefForRuntimeServerID(server: MCPRuntimeServerID): Promise<ArtifactRef> {
		return requiredObject<ArtifactRef>(
			await ArtifactRefForRuntimeServerID(server as Parameters<typeof ArtifactRefForRuntimeServerID>[0]),
			'ArtifactRefForRuntimeServerID'
		);
	}

	async deleteMCPServerSecret(server: ArtifactRef, kind: MCPSecretKind, slot: string): Promise<void> {
		await DeleteMCPServerSecret(
			server as Parameters<typeof DeleteMCPServerSecret>[0],
			kind as Parameters<typeof DeleteMCPServerSecret>[1],
			slot
		);
	}

	async getMCPServerAuthHealth(server: ArtifactRef): Promise<MCPAuthHealth> {
		return requiredObject<MCPAuthHealth>(
			await GetMCPServerAuthHealth(server as Parameters<typeof GetMCPServerAuthHealth>[0]),
			'GetMCPServerAuthHealth'
		);
	}

	async putMCPServerSecret(
		server: ArtifactRef,
		kind: MCPSecretKind,
		slot: string,
		secret: string
	): Promise<MCPSecretWriteResult> {
		return requiredObject<MCPSecretWriteResult>(
			await PutMCPServerSecret(
				server as Parameters<typeof PutMCPServerSecret>[0],
				kind as Parameters<typeof PutMCPServerSecret>[1],
				slot,
				secret
			),
			'PutMCPServerSecret'
		);
	}

	async rootIDForRuntimeCatalogID(catalogID: string): Promise<ArtifactRootID> {
		return requireNonBlankString(
			await RootIDForRuntimeCatalogID(catalogID as Parameters<typeof RootIDForRuntimeCatalogID>[0]),
			'RootIDForRuntimeCatalogID'
		) as ArtifactRootID;
	}

	async runtimeServerIDForArtifact(artifact: ArtifactRef): Promise<MCPRuntimeServerID> {
		return requireNonBlankString(
			await RuntimeServerIDForArtifact(artifact as Parameters<typeof RuntimeServerIDForArtifact>[0]),
			'RuntimeServerIDForArtifact'
		) as MCPRuntimeServerID;
	}

	async updateMCPServerInstallation(
		server: ArtifactRef,
		expectedArtifactRevision: number,
		data: MCPServerData
	): Promise<StoreArtifact> {
		return requiredObject<StoreArtifact>(
			await UpdateMCPServerInstallation(
				server as Parameters<typeof UpdateMCPServerInstallation>[0],
				expectedArtifactRevision,
				data as Parameters<typeof UpdateMCPServerInstallation>[2]
			),
			'UpdateMCPServerInstallation'
		);
	}

	async updateProtectedMCPServerInstallation(
		server: ArtifactRef,
		expectedOverlayRevision: number,
		data: MCPServerData
	): Promise<void> {
		await UpdateProtectedMCPServerInstallation(
			server as Parameters<typeof UpdateProtectedMCPServerInstallation>[0],
			expectedOverlayRevision,
			data as Parameters<typeof UpdateProtectedMCPServerInstallation>[2]
		);
	}
}
