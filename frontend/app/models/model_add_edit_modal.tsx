import type { SubmitEventHandler } from 'react';
import { useMemo, useState } from 'react';
import { FiAlertCircle } from 'react-icons/fi';

import type { ModelCapabilities, ModelDocument } from '@/spec/model';
import { ModelArtifactType, ModelLookupScope } from '@/spec/model';

import { getErrorMessage } from '@/lib/error_utils';

import { useModalDialogController } from '@/hooks/use_dialog_controller';

import type { ModelManagementItem, ModelProviderManagementItem } from '@/apis/model_management';

import { Dropdown } from '@/components/dropdown';
import { MANAGEMENT_MODAL_FORM_CLASS } from '@/components/managementui/management_class_consts';
import { ModalActions } from '@/components/modal/modal_actions';
import { ModalDialog } from '@/components/modal/modal_dialog';
import { ModalField } from '@/components/modal/modal_field';
import { ModalHeader } from '@/components/modal/modal_header';
import { ModalSection } from '@/components/modal/modal_section';
import { ReadOnlyValue } from '@/components/read_only_value';

import type { ModelRuntimeDefaultsErrors, ModelRuntimeDefaultsForm } from '@/models/lib/model_runtime_defaults';
import { mergeModelCapabilities } from '@/models/lib/capabilities';
import { displayModelName, makeUniqueLogicalName } from '@/models/lib/management_form_utils';
import { ModelModalMode } from '@/models/lib/management_types';
import {
	buildModelDefaultsFromRuntimeForm,
	createFallbackModelRuntimeDefaultsForm,
	createModelRuntimeDefaultsForm,
	validateModelRuntimeDefaultsForm,
} from '@/models/lib/model_runtime_defaults';
import { getProviderSDKType } from '@/models/lib/provider_sdk';
import { ModelRuntimeDefaultsFields } from '@/models/model_runtime_defaults_fields';

const COPY_SOURCE_NONE = '__copy_source_none__';

interface ModelFormState {
	displayName: string;
	providerModelID: string;
	enabled: boolean;
	runtime: ModelRuntimeDefaultsForm;
}

interface ModelFormErrors {
	displayName?: string;
	providerModelID?: string;
	form?: string;
	runtime: ModelRuntimeDefaultsErrors;
}

export interface ModelEditorSubmission {
	document: ModelDocument;
	enabled: boolean;
}

interface ModelAddEditModalProps {
	isOpen: boolean;
	mode: ModelModalMode;
	model?: ModelManagementItem;
	initialProvider?: ModelProviderManagementItem;
	allModels?: ModelManagementItem[];
	onClose: () => void;
	onSubmit: (submission: ModelEditorSubmission) => Promise<void>;
}

function safeCapabilities(
	provider: ModelProviderManagementItem,
	model: ModelManagementItem | undefined
): ModelCapabilities | undefined {
	try {
		return mergeModelCapabilities(provider.view.document.capabilities, model?.view.document.capabilities);
	} catch {
		return undefined;
	}
}

function safeInitialRuntime(
	provider: ModelProviderManagementItem,
	model: ModelManagementItem | undefined
): ModelRuntimeDefaultsForm {
	try {
		const capabilities = safeCapabilities(provider, model);
		return createModelRuntimeDefaultsForm(
			model?.view.document.defaults,
			getProviderSDKType(provider.view.document.adapter),
			capabilities,
			!model
		);
	} catch {
		return createFallbackModelRuntimeDefaultsForm();
	}
}

function initialForm(model: ModelManagementItem | undefined, provider: ModelProviderManagementItem): ModelFormState {
	return {
		displayName: model?.view.document.displayName ?? '',
		providerModelID: model?.view.document.providerModelID ?? '',
		enabled: model?.list.enabled ?? true,
		runtime: safeInitialRuntime(provider, model),
	};
}

function validateForm(
	form: ModelFormState,
	mode: ModelModalMode,
	provider: ModelProviderManagementItem,
	model: ModelManagementItem | undefined
): ModelFormErrors {
	const errors: ModelFormErrors = {
		runtime: {},
	};

	try {
		if (!form.displayName.trim()) {
			errors.displayName = 'Display name is required.';
		}

		if (!form.providerModelID.trim()) {
			errors.providerModelID = 'Model name is required.';
		}

		const capabilities = safeCapabilities(provider, model);
		errors.runtime = validateModelRuntimeDefaultsForm(
			form.runtime,
			getProviderSDKType(provider.view.document.adapter),
			capabilities,
			mode === ModelModalMode.Add
		);
	} catch (error) {
		errors.form = getErrorMessage(error, 'Model settings could not be validated.');
	}

	return errors;
}

function flattenErrors(errors: ModelFormErrors): string[] {
	return [errors.displayName, errors.providerModelID, errors.form, ...Object.values(errors.runtime)].filter(
		(value): value is string => Boolean(value)
	);
}

function buildDocument(
	form: ModelFormState,
	model: ModelManagementItem | undefined,
	provider: ModelProviderManagementItem,
	existingLogicalNames: string[]
): ModelDocument {
	const existingDocument = model?.view.document;
	const capabilities = safeCapabilities(provider, model);
	const logicalName =
		existingDocument?.name ??
		makeUniqueLogicalName(form.displayName || form.providerModelID, existingLogicalNames, 'model');

	return {
		...existingDocument,
		type: ModelArtifactType.Model,
		name: logicalName,
		displayName: form.displayName.trim(),
		provider: {
			name: provider.view.document.name,
			...(provider.list.builtIn ? { scope: ModelLookupScope.Builtin } : {}),
		},
		providerModelID: form.providerModelID.trim(),
		defaults: buildModelDefaultsFromRuntimeForm(
			form.runtime,
			getProviderSDKType(provider.view.document.adapter),
			capabilities
		),
	};
}

interface ModelAddEditModalContentProps {
	mode: ModelModalMode;
	model?: ModelManagementItem;
	provider: ModelProviderManagementItem;
	allModels: ModelManagementItem[];
	onSubmit: (submission: ModelEditorSubmission) => Promise<void>;
}

function ModelAddEditModalContent({ mode, model, provider, allModels, onSubmit }: ModelAddEditModalContentProps) {
	const readOnly = mode === ModelModalMode.View;
	const existingLogicalNames = useMemo(() => allModels.map(item => item.view.document.name), [allModels]);
	const [form, setForm] = useState<ModelFormState>(() => initialForm(model, provider));
	const [errors, setErrors] = useState<ModelFormErrors>({
		runtime: {},
	});
	const [copySourceKey, setCopySourceKey] = useState(COPY_SOURCE_NONE);
	const [submitError, setSubmitError] = useState('');
	const [isSubmitting, setIsSubmitting] = useState(false);
	const { requestClose, unmountingRef } = useModalDialogController();

	const capabilities = useMemo(() => safeCapabilities(provider, model), [model, provider]);
	const providerSDKType = useMemo(
		() => getProviderSDKType(provider.view.document.adapter),
		[provider.view.document.adapter]
	);
	const validation = useMemo(() => validateForm(form, mode, provider, model), [form, mode, model, provider]);

	const copySourceItems = useMemo(() => {
		const items: Record<string, { isEnabled: boolean; displayName: string }> = {
			[COPY_SOURCE_NONE]: {
				isEnabled: true,
				displayName: 'Choose a model',
			},
		};

		for (const item of allModels) {
			items[`${item.list.ref.rootID}:${item.list.ref.artifactID}`] = {
				isEnabled: true,
				displayName: `${item.provider?.list.displayName || 'Provider'} / ${displayModelName(item.list.displayName)}`,
			};
		}

		return items;
	}, [allModels]);

	const copySourceKeys = useMemo(() => Object.keys(copySourceItems), [copySourceItems]);

	const title = mode === ModelModalMode.Add ? 'Add Model' : mode === ModelModalMode.Edit ? 'Edit Model' : 'View Model';

	const updateForm = (next: ModelFormState) => {
		setForm(next);
		setErrors(validateForm(next, mode, provider, model));
	};

	const applyCopy = (sourceKey: string) => {
		if (sourceKey === COPY_SOURCE_NONE) {
			return;
		}

		try {
			const source = allModels.find(item => `${item.list.ref.rootID}:${item.list.ref.artifactID}` === sourceKey);
			if (!source) {
				setSubmitError('The selected model is no longer available.');
				return;
			}

			updateForm({
				displayName: `${displayModelName(source.list.displayName)} Copy`,
				providerModelID: source.view.document.providerModelID,
				enabled: true,
				runtime: (() => {
					try {
						return createModelRuntimeDefaultsForm(source.view.document.defaults, providerSDKType, capabilities, true);
					} catch {
						return createFallbackModelRuntimeDefaultsForm();
					}
				})(),
			});
		} catch (error) {
			setSubmitError(getErrorMessage(error, 'Model settings could not be copied.'));
		}
	};

	const handleSubmit: SubmitEventHandler<HTMLFormElement> = event => {
		event.preventDefault();
		event.stopPropagation();

		if (readOnly) {
			requestClose();
			return;
		}

		setErrors(validation);
		if (flattenErrors(validation).length > 0) {
			return;
		}

		let document: ModelDocument;
		try {
			document = buildDocument(form, model, provider, existingLogicalNames);
		} catch (error) {
			setSubmitError(getErrorMessage(error, 'Model settings could not be prepared.'));
			return;
		}

		setIsSubmitting(true);
		setSubmitError('');

		void (async () => {
			try {
				await onSubmit({
					document,
					enabled: form.enabled,
				});
				requestClose(true);
			} catch (error) {
				if (!unmountingRef.current) {
					setSubmitError(getErrorMessage(error, 'Model could not be saved.'));
				}
			} finally {
				if (!unmountingRef.current) {
					setIsSubmitting(false);
				}
			}
		})();
	};

	return (
		<div className="modal-box bg-base-200 flex max-h-[calc(100dvh-1rem)] w-[calc(100%-1rem)] max-w-4xl flex-col overflow-hidden rounded-2xl p-0">
			<ModalHeader
				title={title}
				description={`Provider: ${provider.list.displayName || 'Provider'}`}
				onClose={() => {
					requestClose();
				}}
				closeDisabled={isSubmitting}
			/>

			<form noValidate onSubmit={handleSubmit} className="flex min-h-0 flex-1 flex-col" aria-busy={isSubmitting}>
				<div className={`app-scrollbar-thin min-h-0 flex-1 overflow-y-auto p-4 sm:p-6 ${MANAGEMENT_MODAL_FORM_CLASS}`}>
					{submitError ? (
						<div className="alert alert-error rounded-xl text-sm" role="alert">
							<div className="flex items-center gap-2">
								<FiAlertCircle size={15} />
								<span>{submitError}</span>
							</div>
						</div>
					) : null}

					{flattenErrors(errors).length > 0 ? (
						<div className="alert alert-error rounded-xl text-sm" role="alert">
							<div className="flex items-start gap-2">
								<FiAlertCircle size={15} className="mt-0.5 shrink-0" />
								<ul className="list-disc pl-4">
									{flattenErrors(errors).map((message, index) => (
										<li key={`${message}-${index}`}>{message}</li>
									))}
								</ul>
							</div>
						</div>
					) : null}

					{mode === ModelModalMode.Add && allModels.length > 0 ? (
						<ModalSection
							title="Copy an Existing Model"
							description="Copies user-facing model settings. The selected provider remains unchanged."
						>
							<ModalField label="Copy From">
								<Dropdown<string>
									dropdownItems={copySourceItems}
									orderedKeys={copySourceKeys}
									selectedKey={copySourceKey}
									onChange={key => {
										setCopySourceKey(key);
										applyCopy(key);
									}}
									filterDisabled={false}
									title="Select a model to copy"
									getDisplayName={key => copySourceItems[key]?.displayName ?? 'Choose a model'}
									disabled={isSubmitting}
								/>
							</ModalField>
						</ModalSection>
					) : null}

					<ModalSection title="Identity" description="Set the display label and the provider-facing model name.">
						<ModalField label="Display Name" htmlFor="model-display-name" required error={errors.displayName}>
							{readOnly ? (
								<ReadOnlyValue value={form.displayName || '—'} />
							) : (
								<input
									id="model-display-name"
									type="text"
									className={`input w-full rounded-xl ${errors.displayName ? 'input-error' : ''}`}
									value={form.displayName}
									onChange={event => {
										updateForm({
											...form,
											displayName: event.target.value,
										});
									}}
									autoComplete="off"
									spellCheck="false"
									autoFocus
									disabled={isSubmitting}
								/>
							)}
						</ModalField>

						<ModalField label="Model" htmlFor="provider-model-name" required error={errors.providerModelID}>
							{readOnly ? (
								<ReadOnlyValue value={form.providerModelID || '—'} />
							) : (
								<input
									id="provider-model-name"
									type="text"
									className={`input w-full rounded-xl ${errors.providerModelID ? 'input-error' : ''}`}
									value={form.providerModelID}
									onChange={event => {
										updateForm({
											...form,
											providerModelID: event.target.value,
										});
									}}
									placeholder="gpt-5.4"
									autoComplete="off"
									spellCheck="false"
									disabled={isSubmitting}
								/>
							)}
						</ModalField>
					</ModalSection>

					<ModelRuntimeDefaultsFields
						value={form.runtime}
						errors={errors.runtime}
						enabled={form.enabled}
						readOnly={readOnly}
						disabled={isSubmitting}
						providerSDKType={providerSDKType}
						capabilities={capabilities}
						onEnabledChange={enabled => {
							updateForm({
								...form,
								enabled,
							});
						}}
						onChange={patch => {
							updateForm({
								...form,
								runtime: {
									...form.runtime,
									...patch,
								},
							});
						}}
					/>
				</div>

				<ModalActions>
					<button
						type="button"
						className="btn bg-base-300 rounded-xl"
						disabled={isSubmitting}
						onClick={() => {
							requestClose();
						}}
					>
						{readOnly ? 'Close' : 'Cancel'}
					</button>

					{!readOnly ? (
						<button
							type="submit"
							className="btn btn-primary rounded-xl"
							disabled={isSubmitting || flattenErrors(validation).length > 0}
						>
							{isSubmitting ? 'Saving...' : mode === ModelModalMode.Add ? 'Add Model' : 'Save Changes'}
						</button>
					) : null}
				</ModalActions>
			</form>
		</div>
	);
}

function MissingProviderModal({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) {
	return (
		<ModalDialog isOpen={isOpen} onClose={onClose}>
			<div className="modal-box bg-base-200 w-[calc(100%-1rem)] max-w-lg rounded-2xl p-0">
				<ModalHeader
					title="Model Unavailable"
					description="The provider for this model is no longer available."
					onClose={onClose}
				/>
				<div className="p-6 text-sm">
					Choose an available provider before adding a model, or reload the page if this model was changed elsewhere.
				</div>
				<ModalActions>
					<button type="button" className="btn bg-base-300 rounded-xl" onClick={onClose}>
						Close
					</button>
				</ModalActions>
			</div>
		</ModalDialog>
	);
}

export function ModelAddEditModal({
	isOpen,
	mode,
	model,
	initialProvider,
	allModels = [],
	onClose,
	onSubmit,
}: ModelAddEditModalProps) {
	const provider = model?.provider ?? initialProvider;

	if (!isOpen) {
		return null;
	}

	if (!provider) {
		return <MissingProviderModal isOpen={isOpen} onClose={onClose} />;
	}

	const key =
		mode === ModelModalMode.Add
			? `add-model:${provider.list.ref.rootID}:${provider.list.ref.artifactID}`
			: `${mode}:${model?.list.ref.rootID ?? 'model'}:${
					model?.list.ref.artifactID ?? 'unknown'
				}:${model?.list.revision ?? 0}`;

	return (
		<ModalDialog isOpen={isOpen} onClose={onClose} blockCancel={mode !== ModelModalMode.View}>
			<ModelAddEditModalContent
				key={key}
				mode={mode}
				model={model}
				provider={provider}
				allModels={allModels}
				onSubmit={onSubmit}
			/>
		</ModalDialog>
	);
}
