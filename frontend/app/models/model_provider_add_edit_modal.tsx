import type { ChangeEvent, SubmitEventHandler } from 'react';
import { useMemo, useState } from 'react';

import type { ModelAuthentication, ModelConnection, ModelProviderDocument } from '@/spec/model';
import { ModelAuthenticationMode as AuthenticationMode, ModelArtifactType } from '@/spec/model';

import type { ModelProviderManagementItem } from '@/apis/model_management';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';

import {
	formatJSON,
	parseAdapterParameters,
	parseCapabilities,
	parseOptionalJSONObject,
	parseOptionalLabels,
	parseOptionalReference,
	parseProviderDefaults,
} from '@/models/lib/document';
import { ModelProviderModalMode, ProviderCredentialAction } from '@/models/lib/management_types';

export interface ProviderEditorSubmission {
	document: ModelProviderDocument;
	enabled: boolean;
	credentialAction: ProviderCredentialAction;
	credential: string;
}

interface ProviderFormState {
	name: string;
	displayName: string;
	description: string;
	adapter: string;
	connectionJSON: string;
	authenticationJSON: string;
	defaultModelJSON: string;
	defaultsJSON: string;
	capabilitiesJSON: string;
	adapterParametersJSON: string;
	labelsJSON: string;
	enabled: boolean;
	credential: string;
	credentialAction: ProviderCredentialAction;
}

interface ModelProviderAddEditModalProps {
	isOpen: boolean;
	mode: ModelProviderModalMode;
	provider?: ModelProviderManagementItem;
	onClose: () => void;
	onSubmit: (submission: ProviderEditorSubmission) => Promise<void>;
}

function defaultAuthentication(): ModelAuthentication {
	return {
		mode: AuthenticationMode.APIKeyHeader,
		headerName: 'Authorization',
	};
}

function initialFormState(provider: ModelProviderManagementItem | undefined): ProviderFormState {
	const document = provider?.view.document;

	return {
		name: document?.name ?? '',
		displayName: document?.displayName ?? '',
		description: document?.description ?? '',
		adapter: document?.adapter ?? 'openai.responses',
		connectionJSON: formatJSON(document?.connection),
		authenticationJSON: formatJSON(document?.authentication ?? defaultAuthentication()),
		defaultModelJSON: formatJSON(document?.defaultModel),
		defaultsJSON: formatJSON(document?.defaults),
		capabilitiesJSON: formatJSON(document?.capabilities),
		adapterParametersJSON: formatJSON(document?.adapterParameters),
		labelsJSON: formatJSON(document?.labels),
		enabled: provider?.list.enabled ?? true,
		credential: '',
		credentialAction: ProviderCredentialAction.Unchanged,
	};
}

function buildDocument(state: ProviderFormState): ModelProviderDocument {
	const connection = parseOptionalJSONObject<ModelConnection>(state.connectionJSON, 'Connection');
	const authentication = parseOptionalJSONObject<ModelAuthentication>(state.authenticationJSON, 'Authentication');

	if (!state.name.trim()) {
		throw new Error('Provider name is required.');
	}
	if (!state.adapter.trim()) {
		throw new Error('Provider adapter is required.');
	}
	if (!authentication?.mode) {
		throw new Error('Authentication must include a mode.');
	}

	return {
		type: ModelArtifactType.Provider,
		name: state.name.trim(),
		displayName: state.displayName.trim() || undefined,
		description: state.description.trim() || undefined,
		labels: parseOptionalLabels(state.labelsJSON),
		adapter: state.adapter.trim(),
		connection,
		authentication,
		defaultModel: parseOptionalReference(state.defaultModelJSON),
		defaults: parseProviderDefaults(state.defaultsJSON),
		capabilities: parseCapabilities(state.capabilitiesJSON),
		adapterParameters: parseAdapterParameters(state.adapterParametersJSON),
	};
}

export function ModelProviderAddEditModal({
	isOpen,
	mode,
	provider,
	onClose,
	onSubmit,
}: ModelProviderAddEditModalProps) {
	const readOnly = mode === ModelProviderModalMode.View;
	const [form, setForm] = useState<ProviderFormState>(() => initialFormState(provider));
	const [error, setError] = useState('');
	const [submitting, setSubmitting] = useState(false);

	const title = useMemo(() => {
		switch (mode) {
			case ModelProviderModalMode.Add:
				return 'Add Model Provider';
			case ModelProviderModalMode.Edit:
				return 'Edit Model Provider';
			default:
				return 'View Model Provider';
		}
	}, [mode]);

	if (!isOpen) {
		return null;
	}

	const change = (name: keyof ProviderFormState) => (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
		const value =
			event.target instanceof HTMLInputElement && event.target.type === 'checkbox'
				? event.target.checked
				: event.target.value;

		setForm(
			current =>
				({
					...current,
					[name]: value,
				}) as ProviderFormState
		);
	};

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = async event => {
		event.preventDefault();
		if (readOnly) {
			onClose();
			return;
		}

		setSubmitting(true);
		setError('');

		try {
			await onSubmit({
				document: buildDocument(form),
				enabled: form.enabled,
				credentialAction: form.credentialAction,
				credential: form.credential,
			});
			onClose();
		} catch (submissionError) {
			setError(submissionError instanceof Error ? submissionError.message : 'Provider could not be saved.');
		} finally {
			setSubmitting(false);
		}
	};

	return (
		<ModalDialog isOpen={isOpen} onClose={onClose} blockCancel={!readOnly}>
			<div className="modal-box bg-base-200 max-h-[90vh] max-w-4xl overflow-y-auto rounded-2xl p-0">
				<ModalHeader title={title} onClose={onClose} />

				<form className="space-y-4 p-6" onSubmit={handleSubmit}>
					{error ? <div className="alert alert-error rounded-xl">{error}</div> : null}

					<label>
						<span className="text-sm">Provider name</span>
						<input
							className="input border"
							value={form.name}
							disabled={readOnly || submitting || mode !== ModelProviderModalMode.Add}
							onChange={change('name')}
						/>
					</label>

					<label>
						<span className="text-sm">Display name</span>
						<input
							className="input border"
							value={form.displayName}
							disabled={readOnly || submitting}
							onChange={change('displayName')}
						/>
					</label>

					<label>
						<span className="text-sm">Description</span>
						<textarea
							className="textarea border"
							value={form.description}
							disabled={readOnly || submitting}
							onChange={change('description')}
						/>
					</label>

					<label>
						<span className="text-sm">Adapter</span>
						<input
							className="input border font-mono"
							value={form.adapter}
							disabled={readOnly || submitting}
							onChange={change('adapter')}
							placeholder="openai.responses"
						/>
					</label>

					<label>
						<span className="text-sm">Connection JSON</span>
						<textarea
							className="textarea h-28 border font-mono text-xs"
							value={form.connectionJSON}
							disabled={readOnly || submitting}
							onChange={change('connectionJSON')}
							placeholder='{"origin":"https://api.example.com","path":"/v1/responses"}'
						/>
					</label>

					<label>
						<span className="text-sm">Authentication JSON</span>
						<textarea
							className="textarea h-24 border font-mono text-xs"
							value={form.authenticationJSON}
							disabled={readOnly || submitting}
							onChange={change('authenticationJSON')}
							placeholder='{"mode":"apiKeyHeader","headerName":"Authorization"}'
						/>
					</label>

					<label>
						<span className="text-sm">Default Model Reference JSON</span>
						<textarea
							className="textarea h-20 border font-mono text-xs"
							value={form.defaultModelJSON}
							disabled={readOnly || submitting}
							onChange={change('defaultModelJSON')}
							placeholder='{"name":"my-model"}'
						/>
					</label>

					<label>
						<span className="text-sm">Defaults JSON</span>
						<textarea
							className="textarea h-32 border font-mono text-xs"
							value={form.defaultsJSON}
							disabled={readOnly || submitting}
							onChange={change('defaultsJSON')}
							placeholder='{"timeoutMS":120000,"maxOutputTokens":4096}'
						/>
					</label>

					<label>
						<span className="text-sm">Capabilities JSON</span>
						<textarea
							className="textarea h-32 border font-mono text-xs"
							value={form.capabilitiesJSON}
							disabled={readOnly || submitting}
							onChange={change('capabilitiesJSON')}
							placeholder='{"modalitiesIn":["textIn"],"modalitiesOut":["textOut"]}'
						/>
					</label>

					<label>
						<span className="text-sm">Adapter Parameters JSON</span>
						<textarea
							className="textarea h-24 border font-mono text-xs"
							value={form.adapterParametersJSON}
							disabled={readOnly || submitting}
							onChange={change('adapterParametersJSON')}
						/>
					</label>

					<label>
						<span className="text-sm">Labels JSON</span>
						<textarea
							className="textarea h-20 border font-mono text-xs"
							value={form.labelsJSON}
							disabled={readOnly || submitting}
							onChange={change('labelsJSON')}
						/>
					</label>

					{!readOnly ? (
						<>
							<label className="label cursor-pointer justify-start gap-3">
								<input
									type="checkbox"
									className="toggle toggle-primary"
									checked={form.enabled}
									disabled={submitting}
									onChange={change('enabled')}
								/>
								<span>Enabled</span>
							</label>

							<label>
								<span className="text-sm">Credential action</span>
								<select
									className="select border"
									value={form.credentialAction}
									disabled={submitting}
									onChange={event => {
										setForm(current => ({
											...current,
											credentialAction: event.target.value as ProviderCredentialAction,
										}));
									}}
								>
									<option value={ProviderCredentialAction.Unchanged}>Leave credential unchanged</option>
									<option value={ProviderCredentialAction.Set}>Set credential</option>
									<option value={ProviderCredentialAction.Clear}>Remove credential</option>
								</select>
							</label>

							{form.credentialAction === ProviderCredentialAction.Set ? (
								<label>
									<span className="text-sm">Credential</span>
									<input
										type="password"
										autoComplete="new-password"
										className="input border"
										value={form.credential}
										disabled={submitting}
										onChange={change('credential')}
									/>
								</label>
							) : null}
						</>
					) : null}

					<ModalActions>
						<button type="button" className="btn" disabled={submitting} onClick={onClose}>
							{readOnly ? 'Close' : 'Cancel'}
						</button>

						{!readOnly ? (
							<button type="submit" className="btn btn-primary" disabled={submitting}>
								{submitting ? 'Saving...' : 'Save'}
							</button>
						) : null}
					</ModalActions>
				</form>
			</div>
		</ModalDialog>
	);
}
