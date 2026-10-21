import type { ArtifactRef } from '@/spec/artifact';
import type {
	ManagedMCPCreateRequest,
	ManagedMCPPolicyUpsertRequest,
	ManagedMCPReplaceRequest,
	MCPPolicy,
	MCPPolicyView,
	MCPServerDocument,
	MCPServerSecretsView,
} from '@/spec/mcp';
import {
	MCPApprovalRule,
	MCPExecutionMode,
	MCPHTTPAuthMode,
	MCPInputKind,
	MCPServerType,
	MCPTrustLevel,
} from '@/spec/mcp';

import type { ArtifactRef as WailsArtifactRef } from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/internal/artifactory-go/store/artifact/model/models';
import type { ServerDocument as WailsServerDocument } from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/domain/server/models';
import type {
	ManagedMCPCreateRequest as WailsManagedMCPCreateRequest,
	ManagedMCPPolicyUpsertRequest as WailsManagedMCPPolicyUpsertRequest,
	ManagedMCPReplaceRequest as WailsManagedMCPReplaceRequest,
} from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/internal/llmartifactory-go/mcp/models';
import type { MCPPolicy as WailsMCPPolicy } from '@/apis/wailsbinding/github.com/flexigpt/flexigpt-app/internal/mcppolicy/models';
import { enumFromWails, enumToWails, requiredObject, wailsObjectArrayOrEmpty } from '@/apis/wailsapi/transport';

type MCPToolPolicies = NonNullable<MCPPolicy['toolPolicies']>;
type MCPInstallInputs = NonNullable<MCPServerDocument['configuration']['install']['inputs']>;
type WailsInstallInputs = NonNullable<WailsServerDocument['configuration']['install']['inputs']>;
type WailsInputDeclaration = NonNullable<WailsInstallInputs[string]>;
type WailsServerType = NonNullable<WailsServerDocument['mcpServer']['type']>;
type WailsToolPolicies = NonNullable<WailsMCPPolicy['toolPolicies']>;
type WailsToolPolicy = NonNullable<WailsToolPolicies[string]>;

function artifactRefToWails(ref: ArtifactRef): WailsArtifactRef {
	return {
		rootID: ref.rootID,
		artifactID: ref.artifactID,
	};
}

function installInputsToWails(inputs: MCPInstallInputs | undefined, operation: string): WailsInstallInputs | undefined {
	if (inputs === undefined) {
		return undefined;
	}

	const output: WailsInstallInputs = {};

	for (const [name, declaration] of Object.entries(inputs)) {
		const { kind, ...rest } = declaration;

		output[name] = {
			...rest,
			kind: enumToWails<MCPInputKind, WailsInputDeclaration['kind']>(
				kind,
				MCPInputKind,
				`${operation}.inputs.${name}.kind`
			),
		};
	}

	return output;
}

function mcpServerDocumentToWailsBinding(document: MCPServerDocument, operation: string): WailsServerDocument {
	const checked = mcpServerDocumentToWails(document, operation);
	const { mcpServer, configuration, ...documentRest } = checked;
	const { type, ...serverRest } = mcpServer;
	const { auth, install, ...configurationRest } = configuration;
	const { mode, ...authRest } = auth;
	const { inputs, ...installRest } = install;

	return {
		...documentRest,
		mcpServer: {
			...serverRest,
			type: enumToWails<MCPServerType, WailsServerType>(type, MCPServerType, `${operation}.mcpServer.type`),
		},
		configuration: {
			...configurationRest,
			auth: {
				...authRest,
				mode: enumToWails<MCPHTTPAuthMode, WailsServerDocument['configuration']['auth']['mode']>(
					mode,
					MCPHTTPAuthMode,
					`${operation}.configuration.auth.mode`
				),
			},
			install: {
				...installRest,
				inputs: installInputsToWails(inputs, `${operation}.configuration.install`),
			},
		},
	};
}

function mcpPolicyToWailsBinding(policy: MCPPolicy, operation: string): WailsMCPPolicy {
	const checked = mcpPolicyToWails(policy, operation);
	const { trustLevel, defaultPolicy, toolPolicies, ...policyRest } = checked;
	const { defaultApprovalRule, defaultExecutionMode, ...defaultPolicyRest } = defaultPolicy;
	let wailsToolPolicies: WailsToolPolicies | undefined;

	if (toolPolicies !== undefined) {
		wailsToolPolicies = {};

		for (const [name, override] of Object.entries(toolPolicies)) {
			const { approvalRule, executionMode, ...overrideRest } = override;

			const wailsOverride: WailsToolPolicy = {
				...overrideRest,
			};

			if (approvalRule !== undefined) {
				wailsOverride.approvalRule = enumToWails<MCPApprovalRule, NonNullable<WailsToolPolicy['approvalRule']>>(
					approvalRule,
					MCPApprovalRule,
					`${operation}.toolPolicies.${name}.approvalRule`
				);
			}

			if (executionMode !== undefined) {
				wailsOverride.executionMode = enumToWails<MCPExecutionMode, NonNullable<WailsToolPolicy['executionMode']>>(
					executionMode,
					MCPExecutionMode,
					`${operation}.toolPolicies.${name}.executionMode`
				);
			}

			wailsToolPolicies[name] = wailsOverride;
		}
	}

	return {
		...policyRest,
		trustLevel: enumToWails<MCPTrustLevel, WailsMCPPolicy['trustLevel']>(
			trustLevel,
			MCPTrustLevel,
			`${operation}.trustLevel`
		),
		defaultPolicy: {
			...defaultPolicyRest,
			defaultApprovalRule: enumToWails<MCPApprovalRule, WailsMCPPolicy['defaultPolicy']['defaultApprovalRule']>(
				defaultApprovalRule,
				MCPApprovalRule,
				`${operation}.defaultPolicy.defaultApprovalRule`
			),
			defaultExecutionMode: enumToWails<MCPExecutionMode, WailsMCPPolicy['defaultPolicy']['defaultExecutionMode']>(
				defaultExecutionMode,
				MCPExecutionMode,
				`${operation}.defaultPolicy.defaultExecutionMode`
			),
		},
		...(wailsToolPolicies === undefined ? {} : { toolPolicies: wailsToolPolicies }),
	};
}

export function managedMCPCreateRequestToWails(request: ManagedMCPCreateRequest): WailsManagedMCPCreateRequest {
	return {
		plugin: artifactRefToWails(request.plugin),
		expectedPluginRevision: request.expectedPluginRevision,
		document: mcpServerDocumentToWailsBinding(request.document, 'CreateMCPServer.request.document'),
		enabled: request.enabled,
	};
}

export function managedMCPReplaceRequestToWails(request: ManagedMCPReplaceRequest): WailsManagedMCPReplaceRequest {
	return {
		plugin: artifactRefToWails(request.plugin),
		expectedPluginRevision: request.expectedPluginRevision,
		artifact: artifactRefToWails(request.artifact),
		expectedArtifactRevision: request.expectedArtifactRevision,
		document: mcpServerDocumentToWailsBinding(request.document, 'UpdateMCPServer.request.document'),
		enabled: request.enabled,
	};
}

export function managedMCPPolicyUpsertRequestToWails(
	request: ManagedMCPPolicyUpsertRequest
): WailsManagedMCPPolicyUpsertRequest {
	return {
		plugin: artifactRefToWails(request.plugin),
		expectedPluginRevision: request.expectedPluginRevision,
		name: request.name,
		description: request.description,
		policy: mcpPolicyToWailsBinding(request.policy, 'SaveMCPPolicy.request.policy'),
		enabled: request.enabled,
	};
}

function checkedEnum<T extends string>(
	value: unknown,
	enumValues: Readonly<Record<string, T>>,
	field: string,
	outbound: boolean
): T {
	if (outbound) {
		return enumToWails<T, T>(value, enumValues, field);
	}

	return enumFromWails(value, enumValues, field);
}

function mapInstallInputs(
	inputs: MCPInstallInputs | undefined,
	operation: string,
	outbound: boolean
): MCPInstallInputs | undefined {
	if (inputs === null || inputs === undefined) {
		return undefined;
	}

	const output: MCPInstallInputs = {};

	for (const [name, declaration] of Object.entries(inputs)) {
		output[name] = {
			...declaration,
			kind: checkedEnum(declaration.kind, MCPInputKind, `${operation}.inputs.${name}.kind`, outbound),
		};
	}

	return output;
}

function mapMCPServerDocument(value: unknown, operation: string, outbound: boolean): MCPServerDocument {
	const document = requiredObject<MCPServerDocument>(value, operation);
	const server = requiredObject<MCPServerDocument['mcpServer']>(document.mcpServer, `${operation}.mcpServer`);
	const configuration = requiredObject<MCPServerDocument['configuration']>(
		document.configuration,
		`${operation}.configuration`
	);
	const auth = requiredObject<MCPServerDocument['configuration']['auth']>(
		configuration.auth,
		`${operation}.configuration.auth`
	);
	const install = requiredObject<MCPServerDocument['configuration']['install']>(
		configuration.install,
		`${operation}.configuration.install`
	);

	return {
		...document,
		mcpServer: {
			...server,
			type: checkedEnum(server.type, MCPServerType, `${operation}.mcpServer.type`, outbound),
		},
		configuration: {
			...configuration,
			auth: {
				...auth,
				mode: checkedEnum(auth.mode, MCPHTTPAuthMode, `${operation}.configuration.auth.mode`, outbound),
			},
			install: {
				...install,
				inputs: mapInstallInputs(install.inputs, `${operation}.configuration.install`, outbound),
			},
		},
	};
}

function mapMCPPolicy(value: unknown, operation: string, outbound: boolean): MCPPolicy {
	const policy = requiredObject<MCPPolicy>(value, operation);
	const defaultPolicy = requiredObject<MCPPolicy['defaultPolicy']>(policy.defaultPolicy, `${operation}.defaultPolicy`);
	let toolPolicies: MCPToolPolicies | undefined;

	if (policy.toolPolicies !== null && policy.toolPolicies !== undefined) {
		toolPolicies = {};

		for (const [name, override] of Object.entries(policy.toolPolicies)) {
			toolPolicies[name] = {
				...override,
				approvalRule:
					override.approvalRule === null || override.approvalRule === undefined
						? undefined
						: checkedEnum(
								override.approvalRule,
								MCPApprovalRule,
								`${operation}.toolPolicies.${name}.approvalRule`,
								outbound
							),
				executionMode:
					override.executionMode === null || override.executionMode === undefined
						? undefined
						: checkedEnum(
								override.executionMode,
								MCPExecutionMode,
								`${operation}.toolPolicies.${name}.executionMode`,
								outbound
							),
			};
		}
	}

	return {
		...policy,
		trustLevel: checkedEnum(policy.trustLevel, MCPTrustLevel, `${operation}.trustLevel`, outbound),
		defaultPolicy: {
			...defaultPolicy,
			defaultApprovalRule: checkedEnum(
				defaultPolicy.defaultApprovalRule,
				MCPApprovalRule,
				`${operation}.defaultPolicy.defaultApprovalRule`,
				outbound
			),
			defaultExecutionMode: checkedEnum(
				defaultPolicy.defaultExecutionMode,
				MCPExecutionMode,
				`${operation}.defaultPolicy.defaultExecutionMode`,
				outbound
			),
		},
		toolPolicies,
	};
}

function mcpServerDocumentToWails(document: MCPServerDocument, operation: string): MCPServerDocument {
	return mapMCPServerDocument(document, operation, true);
}

function mcpPolicyToWails(policy: MCPPolicy, operation: string): MCPPolicy {
	return mapMCPPolicy(policy, operation, true);
}

export function mcpServerDocumentFromWails(value: unknown, operation: string): MCPServerDocument {
	return mapMCPServerDocument(value, operation, false);
}

export function mcpPolicyFromWails(value: unknown, operation: string): MCPPolicy {
	return mapMCPPolicy(value, operation, false);
}

export function mcpPolicyViewFromWails(value: unknown, operation: string): MCPPolicyView {
	const view = requiredObject<MCPPolicyView>(value, operation);

	return {
		...view,
		body: mcpPolicyFromWails(view.body, `${operation}.body`),
	};
}

export function mcpServerSecretsViewFromWails(value: unknown, operation: string): MCPServerSecretsView {
	const view = requiredObject<MCPServerSecretsView>(value, operation);

	return {
		...view,
		inputs: wailsObjectArrayOrEmpty(view.inputs, `${operation}.inputs`).map((input, index) => {
			const current = requiredObject<MCPServerSecretsView['inputs'][number]>(input, `${operation}.inputs[${index}]`);

			return {
				...current,
				kind: enumFromWails(current.kind, MCPInputKind, `${operation}.inputs[${index}].kind`),
			};
		}),
	};
}
