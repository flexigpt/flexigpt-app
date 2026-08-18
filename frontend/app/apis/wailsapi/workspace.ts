// oxlint-disable typescript/no-misused-spread
import type {
	ArtifactDefinitionSelector,
	ArtifactRef,
	ArtifactRootID,
	ArtifactSourceBinding,
	ArtifactSourceID,
} from '@/spec/artifact';
import type {
	AdoptWorkspaceOccurrenceBody,
	AttachWorkspaceSourceBody,
	CreateEmptyWorkspaceBody,
	CreateFilesystemWorkspaceBody,
	DetachWorkspaceSourceBody,
	PinWorkspaceArtifactBody,
	ReplaceWorkspacePrimarySourceBody,
	ResolveWorkspaceResourceResult,
	RetireWorkspaceResult,
	SetWorkspaceArtifactEnabledBody,
	SetWorkspaceArtifactRuntimeDisabledBody,
	SetWorkspacePrimarySourceBody,
	SuppressWorkspaceBindingBody,
	UnadoptWorkspaceArtifactBody,
	UnadoptWorkspaceArtifactResult,
	UnsuppressWorkspaceBindingResult,
	UpdateWorkspaceAttachmentBody,
	UpdateWorkspaceBody,
	WorkspaceArtifactView,
	WorkspaceCatalogView,
	WorkspaceContextInspectionView,
	WorkspaceContextLoadPlan,
	WorkspaceContextView,
	WorkspaceLoadPlanView,
	WorkspaceRef,
	WorkspaceRefreshResult,
	WorkspaceSkillLoadView,
	WorkspaceSkillView,
	WorkspaceSuppressionView,
	WorkspaceView,
} from '@/spec/workspace';

import type { IWorkspaceAPI } from '@/apis/interface';
import { rawJSONFromWails, requireWailsBody } from '@/apis/wailsapi/transport';
import {
	AdoptWorkspaceOccurrence,
	AttachWorkspaceSource,
	ComposeWorkspaceContext,
	ComposeWorkspaceLoadPlan,
	CreateEmptyWorkspace,
	CreateFilesystemWorkspace,
	DetachWorkspaceSource,
	GetWorkspace,
	GetWorkspaceArtifact,
	GetWorkspaceCatalog,
	ListWorkspaceArtifacts,
	ListWorkspaceContexts,
	ListWorkspaces,
	ListWorkspaceSkills,
	ListWorkspaceSuppressions,
	LoadWorkspaceContexts,
	LoadWorkspaceSkills,
	PinWorkspaceArtifact,
	PurgeWorkspace,
	PurgeWorkspaceArtifact,
	RefreshWorkspace,
	ReplaceWorkspacePrimarySource,
	ResolveWorkspaceResource,
	RetireWorkspace,
	SetWorkspaceArtifactEnabled,
	SetWorkspaceArtifactRuntimeDisabled,
	SetWorkspacePrimarySource,
	SuppressWorkspaceBinding,
	UnadoptWorkspaceArtifact,
	UnsuppressWorkspaceBinding,
	UpdateWorkspace,
	UpdateWorkspaceAttachment,
} from '@/apis/wailsjs/go/main/WorkspaceWrapper';

export class WailsWorkspaceAPI implements IWorkspaceAPI {
	async createFilesystemWorkspace(rootID: ArtifactRootID, body: CreateFilesystemWorkspaceBody): Promise<WorkspaceView> {
		const response = await CreateFilesystemWorkspace({
			rootID,
			Body: {
				workspaceID: body.workspaceID,
				sourceID: body.sourceID,
				sourceStorageKey: body.sourceStorageKey,
				displayName: body.displayName,
				description: body.description,
				rootPath: body.rootPath,
				discovery: body.discovery,
			},
		} as Parameters<typeof CreateFilesystemWorkspace>[0]);

		return requireWailsBody(response.Body, 'CreateFilesystemWorkspace') as WorkspaceView;
	}

	async createEmptyWorkspace(rootID: ArtifactRootID, body: CreateEmptyWorkspaceBody): Promise<WorkspaceView> {
		const response = await CreateEmptyWorkspace({
			rootID,
			Body: body,
		} as Parameters<typeof CreateEmptyWorkspace>[0]);

		return requireWailsBody(response.Body, 'CreateEmptyWorkspace') as WorkspaceView;
	}

	async getWorkspace(workspace: WorkspaceRef): Promise<WorkspaceView> {
		const response = await GetWorkspace({
			workspace,
		} as Parameters<typeof GetWorkspace>[0]);

		return requireWailsBody(response.Body, 'GetWorkspace') as WorkspaceView;
	}

	async listWorkspaces(rootID: ArtifactRootID): Promise<WorkspaceView[]> {
		const response = await ListWorkspaces({
			rootID,
		} as Parameters<typeof ListWorkspaces>[0]);

		const body = requireWailsBody(response.Body, 'ListWorkspaces');
		return (body.workspaces as WorkspaceView[]) ?? [];
	}

	async updateWorkspace(workspace: WorkspaceRef, body: UpdateWorkspaceBody): Promise<WorkspaceView> {
		const response = await UpdateWorkspace({
			workspace,
			Body: body,
		} as Parameters<typeof UpdateWorkspace>[0]);

		return requireWailsBody(response.Body, 'UpdateWorkspace') as WorkspaceView;
	}

	async replaceWorkspacePrimarySource(
		workspace: WorkspaceRef,
		body: ReplaceWorkspacePrimarySourceBody
	): Promise<WorkspaceView> {
		const response = await ReplaceWorkspacePrimarySource({
			workspace,
			Body: body,
		} as Parameters<typeof ReplaceWorkspacePrimarySource>[0]);

		return requireWailsBody(response.Body, 'ReplaceWorkspacePrimarySource') as WorkspaceView;
	}

	async setWorkspacePrimarySource(
		workspace: WorkspaceRef,
		body: SetWorkspacePrimarySourceBody
	): Promise<WorkspaceView> {
		const response = await SetWorkspacePrimarySource({
			workspace,
			Body: body,
		} as Parameters<typeof SetWorkspacePrimarySource>[0]);

		return requireWailsBody(response.Body, 'SetWorkspacePrimarySource') as WorkspaceView;
	}

	async retireWorkspace(workspace: WorkspaceRef, expectedRevision: number): Promise<RetireWorkspaceResult> {
		const response = await RetireWorkspace({
			workspace,
			expectedRevision,
		} as Parameters<typeof RetireWorkspace>[0]);

		const body = requireWailsBody(response.Body, 'RetireWorkspace');

		return {
			workspace: body.workspace,
			revision: body.revision,
		};
	}

	async purgeWorkspace(workspace: WorkspaceRef, expectedRevision: number): Promise<WorkspaceRef> {
		const response = await PurgeWorkspace({
			workspace,
			expectedRevision,
		} as Parameters<typeof PurgeWorkspace>[0]);

		const body = requireWailsBody(response.Body, 'PurgeWorkspace');
		return body.workspace as WorkspaceRef;
	}

	async attachWorkspaceSource(workspace: WorkspaceRef, body: AttachWorkspaceSourceBody): Promise<WorkspaceView> {
		const response = await AttachWorkspaceSource({
			workspace,
			Body: body,
		} as Parameters<typeof AttachWorkspaceSource>[0]);

		return requireWailsBody(response.Body, 'AttachWorkspaceSource') as WorkspaceView;
	}

	async updateWorkspaceAttachment(
		workspace: WorkspaceRef,
		sourceID: ArtifactSourceID,
		body: UpdateWorkspaceAttachmentBody
	): Promise<WorkspaceView> {
		const response = await UpdateWorkspaceAttachment({
			workspace,
			sourceID,
			Body: body,
		} as Parameters<typeof UpdateWorkspaceAttachment>[0]);

		return requireWailsBody(response.Body, 'UpdateWorkspaceAttachment') as WorkspaceView;
	}

	async detachWorkspaceSource(
		workspace: WorkspaceRef,
		sourceID: ArtifactSourceID,
		body: DetachWorkspaceSourceBody
	): Promise<WorkspaceView> {
		const response = await DetachWorkspaceSource({
			workspace,
			sourceID,
			expectedCollectionRevision: body.expectedCollectionRevision,
			expectedAttachmentRevision: body.expectedAttachmentRevision,
		} as Parameters<typeof DetachWorkspaceSource>[0]);

		return requireWailsBody(response.Body, 'DetachWorkspaceSource') as WorkspaceView;
	}

	async refreshWorkspace(workspace: WorkspaceRef): Promise<WorkspaceRefreshResult> {
		const response = await RefreshWorkspace({
			workspace,
		} as Parameters<typeof RefreshWorkspace>[0]);

		return requireWailsBody(response.Body, 'RefreshWorkspace') as WorkspaceRefreshResult;
	}

	async getWorkspaceCatalog(workspace: WorkspaceRef): Promise<WorkspaceCatalogView> {
		const response = await GetWorkspaceCatalog({
			workspace,
		} as Parameters<typeof GetWorkspaceCatalog>[0]);

		return requireWailsBody(response.Body, 'GetWorkspaceCatalog') as WorkspaceCatalogView;
	}

	async composeWorkspaceLoadPlan(workspace: WorkspaceRef, artifacts: ArtifactRef[]): Promise<WorkspaceLoadPlanView> {
		const response = await ComposeWorkspaceLoadPlan({
			workspace,
			Body: { artifacts },
		} as Parameters<typeof ComposeWorkspaceLoadPlan>[0]);
		const res = requireWailsBody(response.Body, 'ComposeWorkspaceLoadPlan');
		const r = {
			...res,
			items: res.items.map(i => {
				return Object.assign(i, {
					definition: {
						...i.definition,
						body: rawJSONFromWails(i.definition.body, 'item.definition.body'),
					},
				});
			}),
		} as WorkspaceLoadPlanView;
		return r;
	}

	async resolveWorkspaceResource(
		workspace: WorkspaceRef,
		artifact?: ArtifactRef,
		selector?: ArtifactDefinitionSelector
	): Promise<ResolveWorkspaceResourceResult> {
		if ((artifact === undefined) === (selector === undefined)) {
			throw new Error('Provide exactly one of artifact or selector.');
		}

		const response = await ResolveWorkspaceResource({
			workspace,
			Body: { artifact, selector },
		} as Parameters<typeof ResolveWorkspaceResource>[0]);
		const res = requireWailsBody(response.Body, 'ResolveWorkspaceResource');
		const r = {
			...res,
			definition: {
				...res.definition,
				body: rawJSONFromWails(res.definition.body, 'workspace.definition.body'),
			},
		} as ResolveWorkspaceResourceResult;
		return r;
	}

	async getWorkspaceArtifact(workspace: WorkspaceRef, artifact: ArtifactRef): Promise<WorkspaceArtifactView> {
		const response = await GetWorkspaceArtifact({
			workspace,
			artifact,
		} as Parameters<typeof GetWorkspaceArtifact>[0]);

		return requireWailsBody(response.Body, 'GetWorkspaceArtifact') as WorkspaceArtifactView;
	}

	async listWorkspaceArtifacts(workspace: WorkspaceRef): Promise<WorkspaceArtifactView[]> {
		const response = await ListWorkspaceArtifacts({
			workspace,
		} as Parameters<typeof ListWorkspaceArtifacts>[0]);

		const body = requireWailsBody(response.Body, 'ListWorkspaceArtifacts');
		return (body.artifacts as WorkspaceArtifactView[]) ?? [];
	}

	async adoptWorkspaceOccurrence(
		workspace: WorkspaceRef,
		body: AdoptWorkspaceOccurrenceBody
	): Promise<WorkspaceArtifactView> {
		const response = await AdoptWorkspaceOccurrence({
			workspace,
			Body: body,
		} as Parameters<typeof AdoptWorkspaceOccurrence>[0]);

		return requireWailsBody(response.Body, 'AdoptWorkspaceOccurrence') as WorkspaceArtifactView;
	}

	async pinWorkspaceArtifact(workspace: WorkspaceRef, body: PinWorkspaceArtifactBody): Promise<WorkspaceArtifactView> {
		const response = await PinWorkspaceArtifact({
			workspace,
			Body: body,
		} as Parameters<typeof PinWorkspaceArtifact>[0]);

		return requireWailsBody(response.Body, 'PinWorkspaceArtifact') as WorkspaceArtifactView;
	}

	async listWorkspaceSuppressions(workspace: WorkspaceRef): Promise<WorkspaceSuppressionView[]> {
		const response = await ListWorkspaceSuppressions({
			workspace,
		} as Parameters<typeof ListWorkspaceSuppressions>[0]);

		const body = requireWailsBody(response.Body, 'ListWorkspaceSuppressions');
		return (body.suppressions as WorkspaceSuppressionView[]) ?? [];
	}

	async suppressWorkspaceBinding(
		workspace: WorkspaceRef,
		body: SuppressWorkspaceBindingBody
	): Promise<WorkspaceSuppressionView> {
		const response = await SuppressWorkspaceBinding({
			workspace,
			Body: body,
		} as Parameters<typeof SuppressWorkspaceBinding>[0]);

		return requireWailsBody(response.Body, 'SuppressWorkspaceBinding') as WorkspaceSuppressionView;
	}

	async unsuppressWorkspaceBinding(
		workspace: WorkspaceRef,
		binding: ArtifactSourceBinding,
		expectedRevision: number
	): Promise<UnsuppressWorkspaceBindingResult> {
		const response = await UnsuppressWorkspaceBinding({
			workspace,
			binding,
			expectedRevision: expectedRevision,
		} as Parameters<typeof UnsuppressWorkspaceBinding>[0]);

		return requireWailsBody(response.Body, 'UnsuppressWorkspaceBinding');
	}

	async listWorkspaceContexts(workspace: WorkspaceRef): Promise<WorkspaceContextView[]> {
		const response = await ListWorkspaceContexts({
			workspace,
		} as Parameters<typeof ListWorkspaceContexts>[0]);

		const body = requireWailsBody(response.Body, 'ListWorkspaceContexts');
		return (body.contexts as WorkspaceContextView[]) ?? [];
	}

	async loadWorkspaceContexts(
		workspace: WorkspaceRef,
		artifacts?: ArtifactRef[]
	): Promise<WorkspaceContextInspectionView> {
		const response = await LoadWorkspaceContexts({
			workspace,
			Body: { artifacts },
		} as Parameters<typeof LoadWorkspaceContexts>[0]);

		return requireWailsBody(response.Body, 'LoadWorkspaceContexts') as WorkspaceContextInspectionView;
	}

	async composeWorkspaceContext(workspace: WorkspaceRef, artifacts?: ArtifactRef[]): Promise<WorkspaceContextLoadPlan> {
		const response = await ComposeWorkspaceContext({
			workspace,
			Body: { artifacts },
		} as Parameters<typeof ComposeWorkspaceContext>[0]);

		return requireWailsBody(response.Body, 'ComposeWorkspaceContext') as WorkspaceContextLoadPlan;
	}

	async listWorkspaceSkills(workspace: WorkspaceRef): Promise<WorkspaceSkillView[]> {
		const response = await ListWorkspaceSkills({
			workspace,
		} as Parameters<typeof ListWorkspaceSkills>[0]);

		const body = requireWailsBody(response.Body, 'ListWorkspaceSkills');
		return (body.skills as WorkspaceSkillView[]) ?? [];
	}

	async loadWorkspaceSkills(workspace: WorkspaceRef, artifacts: ArtifactRef[]): Promise<WorkspaceSkillLoadView> {
		const response = await LoadWorkspaceSkills({
			workspace,
			Body: { artifacts },
		} as Parameters<typeof LoadWorkspaceSkills>[0]);

		return requireWailsBody(response.Body, 'LoadWorkspaceSkills') as WorkspaceSkillLoadView;
	}

	async setWorkspaceArtifactEnabled(
		workspace: WorkspaceRef,
		artifact: ArtifactRef,
		body: SetWorkspaceArtifactEnabledBody
	): Promise<WorkspaceArtifactView> {
		const response = await SetWorkspaceArtifactEnabled({
			workspace,
			artifact,
			Body: body,
		} as Parameters<typeof SetWorkspaceArtifactEnabled>[0]);

		return requireWailsBody(response.Body, 'SetWorkspaceArtifactEnabled') as WorkspaceArtifactView;
	}

	async unadoptWorkspaceArtifact(
		workspace: WorkspaceRef,
		artifact: ArtifactRef,
		body: UnadoptWorkspaceArtifactBody
	): Promise<UnadoptWorkspaceArtifactResult> {
		const response = await UnadoptWorkspaceArtifact({
			workspace,
			artifact,
			expectedRevision: body.expectedRevision,
			suppress: body.suppress,
		} as Parameters<typeof UnadoptWorkspaceArtifact>[0]);

		const wbody = requireWailsBody(response.Body, 'UnadoptWorkspaceArtifact');
		return {
			artifact: wbody.artifact,
		};
	}

	async purgeWorkspaceArtifact(
		workspace: WorkspaceRef,
		artifact: ArtifactRef,
		expectedRevision: number
	): Promise<ArtifactRef> {
		const response = await PurgeWorkspaceArtifact({
			workspace,
			artifact,
			expectedRevision,
		} as Parameters<typeof PurgeWorkspaceArtifact>[0]);

		return requireWailsBody(response.Body, 'PurgeWorkspaceArtifact').artifact;
	}

	async setWorkspaceArtifactRuntimeDisabled(
		workspace: WorkspaceRef,
		artifact: ArtifactRef,
		body: SetWorkspaceArtifactRuntimeDisabledBody
	): Promise<WorkspaceArtifactView> {
		const response = await SetWorkspaceArtifactRuntimeDisabled({
			workspace,
			artifact,
			Body: body,
		} as Parameters<typeof SetWorkspaceArtifactRuntimeDisabled>[0]);

		return requireWailsBody(response.Body, 'SetWorkspaceArtifactRuntimeDisabled') as WorkspaceArtifactView;
	}
}
