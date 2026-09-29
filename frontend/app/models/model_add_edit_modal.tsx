import type { ChangeEvent, SubmitEventHandler } from 'react';
import { useMemo, useState } from 'react';

import type { ModelDocument } from '@/spec/model';
import { ModelArtifactType, ModelLookupScope } from '@/spec/model';

import type { ModelManagementItem, ModelProviderManagementItem } from '@/apis/model_management';

import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalHeader } from '@/components/modal/modal_header';

import {
	formatJSON,
	parseAdapterParameters,
	parseCapabilities,
	parseModelDefaults,
	parseOptionalLabels,
} from '@/models/lib/document';
import { ModelModalMode } from '@/models/lib/management_types';

export interface ModelEditorSubmission {
	document: ModelDocument;
	enabled: boolean;
}

interface ModelFormState {
	name: string;
	displayName: string;
	description: string;
	providerRefKey: string;
	providerModelID: string;
	defaultsJSON: string;
	capabilitiesJSON: string;
	adapterParametersJSON: string;
	labelsJSON: string;
	enabled: boolean;
}

interface ModelAddEditModalProps {
	isOpen: boolean;
	mode: ModelModalMode;
	model?: ModelManagementItem;
	providers: ModelProviderManagementItem[];
	initialProvider?: ModelProviderManagementItem;
	onClose: () => void;
	onSubmit: (submission: ModelEditorSubmission) => Promise<void>;
}

function providerKey(provider: ModelProviderManagementItem): string {
	return `${provider.list.ref.rootID}\u0000${provider.list.ref.artifactID}`;
}

function initialForm(
	model: ModelManagementItem | undefined,
	provider: ModelProviderManagementItem | undefined
): ModelFormState {
	const document = model?.view.document;

	return {
		name: document?.name ?? '',
		displayName: document?.displayName ?? '',
		description: document?.description ?? '',
		providerRefKey: provider ? providerKey(provider) : '',
		providerModelID: document?.providerModelID ?? '',
		defaultsJSON: formatJSON(document?.defaults),
		capabilitiesJSON: formatJSON(document?.capabilities),
		adapterParametersJSON: formatJSON(document?.adapterParameters),
		labelsJSON: formatJSON(document?.labels),
		enabled: model?.list.enabled ?? true,
	};
}

function buildDocument(state: ModelFormState, providers: ModelProviderManagementItem[]): ModelDocument {
	const provider = providers.find(item => providerKey(item) === state.providerRefKey);

	if (!provider) {
		throw new Error('A provider must be selected.');
	}
	if (!state.name.trim()) {
		throw new Error('Model name is required.');
	}
	if (!state.providerModelID.trim()) {
		throw new Error('Provider model ID is required.');
	}

	return {
		type: ModelArtifactType.Model,
		name: state.name.trim(),
		displayName: state.displayName.trim() || undefined,
		description: state.description.trim() || undefined,
		labels: parseOptionalLabels(state.labelsJSON),
		provider: {
			name: provider.view.document.name,
			...(provider.list.builtIn ? { scope: ModelLookupScope.Builtin } : {}),
		},
		providerModelID: state.providerModelID.trim(),
		defaults: parseModelDefaults(state.defaultsJSON),
		capabilities: parseCapabilities(state.capabilitiesJSON),
		adapterParameters: parseAdapterParameters(state.adapterParametersJSON),
	};
}

export function ModelAddEditModal({
	isOpen,
	mode,
	model,
	providers,
	initialProvider,
	onClose,
	onSubmit,
}: ModelAddEditModalProps) {
	const readOnly = mode === ModelModalMode.View;
	const initialProviderAttr = model?.provider ?? initialProvider;
	const [form, setForm] = useState<ModelFormState>(() => initialForm(model, initialProviderAttr));
	const [error, setError] = useState('');
	const [submitting, setSubmitting] = useState(false);

	const title = useMemo(() => {
		switch (mode) {
			case ModelModalMode.Add:
				return 'Add Model';
			case ModelModalMode.Edit:
				return 'Edit Model';
			default:
				return 'View Model';
		}
	}, [mode]);

	if (!isOpen) {
		return null;
	}

	const change = (name: keyof ModelFormState) => (event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
		const value =
			event.target instanceof HTMLInputElement && event.target.type === 'checkbox'
				? event.target.checked
				: event.target.value;

		setForm(
			current =>
				({
					...current,
					[name]: value,
				}) as ModelFormState
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
				document: buildDocument(form, providers),
				enabled: form.enabled,
			});
			onClose();
		} catch (submissionError) {
			setError(submissionError instanceof Error ? submissionError.message : 'Model could not be saved.');
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
						<span className="text-sm">Model name</span>
						<input
							className="input border"
							value={form.name}
							disabled={readOnly || submitting || mode !== ModelModalMode.Add}
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
						<span className="text-sm">Provider</span>
						<select
							className="select border"
							value={form.providerRefKey}
							disabled={readOnly || submitting}
							onChange={event => {
								setForm(current => ({
									...current,
									providerRefKey: event.target.value,
								}));
							}}
						>
							<option value="">Select provider</option>
							{providers.map(provider => (
								<option key={providerKey(provider)} value={providerKey(provider)}>
									{provider.list.displayName} ({provider.view.document.name})
								</option>
							))}
						</select>
					</label>

					<label>
						<span className="text-sm">Provider Model ID</span>
						<input
							className="input border font-mono"
							value={form.providerModelID}
							disabled={readOnly || submitting}
							onChange={change('providerModelID')}
							placeholder="gpt-5.4"
						/>
					</label>

					<label>
						<span className="text-sm">Defaults JSON</span>
						<textarea
							className="textarea h-32 border font-mono text-xs"
							value={form.defaultsJSON}
							disabled={readOnly || submitting}
							onChange={change('defaultsJSON')}
							placeholder='{"temperature":0.2,"maxOutputTokens":4096}'
						/>
					</label>

					<label>
						<span className="text-sm">Capabilities JSON</span>
						<textarea
							className="textarea h-32 border font-mono text-xs"
							value={form.capabilitiesJSON}
							disabled={readOnly || submitting}
							onChange={change('capabilitiesJSON')}
						/>
					</label>

					<label>
						<span>Adapter Parameters JSON</span>
						<textarea
							className="textarea h-24 border font-mono text-xs"
							value={form.adapterParametersJSON}
							disabled={readOnly || submitting}
							onChange={change('adapterParametersJSON')}
						/>
					</label>

					<label>
						<span>Labels JSON</span>
						<textarea
							className="textarea h-20 border font-mono text-xs"
							value={form.labelsJSON}
							disabled={readOnly || submitting}
							onChange={change('labelsJSON')}
						/>
					</label>

					{!readOnly ? (
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
