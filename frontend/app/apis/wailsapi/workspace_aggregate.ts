import type { ArtifactRef } from '@/spec/artifact';
import type { WorkspaceArtifactView } from '@/spec/workspace_store';

import type { IWorkspaceAggregateAPI } from '@/apis/interface';
import { requiredObject } from '@/apis/wailsapi/transport';
import { SetWorkspaceArtifactRuntimeDisabled } from '@/apis/wailsjs/go/main/WorkspaceAggregateWrapper';

export class WailsWorkspaceAggregateAPI implements IWorkspaceAggregateAPI {
	async setWorkspaceArtifactRuntimeDisabled(
		workspace: ArtifactRef,
		artifact: ArtifactRef,
		expectedRevision: number,
		runtimeDisabled: boolean
	): Promise<WorkspaceArtifactView> {
		return requiredObject<WorkspaceArtifactView>(
			await SetWorkspaceArtifactRuntimeDisabled(
				workspace as Parameters<typeof SetWorkspaceArtifactRuntimeDisabled>[0],
				artifact as Parameters<typeof SetWorkspaceArtifactRuntimeDisabled>[1],
				expectedRevision,
				runtimeDisabled
			),
			'SetWorkspaceArtifactRuntimeDisabled'
		);
	}
}
