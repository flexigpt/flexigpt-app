import { useCallback, useEffect, useMemo, useState } from 'react';
import { FiPlus, FiRefreshCw } from 'react-icons/fi';

import type {
	ModelManagementItem,
	ModelManagementSnapshot,
	ModelProviderManagementItem,
} from '@/apis/model_management';
import { modelManagementAPI } from '@/apis/baseapi';

import { Loader } from '@/components/loader';
import { ManagementPageContent } from '@/components/managementui/management_page_content';
import { ManagementPageHeader } from '@/components/managementui/management_page_header';
import { PageFrame } from '@/components/page_frame';

import type { ModelEditorSubmission } from '@/models/model_add_edit_modal';
import type { ProviderEditorSubmission } from '@/models/model_provider_add_edit_modal';
import { ModelModalMode, ModelProviderModalMode, ProviderCredentialAction } from '@/models/lib/management_types';
import { ModelAddEditModal } from '@/models/model_add_edit_modal';
import { ModelProviderAddEditModal } from '@/models/model_provider_add_edit_modal';
import { ModelProviderCard } from '@/models/model_provider_card';

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

function providerKey(provider: ModelProviderManagementItem): string {
	return `${provider.list.ref.rootID}\u0000${provider.list.ref.artifactID}`;
}

function confirmation(message: string): boolean {
	// oxlint-disable-next-line no-alert
	return window.confirm(message);
}

// oxlint-disable-next-line no-restricted-exports
export default function ModelsPage() {
	const [snapshot, setSnapshot] = useState<ModelManagementSnapshot | null>(null);
	const [loading, setLoading] = useState(true);
	const [busy, setBusy] = useState(false);
	const [error, setError] = useState('');

	const [providerDialog, setProviderDialog] = useState<ProviderDialogState | null>(null);
	const [modelDialog, setModelDialog] = useState<ModelDialogState | null>(null);
	const [credentialDialog, setCredentialDialog] = useState<CredentialDialogState | null>(null);
	const [credential, setCredential] = useState('');

	const reload = useCallback(async () => {
		setLoading(true);
		setError('');

		try {
			setSnapshot(await modelManagementAPI.loadSnapshot());
		} catch (loadError) {
			setError(loadError instanceof Error ? loadError.message : 'Models could not be loaded.');
		} finally {
			setLoading(false);
		}
	}, []);

	useEffect(() => {
		// oxlint-disable-next-line react/set-state-in-effect
		void reload();
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
			setBusy(true);
			setError('');

			try {
				await operation();
				await reload();
			} catch (operationError) {
				setError(operationError instanceof Error ? operationError.message : 'Model operation failed.');
			} finally {
				setBusy(false);
			}
		},
		[reload]
	);

	const submitProvider = async (submission: ProviderEditorSubmission) => {
		const provider = providerDialog?.provider;

		await run(async () => {
			if (provider) {
				await modelManagementAPI.replaceProvider({
					provider: provider.list.ref,
					expectedArtifactRevision: provider.list.revision,
					document: submission.document,
					enabled: submission.enabled,
				});
			} else {
				await modelManagementAPI.createProvider({
					rootID: '',
					document: submission.document,
					enabled: submission.enabled,
				});
			}

			if (provider && submission.credentialAction !== ProviderCredentialAction.Unchanged) {
				await modelManagementAPI.setProviderCredential(
					provider.list.ref,
					provider.list.runtimeOverlayRevision ?? 0,
					submission.credentialAction === ProviderCredentialAction.Clear ? '' : submission.credential
				);
			}
		});
	};

	const submitModel = async (submission: ModelEditorSubmission) => {
		const model = modelDialog?.model;
		const provider = modelDialog?.provider;

		if (!provider) {
			throw new Error('A provider must be selected.');
		}

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
		});
	};

	const submitCredential = async () => {
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

		setCredential('');
		setCredentialDialog(null);
	};

	const providerDefault = async (provider: ModelProviderManagementItem, model: ModelManagementItem) => {
		await run(async () => {
			await modelManagementAPI.setProviderDefaultModel(provider, model);
		});
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
								disabled={loading || busy}
								onClick={() => void reload()}
							>
								<FiRefreshCw size={16} />
								Refresh
							</button>

							<button
								type="button"
								className="btn btn-sm btn-primary rounded-xl"
								disabled={busy}
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

					<div className="space-y-4">
						{snapshot?.providers.map(provider => (
							<ModelProviderCard
								key={providerKey(provider)}
								provider={provider}
								models={modelsByProvider.get(providerKey(provider)) ?? []}
								busy={busy}
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
								onToggleProvider={p => {
									void run(() => modelManagementAPI.setProviderEnabled(p.list.ref, p.list.revision, !p.list.enabled));
								}}
								onDeleteProvider={p => {
									if (
										!confirmation(
											`Delete provider "${p.list.displayName}"? Linked Models remain stored but become unavailable until a Provider is restored.`
										)
									) {
										return;
									}

									void run(() => modelManagementAPI.deleteProvider(p.list.ref, p.list.revision));
								}}
								onCredential={p => {
									setCredential('');
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
								onToggleModel={model => {
									void run(() =>
										modelManagementAPI.setModelEnabled(model.list.ref, model.list.revision, !model.list.enabled)
									);
								}}
								onDeleteModel={model => {
									if (!confirmation(`Delete model "${model.list.displayName}"?`)) {
										return;
									}

									void run(() => modelManagementAPI.deleteModel(model.list.ref, model.list.revision));
								}}
								onSetDefaultModel={providerDefault}
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

			<ModelProviderAddEditModal
				isOpen={providerDialog !== null}
				mode={providerDialog?.mode ?? ModelProviderModalMode.Add}
				provider={providerDialog?.provider}
				onClose={() => {
					setProviderDialog(null);
				}}
				onSubmit={submitProvider}
			/>

			<ModelAddEditModal
				isOpen={modelDialog !== null}
				mode={modelDialog?.mode ?? ModelModalMode.Add}
				model={modelDialog?.model}
				providers={snapshot?.providers ?? []}
				initialProvider={modelDialog?.provider}
				onClose={() => {
					setModelDialog(null);
				}}
				onSubmit={submitModel}
			/>

			{credentialDialog ? (
				<div className="modal modal-open">
					<div className="modal-box bg-base-200 rounded-2xl">
						<h3 className="text-lg font-semibold">Credential for {credentialDialog.provider.list.displayName}</h3>

						<p className="mt-2 text-sm opacity-70">Leave this blank and save to remove the credential.</p>

						<input
							type="password"
							autoComplete="new-password"
							className="input mt-4 w-full border"
							value={credential}
							disabled={busy}
							onChange={event => {
								setCredential(event.target.value);
							}}
						/>

						<div className="modal-action">
							<button
								type="button"
								className="btn"
								disabled={busy}
								onClick={() => {
									setCredential('');
									setCredentialDialog(null);
								}}
							>
								Cancel
							</button>

							<button type="button" className="btn btn-primary" disabled={busy} onClick={() => void submitCredential()}>
								Save Credential
							</button>
						</div>
					</div>
				</div>
			) : null}
		</PageFrame>
	);
}
