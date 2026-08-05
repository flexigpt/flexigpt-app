// oxlint-disable typescript/no-misused-spread
import type {
	ArtifactDefinitionSelector,
	ArtifactDefinitionView,
	ArtifactRef,
	ArtifactRootID,
	ArtifactSourceBinding,
	ArtifactSourceID,
} from '@/spec/artifact';
import {
	ArtifactAdoptionMode,
	ArtifactDiagnosticSeverity,
	ArtifactOccurrenceState,
	ArtifactState,
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
import {
	WorkspaceAttachmentRole,
	WorkspaceContextCompositionStatus,
	WorkspaceContextMediaType,
	WorkspaceContextRole,
	WorkspaceMode,
	WorkspaceSkillInsert,
} from '@/spec/workspace';

import type { IWorkspaceAPI } from '@/apis/interface';
import { enumFromWails, rawJSONFromWails, requireWailsBody, toFrontendDate } from '@/apis/wailsapi/transport';
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
import type { workspace as wailsWorkspace } from '@/apis/wailsjs/go/models';

function artifactDefinitionFromWails(
	definition: wailsWorkspace.WorkspaceDefinitionView,
	operation: string
): ArtifactDefinitionView {
	return {
		...definition,
		body: rawJSONFromWails(definition.body, `${operation}.definition.body`),
	} as ArtifactDefinitionView;
}

function diagnosticsFromWails(
	values:
		| Array<{
				severity: unknown;
				code: string;
				message: string;
				location?: {
					locator?: string;
					subresourceLocator?: string;
					line?: number;
					column?: number;
				};
		  }>
		| undefined
) {
	return values?.map(value => ({
		severity: enumFromWails(value.severity, ArtifactDiagnosticSeverity, 'workspace.diagnostic.severity'),
		code: value.code,
		message: value.message,
		...(value.location === undefined ? {} : { location: { ...value.location } }),
	}));
}

function workspaceViewFromWails(view: wailsWorkspace.WorkspaceView): WorkspaceView {
	return {
		...view,
		mode: enumFromWails(view.mode, WorkspaceMode, 'workspace.mode'),
		attachments: view.attachments.map(attachment => ({
			...attachment,
			role: enumFromWails(attachment.role, WorkspaceAttachmentRole, 'workspace.attachment.role'),
			diagnostics: diagnosticsFromWails(attachment.diagnostics),
		})),
	} as WorkspaceView;
}

function workspaceArtifactFromWails(view: wailsWorkspace.WorkspaceArtifactView): WorkspaceArtifactView {
	return {
		...view,
		state: enumFromWails(view.state, ArtifactState, 'workspace.artifact.state'),
		adoption: enumFromWails(view.adoption, ArtifactAdoptionMode, 'workspace.artifact.adoption'),
		diagnostics: diagnosticsFromWails(view.diagnostics),
	} as WorkspaceArtifactView;
}

function workspaceOccurrenceFromWails(
	view: wailsWorkspace.WorkspaceOccurrenceView
): WorkspaceCatalogView['occurrences'][number] {
	return {
		...view,
		state: enumFromWails(view.state, ArtifactOccurrenceState, 'workspace.occurrence.state'),
		diagnostics: diagnosticsFromWails(view.diagnostics),
	} as WorkspaceCatalogView['occurrences'][number];
}

function workspaceResourceFromWails(
	view: wailsWorkspace.WorkspaceResourceView
): WorkspaceCatalogView['resources'][number] {
	return {
		...view,
		artifact: workspaceArtifactFromWails(view.artifact),
		diagnostics: diagnosticsFromWails(view.diagnostics),
	} as WorkspaceCatalogView['resources'][number];
}

function workspaceLoadPlanFromWails(plan: wailsWorkspace.WorkspaceLoadPlanView): WorkspaceLoadPlanView {
	return {
		...plan,
		items: plan.items.map(item => ({
			...item,
			artifact: workspaceArtifactFromWails(item.artifact),
			definition: artifactDefinitionFromWails(item.definition, 'ComposeWorkspaceLoadPlan'),
		})),
		diagnostics: diagnosticsFromWails(plan.diagnostics),
	} as WorkspaceLoadPlanView;
}

function resolvedWorkspaceResourceFromWails(
	result: wailsWorkspace.ResolveWorkspaceResourceResponseBody
): ResolveWorkspaceResourceResult {
	return {
		...result,
		resource: workspaceResourceFromWails(result.resource),
		definition: artifactDefinitionFromWails(result.definition, 'ResolveWorkspaceResource'),
	} as ResolveWorkspaceResourceResult;
}

function workspaceCatalogFromWails(value: wailsWorkspace.WorkspaceCatalogView): WorkspaceCatalogView {
	return {
		...value,
		workspace: workspaceViewFromWails(value.workspace),
		resources: value.resources.map(workspaceResourceFromWails),
		groups: value.groups.map(group => ({
			...group,
			resources: group.resources.map(workspaceResourceFromWails),
			unrecorded: group.unrecorded.map(workspaceOccurrenceFromWails),
		})),
		occurrences: value.occurrences.map(workspaceOccurrenceFromWails),
		validOccurrences: value.validOccurrences.map(workspaceOccurrenceFromWails),
		invalidOccurrences: value.invalidOccurrences.map(workspaceOccurrenceFromWails),
		missingOccurrences: value.missingOccurrences.map(workspaceOccurrenceFromWails),
		unrecordedOccurrences: value.unrecordedOccurrences.map(workspaceOccurrenceFromWails),
		unresolvedArtifacts: value.unresolvedArtifacts.map(workspaceArtifactFromWails),
		diagnostics: diagnosticsFromWails(value.diagnostics),
	} as WorkspaceCatalogView;
}

function workspaceRefreshFromWails(value: wailsWorkspace.WorkspaceRefreshResult): WorkspaceRefreshResult {
	return {
		...value,
		diagnostics: diagnosticsFromWails(value.diagnostics),
	} as WorkspaceRefreshResult;
}

function workspaceSuppressionFromWails(suppression: wailsWorkspace.WorkspaceSuppressionView): WorkspaceSuppressionView {
	return {
		...suppression,
		createdAt: toFrontendDate(suppression.createdAt, 'workspaceSuppression.createdAt'),
		modifiedAt: toFrontendDate(suppression.modifiedAt, 'workspaceSuppression.modifiedAt'),
	} as WorkspaceSuppressionView;
}

function workspaceContextContributionFromWails(
	value: wailsWorkspace.WorkspaceContextContribution
): WorkspaceContextLoadPlan['contributions'][number] {
	return {
		...value,
		role: enumFromWails(value.role, WorkspaceContextRole, 'workspace.context.role'),
		mediaType: enumFromWails(value.mediaType, WorkspaceContextMediaType, 'workspace.context.mediaType'),
	} as WorkspaceContextLoadPlan['contributions'][number];
}

function workspaceContextDecisionFromWails(
	value: wailsWorkspace.WorkspaceContextDecision
): WorkspaceContextLoadPlan['decisions'][number] {
	return {
		...value,
		status: enumFromWails(value.status, WorkspaceContextCompositionStatus, 'workspace.context.status'),
	} as WorkspaceContextLoadPlan['decisions'][number];
}

function workspaceContextFromWails(value: wailsWorkspace.WorkspaceContextView): WorkspaceContextView {
	return {
		...value,
		role: enumFromWails(value.role, WorkspaceContextRole, 'workspace.context.role'),
		mediaType: enumFromWails(value.mediaType, WorkspaceContextMediaType, 'workspace.context.mediaType'),
		state: enumFromWails(value.state, ArtifactState, 'workspace.context.state'),
		diagnostics: diagnosticsFromWails(value.diagnostics),
	} as WorkspaceContextView;
}

function workspaceContextInspectionFromWails(
	value: wailsWorkspace.WorkspaceContextInspectionView
): WorkspaceContextInspectionView {
	return {
		...value,
		contributions: value.contributions.map(workspaceContextContributionFromWails),
		diagnostics: diagnosticsFromWails(value.diagnostics),
	} as WorkspaceContextInspectionView;
}

function workspaceContextLoadPlanFromWails(value: wailsWorkspace.WorkspaceContextLoadPlan): WorkspaceContextLoadPlan {
	return {
		...value,
		contributions: value.contributions.map(workspaceContextContributionFromWails),
		decisions: value.decisions.map(workspaceContextDecisionFromWails),
		diagnostics: diagnosticsFromWails(value.diagnostics),
	} as WorkspaceContextLoadPlan;
}

function workspaceSkillFromWails(skill: wailsWorkspace.WorkspaceSkillView): WorkspaceSkillView {
	return {
		...skill,
		state: enumFromWails(skill.state, ArtifactState, 'workspace.skill.state'),
		diagnostics: diagnosticsFromWails(skill.diagnostics),
		skill: {
			...skill.skill,
			insert: enumFromWails(skill.skill.insert, WorkspaceSkillInsert, 'workspace.skill.insert'),
			createdAt: toFrontendDate(skill.skill.createdAt, 'workspaceSkill.skill.createdAt'),
			modifiedAt: toFrontendDate(skill.skill.modifiedAt, 'workspaceSkill.skill.modifiedAt'),
		},
	} as WorkspaceSkillView;
}

function workspaceSkillLoadFromWails(load: wailsWorkspace.WorkspaceSkillLoadView): WorkspaceSkillLoadView {
	return {
		...load,
		skills: (load.skills ?? []).map(s => {
			return workspaceSkillFromWails(s);
		}),
		diagnostics: diagnosticsFromWails(load.diagnostics),
	} as WorkspaceSkillLoadView;
}

export class WailsWorkspaceAPI implements IWorkspaceAPI {
	async createFilesystemWorkspace(rootID: ArtifactRootID, body: CreateFilesystemWorkspaceBody): Promise<WorkspaceView> {
		const response = await CreateFilesystemWorkspace({
			rootID,
			Body: body,
		} as Parameters<typeof CreateFilesystemWorkspace>[0]);

		return workspaceViewFromWails(
			requireWailsBody(response.Body, 'CreateFilesystemWorkspace') as wailsWorkspace.WorkspaceView
		);
	}

	async createEmptyWorkspace(rootID: ArtifactRootID, body: CreateEmptyWorkspaceBody): Promise<WorkspaceView> {
		const response = await CreateEmptyWorkspace({
			rootID,
			Body: body,
		} as Parameters<typeof CreateEmptyWorkspace>[0]);

		return workspaceViewFromWails(
			requireWailsBody(response.Body, 'CreateEmptyWorkspace') as wailsWorkspace.WorkspaceView
		);
	}

	async getWorkspace(workspace: WorkspaceRef): Promise<WorkspaceView> {
		const response = await GetWorkspace({
			workspace,
		} as Parameters<typeof GetWorkspace>[0]);

		return workspaceViewFromWails(requireWailsBody(response.Body, 'GetWorkspace'));
	}

	async listWorkspaces(rootID: ArtifactRootID): Promise<WorkspaceView[]> {
		const response = await ListWorkspaces({
			rootID,
		} as Parameters<typeof ListWorkspaces>[0]);

		const body = requireWailsBody(response.Body, 'ListWorkspaces');
		return (body.workspaces ?? []).map(w => {
			return workspaceViewFromWails(w);
		});
	}

	async updateWorkspace(workspace: WorkspaceRef, body: UpdateWorkspaceBody): Promise<WorkspaceView> {
		const response = await UpdateWorkspace({
			workspace,
			Body: body,
		} as Parameters<typeof UpdateWorkspace>[0]);

		return workspaceViewFromWails(requireWailsBody(response.Body, 'UpdateWorkspace'));
	}

	async replaceWorkspacePrimarySource(
		workspace: WorkspaceRef,
		body: ReplaceWorkspacePrimarySourceBody
	): Promise<WorkspaceView> {
		const response = await ReplaceWorkspacePrimarySource({
			workspace,
			Body: body,
		} as Parameters<typeof ReplaceWorkspacePrimarySource>[0]);

		return workspaceViewFromWails(requireWailsBody(response.Body, 'ReplaceWorkspacePrimarySource'));
	}

	async setWorkspacePrimarySource(
		workspace: WorkspaceRef,
		body: SetWorkspacePrimarySourceBody
	): Promise<WorkspaceView> {
		const response = await SetWorkspacePrimarySource({
			workspace,
			Body: body,
		} as Parameters<typeof SetWorkspacePrimarySource>[0]);

		return workspaceViewFromWails(requireWailsBody(response.Body, 'SetWorkspacePrimarySource'));
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

		return workspaceViewFromWails(
			requireWailsBody(response.Body, 'AttachWorkspaceSource') as wailsWorkspace.WorkspaceView
		);
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

		return workspaceViewFromWails(
			requireWailsBody(response.Body, 'UpdateWorkspaceAttachment') as wailsWorkspace.WorkspaceView
		);
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

		return workspaceViewFromWails(requireWailsBody(response.Body, 'DetachWorkspaceSource'));
	}

	async refreshWorkspace(workspace: WorkspaceRef): Promise<WorkspaceRefreshResult> {
		const response = await RefreshWorkspace({
			workspace,
		} as Parameters<typeof RefreshWorkspace>[0]);

		return workspaceRefreshFromWails(requireWailsBody(response.Body, 'RefreshWorkspace'));
	}

	async getWorkspaceCatalog(workspace: WorkspaceRef): Promise<WorkspaceCatalogView> {
		const response = await GetWorkspaceCatalog({
			workspace,
		} as Parameters<typeof GetWorkspaceCatalog>[0]);

		return workspaceCatalogFromWails(requireWailsBody(response.Body, 'GetWorkspaceCatalog'));
	}

	async composeWorkspaceLoadPlan(workspace: WorkspaceRef, artifacts: ArtifactRef[]): Promise<WorkspaceLoadPlanView> {
		const response = await ComposeWorkspaceLoadPlan({
			workspace,
			Body: { artifacts },
		} as Parameters<typeof ComposeWorkspaceLoadPlan>[0]);

		const body = requireWailsBody(response.Body, 'ComposeWorkspaceLoadPlan');
		return workspaceLoadPlanFromWails(body);
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

		const body = requireWailsBody(response.Body, 'ResolveWorkspaceResource');
		return resolvedWorkspaceResourceFromWails(body);
	}

	async getWorkspaceArtifact(workspace: WorkspaceRef, artifact: ArtifactRef): Promise<WorkspaceArtifactView> {
		const response = await GetWorkspaceArtifact({
			workspace,
			artifact,
		} as Parameters<typeof GetWorkspaceArtifact>[0]);

		return workspaceArtifactFromWails(requireWailsBody(response.Body, 'GetWorkspaceArtifact'));
	}

	async listWorkspaceArtifacts(workspace: WorkspaceRef): Promise<WorkspaceArtifactView[]> {
		const response = await ListWorkspaceArtifacts({
			workspace,
		} as Parameters<typeof ListWorkspaceArtifacts>[0]);

		const body = requireWailsBody(response.Body, 'ListWorkspaceArtifacts');
		return (body.artifacts ?? []).map(w => {
			return workspaceArtifactFromWails(w);
		});
	}

	async adoptWorkspaceOccurrence(
		workspace: WorkspaceRef,
		body: AdoptWorkspaceOccurrenceBody
	): Promise<WorkspaceArtifactView> {
		const response = await AdoptWorkspaceOccurrence({
			workspace,
			Body: body,
		} as Parameters<typeof AdoptWorkspaceOccurrence>[0]);

		return workspaceArtifactFromWails(requireWailsBody(response.Body, 'AdoptWorkspaceOccurrence'));
	}

	async pinWorkspaceArtifact(workspace: WorkspaceRef, body: PinWorkspaceArtifactBody): Promise<WorkspaceArtifactView> {
		const response = await PinWorkspaceArtifact({
			workspace,
			Body: body,
		} as Parameters<typeof PinWorkspaceArtifact>[0]);

		return workspaceArtifactFromWails(requireWailsBody(response.Body, 'PinWorkspaceArtifact'));
	}

	async listWorkspaceSuppressions(workspace: WorkspaceRef): Promise<WorkspaceSuppressionView[]> {
		const response = await ListWorkspaceSuppressions({
			workspace,
		} as Parameters<typeof ListWorkspaceSuppressions>[0]);

		const body = requireWailsBody(response.Body, 'ListWorkspaceSuppressions');
		return (body.suppressions ?? []).map(b => {
			return workspaceSuppressionFromWails(b);
		});
	}

	async suppressWorkspaceBinding(
		workspace: WorkspaceRef,
		body: SuppressWorkspaceBindingBody
	): Promise<WorkspaceSuppressionView> {
		const response = await SuppressWorkspaceBinding({
			workspace,
			Body: body,
		} as Parameters<typeof SuppressWorkspaceBinding>[0]);

		return workspaceSuppressionFromWails(requireWailsBody(response.Body, 'SuppressWorkspaceBinding'));
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
		return (body.contexts ?? []).map(w => {
			return workspaceContextFromWails(w);
		});
	}

	async loadWorkspaceContexts(
		workspace: WorkspaceRef,
		artifacts?: ArtifactRef[]
	): Promise<WorkspaceContextInspectionView> {
		const response = await LoadWorkspaceContexts({
			workspace,
			Body: { artifacts },
		} as Parameters<typeof LoadWorkspaceContexts>[0]);

		return workspaceContextInspectionFromWails(requireWailsBody(response.Body, 'LoadWorkspaceContexts'));
	}

	async composeWorkspaceContext(workspace: WorkspaceRef, artifacts?: ArtifactRef[]): Promise<WorkspaceContextLoadPlan> {
		const response = await ComposeWorkspaceContext({
			workspace,
			Body: { artifacts },
		} as Parameters<typeof ComposeWorkspaceContext>[0]);

		return workspaceContextLoadPlanFromWails(requireWailsBody(response.Body, 'ComposeWorkspaceContext'));
	}

	async listWorkspaceSkills(workspace: WorkspaceRef): Promise<WorkspaceSkillView[]> {
		const response = await ListWorkspaceSkills({
			workspace,
		} as Parameters<typeof ListWorkspaceSkills>[0]);

		const body = requireWailsBody(response.Body, 'ListWorkspaceSkills');
		return (body.skills ?? []).map(s => {
			return workspaceSkillFromWails(s);
		});
	}

	async loadWorkspaceSkills(workspace: WorkspaceRef, artifacts: ArtifactRef[]): Promise<WorkspaceSkillLoadView> {
		const response = await LoadWorkspaceSkills({
			workspace,
			Body: { artifacts },
		} as Parameters<typeof LoadWorkspaceSkills>[0]);

		const body = requireWailsBody(response.Body, 'LoadWorkspaceSkills');
		return workspaceSkillLoadFromWails(body);
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

		return workspaceArtifactFromWails(requireWailsBody(response.Body, 'SetWorkspaceArtifactEnabled'));
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

		return workspaceArtifactFromWails(requireWailsBody(response.Body, 'SetWorkspaceArtifactRuntimeDisabled'));
	}
}
