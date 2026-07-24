import type {
	AttachWorkspaceSourcePayload,
	CreateEmptyWorkspacePayload,
	CreateFilesystemWorkspacePayload,
	DeleteWorkspaceResult,
	UpdateWorkspaceAttachmentPayload,
	UpdateWorkspacePayload,
	WorkspaceCatalogView,
	WorkspaceContextInspectionView,
	WorkspaceContextLoadPlan,
	WorkspaceContextView,
	WorkspaceRecordID,
	WorkspaceRecordView,
	WorkspaceRefreshResult,
	WorkspaceRootID,
	WorkspaceSkillLoadView,
	WorkspaceSkillView,
	WorkspaceSourceID,
	WorkspaceView,
} from '@/spec/workspace';

import type { IWorkspaceAPI } from '@/apis/interface';
import {
	AttachWorkspaceSource,
	ComposeWorkspaceContext,
	CreateEmptyWorkspace,
	CreateFilesystemWorkspace,
	DeleteWorkspace,
	DeleteWorkspaceRecord,
	DetachWorkspaceSource,
	GetWorkspace,
	GetWorkspaceCatalog,
	GetWorkspaceRecord,
	ListWorkspaceContexts,
	ListWorkspaces,
	ListWorkspaceSkills,
	LoadWorkspaceContexts,
	LoadWorkspaceSkills,
	RefreshWorkspace,
	SetWorkspaceRecordEnabled,
	SetWorkspaceRecordRuntimeDisabled,
	UpdateWorkspace,
	UpdateWorkspaceAttachment,
} from '@/apis/wailsjs/go/main/WorkspaceWrapper';
import type { workspace as workspaceModel } from '@/apis/wailsjs/go/models';

function requireResponseBody<T>(body: T | null | undefined, operation: string): T {
	if (body === null || body === undefined) {
		throw new Error(`${operation} returned an empty response body`);
	}
	return body;
}

/**
 * Flattened Workspace transport bridge.
 *
 * The public frontend API intentionally does not expose Go request/response
 * `Body` wrappers, internal source configuration, or raw Workspace data.
 */
export class WailsWorkspaceAPI implements IWorkspaceAPI {
	async createFilesystemWorkspace(payload: CreateFilesystemWorkspacePayload): Promise<WorkspaceView> {
		const request = {
			Body: payload as workspaceModel.CreateFilesystemWorkspaceRequestBody,
		} as workspaceModel.CreateFilesystemWorkspaceRequest;

		const response = await CreateFilesystemWorkspace(request);
		return requireResponseBody(response.Body, 'CreateFilesystemWorkspace') as WorkspaceView;
	}

	async createEmptyWorkspace(payload: CreateEmptyWorkspacePayload): Promise<WorkspaceView> {
		const request = {
			Body: payload as workspaceModel.CreateEmptyWorkspaceRequestBody,
		} as workspaceModel.CreateEmptyWorkspaceRequest;

		const response = await CreateEmptyWorkspace(request);
		return requireResponseBody(response.Body, 'CreateEmptyWorkspace') as WorkspaceView;
	}

	async getWorkspace(rootID: WorkspaceRootID): Promise<WorkspaceView> {
		const response = await GetWorkspace({
			RootID: rootID,
		} as workspaceModel.GetWorkspaceRequest);

		return requireResponseBody(response.Body, 'GetWorkspace') as WorkspaceView;
	}

	async listWorkspaces(): Promise<WorkspaceView[]> {
		const response = await ListWorkspaces({} as workspaceModel.ListWorkspacesRequest);
		const body = requireResponseBody(response.Body, 'ListWorkspaces');

		return body.workspaces as WorkspaceView[];
	}

	async updateWorkspace(rootID: WorkspaceRootID, payload: UpdateWorkspacePayload): Promise<WorkspaceView> {
		const request = {
			RootID: rootID,
			Body: payload as workspaceModel.UpdateWorkspaceRequestBody,
		} as workspaceModel.UpdateWorkspaceRequest;

		const response = await UpdateWorkspace(request);
		return requireResponseBody(response.Body, 'UpdateWorkspace') as WorkspaceView;
	}

	async deleteWorkspace(rootID: WorkspaceRootID, expectedRevision: number): Promise<DeleteWorkspaceResult> {
		const response = await DeleteWorkspace({
			RootID: rootID,
			ExpectedRevision: expectedRevision,
		} as workspaceModel.DeleteWorkspaceRequest);

		return requireResponseBody(response.Body, 'DeleteWorkspace') as DeleteWorkspaceResult;
	}

	async attachWorkspaceSource(
		rootID: WorkspaceRootID,
		sourceID: WorkspaceSourceID,
		payload: AttachWorkspaceSourcePayload
	): Promise<WorkspaceView> {
		const request = {
			RootID: rootID,
			Body: {
				sourceID,
				...payload,
			} as workspaceModel.AttachWorkspaceSourceRequestBody,
		} as workspaceModel.AttachWorkspaceSourceRequest;

		const response = await AttachWorkspaceSource(request);
		return requireResponseBody(response.Body, 'AttachWorkspaceSource') as WorkspaceView;
	}

	async updateWorkspaceAttachment(
		rootID: WorkspaceRootID,
		sourceID: WorkspaceSourceID,
		payload: UpdateWorkspaceAttachmentPayload
	): Promise<WorkspaceView> {
		const request = {
			RootID: rootID,
			SourceID: sourceID,
			Body: payload as workspaceModel.UpdateWorkspaceAttachmentRequestBody,
		} as workspaceModel.UpdateWorkspaceAttachmentRequest;

		const response = await UpdateWorkspaceAttachment(request);
		return requireResponseBody(response.Body, 'UpdateWorkspaceAttachment') as WorkspaceView;
	}

	async detachWorkspaceSource(
		rootID: WorkspaceRootID,
		sourceID: WorkspaceSourceID,
		expectedRootRevision: number,
		expectedAttachmentRevision: number
	): Promise<WorkspaceView> {
		const response = await DetachWorkspaceSource({
			RootID: rootID,
			SourceID: sourceID,
			ExpectedRootRevision: expectedRootRevision,
			ExpectedAttachmentRevision: expectedAttachmentRevision,
		} as workspaceModel.DetachWorkspaceSourceRequest);

		return requireResponseBody(response.Body, 'DetachWorkspaceSource') as WorkspaceView;
	}

	async refreshWorkspace(rootID: WorkspaceRootID): Promise<WorkspaceRefreshResult> {
		const response = await RefreshWorkspace({
			RootID: rootID,
		} as workspaceModel.RefreshWorkspaceRequest);

		return requireResponseBody(response.Body, 'RefreshWorkspace') as WorkspaceRefreshResult;
	}

	async getWorkspaceCatalog(rootID: WorkspaceRootID): Promise<WorkspaceCatalogView> {
		const response = await GetWorkspaceCatalog({
			RootID: rootID,
		} as workspaceModel.GetWorkspaceCatalogRequest);

		return requireResponseBody(response.Body, 'GetWorkspaceCatalog') as WorkspaceCatalogView;
	}

	async getWorkspaceRecord(rootID: WorkspaceRootID, recordID: WorkspaceRecordID): Promise<WorkspaceRecordView> {
		const response = await GetWorkspaceRecord({
			RootID: rootID,
			RecordID: recordID,
		} as workspaceModel.GetWorkspaceRecordRequest);

		return requireResponseBody(response.Body, 'GetWorkspaceRecord') as WorkspaceRecordView;
	}

	async listWorkspaceContexts(rootID: WorkspaceRootID): Promise<WorkspaceContextView[]> {
		const response = await ListWorkspaceContexts({
			RootID: rootID,
		} as workspaceModel.ListWorkspaceContextsRequest);
		const body = requireResponseBody(response.Body, 'ListWorkspaceContexts');

		return body.contexts as WorkspaceContextView[];
	}

	async loadWorkspaceContexts(
		rootID: WorkspaceRootID,
		recordIDs?: WorkspaceRecordID[]
	): Promise<WorkspaceContextInspectionView> {
		const request = {
			RootID: rootID,
			Body: {
				recordIDs,
			} as workspaceModel.LoadWorkspaceContextsRequestBody,
		} as workspaceModel.LoadWorkspaceContextsRequest;

		const response = await LoadWorkspaceContexts(request);
		return requireResponseBody(response.Body, 'LoadWorkspaceContexts') as WorkspaceContextInspectionView;
	}

	async composeWorkspaceContext(
		rootID: WorkspaceRootID,
		recordIDs?: WorkspaceRecordID[]
	): Promise<WorkspaceContextLoadPlan> {
		const request = {
			RootID: rootID,
			Body: {
				recordIDs,
			} as workspaceModel.ComposeWorkspaceContextRequestBody,
		} as workspaceModel.ComposeWorkspaceContextRequest;

		const response = await ComposeWorkspaceContext(request);
		return requireResponseBody(response.Body, 'ComposeWorkspaceContext') as WorkspaceContextLoadPlan;
	}

	async listWorkspaceSkills(rootID: WorkspaceRootID): Promise<WorkspaceSkillView[]> {
		const response = await ListWorkspaceSkills({
			RootID: rootID,
		} as workspaceModel.ListWorkspaceSkillsRequest);
		const body = requireResponseBody(response.Body, 'ListWorkspaceSkills');

		return body.skills as WorkspaceSkillView[];
	}

	async loadWorkspaceSkills(rootID: WorkspaceRootID, recordIDs: WorkspaceRecordID[]): Promise<WorkspaceSkillLoadView> {
		const request = {
			RootID: rootID,
			Body: {
				recordIDs,
			} as workspaceModel.LoadWorkspaceSkillsRequestBody,
		} as workspaceModel.LoadWorkspaceSkillsRequest;

		const response = await LoadWorkspaceSkills(request);
		return requireResponseBody(response.Body, 'LoadWorkspaceSkills') as WorkspaceSkillLoadView;
	}

	async setWorkspaceRecordEnabled(
		rootID: WorkspaceRootID,
		recordID: WorkspaceRecordID,
		expectedRevision: number,
		enabled: boolean
	): Promise<WorkspaceRecordView> {
		const request = {
			RootID: rootID,
			RecordID: recordID,
			Body: {
				expectedRevision,
				enabled,
			} as workspaceModel.SetWorkspaceRecordEnabledRequestBody,
		} as workspaceModel.SetWorkspaceRecordEnabledRequest;

		const response = await SetWorkspaceRecordEnabled(request);
		return requireResponseBody(response.Body, 'SetWorkspaceRecordEnabled') as WorkspaceRecordView;
	}

	async deleteWorkspaceRecord(
		rootID: WorkspaceRootID,
		recordID: WorkspaceRecordID,
		expectedRevision: number
	): Promise<void> {
		const response = await DeleteWorkspaceRecord({
			RootID: rootID,
			RecordID: recordID,
			ExpectedRevision: expectedRevision,
		} as workspaceModel.DeleteWorkspaceRecordRequest);

		requireResponseBody(response.Body, 'DeleteWorkspaceRecord');
	}

	async setWorkspaceRecordRuntimeDisabled(
		rootID: WorkspaceRootID,
		recordID: WorkspaceRecordID,
		expectedRevision: number,
		runtimeDisabled: boolean
	): Promise<WorkspaceRecordView> {
		const request = {
			RootID: rootID,
			RecordID: recordID,
			Body: {
				expectedRevision,
				runtimeDisabled,
			} as workspaceModel.SetWorkspaceRecordRuntimeDisabledRequestBody,
		} as workspaceModel.SetWorkspaceRecordRuntimeDisabledRequest;

		const response = await SetWorkspaceRecordRuntimeDisabled(request);
		return requireResponseBody(response.Body, 'SetWorkspaceRecordRuntimeDisabled') as WorkspaceRecordView;
	}
}
