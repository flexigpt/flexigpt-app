import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FiPlus, FiRefreshCw } from 'react-icons/fi';

import type { ArtifactRef } from '@/spec/artifact';
import { ArtifactState } from '@/spec/artifact';
import { ModelLookupScope } from '@/spec/model';

import { getErrorMessage } from '@/lib/error_utils';

import { usePendingActions } from '@/hooks/use_pending_actions';

import type {
	ModelManagementItem,
	ModelManagementSnapshot,
	ModelProviderManagementItem,
} from '@/apis/model_management';
import { modelManagementAPI } from '@/apis/baseapi';

import { ActionDeniedAlertModal } from '@/components/action_denied_modal';
import { DeleteConfirmationModal } from '@/components/delete_confirmation_modal';
import { Loader } from '@/components/loader';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { PageFrame } from '@/components/page_frame';

import type { ModelEditorSubmission } from '@/models/model_add_edit_modal';
import type { ProviderEditorSubmission } from '@/models/model_provider_add_edit_modal';
import { DefaultProviderControl } from '@/models/default_provider_control';
import { ModelModalMode, ModelProviderModalMode, ProviderCredentialAction } from '@/models/lib/management_types';
import { ModelAddEditModal } from '@/models/model_add_edit_modal';
import { ModelProviderAddEditModal } from '@/models/model_provider_add_edit_modal';
import { ModelProviderCard } from '@/models/model_provider_card';
import { ProviderCredentialModal } from '@/models/provider_credential_modal';

interface ProviderDialogState {
	mode: ModelProviderModalMode;
	provider?: ModelProviderManagementItem;
}

interface ModelDialogState {
	mode: ModelModalMode;
	model?: ModelManagementItem;
	provider?: ModelProviderManagementItem;
}

interface CredentialDialogState {
	provider: ModelProviderManagementItem;
}

type DeleteTarget =
	{ kind: 'provider'; provider: ModelProviderManagementItem } | { kind: 'model'; model: ModelManagementItem };

function providerKey(provider: ModelProviderManagementItem): string {
	return `${provider.list.ref.rootID}\u0000${provider.list.ref.artifactID}`;
}

function artifactRefsEqual(left: ArtifactRef | undefined, right: ArtifactRef | undefined): boolean {
	return left?.rootID === right?.rootID && left?.artifactID === right?.artifactID;
}

function isProviderDefault(provider: ModelProviderManagementItem, model: ModelManagementItem): boolean {
	const reference = provider.view.document.defaultModel;
	if (!reference || reference.name !== model.view.document.name) {
		return false;
	}

	if (reference.scope === ModelLookupScope.Builtin) {
		return model.list.builtIn;
	}

	return !model.list.builtIn && model.list.ref.rootID === provider.list.ref.rootID;
}

// oxlint-disable-next-line no-restricted-exports
export default function ModelsPage() {
	const [snapshot, setSnapshot] = useState<ModelManagementSnapshot | null>(null);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState('');

	const reloadSequenceRef = useRef(0);
	const [providerDialog, setProviderDialog] = useState<ProviderDialogState | null>(null);
	const [modelDialog, setModelDialog] = useState<ModelDialogState | null>(null);
	const [credentialDialog, setCredentialDialog] = useState<CredentialDialogState | null>(null);
	const [deleteTarget, setDeleteTarget] = useState<DeleteTarget | null>(null);
	const [deniedMessage, setDeniedMessage] = useState('');
	const [showDenied, setShowDenied] = useState(false);

	const { isPending: isDefaultProviderPending, runAction: runDefaultProviderAction } = usePendingActions();
	const showDeniedMessage = useCallback((message: string) => {
		setDeniedMessage(message);
		setShowDenied(true);
	}, []);

	const reload = useCallback(async () => {
		const requestSequence = reloadSequenceRef.current + 1;
		reloadSequenceRef.current = requestSequence;
		setLoading(true);
		setError('');

		try {
			const nextSnapshot = await modelManagementAPI.loadSnapshot();
			if (reloadSequenceRef.current === requestSequence) {
				setSnapshot(nextSnapshot);
			}
		} catch (loadError) {
			if (reloadSequenceRef.current === requestSequence) {
				setError(loadError instanceof Error ? loadError.message : 'Models could not be loaded.');
			}
			throw loadError;
		} finally {
			if (reloadSequenceRef.current === requestSequence) {
				setLoading(false);
			}
		}
	}, []);

	useEffect(() => {
		// oxlint-disable-next-line react/set-state-in-effect
		void reload().catch(() => undefined);
	}, [reload]);

	const modelsByProvider = useMemo(() => {
		const result = new Map<string, ModelManagementItem[]>();

		for (const model of snapshot?.models ?? []) {
			if (!model.provider) {
				continue;
			}

			const key = providerKey(model.provider);
			const current = result.get(key) ?? [];
			current.push(model);
			result.set(key, current);
		}

		for (const models of result.values()) {
			models.sort((left, right) =>
				left.list.displayName.localeCompare(right.list.displayName, undefined, {
					numeric: true,
					sensitivity: 'base',
				})
			);
		}

		return result;
	}, [snapshot]);

	const run = useCallback(
		async (operation: () => Promise<void>) => {
			setError('');

			try {
				await operation();
			} catch (operationError) {
				const message = operationError instanceof Error ? operationError.message : 'Model operation failed.';
				setError(message);
				throw operationError instanceof Error ? operationError : new Error(message);
			}

			try {
				await reload();
			} catch {
				setError('The change was saved, but the page could not be refreshed. Reload before making another change.');
			}
		},
		[reload]
	);

	const submitProvider = async (submission: ProviderEditorSubmission) => {
		const provider = providerDialog?.provider;
		let providerWasWritten = false;

		try {
			await run(async () => {
				let credentialProviderRef = provider?.list.ref;
				let credentialOverlayRevision = provider?.list.runtimeOverlayRevision ?? 0;

				if (provider) {
					await modelManagementAPI.replaceProvider({
						provider: provider.list.ref,
						expectedArtifactRevision: provider.list.revision,
						document: submission.document,
						enabled: submission.enabled,
					});
				} else {
					const created = await modelManagementAPI.createProvider({
						rootID: '',
						document: submission.document,
						enabled: submission.enabled,
					});

					credentialProviderRef = {
						rootID: created.artifact.rootID,
						artifactID: created.artifact.id,
					};
					credentialOverlayRevision = 0;
				}
				providerWasWritten = true;

				if (submission.credentialAction !== ProviderCredentialAction.Unchanged && credentialProviderRef) {
					await modelManagementAPI.setProviderCredential(
						credentialProviderRef,
						credentialOverlayRevision,
						submission.credentialAction === ProviderCredentialAction.Clear ? '' : submission.credential
					);
				}
			});
		} catch (submissionError) {
			if (!providerWasWritten) {
				throw submissionError;
			}

			await reload().catch(() => undefined);
			const message =
				'Provider settings were saved, but its credential could not be updated. Use Manage Credential to complete setup.';
			setError(message);
			showDeniedMessage(message);
		}
	};

	const submitModel = async (submission: ModelEditorSubmission) => {
		const model = modelDialog?.model;
		const provider = modelDialog?.provider;
		let modelWasCreated = false;

		if (!provider) {
			throw new Error('A provider must be selected.');
		}

		try {
			await run(async () => {
				if (model) {
					await modelManagementAPI.replaceModel({
						model: model.list.ref,
						expectedArtifactRevision: model.list.revision,
						document: submission.document,
						enabled: submission.enabled,
					});
					return;
				}

				await modelManagementAPI.createModel({
					rootID: provider.list.builtIn ? '' : provider.list.ref.rootID,
					document: submission.document,
					enabled: submission.enabled,
				});
				modelWasCreated = true;

				if (submission.enabled && !provider.list.builtIn && !provider.view.document.defaultModel) {
					await modelManagementAPI.replaceProvider({
						provider: provider.list.ref,
						expectedArtifactRevision: provider.list.revision,
						document: {
							...provider.view.document,
							defaultModel: {
								name: submission.document.name,
							},
						},
						enabled: provider.list.enabled,
					});
				}
			});
		} catch (submissionError) {
			if (!modelWasCreated) {
				throw submissionError;
			}

			await reload().catch(() => undefined);
			const message =
				'The model was saved, but selecting it as the provider default failed. Choose a default model manually.';
			setError(message);
			showDeniedMessage(message);
		}
	};

	const saveCredential = async (credential: string) => {
		const provider = credentialDialog?.provider;
		if (!provider) {
			return;
		}

		await run(async () => {
			await modelManagementAPI.setProviderCredential(
				provider.list.ref,
				provider.list.runtimeOverlayRevision ?? 0,
				credential
			);
		});
	};

	const confirmDelete = async () => {
		if (!deleteTarget) {
			return;
		}

		try {
			if (deleteTarget.kind === 'provider') {
				if (artifactRefsEqual(snapshot?.defaultProvider, deleteTarget.provider.list.ref)) {
					showDeniedMessage('Choose another default provider before deleting this provider.');
					return;
				}

				const linkedModels = modelsByProvider.get(providerKey(deleteTarget.provider)) ?? [];
				if (linkedModels.length > 0) {
					showDeniedMessage('Remove all linked models before deleting this provider.');
					return;
				}

				await run(() =>
					modelManagementAPI.deleteProvider(deleteTarget.provider.list.ref, deleteTarget.provider.list.revision)
				);
			} else {
				const provider = deleteTarget.model.provider;
				if (provider && isProviderDefault(provider, deleteTarget.model)) {
					showDeniedMessage('Choose another default model before deleting the current default model.');
					return;
				}

				await run(() => modelManagementAPI.deleteModel(deleteTarget.model.list.ref, deleteTarget.model.list.revision));
			}

			setDeleteTarget(null);
		} catch {
			// `run` already exposes the actionable error and leaves the confirmation open.
		}
	};

	if (loading && !snapshot) {
		return <Loader text="Loading models..." />;
	}

	return (
		<PageFrame>
			<div className="flex size-full flex-col overflow-auto">
				<ManagementPageHeader
					title="Models"
					description="Manage Artifact-backed model providers, credentials, defaults, and models."
					actions={
						<>
							<button
								type="button"
								className="btn btn-sm btn-ghost rounded-xl"
								disabled={loading}
								onClick={() => void reload().catch(() => undefined)}
							>
								<FiRefreshCw size={16} />
								Refresh
							</button>

							<button
								type="button"
								className="btn btn-sm btn-primary rounded-xl"
								disabled={loading}
								onClick={() => {
									setProviderDialog({
										mode: ModelProviderModalMode.Add,
									});
								}}
							>
								<FiPlus size={16} />
								Add Provider
							</button>
						</>
					}
				/>

				<ManagementPageContent>
					{error ? <div className="alert alert-error mb-4 rounded-xl">{error}</div> : null}

					{snapshot ? (
						<DefaultProviderControl
							providers={snapshot.providers}
							defaultProvider={snapshot.defaultProvider}
							pending={isDefaultProviderPending('set-default-provider')}
							onChange={provider => {
								void runDefaultProviderAction('set-default-provider', () =>
									run(() => modelManagementAPI.setDefaultModelProvider(provider.list.ref))
								).catch((actionError: unknown) => {
									showDeniedMessage(getErrorMessage(actionError, 'Failed changing default provider.'));
								});
							}}
						/>
					) : null}

					<div className="space-y-4">
						{snapshot?.providers.map(provider => (
							<ModelProviderCard
								key={providerKey(provider)}
								provider={provider}
								models={modelsByProvider.get(providerKey(provider)) ?? []}
								isGlobalDefault={artifactRefsEqual(snapshot.defaultProvider, provider.list.ref)}
								onViewProvider={p => {
									setProviderDialog({
										mode: ModelProviderModalMode.View,
										provider: p,
									});
								}}
								onEditProvider={p => {
									setProviderDialog({
										mode: ModelProviderModalMode.Edit,
										provider: p,
									});
								}}
								onToggleProvider={async p => {
									if (artifactRefsEqual(snapshot?.defaultProvider, p.list.ref) && p.list.enabled) {
										throw new Error('Choose another default provider before disabling this provider.');
									}

									await run(() => modelManagementAPI.setProviderEnabled(p.list.ref, p.list.revision, !p.list.enabled));
								}}
								onDeleteProvider={p => {
									if (artifactRefsEqual(snapshot?.defaultProvider, p.list.ref)) {
										showDeniedMessage('Choose another default provider before deleting this provider.');
										return;
									}

									const linkedModels = modelsByProvider.get(providerKey(p)) ?? [];
									if (linkedModels.length > 0) {
										showDeniedMessage('Remove all linked models before deleting this provider.');
										return;
									}

									setDeleteTarget({ kind: 'provider', provider: p });
								}}
								onCredential={p => {
									setCredentialDialog({
										provider: p,
									});
								}}
								onAddModel={p => {
									setModelDialog({
										mode: ModelModalMode.Add,
										provider: p,
									});
								}}
								onViewModel={model => {
									setModelDialog({
										mode: ModelModalMode.View,
										model,
										provider: model.provider,
									});
								}}
								onEditModel={model => {
									setModelDialog({
										mode: ModelModalMode.Edit,
										model,
										provider: model.provider,
									});
								}}
								onToggleModel={async model => {
									if (model.provider && model.list.enabled && isProviderDefault(model.provider, model)) {
										throw new Error('Choose another default model before disabling this model.');
									}

									await run(() =>
										modelManagementAPI.setModelEnabled(model.list.ref, model.list.revision, !model.list.enabled)
									);
								}}
								onDeleteModel={model => {
									if (model.provider && isProviderDefault(model.provider, model)) {
										showDeniedMessage('Choose another default model before deleting the current default model.');
										return;
									}

									setDeleteTarget({ kind: 'model', model });
								}}
								onSetDefaultModel={async (p, model) => {
									if (p.list.builtIn) {
										throw new Error('Built-in provider defaults are source-owned.');
									}
									if (!p.list.enabled || p.list.state !== ArtifactState.Available) {
										throw new Error('Enable an available provider before choosing its default model.');
									}
									if (!model.list.enabled || model.list.state !== ArtifactState.Available) {
										throw new Error('Enable an available model before choosing it as the default.');
									}

									await run(() => modelManagementAPI.setProviderDefaultModel(p, model));
								}}
							/>
						))}

						{snapshot?.providers.length === 0 ? (
							<div className="bg-base-100 rounded-2xl p-6 text-sm opacity-70 shadow-sm">
								No model providers are installed.
							</div>
						) : null}
					</div>
				</ManagementPageContent>
			</div>

			{providerDialog ? (
				<ModelProviderAddEditModal
					key={`${providerDialog.mode}:${providerDialog.provider?.list.ref.rootID ?? 'new'}:${providerDialog.provider?.list.ref.artifactID ?? 'provider'}`}
					isOpen
					mode={providerDialog.mode}
					provider={providerDialog.provider}
					allProviders={snapshot?.providers ?? []}
					onClose={() => {
						setProviderDialog(null);
					}}
					onSubmit={submitProvider}
				/>
			) : null}

			{modelDialog ? (
				<ModelAddEditModal
					key={`${modelDialog.mode}:${modelDialog.model?.list.ref.rootID ?? 'new'}:${modelDialog.model?.list.ref.artifactID ?? 'model'}`}
					isOpen
					mode={modelDialog.mode}
					model={modelDialog.model}
					initialProvider={modelDialog.provider}
					allModels={snapshot?.models ?? []}
					onClose={() => {
						setModelDialog(null);
					}}
					onSubmit={submitModel}
				/>
			) : null}

			{credentialDialog ? (
				<ProviderCredentialModal
					isOpen
					provider={credentialDialog.provider}
					onClose={() => {
						setCredentialDialog(null);
					}}
					onSave={saveCredential}
				/>
			) : null}

			<DeleteConfirmationModal
				isOpen={deleteTarget !== null}
				onClose={() => {
					setDeleteTarget(null);
				}}
				onConfirm={confirmDelete}
				confirmButtonText="Delete"
				title={deleteTarget?.kind === 'provider' ? 'Delete Provider' : 'Delete Model'}
				message={
					deleteTarget?.kind === 'provider'
						? `Delete provider “${deleteTarget.provider.list.displayName}”? This action cannot be undone.`
						: `Delete model “${deleteTarget?.model.list.displayName ?? ''}”? This action cannot be undone.`
				}
			/>

			<ActionDeniedAlertModal
				isOpen={showDenied}
				onClose={() => {
					setShowDenied(false);
					setDeniedMessage('');
				}}
				message={deniedMessage}
			/>
		</PageFrame>
	);
}
